package workers

import (
	"archive/tar"
	"io"
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
	"golang.org/x/sync/errgroup"
)

type Worker struct {
	ID string

	Stdin     io.WriteCloser
	stdinPipe io.ReadCloser

	Stdout     io.ReadCloser
	stdoutPipe io.WriteCloser

	Stderr     io.ReadCloser
	stderrPipe io.WriteCloser

	Status      ExecStatus
	ExecTime    time.Duration
	MemoryUsage int64

	cli         *client.Client
	eg          errgroup.Group
	timeLimit   time.Duration
	memoryLimit int64
}

type WorkerConfig struct {
	TimeLimit   time.Duration
	MemoryLimit int64
}

type ExecStatus int

const (
	StatusCreated             ExecStatus = 0
	StatusRunning             ExecStatus = 1
	StatusFinished            ExecStatus = 2
	StatusTimeLimitExceeded   ExecStatus = 3
	StatusMemoryLimitExceeded ExecStatus = 4
	StatusRuntimeError        ExecStatus = 5
	StatusUnknownError        ExecStatus = 6

	OutputLimit      = 10 * 1024 * 1024
	errorOutputLimit = 512
	Workspace        = "/tmp/koj-workspace/"
	errorString      = "runtime_error"
)

var (
	ErrTimeTextParse = errors.New("time.txtの内容がパースできません。")
	errRuntime       = errors.New("runtime error")
)

func NewWorker(img string, timeLimit time.Duration, memoryLimit int64, cmd []string) (*Worker, error) {
	ctx := context.Background()
	cli, err := client.NewEnvClient()
	if err != nil {
		return nil, err
	}

	// 下のやつ、echo $?したら必ず0になってよくわからず
	runCmd := []string{
		"/usr/bin/time", "-f", "%e %M", "-o", "time.txt",
		"timeout", strconv.FormatFloat(timeLimit.Seconds()+0.01, 'f', 4, 64),
		"/usr/bin/sudo", "-u", "nobody", "--",
		"/bin/sh", "-c", strings.Join(cmd, " ") + " || echo " + errorString + " >error.txt",
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
			PidsLimit:  10,
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
		Status:      StatusCreated,
		timeLimit:   timeLimit,
		memoryLimit: memoryLimit,
		cli:         cli,
		eg:          errgroup.Group{},
	}
	w.Stdout, w.stdoutPipe = io.Pipe()
	w.Stderr, w.stderrPipe = io.Pipe()
	w.stdinPipe, w.Stdin = io.Pipe()
	return w, nil
}

func (w *Worker) Start() error {
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
		return err
	}

	err = w.cli.ContainerStart(ctx, w.ID, types.ContainerStartOptions{})
	if err != nil {
		logger.AppLog.Errorf("error %+v", err)
		hijacked.Close()
		return err
	}
	w.Status = StatusRunning

	w.observeIO(hijacked)
	w.observeContainer(hijacked)

	return nil
}

func (w *Worker) observeIO(response types.HijackedResponse) {
	w.eg.Go(func() error {
		defer response.CloseWrite()
		_, err := io.Copy(response.Conn, w.stdinPipe)
		if err != nil && err != io.EOF {
			logger.AppLog.Errorf("%+v", err)
			return err
		}
		return nil
	})
	w.eg.Go(func() error {
		_, err := stdcopy.StdCopy(w.stdoutPipe, w.stderrPipe, response.Reader)
		if err != nil && err != io.EOF {
			logger.AppLog.Errorf("%+v", err)
			return err
		}
		return nil
	})
}

func (w *Worker) observeContainer(status types.HijackedResponse) {
	w.eg.Go(func() error {
		ctx := context.Background()
		_, err := w.cli.ContainerWait(ctx, w.ID)
		if err != nil {
			logger.AppLog.Errorf("error %+v", err)
			return err
		}

		exitStatus, err := w.getFromContainer(Workspace+"error.txt", errorOutputLimit)
		// 終了コードが0のときerror.txtは出力されないので、そのようなエラーは無視する
		if err != nil && err != io.EOF && !strings.Contains(err.Error(), "Could not find the file") {
			logger.AppLog.Errorf("error %+v", err)
			return err
		}

		timeText, err := w.getFromContainer(Workspace+"time.txt", 128)
		if err != nil && err != io.EOF {
			logger.AppLog.Errorf("error %+v", err)
			return err
		}

		timeMillis, memoryUsage, err := parseTimeText(string(timeText))
		if err != nil {
			logger.AppLog.Errorf("error: %v %+v", string(timeText), err)
			return err
		}
		memoryUsage *= 1024
		w.ExecTime = time.Duration(timeMillis) * time.Millisecond
		w.MemoryUsage = memoryUsage

		if 0 < len(exitStatus) {
			w.Status = StatusRuntimeError
		} else {
			switch {
			case int64(w.timeLimit.Seconds()*1000.0+0.0001) <= timeMillis:
				w.Status = StatusTimeLimitExceeded
				logger.AppLog.Debugf("time limit(%v) exceeded:%v", w.timeLimit, timeMillis)
			case w.memoryLimit <= memoryUsage:
				w.Status = StatusMemoryLimitExceeded
				logger.AppLog.Debugf("memory limit(%v) exceeded:%v", w.memoryLimit, memoryUsage)
			default:
				w.Status = StatusFinished
			}
		}

		return nil
	})
}

func (w *Worker) Wait() error {
	if err := w.eg.Wait(); err != nil {
		w.Status = StatusUnknownError
		return err
	}
	return nil
}

func (w *Worker) Output() (string, error) {
	if err := w.Start(); err != nil {
		return "", err
	}
	if err := w.Wait(); err != nil {
		return "", err
	}
	buf := make([]byte, 0, OutputLimit)
	n, err := w.Stdout.Read(buf)
	if err != nil {
		return "", err
	}
	return string(buf[:n]), nil
}

func (w *Worker) CopyTo(basename string, dist *Worker) error {
	const limit = 10 * 1024 * 1024
	name := Workspace + basename
	content, err := w.getFromContainer(name, limit)
	if err != nil {
		logger.AppLog.Errorf("%+v", err)
		return err
	}
	return dist.CopyContentToContainer(content, basename)
}

func (w *Worker) CopyContentToContainer(content []byte, name string) error {
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

func (w *Worker) Remove() error {
	ctx := context.Background()
	err := w.cli.ContainerRemove(ctx, w.ID, types.ContainerRemoveOptions{
		Force: true,
	})
	w.cli.Close()

	return err
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

func createTempDir() error {
	_, err := os.Stat(Workspace)
	if err != nil {
		return os.Mkdir(Workspace, os.ModePerm)
	}

	return nil
}
