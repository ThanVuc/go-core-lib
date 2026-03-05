package eventbus

import (
	"context"
	"sync"
	"time"

	"github.com/thanvuc/go-core-lib/log"
	"github.com/wagslane/go-rabbitmq"
	"golang.org/x/sync/errgroup"
)

type RabbitMQConnector struct {
	conn      *rabbitmq.Conn
	uri       string
	consumers []*rabbitmq.Consumer
	logger    log.Logger
}

func NewConnector(uri string, logger log.Logger) (*RabbitMQConnector, error) {
	conn, err := rabbitmq.NewConn(
		uri,
		rabbitmq.WithConnectionOptionsLogging,
	)

	if err != nil {
		conn.Close()
		return nil, err
	}

	logger.Info("RabbitMQ connection established", "")

	return &RabbitMQConnector{
		conn:   conn,
		uri:    uri,
		logger: logger,
	}, nil
}

func (r *RabbitMQConnector) Close(wg *sync.WaitGroup) {
	defer wg.Done()

	var consumerWg sync.WaitGroup
	for _, consumer := range r.consumers {
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

	if r.conn != nil {
		r.conn.Close()
	}

	r.logger.Info("RabbitMQ connection, publisher, consumer closed", "")
}

func (r *RabbitMQConnector) Shutdown(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)

	for _, consumer := range r.consumers {
		if consumer == nil {
			continue
		}

		c := consumer

		g.Go(func() error {
			shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()

			c.CloseWithContext(shutdownCtx)
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return err
	}

	if r.conn != nil {
		r.conn.Close()
	}

	r.logger.Info("RabbitMQ connection, publisher, consumer closed", "")

	return nil
}
