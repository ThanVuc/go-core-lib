package eventbus

import (
	"context"
	"sync"
	"time"

	"github.com/thanvuc/go-core-lib/log"
	"github.com/wagslane/go-rabbitmq"
	"go.uber.org/zap"
)

type RabbitMQConnector struct {
	conn      *rabbitmq.Conn
	publisher *rabbitmq.Publisher
	uri       string
	consumer  []*rabbitmq.Consumer
	logger    log.Logger
}

func NewConnector(uri string, logger log.Logger) (*RabbitMQConnector, error) {
	conn, err := rabbitmq.NewConn(
		uri,
		rabbitmq.WithConnectionOptionsLogging,
	)

	if err != nil {
		return nil, err
	}

	publisher, err := rabbitmq.NewPublisher(
		conn,
		rabbitmq.WithPublisherOptionsLogging,
	)

	if err != nil {
		conn.Close()
		return nil, err
	}

	logger.Info("RabbitMQ connection established", "", zap.String("uri", uri))

	return &RabbitMQConnector{
		conn:      conn,
		publisher: publisher,
		uri:       uri,
		logger:    logger,
	}, nil
}

func (r *RabbitMQConnector) Close(wg *sync.WaitGroup) {
	defer wg.Done()

	if r.publisher != nil {
		r.publisher.Close()
	}

	var consumerWg sync.WaitGroup
	for _, consumer := range r.consumer {
		if consumer != nil {
			consumerWg.Add(1)
			go func(c *rabbitmq.Consumer) {
				defer consumerWg.Done()
				shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				c.CloseWithContext(shutdownCtx)
			}(consumer)
		}
	}

	consumerWg.Wait()

	if r.conn != nil {
		r.conn.Close()
	}

	r.logger.Info("RabbitMQ connection, publisher, consumer closed", "", zap.String("uri", r.uri))
}
