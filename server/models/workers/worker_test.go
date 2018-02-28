package workers

import (
	"fmt"
	"io/ioutil"
	"math"
	"strings"
	"testing"
	"time"
)

const image = "koneko-online-judge-image-cpp"

func TestWorkerOutput(t *testing.T) {
	const (
		output    = "output"
		stderr    = "err"
		scriptTmp = "#!/bin/sh\n" +
			"echo " + output + "\n" +
			"echo " + stderr + " 1>&2\n" +
			"exit %v\n"
		filename = "run.sh"
	)

	exitCodes := []int{0, 1}
	execStatuses := []ExecStatus{StatusFinished, StatusRuntimeError}

	for i := range exitCodes {
		func() {
			script := fmt.Sprintf(scriptTmp, exitCodes[i])
			w, err := NewWorker(image, time.Second, 128*1024*1024, []string{"./" + filename})
			if err != nil {
				t.Fatal(err)
			}
			defer w.Remove()

			if err := w.CopyContentToContainer([]byte(script), filename); err != nil {
				t.Fatalf("on case %v: %+v", i, err)
			}

			res, err := w.Output()
			if err != nil {
				t.Fatalf("on case %v: %+v", i, err)
			}

			if w.Status != execStatuses[i] {
				t.Errorf("invalid ExecStatus on case #%v: test case -> %v, actual -> %v", i, execStatuses[i], res)
			}

			if strings.TrimSpace(res) != output {
				t.Errorf("invalid stdout on case #%v: test case -> %v, actual -> %v", i, output, res)
			}

			resErr, err := ioutil.ReadAll(w.Stderr)
			if err != nil {
				t.Errorf("could not read stderr on case #%v: #%+v", i, err)
			}
			if strings.TrimSpace(string(resErr)) != stderr {
				t.Errorf("invalid stderr on case #%v: test case -> %v, actual -> %v", i, stderr, string(resErr))
			}

			t.Logf("exec result on case #%v: %+v", i, res)
		}()
	}
}

func TestWorkerTimeLimit(t *testing.T) {
	cmd := []string{"/bin/sleep", "1s"}
	timeLimits := []time.Duration{500 * time.Millisecond, 5 * time.Second}
	execStatuses := []ExecStatus{StatusTimeLimitExceeded, StatusFinished}
	execTimes := []time.Duration{500 * time.Millisecond, 1000 * time.Millisecond}

	for i := range timeLimits {
		func() {
			w, err := NewWorker(image, timeLimits[i], 128*1024*1024, cmd)
			if err != nil {
				t.Fatalf("on case %v: %+v", i, err)
			}
			defer w.Remove()

			res, err := w.Output()
			if err != nil {
				t.Fatalf("on case %v: %+v", i, err)
			}

			if w.Status != execStatuses[i] {
				t.Errorf("invalid ExecStatus on case #%v: test case -> %v, actual -> %v", i, execStatuses[i], w.Status)
			}

			diff := w.ExecTime.Seconds() - execTimes[i].Seconds()
			if 1.0 < math.Abs(diff) {
				t.Errorf("invalid exec time on case #%v: test case -> %v, actual -> %v", i, execTimes, w.ExecTime)
			}

			t.Logf("exec result on case #%v: %+v", i, res)
		}()
	}
}

func TestWorkerMemoryLimit(t *testing.T) {
	const memoryLimit = 1 * 1024 * 1024
	cmd := []string{"/bin/sh", "-c", "/dev/null < $(yes)"}
	w, err := NewWorker(image, time.Second, memoryLimit, cmd)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	defer w.Remove()

	res, err := w.Output()
	if err != nil {
		t.Fatalf("%+v", err)
	}

	if w.Status != StatusMemoryLimitExceeded {
		t.Errorf("invalid ExecStatus: test case -> %v, actual -> %v", StatusMemoryLimitExceeded, w.Status)
	}

	diff := float64(w.MemoryUsage - memoryLimit)
	if float64(25*1024*1024) < math.Abs(diff) {
		t.Errorf("invalid memory usage: test case -> %v, actual -> %v", memoryLimit, w.MemoryUsage)
	}

	t.Logf("exec result: %+v", res)
}
