package main

import (
	"io/ioutil"
	"os"
	"strconv"
	"time"

	"github.com/labstack/gommon/log"
)

const (
	dataDir     = "./judge_data/"
	inputDir    = dataDir + "input/"
	outputDir   = dataDir + "output/"
	statusDir   = dataDir + "status/"
	outputLimit = 10 * 1024 * 1024
	stderrLimit = 256
)

func main() {
	defer func() {
		if err := recover(); err != nil {
			log.Error(err)
		}
	}()

	f, err := os.Create(dataDir + "err")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	log.SetOutput(f)

	if len(os.Args) < 4 {
		log.Fatal("invalid arg(s)")
	}

	err = loadSeccompContext()
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

	os.RemoveAll(outputDir)
	err = os.Mkdir(outputDir, 0777)
	if err != nil {
		log.Fatal(err)
	}

	os.RemoveAll(statusDir)
	err = os.Mkdir(statusDir, 0777)
	if err != nil {
		log.Fatal(err)
	}

	for _, i := range inputs {
		e := NewExecutor(tl, ml, i, os.Args[3:])
		if err := e.ExecMonitored(); err != nil {
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
