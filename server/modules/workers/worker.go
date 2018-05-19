package workers

import (
	"archive/tar"
	"bufio"
	"crypto/rand"
	"encoding/base64"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/gedorinku/koneko-online-judge/server/logger"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
)

type Worker struct {
	ID          string
	cli         *client.Client
	TimeLimit   time.Duration
	MemoryLimit int64
	separator   string
}

type ExecStatus int

const (
	StatusFinished            ExecStatus = 0
	StatusTimeLimitExceeded   ExecStatus = 1
	StatusMemoryLimitExceeded ExecStatus = 2
	StatusRuntimeError        ExecStatus = 3
	StatusUnknownError        ExecStatus = 4
	StatusOutputLimitExceeded ExecStatus = 5
	outputLimit                          = 10 * 1024 * 1024
	errorOutputLimit                     = 512
	Workspace                            = "/tmp/koj-workspace/"
	errorString                          = "runtime_error"
	exitCodeFile                         = "exit.txt"
)

var (
	ErrTimeTextParse = errors.New("time.txtの内容がパースできません。")
	errRuntime       = errors.New("runtime error")
)

type ExecResult struct {
	Status      ExecStatus
	ExecTime    time.Duration
	MemoryUsage int64
	Stdout      string
	Stderr      string
}

func NewWorker(img string, timeLimit time.Duration, memoryLimit int64, cmd []string) (*Worker, error) {
	ctx := context.Background()
	cli, err := client.NewEnvClient()
	if err != nil {
		return nil, err
	}

	sp, err := newSeparator()
	if err != nil {
		return nil, err
	}

	outputCmd := "echo -n " + sp + "$?" + sp
	runCmd := []string{
		"/usr/bin/time", "-f", sp + "%e %M",
		"timeout", strconv.FormatFloat(timeLimit.Seconds()+0.01, 'f', 4, 64),
		"/usr/bin/sudo", "-u", "nobody", "--",
		"/bin/bash", "-c", strings.Join(cmd, " ") + ";" + outputCmd,
	}
	logger.AppLog.Debugf("run command", runCmd)

	cfg := &container.Config{
		Image:        img,
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		OpenStdin:    true,
		StdinOnce:    true,
		Tty:          false,
		WorkingDir:   Workspace,
		Cmd:          runCmd,
	}
	hcfg := &container.HostConfig{
		Resources: container.Resources{
			CpusetCpus: "0",
			PidsLimit:  15,
			Memory:     memoryLimit + 10*1024*1024,
		},
		NetworkMode: "none",
	}

	res, err := cli.ContainerCreate(ctx, cfg, hcfg, &network.NetworkingConfig{}, "")
	if err != nil {
		logger.AppLog.Errorf("error %v %+v", img, err)
		return nil, err
	}

	w := &Worker{
		ID:          res.ID,
		TimeLimit:   timeLimit,
		MemoryLimit: memoryLimit,
		cli:         cli,
		separator:   sp,
	}
	return w, nil
}

func newSeparator() (string, error) {
	s := make([]byte, 16)
	_, err := rand.Read(s)
	if err != nil {
		logger.AppLog.Error(err)
		return "", err
	}
	return base64.URLEncoding.EncodeToString(s), nil
}

func (w Worker) Run(input string) (*ExecResult, error) {
	start := time.Now()

	ctx := context.Background()
	opt := types.ContainerAttachOptions{
		Stream: true,
		Stdin:  true,
		Stdout: true,
		Stderr: true,
	}
	hijacked, err := w.cli.ContainerAttach(ctx, w.ID, opt)
	if err != nil {
		logger.AppLog.Errorf("error  %+v", err)
		return nil, err
	}
	defer hijacked.Close()

	{
		end := time.Now()
		logger.AppLog.Debugf("attach%f秒\n", (end.Sub(start)).Seconds())
	}

	startErrChan := make(chan error)
	go func() {
		err = w.cli.ContainerStart(ctx, w.ID, types.ContainerStartOptions{})
		startErrChan <- err
	}()

	{
		end := time.Now()
		logger.AppLog.Debugf("start%f秒\n", (end.Sub(start)).Seconds())
	}

	stdout, err := ioutil.TempFile("", "stdout")
	if err != nil {
		logger.AppLog.Errorf("error %+v", err)
		return nil, err
	}
	defer removeTempFile(stdout)

	stderr, err := ioutil.TempFile("", "stderr")
	if err != nil {
		logger.AppLog.Errorf("error %+v", err)
		return nil, err
	}
	defer removeTempFile(stderr)

	{
		end := time.Now()
		logger.AppLog.Debugf("temp create%f秒\n", (end.Sub(start)).Seconds())
	}

	streamErrChan := make(chan error)
	go func() {
		_, err := hijacked.Conn.Write([]byte(input))
		if err != nil {
			streamErrChan <- err
			return
		}

		hijacked.CloseWrite()

		_, err = stdcopy.StdCopy(stdout, stderr, hijacked.Reader)
		streamErrChan <- err
	}()

	if err := <-startErrChan; err != nil {
		logger.AppLog.Errorf("error %+v", err)
		return nil, err
	}
	if err := <-streamErrChan; err != nil {
		logger.AppLog.Errorf("error %+v", err)
		return nil, err
	}

	{
		end := time.Now()
		logger.AppLog.Debugf("wait chan%f秒\n", (end.Sub(start)).Seconds())
	}

	_, err = stdout.Seek(0, 0)
	if err != nil {
		logger.AppLog.Errorf("error %+v", err)
		return nil, err
	}
	rawStdout, err := w.parseStdout(stdout)
	if err != nil {
		logger.AppLog.Error(err)
		return nil, err
	}
	logger.AppLog.Debug("stdoutlen", len(rawStdout))
	if len(rawStdout) == 1 {
		rawStdout = append(rawStdout, "255")
	}
	// TODO ここでOLEの処理
	{
		end := time.Now()
		logger.AppLog.Debugf("stdout%f秒\n", (end.Sub(start)).Seconds())
	}

	_, err = stderr.Seek(0, 0)
	if err != nil {
		logger.AppLog.Errorf("error %+v", err)
		return nil, err
	}
	rawStderr, err := w.parseStdout(stderr)
	if err != nil {
		logger.AppLog.Error(err)
		return nil, err
	}
	logger.AppLog.Debug("stderrlen", len(rawStderr))
	{
		end := time.Now()
		logger.AppLog.Debugf("stderr%f秒\n", (end.Sub(start)).Seconds())
	}

	exitCode := rawStdout[1]
	logger.AppLog.Debug(string(exitCode))

	timeText := rawStderr[1]
	logger.AppLog.Debug(string(timeText))

	timeMillis, memoryUsage, err := parseTimeText(string(timeText))
	if err != nil {
		logger.AppLog.Errorf("error: %v %+v", string(timeText), err)
		return nil, err
	}
	memoryUsage *= 1024
	end := time.Now()
	logger.AppLog.Debugf("終わり%f秒\n", (end.Sub(start)).Seconds())

	var status ExecStatus
	runtimeErr := checkRuntimeError(exitCode)
	switch runtimeErr {
	case errRuntime:
		status = StatusRuntimeError
	case nil:
		switch {
		case int64(w.TimeLimit.Seconds()*1000.0+0.0001) <= timeMillis:
			status = StatusTimeLimitExceeded
			logger.AppLog.Debugf("time limit(%v) exceeded:%v", w.TimeLimit, timeMillis)
		case w.MemoryLimit <= memoryUsage:
			status = StatusMemoryLimitExceeded
			logger.AppLog.Debugf("memory limit(%v) exceeded:%v", w.MemoryLimit, memoryUsage)
		default:
			status = StatusFinished
		}
	default:
		return nil, runtimeErr
	}

	return &ExecResult{
		Status:      status,
		ExecTime:    time.Duration(timeMillis) * time.Millisecond,
		MemoryUsage: memoryUsage,
		Stdout:      rawStdout[0],
		Stderr:      rawStderr[0],
	}, nil
}

func (w Worker) CopyTo(filename string, dist *Worker) error {
	const limit = 10 * 1024 * 1024
	name := Workspace + filename
	content, err := w.getFromContainer(name, limit)
	if err != nil && err != io.EOF {
		logger.AppLog.Errorf("%+v", err)
		return err
	}
	return dist.CopyContentToContainer(content, filename)
}

func (w Worker) CopyContentToContainer(content []byte, name string) error {
	createTempDir()
	f, err := os.Create(Workspace + name)
	if err != nil {
		logger.AppLog.Errorf("could not create temp file %+v", err)
		return err
	}
	f.Write(content)
	f.Close()
	err = os.Chmod(f.Name(), 0777)
	if err != nil {
		logger.AppLog.Errorf("could not change temp file mode %+v", err)
		return err
	}
	defer removeTempFile(f)

	srcInfo, err := archive.CopyInfoSourcePath(f.Name(), false)
	if err != nil {
		logger.AppLog.Errorf("", err)
		return err
	}

	srcArchive, err := archive.TarResource(srcInfo)
	if err != nil {
		logger.AppLog.Errorf("%+v", err)
		return err
	}
	defer srcArchive.Close()

	dstInfo := archive.CopyInfo{Path: Workspace + name}
	dstDir, preparedArchive, err := archive.PrepareArchiveCopy(srcArchive, srcInfo, dstInfo)
	if err != nil {
		logger.AppLog.Errorf("%+v", err)
		return err
	}

	ctx := context.Background()
	return w.cli.CopyToContainer(ctx, w.ID, dstDir, preparedArchive, types.CopyToContainerOptions{})
}

func (w Worker) Remove() error {
	ctx := context.Background()
	err := w.cli.ContainerRemove(ctx, w.ID, types.ContainerRemoveOptions{
		Force: true,
	})
	w.cli.Close()

	return err
}

func (w Worker) parseStdout(r io.Reader) ([]string, error) {
	scanner := bufio.NewScanner(r)
	scanner.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}
		if i := strings.Index(string(data), w.separator); i >= 0 {
			return i + len(w.separator), data[0:i], nil
		}

		if atEOF {
			return len(data), data, nil
		}

		return 0, nil, nil
	})

	res := make([]string, 0, 3)
	for scanner.Scan() {
		res = append(res, scanner.Text())
	}

	return res, scanner.Err()
}

func (w Worker) getFromContainer(path string, limit int64) ([]byte, error) {
	ctx := context.Background()
	f, _, err := w.cli.CopyFromContainer(ctx, w.ID, path)
	if err != nil {
		logger.AppLog.Errorf("%+v", err)
		return nil, err
	}

	r := tar.NewReader(f)
	r.Next()
	buf := make([]byte, limit)
	n, err := r.Read(buf)
	if err != nil && err != io.EOF {
		logger.AppLog.Errorf("%+v", err)
		return nil, err
	}

	return buf[0:n], err
}

func removeTempFile(file *os.File) {
	file.Close()
	err := os.Remove(file.Name())
	if err != nil {
		logger.AppLog.Errorf("temp file remove fail: %+v", err)
	}
}

func parseTimeText(time string) (int64, int64, error) {
	time = strings.TrimSpace(time)
	lines := strings.Split(time, "\n")
	if i := strings.Index(lines[0], "Command"); i != -1 {
		time = lines[1]
	} else {
		time = lines[0]
	}
	args := strings.Split(strings.TrimSpace(time), " ")
	if len(args) < 2 {
		return 0, 0, ErrTimeTextParse
	}
	t, err := strconv.ParseFloat(args[0], 64)
	if err != nil {
		logger.AppLog.Errorf("parse error %+v", err)
		return 0, 0, ErrTimeTextParse
	}

	m, err := strconv.ParseFloat(args[1], 64)
	if err != nil {
		logger.AppLog.Errorf("parse error %+v", err)
		return 0, 0, ErrTimeTextParse
	}

	// GNU Timeにはメモリ使用量が4倍に表示されるバグがある。
	// https://bugzilla.redhat.com/show_bug.cgi?id=702826
	return int64(t * 1000), int64(m / 4), nil
}

func checkRuntimeError(exitCode string) error {
	code, err := strconv.Atoi(strings.TrimSpace(exitCode))
	if err != nil {
		logger.AppLog.Errorf("parse error: %+v", err)
		return err
	}

	if code != 0 {
		return errRuntime
	}
	return nil
}

func createTempDir() error {
	_, err := os.Stat(Workspace)
	if err != nil {
		return os.Mkdir(Workspace, 0700)
	}

	return nil
}
