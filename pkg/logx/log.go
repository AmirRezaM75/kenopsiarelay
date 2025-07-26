package logx

import (
	"errors"
	"log"
	"syscall"

	"go.uber.org/zap"
)

var Logger *zap.SugaredLogger

func NewLogger() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatalf(`level=error msg="%s" desc="%s"`, err.Error(), "could not create new zap instance")
	}

	defer func(logger *zap.Logger) {
		err := logger.Sync()
		if errors.Is(err, syscall.EINVAL) {
			// https://github.com/uber-go/zap/issues/328
			return
		}
		if err != nil {
			log.Printf(`level=error msg="%s" desc="%s"`, err.Error(), "could not sync (flush) logger")
		}
	}(logger)

	Logger = logger.Sugar()
}
