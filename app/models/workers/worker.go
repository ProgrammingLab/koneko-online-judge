package workers

import (
	"archive/tar"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/pkg/errors"
	"github.com/revel/revel"
	"golang.org/x/net/context"
)

type Worker struct {
	ID          string
	cli         *client.Client
	TimeLimit   int64
	MemoryLimit int64
}

type ExecStatus int

const (
	StatusFinished            = 0
	StatusTimeLimitExceeded   = 1
	StatusMemoryLimitExceeded = 2
	StatusRuntimeError        = 3
	StatusUnknownError        = 4
	outputLimit               = 10 * 1024 * 1024
	errorOutputLimit          = 512
	Workspace                 = "/tmp/koj-workspace/"
	errorString               = "runtime_error"
)

var (
	TimeTextParseError = errors.New("time.txtの内容がパースできません。")
	runtimeError       = errors.New("runtime error")
)

type ExecResult struct {
	Status      ExecStatus
	ExecTime    int64
	MemoryUsage int64
	Stdout      string
	Stderr      string
}

// memoryLimitはバイト、timeLimitはミリ秒単位
func NewWorker(img string, timeLimit int64, memoryLimit int64, cmd []string) (*Worker, error) {
	ctx := context.Background()
	cli, err := client.NewEnvClient()
	if err != nil {
		return nil, err
	}

	// 下のやつ、echo $?したら必ず0になってよくわからず
	runCmd := []string{
		"/usr/bin/time", "-f", "%e %M", "-o", "time.txt",
		"timeout", strconv.FormatInt(timeLimit/1000, 10),
		"/usr/bin/sudo", "-u", "nobody",
		"/bin/sh", "-c", strings.Join(cmd, " ") + " 2>error.txt || echo " + errorString + " 1>&2",
	}
	runCmd = append(runCmd, cmd...)

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
			PidsLimit:  10,
			Memory:     memoryLimit + 10*1024*1024,
		},
		NetworkMode: "none",
	}

	var res container.ContainerCreateCreatedBody
	res, err = cli.ContainerCreate(ctx, cfg, hcfg, &network.NetworkingConfig{}, "")
	if err != nil {
		revel.AppLog.Errorf("error %v", img, err)
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

func (w Worker) Run(input string) (*ExecResult, error) {
	ctx := context.Background()
	opt := types.ContainerAttachOptions{
		Stream: true,
		Stdin:  true,
		Stdout: true,
		Stderr: true,
	}
	hijacked, err := w.cli.ContainerAttach(ctx, w.ID, opt)
	if err != nil {
		revel.AppLog.Errorf("error", err)
		return nil, err
	}
	defer hijacked.Close()

	err = w.cli.ContainerStart(ctx, w.ID, types.ContainerStartOptions{})
	if err != nil {
		revel.AppLog.Errorf("error", err)
		return nil, err
	}

	stdout, err := ioutil.TempFile("", "stdout")
	if err != nil {
		revel.AppLog.Errorf("error", err)
		return nil, err
	}
	defer removeTempFile(stdout)

	stderr, err := ioutil.TempFile("", "stderr")
	if err != nil {
		revel.AppLog.Errorf("error", err)
		return nil, err
	}
	defer removeTempFile(stderr)

	streamErrChan := make(chan error)
	// 関数分けようとしたら、hijackedを引数にとる関数をつくろうとするとコンパイルエラーになる(意味不明)のでつらい
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

	err = w.cli.ContainerStart(ctx, w.ID, types.ContainerStartOptions{})
	if err != nil {
		revel.AppLog.Errorf("error", err)
		return nil, err
	}

	if err := <-streamErrChan; err != nil {
		revel.AppLog.Errorf("error", err)
		return nil, err
	}

	timeText, err := w.getFromContainer(Workspace+"time.txt", 128)
	if err != nil {
		revel.AppLog.Errorf("error", err)
		return nil, err
	}

	timeMillis, memoryUsage, err := parseTimeText(string(timeText))
	if err != nil {
		revel.AppLog.Errorf("error: %v", string(timeText), err)
		return nil, err
	}

	_, err = stdout.Seek(0, 0)
	if err != nil {
		revel.AppLog.Errorf("error", err)
		return nil, err
	}
	stdoutBuf := make([]byte, outputLimit)
	n, err := stdout.Read(stdoutBuf)
	if err != nil && err != io.EOF {
		revel.AppLog.Errorf("error", err)
		return nil, err
	}
	stdoutBuf = stdoutBuf[0:n]
	stderrString, err := w.getFromContainer(Workspace+"error.txt", errorOutputLimit)
	if err != nil && err != io.EOF {
		revel.AppLog.Errorf("error", err)
		return nil, err
	}

	var status ExecStatus
	switch checkRuntimeError(stderr) {
	case runtimeError:
		status = StatusRuntimeError
	case nil:
		break
	default:
		return nil, nil
	}

	switch {
	case w.TimeLimit < timeMillis:
		status = StatusTimeLimitExceeded
		revel.AppLog.Debugf("time limit(%v) exceeded:%v", w.TimeLimit, timeMillis)
	case w.MemoryLimit < memoryUsage:
		status = StatusMemoryLimitExceeded
		revel.AppLog.Debugf("memory limit(%v) exceeded:%v", w.MemoryLimit, memoryUsage)
	default:
		status = StatusFinished
	}

	return &ExecResult{
		Status:      status,
		ExecTime:    timeMillis,
		MemoryUsage: memoryUsage,
		Stdout:      string(stdoutBuf),
		Stderr:      string(stderrString),
	}, nil
}

func (w Worker) CopyTo(basename string, dist *Worker) error {
	const limit = 10 * 1024 * 1024
	name := Workspace + basename
	content, err := w.getFromContainer(name, limit)
	if err != nil {
		revel.AppLog.Errorf("", err)
		return err
	}
	return dist.CopyContentToContainer(content, basename)
}

func (w Worker) CopyContentToContainer(content []byte, name string) error {
	createTempDir()
	f, err := os.Create(Workspace + name)
	if err != nil {
		revel.AppLog.Errorf("could not create temp file", err)
		return err
	}
	f.Write(content)
	f.Close()
	err = os.Chmod(f.Name(), 0777)
	if err != nil {
		revel.AppLog.Errorf("could not change temp file mode", err)
		return err
	}
	defer removeTempFile(f)

	srcInfo, err := archive.CopyInfoSourcePath(f.Name(), false)
	if err != nil {
		revel.AppLog.Errorf("", err)
		return err
	}

	srcArchive, err := archive.TarResource(srcInfo)
	if err != nil {
		revel.AppLog.Errorf("", err)
		return err
	}
	defer srcArchive.Close()

	// 最後に`/`をつけてはいけない
	dstInfo := archive.CopyInfo{Path: Workspace + name}
	dstDir, preparedArchive, err := archive.PrepareArchiveCopy(srcArchive, srcInfo, dstInfo)
	if err != nil {
		revel.AppLog.Errorf("", err)
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

func (w Worker) getFromContainer(path string, limit int64) ([]byte, error) {
	ctx := context.Background()
	f, _, err := w.cli.CopyFromContainer(ctx, w.ID, path)
	if err != nil {
		return nil, err
	}

	r := tar.NewReader(f)
	r.Next()
	buf := make([]byte, limit)
	n, err := r.Read(buf)
	if err != nil {
		return nil, err
	}

	return buf[0:n], err
}

func removeTempFile(file *os.File) {
	file.Close()
	err := os.Remove(file.Name())
	if err != nil {
		revel.AppLog.Errorf("temp file remove fail:", err)
	}
}

func parseTimeText(time string) (int64, int64, error) {
	args := strings.Split(strings.TrimSpace(time), " ")
	if len(args) < 2 {
		return 0, 0, TimeTextParseError
	}
	t, err := strconv.ParseFloat(args[0], 64)
	if err != nil {
		revel.AppLog.Errorf("parse error", err)
		return 0, 0, TimeTextParseError
	}

	m, err := strconv.ParseFloat(args[1], 64)
	if err != nil {
		revel.AppLog.Errorf("parse error", err)
		return 0, 0, TimeTextParseError
	}

	return int64(t), int64(m), nil
}

func checkRuntimeError(stderr *os.File) error {
	stderrString, err := ioutil.ReadAll(stderr)
	if err != nil {
		return err
	}

	index := strings.Index(string(stderrString), errorString)
	if index == -1 {
		return nil
	}
	return runtimeError
}

func createTempDir() error {
	_, err := os.Stat(Workspace)
	if err != nil {
		return os.Mkdir(Workspace, os.ModePerm)
	}

	return nil
}
