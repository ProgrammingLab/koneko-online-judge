package workers

import (
	"fmt"
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
			w, err := NewTimeoutWorker(image, time.Second, 128*1024*1024, []string{"./" + filename})
			if err != nil {
				t.Fatal(err)
			}
			defer w.Remove()

			if err := w.CopyContentToContainer([]byte(script), Workspace+filename); err != nil {
				t.Fatalf("on case %v: %+v", i, err)
			}

			res, err := w.Run("", true)
			if err != nil {
				t.Fatalf("on case %v: %+v", i, err)
			}

			if res.Status != execStatuses[i] {
				t.Errorf("invalid ExecStatus on case #%v: test case -> %v, actual -> %v", i, execStatuses[i], res.Status)
			}

			if strings.TrimSpace(res.Stdout) != output {
				t.Errorf("invalid stdout on case #%v: test case -> %v, actual -> %v", i, output, res.Stdout)
			}

			if strings.Index(res.Stderr, stderr) == -1 {
				t.Errorf("invalid stderr on case #%v: test case -> %v, actual -> %v", i, stderr, res.Stderr)
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
			w, err := NewTimeoutWorker(image, timeLimits[i], 128*1024*1024, cmd)
			if err != nil {
				t.Fatalf("on case %v: %+v", i, err)
			}
			defer w.Remove()

			res, err := w.Run("", true)
			if err != nil {
				t.Fatalf("on case %v: %+v", i, err)
			}

			if res.Status != execStatuses[i] {
				t.Errorf("invalid ExecStatus on case #%v: test case -> %v, actual -> %v", i, execStatuses[i], res.Status)
			}

			diff := res.ExecTime.Seconds() - execTimes[i].Seconds()
			if 1.0 < math.Abs(diff) {
				t.Errorf("invalid exec time on case #%v: test case -> %v, actual -> %v", i, execTimes, res.ExecTime)
			}

			t.Logf("exec result on case #%v: %+v", i, res)
		}()
	}
}

func TestWorkerMemoryLimit(t *testing.T) {
	const memoryLimit = 1 * 1024 * 1024
	cmd := []string{"/bin/sh", "-c", "/dev/null < $(yes)"}
	w, err := NewTimeoutWorker(image, time.Second, memoryLimit, cmd)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	defer w.Remove()

	res, err := w.Run("", true)
	if err != nil {
		t.Fatalf("%+v", err)
	}

	if res.Status != StatusMemoryLimitExceeded {
		t.Errorf("invalid ExecStatus: test case -> %v, actual -> %v", StatusMemoryLimitExceeded, res.Status)
	}

	diff := float64(res.MemoryUsage - memoryLimit)
	if float64(25*1024*1024) < math.Abs(diff) {
		t.Errorf("invalid memory usage: test case -> %v, actual -> %v", memoryLimit, res.MemoryUsage)
	}

	t.Logf("exec result: %+v", res)
}
