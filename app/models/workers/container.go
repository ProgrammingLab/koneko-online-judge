package workers

import (
	"strings"
	"encoding/base64"
	"crypto/rand"
	"github.com/pkg/errors"
	"github.com/revel/revel"
	"github.com/docker/docker/api/types"
	"golang.org/x/net/context"
	"github.com/docker/docker/client"
	"github.com/docker/docker/api/types/container"
	"io/ioutil"
	"os"
	"os/exec"
)

type JudgementWorker struct {
	container      types.Container
	context        context.Context
	client         *client.Client
	ID             string
	sourceFileName string
}

const (
	workspace        = "/tmp/koj-workspace"
	timeCommand      = "/usr/bin/time"
	timeOutCommand   = "timeout"
	imageNamePrefix  = "koneko-online-judge-image-"
	inputFileName    = "input.txt"
	outputLimit      = 1024 * 1024 * 10
	errorOutputLimit = 1024
)

var WaitStatusIsUnimplementedErr = errors.New("waitStatus is unimplemented")

func NewJudgementWorker(imageSuffix string, memoryLimit int) (*JudgementWorker, error) {
	worker := &JudgementWorker{}
	worker.context = context.Background()
	cli, err := client.NewEnvClient()
	if err != nil {
		return nil, err
	}
	worker.client = cli

	var res container.ContainerCreateCreatedBody
	res, err = cli.ContainerCreate(worker.context, &container.Config{
		Image:        imageNamePrefix + imageSuffix,
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		WorkingDir:   workspace,
	}, &container.HostConfig{
		Resources: container.Resources{
			CpusetCpus: "0",
			PidsLimit:  10,
			Memory:     int64(1024 * 1024 * memoryLimit),
		},
	}, nil, "")

	worker.ID = res.ID
	return worker, nil
}

func (w *JudgementWorker) Start(sourceFileName string, sourceCode, input *string) error {
	if err := w.writeFileToTempDirectory(sourceFileName, sourceCode); err != nil {
		return err
	}
	if err := w.writeFileToTempDirectory(inputFileName, input); err != nil {
		return err
	}

	return w.client.ContainerStart(w.context, w.ID, types.ContainerStartOptions{})
}

func (w *JudgementWorker) Compile(command string) (int, string) {
	id, _ := w.client.ContainerExecCreate(w.context, w.ID, types.ExecConfig{
		AttachStderr: true,
		AttachStdout: true,
		WorkingDir:   workspace,
		Cmd:          strings.Split(timeOutCommand+" 5 "+command, " "),
	})
	revel.AppLog.Debugf(timeOutCommand + " 5 " + command)
	resp, err := w.client.ContainerExecAttach(w.context, id.ID, types.ExecStartCheck{})
	if err != nil {
		revel.AppLog.Errorf("docker exec", err)
	}

	if r := resp.Reader; r != nil {
		log := make([]byte, errorOutputLimit)
		n, _ := r.Read(log)
		revel.AppLog.Errorf(string(log))
		return 0, string(log[0:n])
	}

	revel.AppLog.Errorf("docker exec something wrong")
	return -1, ""
}

func (w *JudgementWorker) writeFileToTempDirectory(name string, text *string) error {
	if err := makeTempDirectory(); err != nil {
		return err
	}

	tempFile := workspace + "/" + name
	if err := ioutil.WriteFile(tempFile, []byte((*text)[:]), os.ModePerm); err != nil {
		return err
	}

	copyCommandArgs := []string{"cp", tempFile, w.ID + ":" + tempFile}
	if err := exec.Command("docker", copyCommandArgs...).Run(); err != nil {
		return err
	}

	return os.Remove(tempFile)
}

func makeTempDirectory() error {
	_, err := os.Stat(workspace)
	if err != nil {
		return os.Mkdir(workspace, os.ModePerm)
	}

	return nil
}

func generateRandomPassword() string {
	const length = 32
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		panic(err)
	}

	return base64.StdEncoding.EncodeToString(bytes)
}
