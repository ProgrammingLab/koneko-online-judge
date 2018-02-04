package jobs

import (
	"sync"

	"github.com/gedorinku/koneko-online-judge/server/logger"
	"github.com/pkg/errors"
)

type Job interface {
	Run()
}

const QueueSize = 10000

var (
	ErrStuffedQueue = errors.New("job queueがいっぱいです")
	queue           = make(chan Job, QueueSize)
	once            sync.Once
)

func InitRunner() {
	once.Do(func() {
		go run()
	})
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
		return ErrStuffedQueue
	}
	queue <- job
	return nil
}
