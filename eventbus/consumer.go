package eventbus

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/thanvuc/go-core-lib/log"
	"github.com/wagslane/go-rabbitmq"
	"go.uber.org/zap"
)

type Consumer interface {
	Consume(ctx context.Context, handler rabbitmq.Handler) error
}

type consumer struct {
	logger       log.Logger
	connector    *RabbitMQConnector
	ownConsumer  *rabbitmq.Consumer
	exchange     ExchangeName
	exchangeType ExchangeType
	routingKey   string
	queueName    string
	concurrency  int

	closeOnce sync.Once
}

func NewConsumer(
	connector *RabbitMQConnector,
	exchange ExchangeName,
	exchangeType ExchangeType,
	routingKey string,
	queueName string,
	concurrency int,
) Consumer {
	if connector == nil {
		panic("connector cannot be nil")
	}

	ownConsumerInstance, err := rabbitmq.NewConsumer(
		connector.conn,
		queueName,
		rabbitmq.WithConsumerOptionsExchangeName(string(exchange)),
		rabbitmq.WithConsumerOptionsExchangeKind(string(exchangeType)),
		rabbitmq.WithConsumerOptionsRoutingKey(routingKey),
		rabbitmq.WithConsumerOptionsConcurrency(concurrency),
		rabbitmq.WithConsumerOptionsBinding(rabbitmq.Binding{
			RoutingKey: routingKey,
			BindingOptions: rabbitmq.BindingOptions{
				Declare: true,
			},
		}),
		rabbitmq.WithConsumerOptionsExchangeDurable,
		rabbitmq.WithConsumerOptionsQueueDurable,
		rabbitmq.WithConsumerOptionsExchangeDeclare,
	)

	if err != nil {
		panic(err)
	}

	connector.consumers = append(connector.consumers, ownConsumerInstance)

	return &consumer{
		logger:       connector.logger,
		connector:    connector,
		ownConsumer:  ownConsumerInstance,
		exchange:     exchange,
		exchangeType: exchangeType,
		routingKey:   routingKey,
		queueName:    queueName,
		concurrency:  concurrency,
	}
}

func (c *consumer) Consume(ctx context.Context, handler rabbitmq.Handler) error {
	if handler == nil || c.ownConsumer == nil {
		c.logger.Error("Consume failed: handler or consumer is nil", "")
		return errors.New("handler or consumer is nil")
	}

	done := make(chan struct{})

	go func() {
		select {
		case <-ctx.Done():
			c.logger.Info("Context cancelled, closing consumer", "")
			c.Close()
		case <-done:
			return
		}
	}()

	return c.ownConsumer.Run(handler)
}

func (c *consumer) Close() {
	c.closeOnce.Do(func() {
		if c.ownConsumer != nil {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			c.ownConsumer.CloseWithContext(shutdownCtx)
		}

		c.logger.Info(
			"Consumer closed",
			"",
			zap.String("exchange", string(c.exchange)),
			zap.String("queue", c.queueName),
		)
	})
}
