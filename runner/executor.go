package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/gedorinku/koneko-online-judge/server/modules/workers"
)

var (
	errOutputLimitExceeded = errors.New("output limit exceeded")
	errNotSupportedOS      = errors.New("the os is not supported")
)

type outputWriteResult struct {
	n   int64
	err error
}

type Executor struct {
	TimeLimit   time.Duration
	MemoryLimit int64
	Input       os.FileInfo
	Cmd         []string
}

func NewExecutor(timeLimit time.Duration, memoryLimit int64, input os.FileInfo, cmd []string) Executor {
	return Executor{
		timeLimit,
		memoryLimit,
		input,
		cmd,
	}
}

func (e Executor) ExecMonitored() error {
	in, err := os.Open(inputDir + e.Input.Name())
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(outputDir + e.Input.Name())
	if err != nil {
		return err
	}
	defer out.Close()
	pr, pw := io.Pipe()
	defer pr.Close()

	ch := make(chan outputWriteResult, 1)

	go func() {
		defer pw.Close()
		n, err := io.CopyN(out, pr, outputLimit)
		if err == io.EOF {
			err = nil
		}
		// err == nilの時だけn == outputLimitになる
		if n == outputLimit {
			err = errOutputLimitExceeded
		}
		ch <- outputWriteResult{n, err}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), e.TimeLimit+50*time.Millisecond)
	defer cancel()
	cmd := []string{"/usr/bin/sudo", "-u", "nobody", "--"}
	cmd = append(cmd, e.Cmd...)
	c := exec.CommandContext(ctx, cmd[0], cmd[1:]...)
	c.Stdin = in
	c.Stdout = pw

	start := time.Now()
	err = c.Run()
	dur := time.Now().Sub(start)
	pw.Close()
	if err != nil && err != io.ErrClosedPipe && err != context.DeadlineExceeded {
		_, ok := err.(*exec.ExitError)
		if !ok {
			return err
		}
	}

	writeRes := <-ch
	if writeRes.err != nil && writeRes.err != errOutputLimitExceeded {
		return writeRes.err
	}

	return e.saveExecResult(c, writeRes, dur)
}

func (e Executor) saveExecResult(cmd *exec.Cmd, writeRes outputWriteResult, duration time.Duration) error {
	usage, ok := cmd.ProcessState.SysUsage().(*syscall.Rusage)
	if !ok {
		return errNotSupportedOS
	}
	memory := usage.Maxrss * 1024
	wst, ok := cmd.ProcessState.Sys().(syscall.WaitStatus)
	if !ok {
		return errNotSupportedOS
	}
	exitStatus := wst.ExitStatus()

	status := workers.StatusFinished
	switch {
	case e.MemoryLimit < memory:
		status = workers.StatusMemoryLimitExceeded
	case e.TimeLimit < duration:
		status = workers.StatusTimeLimitExceeded
	case writeRes.err == errOutputLimitExceeded:
		status = workers.StatusOutputLimitExceeded
	case exitStatus != 0:
		status = workers.StatusRuntimeError
	}

	res := workers.ExecResult{
		Status:      status,
		ExecTime:    duration,
		MemoryUsage: memory,
	}

	st, err := os.Create(statusDir + e.Input.Name())
	if err != nil {
		return err
	}
	defer st.Close()
	en := json.NewEncoder(st)
	return en.Encode(res)
}
