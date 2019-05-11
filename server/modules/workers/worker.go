package workers

import (
	"archive/tar"
	"crypto/rand"
	"encoding/base64"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/ProgrammingLab/koneko-online-judge/server/logger"
	"github.com/ProgrammingLab/koneko-online-judge/server/modules/unique"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
)

type Worker struct {
	ID               string
	cli              *client.Client
	TimeLimit        time.Duration
	MemoryLimit      int64
	HostJudgeDataDir string
	separator        string
	stdout           *os.File
	stderr           *os.File
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
	JudgeDataDir                         = Workspace + "judge_data"
	errorString                          = "runtime_error"
	exitCodeFile                         = "exit.txt"
)

var (
	ErrTimeTextParse = errors.New("time.txtの内容がパースできません。")
	errRuntime       = errors.New("runtime error")
)

type ExecResult struct {
	Status      ExecStatus    `json:"status"`
	ExecTime    time.Duration `json:"execTime"`
	MemoryUsage int64         `json:"memoryUsage"`
	Stdout      string        `json:"stdout"`
	Stderr      string        `json:"stderr"`
}

func NewTimeoutWorker(img string, timeLimit time.Duration, memoryLimit int64, cmd []string) (*Worker, error) {
	sp, err := newSeparator()
	if err != nil {
		return nil, err
	}

	outputCmd := "echo -n " + sp + "$?"
	runCmd := []string{
		"/usr/bin/time", "-f", sp + "%e %M",
		"timeout", strconv.FormatFloat(timeLimit.Seconds()+0.01, 'f', 4, 64),
		"/usr/bin/sudo", "-u", "nobody", "--",
		"/bin/bash", "-c", strings.Join(cmd, " ") + ";" + outputCmd,
	}

	w, err := newWorker(img, timeLimit, memoryLimit, runCmd, nil)
	if err != nil {
		logger.AppLog.Error(err)
		return nil, err
	}
	w.separator = sp

	return w, err
}

func NewJudgementWorker(img string, timeLimit time.Duration, memoryLimit int64, cmd []string) (*Worker, error) {
	sp, err := newSeparator()
	if err != nil {
		return nil, err
	}

	runCmd := []string{
		"./runner",
		strconv.FormatInt(int64(timeLimit), 10),
		strconv.FormatInt(int64(memoryLimit), 10),
	}
	runCmd = append(runCmd, cmd...)

	judgeData, err := createJudgeDataDir()
	if err != nil {
		logger.AppLog.Error(err)
		return nil, err
	}

	mounts := []mount.Mount{
		{
			Type:     mount.TypeBind,
			Source:   judgeData,
			Target:   JudgeDataDir,
			ReadOnly: false,
		},
	}

	w, err := newWorker(img, timeLimit, memoryLimit, runCmd, mounts)
	if err != nil {
		logger.AppLog.Error(err)
		return nil, err
	}
	w.separator = sp
	w.HostJudgeDataDir = judgeData

	runnerAbs, err := filepath.Abs("../runner/runner")
	if err != nil {
		logger.AppLog.Error(err)
		w.Remove()
		return nil, err
	}
	if err := w.CopyFileToContainer(runnerAbs, Workspace+"runner"); err != nil {
		logger.AppLog.Error(err)
		return nil, err
	}

	return w, err
}

func newWorker(img string, timeLimit time.Duration, memoryLimit int64, cmd []string, mounts []mount.Mount) (*Worker, error) {
	ctx := context.Background()
	cli, err := client.NewEnvClient()
	if err != nil {
		return nil, err
	}

	cfg := &container.Config{
		Image:        img,
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		OpenStdin:    true,
		StdinOnce:    true,
		Tty:          false,
		WorkingDir:   Workspace,
		Cmd:          cmd,
	}
	hcfg := &container.HostConfig{
		Resources: container.Resources{
			CpusetCpus: "0",
			PidsLimit:  50,
			Memory:     memoryLimit + 10*1024*1024,
		},
		NetworkMode: "none",
		Mounts:      mounts,
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

func (w *Worker) Run(input string, parseOutput bool) (*ExecResult, error) {
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

	startErrChan := make(chan error)
	go func() {
		err = w.cli.ContainerStart(ctx, w.ID, types.ContainerStartOptions{})
		startErrChan <- err
	}()

	w.stdout, err = ioutil.TempFile(Workspace, "stdout"+w.ID[:16])
	if err != nil {
		logger.AppLog.Errorf("error %+v", err)
		return nil, err
	}

	w.stderr, err = ioutil.TempFile(Workspace, "stderr"+w.ID[:16])
	if err != nil {
		logger.AppLog.Errorf("error %+v", err)
		return nil, err
	}

	streamErrChan := make(chan error)
	go func() {
		_, err := hijacked.Conn.Write([]byte(input))
		if err != nil {
			streamErrChan <- err
			return
		}

		hijacked.CloseWrite()

		_, err = stdcopy.StdCopy(w.stdout, w.stderr, hijacked.Reader)
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

	if !parseOutput {
		return &ExecResult{
			Status:      StatusFinished,
			ExecTime:    0,
			MemoryUsage: 0,
			Stdout:      "",
			Stderr:      "",
		}, nil
	}

	_, err = w.stdout.Seek(0, 0)
	if err != nil {
		logger.AppLog.Errorf("error %+v", err)
		return nil, err
	}
	rawStdout, err := w.parseOutput(w.stdout)
	if err != nil {
		logger.AppLog.Error(err)
		return nil, err
	}
	if len(rawStdout) == 1 {
		rawStdout = append(rawStdout, "255")
	}

	_, err = w.stderr.Seek(0, 0)
	if err != nil {
		logger.AppLog.Errorf("error %+v", err)
		return nil, err
	}
	rawStderr, err := w.parseOutput(w.stderr)
	if err != nil {
		logger.AppLog.Error(err)
		return nil, err
	}

	exitCode := rawStdout[1]
	timeText := rawStderr[1]

	timeMillis, memoryUsage, err := parseTimeText(string(timeText))
	if err != nil {
		logger.AppLog.Errorf("error: %v %+v", string(timeText), err)
		return nil, err
	}
	memoryUsage *= 1024

	return &ExecResult{
		Status:      checkStatus(timeMillis, memoryUsage, w.TimeLimit, w.MemoryLimit, exitCode),
		ExecTime:    time.Duration(timeMillis) * time.Millisecond,
		MemoryUsage: memoryUsage,
		Stdout:      rawStdout[0],
		Stderr:      rawStderr[0],
	}, nil
}

func (w *Worker) CopyTo(filename string, dist *Worker) error {
	const limit = 10 * 1024 * 1024
	content, err := w.getFromContainer(filename, limit)
	if err != nil && err != io.EOF {
		logger.AppLog.Errorf("%+v", err)
		return err
	}
	return dist.CopyContentToContainer(content, filename)
}

func (w *Worker) CopyContentToContainer(content []byte, name string) error {
	createTempDir()
	base := path.Base(name)
	f, err := os.Create(Workspace + base)
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

	dstInfo := archive.CopyInfo{Path: name}
	dstDir, preparedArchive, err := archive.PrepareArchiveCopy(srcArchive, srcInfo, dstInfo)
	if err != nil {
		logger.AppLog.Errorf("%+v", err)
		return err
	}

	ctx := context.Background()
	return w.cli.CopyToContainer(ctx, w.ID, dstDir, preparedArchive, types.CopyToContainerOptions{})
}

func (w *Worker) CopyFileToContainer(src, dst string) error {
	f, err := os.Open(src)
	if err != nil {
		logger.AppLog.Error(err)
		return err
	}
	defer f.Close()

	srcInfo, err := archive.CopyInfoSourcePath(f.Name(), false)
	if err != nil {
		logger.AppLog.Error(err)
		return err
	}

	srcArchive, err := archive.TarResource(srcInfo)
	if err != nil {
		logger.AppLog.Error(err)
		return err
	}
	defer srcArchive.Close()

	dstInfo := archive.CopyInfo{Path: dst}
	dstDir, preparedArchive, err := archive.PrepareArchiveCopy(srcArchive, srcInfo, dstInfo)
	if err != nil {
		logger.AppLog.Error(err)
		return err
	}

	ctx := context.Background()
	return w.cli.CopyToContainer(ctx, w.ID, dstDir, preparedArchive, types.CopyToContainerOptions{})
}

func (w *Worker) Remove() error {
	if w.stdout != nil {
		removeTempFile(w.stdout)
		w.stdout = nil
	}
	if w.stderr != nil {
		removeTempFile(w.stderr)
		w.stderr = nil
	}

	ctx := context.Background()
	err := w.cli.ContainerRemove(ctx, w.ID, types.ContainerRemoveOptions{
		Force: true,
	})
	w.cli.Close()

	if w.HostJudgeDataDir != "" {
		os.RemoveAll(w.HostJudgeDataDir)
	}

	return err
}

func (w *Worker) parseOutput(r io.Reader) ([]string, error) {
	raw := make([]byte, outputLimit)
	n, err := r.Read(raw)
	if err != nil {
		if err != io.EOF {
			return nil, err
		}
		return []string{""}, nil
	}
	out := string(raw[0:n])

	res := make([]string, 0, 3)
	for {
		i := strings.Index(out, w.separator)
		if i == -1 {
			i = len(out)
		}
		res = append(res, out[:i])
		if i == len(out) {
			break
		}
		out = out[i+len(w.separator):]
	}

	return res, nil
}

func (w *Worker) getFromContainer(path string, limit int64) ([]byte, error) {
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

func createJudgeDataDir() (string, error) {
	id, err := unique.GenerateRandomBase62String(12)
	if err != nil {
		logger.AppLog.Error(err)
		return "", err
	}
	tmp := "/tmp/judge_data/judge_data" + id
	return tmp, os.MkdirAll(tmp, 0700)
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

func checkStatus(timeMillis, memoryUsage int64, timeLimit time.Duration, memoryLimit int64, exitCode string) ExecStatus {
	runtimeErr := checkRuntimeError(exitCode)
	switch {
	case int64(timeLimit.Seconds()*1000.0+0.0001) <= timeMillis:
		logger.AppLog.Debugf("time limit(%v) exceeded:%v", timeLimit, timeMillis)
		return StatusTimeLimitExceeded
	case memoryLimit <= memoryUsage:
		logger.AppLog.Debugf("memory limit(%v) exceeded:%v", memoryLimit, memoryUsage)
		return StatusMemoryLimitExceeded
	case runtimeErr == errRuntime:
		return StatusRuntimeError
	case runtimeErr != nil:
		logger.AppLog.Error(runtimeErr)
		return StatusUnknownError
	default:
		return StatusFinished
	}
}
