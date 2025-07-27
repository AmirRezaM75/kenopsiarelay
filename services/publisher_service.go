package services

import (
	"context"
	"fmt"

	"github.com/amirrezam75/kenopsiarelay/pkg/logx"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type PublisherService struct {
	broker *redis.Client
}

func NewPublisherService(host, port, password string) PublisherService {
	broker := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", host, port),
		Password: password,
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
