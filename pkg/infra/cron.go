package infra

import (
	"ley/pkg/log"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

var globalCron *cron.Cron

func InitCron() {
	globalCron = cron.New(cron.WithLogger(CronLogger()))
	globalCron.Start()
}

type CustomCronLogger struct {
	logger *zap.Logger
}

func (cl *CustomCronLogger) Info(msg string, keysAndValues ...interface{}) {
	cl.logger.Sugar().Info(msg, keysAndValues)
}

func (cl *CustomCronLogger) Error(err error, msg string, keysAndValues ...interface{}) {
	cl.logger.Sugar().Error(err, msg, keysAndValues)
}
func CronLogger() cron.Logger {
	return &CustomCronLogger{
		logger: log.GetLogger().Logger,
	}
}

func CloseCron() {
	globalCron.Stop()
}
