package models

import (
	"github.com/garyburd/redigo/redis"
	"github.com/gocraft/work"
)

type jobContext struct{}

const (
	redisNamespace      = "koneko_online_judge"
	submissionJobArgKey = "submission_id"
	judgementJobName    = "judgement"
)

var (
	redisPool = &redis.Pool{
		MaxActive: 3,
		MaxIdle:   3,
		Wait:      true,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", ":6379")
		},
	}
	enqueuer   = work.NewEnqueuer(redisNamespace, redisPool)
	workerPool = work.NewWorkerPool(jobContext{}, 1, redisNamespace, redisPool)
)

func InitJobs() {
	workerPool.Job(judgementJobName, (*jobContext).Judge)
	workerPool.Start()
}

func StopPool() {
	workerPool.Stop()
}

func (c *jobContext) Judge(job *work.Job) error {
	id := job.ArgInt64(submissionJobArgKey)
	if err := job.ArgError(); err != nil {
		return err
	}

	judge := judgementJob{
		submissionID: uint(id),
	}
	defer judge.Close()
	judge.Run()

	return nil
}
