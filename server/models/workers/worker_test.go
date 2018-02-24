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
			w, err := NewWorker(image, 1000, 128*1024*1024, []string{"./" + filename})
			if err != nil {
				t.Fatal(err)
			}
			defer w.Remove()

			if err := w.CopyContentToContainer([]byte(script), filename); err != nil {
				t.Fatalf("on case %v: %+v", i, err)
			}

			res, err := w.Run("")
			if err != nil {
				t.Fatalf("on case %v: %+v", i, err)
			}

			if res.Status != execStatuses[i] {
				t.Errorf("invalid ExecStatus on case #%v: test case -> %v, actual -> %v", i, execStatuses[i], res.Status)
			}

			if strings.TrimSpace(res.Stdout) != output {
				t.Errorf("invalid stdout on case #%v: test case -> %v, actual -> %v", i, output, res.Stdout)
			}

			if strings.TrimSpace(res.Stderr) != stderr {
				t.Errorf("invalid stderr on case #%v: test case -> %v, actual -> %v", i, stderr, res.Stderr)
			}
		}()
	}
}

func TestWorkerTimeLimit(t *testing.T) {
	cmd := []string{"/bin/sleep", "1s"}
	timeLimits := []time.Duration{time.Second / 2, 5 * time.Second}
	execStatuses := []ExecStatus{StatusTimeLimitExceeded, StatusFinished}
	execTimes := []int64{500, 1000}

	for i := range timeLimits {
		func() {
			millis := timeLimits[i].Seconds() * 1000
			w, err := NewWorker(image, int64(millis), 128*1024*1024, cmd)
			if err != nil {
				t.Fatalf("on case %v: %+v", i, err)
			}
			defer w.Remove()

			res, err := w.Run("")
			if err != nil {
				t.Fatalf("on case %v: %+v", i, err)
			}

			if res.Status != execStatuses[i] {
				t.Errorf("invalid ExecStatus on case #%v: test case -> %v, actual -> %v", i, execStatuses[i], res.Status)
			}

			diff := float64(res.ExecTime - execTimes[i])
			if 1000 < math.Abs(diff) {
				t.Errorf("invalid exec time on case #%v: test case -> %v, actual -> %v", i, millis, res.ExecTime)
			}

			t.Logf("exec result: %+v", res)
		}()
	}
}

func TestWorkerMemoryLimit(t *testing.T) {
	const memoryLimit = 1 * 1024 * 1024
	cmd := []string{"/bin/sh", "-c", "/dev/null < $(yes yeaaaaaaaah)"}
	w, err := NewWorker(image, int64(10*time.Second), memoryLimit, cmd)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	defer w.Remove()

	res, err := w.Run("")
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
}
