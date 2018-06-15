package main

import (
	"context"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/labstack/gommon/log"
)

const (
	dataDir   = "./judge_data/"
	inputDir  = dataDir + "input/"
	outputDir = dataDir + "output/"
)

func main() {
	if len(os.Args) < 4 {
		log.Fatal("invalid arg(s)")
	}

	err := loadSeccompContext()
	if err != nil {
		log.Fatal(err)
	}

	tl, err := getTimeLimit()
	if err != nil {
		log.Fatal(err)
	}

	ml, err := getMemoryLimitByte()
	if err != nil {
		log.Fatal(err)
	}

	inputs, err := ioutil.ReadDir(inputDir)
	if err != nil {
		log.Fatal(err)
	}

	err = os.Mkdir(outputDir, 0700)
	if err != nil {
		log.Fatal(err)
	}

	for _, i := range inputs {
		if err := execMonitored(tl, ml, i, os.Args[3:]); err != nil {
			log.Fatal(err)
		}
	}
}

func getTimeLimit() (time.Duration, error) {
	t, err := strconv.ParseInt(os.Args[1], 10, 64)
	return time.Duration(t), err
}

func getMemoryLimitByte() (int64, error) {
	return strconv.ParseInt(os.Args[2], 10, 64)
}

func execMonitored(timeLimit time.Duration, memoryLimit int64, input os.FileInfo, cmd []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeLimit+50*time.Millisecond)
	defer cancel()

	in, err := os.Open(inputDir + input.Name())
	if err != nil {
		return err
	}
	out, err := os.Create(outputDir + input.Name())
	if err != nil {
		return err
	}
	c := exec.CommandContext(ctx, cmd[0], cmd[1:]...)
	c.Stdin = in
	c.Stdout = out

	if err := c.Run(); err != nil {
		return err
	}

	return nil
}
