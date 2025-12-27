package cronjob

import (
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/thanvuc/go-core-lib/cache"
)

// CronScheduler defines the interface for a single scheduler
// Only supports job > 2 minutes interval
type CronScheduler interface {
	ScheduleCronJob(schedule string, jobFunc func()) (cron.EntryID, error)
	Start()
	Stop()
}

// RobfigCron implements CronScheduler
type cronScheduler struct {
	cron        *cron.Cron
	logger      cron.Logger
	redisClient *cache.RedisCache
	lockKey     string
}

func NewCronScheduler(
	redisClient *cache.RedisCache,
	cronName string,
	cronOpts ...cron.Option,
) *cronScheduler {
	lockKey := "cronjob-lock-" + cronName
	return &cronScheduler{
		lockKey:     lockKey,
		redisClient: redisClient,
		cron:        cron.New(cronOpts...),
		logger:      cron.DefaultLogger,
	}
}

// ScheduleCronJob schedules a cron job with distributed locking
func (r *cronScheduler) ScheduleCronJob(schedule string, jobFunc func()) error {
	if schedule == "" {
		return nil
	}

	// Schedule the cron job
	if _, err := r.cron.AddFunc(schedule, func() {
		defer cache.RenewTTL(r.redisClient, r.lockKey, 2*time.Minute) // optional post-job TTL
		// Recover from panic to ensure lock release
		defer func(redisClient *cache.RedisCache, lockKey string, logger cron.Logger) {
			if r := recover(); r != nil {
				err := fmt.Errorf("%v", r) // convert recovered panic to error
				logger.Error(err, "cron job panic recovered",
					"lockKey", lockKey,
				)
				cache.Delete(redisClient, lockKey)
			}
		}(r.redisClient, r.lockKey, r.logger)

		// Try to acquire lock atomically
		ok, err := cache.SetNX(r.redisClient, r.lockKey, true, 0)
		if err != nil {
			return
		}
		if !ok {
			// Lock already exists, skip scheduling
			return
		}

		jobFunc()
	}); err != nil {
		cache.Delete(r.redisClient, r.lockKey)
		return err
	}

	return nil
}

func (r *cronScheduler) Start() {
	r.cron.Start()
}

func (r *cronScheduler) Stop() {
	r.cron.Stop()
	cache.Delete(r.redisClient, r.lockKey)
}
