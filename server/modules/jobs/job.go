package jobs

import (
	"github.com/gedorinku/koneko-online-judge/server/logger"
	"github.com/pkg/errors"
)

type Job interface {
	Run()
}

const QueueSize = 10000

var (
	StuffedQueueError       = errors.New("job queueがいっぱいです")
	AlreadyInitializedError = errors.New("job runnerは初期化済みです")
	queue                   = make(chan Job, QueueSize)
	initialized             = false
)

func InitRunner() error {
	if initialized {
		return AlreadyInitializedError
	}
	go run()
	initialized = true
	return nil
}

func run() {
	defer func() {
		if err := recover(); err != nil {
			logger.AppLog.Errorf("job recover:", err)
			go run()
		}
	}()

	for {
		job := <-queue
		job.Run()
	}
}

func Now(job Job) error {
	if len(queue) == QueueSize {
		return StuffedQueueError
	}
	queue <- job
	return nil
}
