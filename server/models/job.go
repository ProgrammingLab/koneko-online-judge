package models

import (
	"github.com/garyburd/redigo/redis"
	"github.com/gedorinku/koneko-online-judge/server/conf"
	"github.com/gedorinku/koneko-online-judge/server/logger"
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
			cfg := conf.GetConfig().Koneko
			return redis.Dial("tcp", cfg.RedisHost)
		},
	}
	enqueuer          = work.NewEnqueuer(redisNamespace, redisPool)
	workerPool   *work.WorkerPool
	workerClient  = work.NewClient(redisNamespace, redisPool)
)

func InitJobs() {
	cfg := conf.GetConfig().Judgement
	workerPool = work.NewWorkerPool(jobContext{}, uint(cfg.Concurrently), redisNamespace, redisPool)
	workerPool.Job(judgementJobName, (*jobContext).Judge)
	workerPool.Start()
}

func StopPool() {
	workerPool.Stop()
}

func GetWorkers() ([]*work.WorkerObservation, error) {
	w, err := workerClient.WorkerObservations()
	if err != nil {
		logger.AppLog.Errorf("error: %+v", err)
	}
	return w, err
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
