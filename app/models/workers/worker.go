package workers

import (
	"archive/tar"
	"strconv"
	"io/ioutil"
	"os"
	"strings"
	"golang.org/x/net/context"
	"github.com/docker/docker/client"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/revel/revel"
	"github.com/pkg/errors"
	"bytes"
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
	res, err = cli.ContainerCreate(ctx, cfg, hcfg, nil, "")
	if err != nil {
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
		return nil, err
	}
	defer hijacked.Close()

	err = w.cli.ContainerStart(ctx, w.ID, types.ContainerStartOptions{})
	if err != nil {
		return nil, err
	}

	stdout, err := ioutil.TempFile("", "stdout")
	if err != nil {
		return nil, err
	}
	defer removeTempFile(stdout)

	stderr, err := ioutil.TempFile("", "stderr")
	if err != nil {
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

		hijacked.Close()

		_, err = stdcopy.StdCopy(stdout, stderr, hijacked.Reader)
		streamErrChan <- err
	}()

	err = w.cli.ContainerStart(ctx, w.ID, types.ContainerStartOptions{})
	if err != nil {
		return nil, err
	}

	if err := <-streamErrChan; err != nil {
		return nil, err
	}

	timeText, err := w.getFromContainer("/tmp/time.txt", 128)
	if err != nil {
		return nil, err
	}

	timeMillis, memoryUsage, err := parseTimeText(string(timeText))
	if err != nil {
		return nil, err
	}

	stdoutBuf := make([]byte, outputLimit)
	_, err = stdout.Read(stdoutBuf)
	if err != nil {
		return nil, err
	}
	stderrString, err := w.getFromContainer("/tmp/error.txt", errorOutputLimit)

	var status ExecStatus
	switch checkRuntimeError(stderr) {
	case runtimeError:
		status = StatusRuntimeError
	case nil:
		break
	default:
		return nil, err
	}

	switch {
	case w.TimeLimit < timeMillis:
		status = StatusTimeLimitExceeded
	case w.MemoryLimit < memoryUsage:
		status = StatusMemoryLimitExceeded
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

func (w Worker) CopyTo(srcPath, distPath string, container string) error {
	ctx := context.Background()
	r, _, err := w.cli.CopyFromContainer(ctx, w.ID, srcPath)
	if err != nil {
		return err
	}
	return w.cli.CopyToContainer(ctx, w.ID, distPath, r, types.CopyToContainerOptions{})
}

func (w Worker) PutFileTo(content []byte, name string) error {
	const (
		headerSize = 512
		binaryZero = 1024
	)
	buf := bytes.NewBuffer(make([]byte, headerSize+len(content)+binaryZero))
	writer := tar.NewWriter(buf)
	writer.WriteHeader(&tar.Header{
		Name: name,
		Size: int64(len(content)),
	})
	writer.Write(content)
	writer.Close()

	ctx := context.Background()
	return w.cli.CopyToContainer(ctx, w.ID, Workspace+name, buf, types.CopyToContainerOptions{})
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
	_, err = r.Read(buf)
	if err != nil {
		return nil, err
	}

	return buf, err
}

func removeTempFile(file *os.File) {
	err := os.Remove(file.Name())
	revel.AppLog.Errorf("temp file remove fail:", err)
}

func parseTimeText(time string) (int64, int64, error) {
	args := strings.Split(time, " ")
	if len(args) < 2 {
		return 0, 0, TimeTextParseError
	}
	t, err := strconv.ParseFloat(args[0], 64)
	if err != nil {
		return 0, 0, TimeTextParseError
	}

	m, err := strconv.ParseFloat(args[1], 64)
	if err != nil {
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
