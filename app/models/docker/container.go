package docker

import (
	"fmt"
	"os/exec"
	"strings"
	"os"
	"io/ioutil"
	"encoding/base64"
	"crypto/rand"
	"syscall"
	"github.com/pkg/errors"
	"github.com/revel/revel"
)

type Container struct {
	ID             string
	sourceFileName string
	cmd            *exec.Cmd
}

const (
	workspace        = "/tmp/koj-workspace"
	timeCommand      = "/usr/bin/time"
	createArgsFormat = "create -i --net none --cpuset-cpus 0 --memory %vm -w " + workspace + " %v"
	imageNamePrefix  = "koneko-online-judge-image-"
	inputFileName    = "input.txt"
	outputLimit      = 1024 * 1024 * 10
	errorOutputLimit = 1024
)

var WaitStatusIsUnimplementedErr = errors.New("waitStatus is unimplemented")

// docker createする。memoryLimitの単位はMiB
func CreateContainer(imageSuffix string, memoryLimit int, sourceFileName string) *Container {
	imageName := imageNamePrefix + imageSuffix
	options := fmt.Sprintf(createArgsFormat, memoryLimit, imageName)
	cmd := exec.Command("docker", strings.Split(options, " ")...)
	id, err := cmd.Output()
	if err != nil {
		return nil
	}

	return &Container{
		ID:             string(id[0:16]),
		sourceFileName: sourceFileName,
	}
}

func (c *Container) Start(sourceCode, input *string) error {
	if err := c.writeFileToTempDirectory(c.sourceFileName, sourceCode); err != nil {
		return err
	}
	if err := c.writeFileToTempDirectory(inputFileName, input); err != nil {
		return err
	}

	startArgs := []string{"start", "-i", c.ID}
	c.cmd = exec.Command("docker", startArgs...)
	if err := c.cmd.Start(); err != nil {
		return err
	}

	return nil
}

// exit code, stderrを返す
func (c *Container) Compile(command string) (int, string) {
	revel.AppLog.Info("compile")
	args := strings.Split("exec"+timeCommand+" 5 "+command, " ")
	cmd := exec.Command("docker", args...)
	stderr, _ := cmd.StderrPipe()
	defer stderr.Close()

	cmd.Start()
	cmd.Wait()

	log := make([]byte, errorOutputLimit)
	n, _ := stderr.Read(log)

	if s, ok := cmd.ProcessState.Sys().(syscall.WaitStatus); ok {
		status := s.ExitStatus()
		return status, string(log[0:n])
	}

	panic(WaitStatusIsUnimplementedErr)
}

func (c *Container) writeFileToTempDirectory(name string, text *string) error {
	if err := makeTempDirectory(); err != nil {
		return err
	}

	tempFile := workspace + "/" + name
	if err := ioutil.WriteFile(tempFile, []byte((*text)[:]), os.ModePerm); err != nil {
		return err
	}

	copyCommandArgs := []string{"cp", tempFile, c.ID + ":" + tempFile}
	if err := exec.Command("docker", copyCommandArgs...).Run(); err != nil {
		return err
	}

	return os.Remove(tempFile)
}

func makeTempDirectory() error {
	return os.Mkdir(workspace, os.ModePerm)
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
