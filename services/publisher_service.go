package services

import (
	"context"
	"fmt"
	"github.com/amirrezam75/kenopsiarelay/pkg/logx"
	"os"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type PublisherService struct {
	broker *redis.Client
}

func NewPublisherService() PublisherService {
	broker := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", os.Getenv("BROKER_REDIS_HOST"), os.Getenv("BROKER_REDIS_PORT")), // TODO: config
		Password: os.Getenv("BROKER_REDIS_PASSWORD"),
		DB:       0,
	})
	return PublisherService{broker: broker}
}

func (publisherService PublisherService) Publish(message string) error {
	if message == "" {
		return nil
	}

	err := publisherService.broker.Publish(context.Background(), "game-service", message).Err()

	if err != nil {
		logx.Logger.Error(
			err.Error(),
			zap.String("desc", "could not publish message"),
			zap.String("message", message),
		)

		return err
	}

	return nil
}
