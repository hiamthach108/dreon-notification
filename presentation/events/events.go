package events

import (
	"errors"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-amqp/v3/pkg/amqp"
	"github.com/hiamthach108/dreon-notification/config"
)

func amqpConfigFrom(cfg *config.AppConfig) (amqp.Config, error) {
	if cfg.RabbitMQ.URL == "" {
		return amqp.Config{}, errors.New("RABBITMQ_URL is required")
	}
	return amqp.NewDurableQueueConfig(cfg.RabbitMQ.URL), nil
}

// NewAMQPPublisher creates an AMQP publisher from app config.
func NewAMQPPublisher(cfg *config.AppConfig, log watermill.LoggerAdapter) (*amqp.Publisher, error) {
	ac, err := amqpConfigFrom(cfg)
	if err != nil {
		return nil, err
	}
	return amqp.NewPublisher(ac, log)
}

// NewAMQPSubscriber creates an AMQP subscriber from app config.
func NewAMQPSubscriber(cfg *config.AppConfig, log watermill.LoggerAdapter) (*amqp.Subscriber, error) {
	ac, err := amqpConfigFrom(cfg)
	if err != nil {
		return nil, err
	}
	return amqp.NewSubscriber(ac, log)
}
