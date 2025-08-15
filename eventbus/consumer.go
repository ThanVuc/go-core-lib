package eventbus

import (
	"context"
	"errors"
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

	connector.consumers = append(connector.consumers, ownConsumerInstance)

	if err != nil {
		panic(err)
	}

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

	return c.ownConsumer.Run(handler)
}

func (c *consumer) Close() {
	if c.ownConsumer != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		c.ownConsumer.CloseWithContext(shutdownCtx)
	}

	c.logger.Info("Consumer closed", "", zap.String("exchange", string(c.exchange)), zap.String("queue", c.queueName))
}
