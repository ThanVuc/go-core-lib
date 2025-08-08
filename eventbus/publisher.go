package eventbus

import (
	"context"
	"fmt"
	"time"

	"github.com/thanvuc/go-core-lib/log"
	"github.com/wagslane/go-rabbitmq"
	"go.uber.org/zap"
)

type Publisher interface {
	Publish(ctx context.Context, request_id string, body []byte, headers map[string]interface{}) error
	SafetyPublish(ctx context.Context, request_id string, body []byte, headers map[string]interface{}) error
}

type publisher struct {
	sharedPublisher *rabbitmq.Publisher
	exchange        ExchangeName
	routingKey      []string
	logger          log.Logger
	maxRetries      int
	retryDelay      int
	dlqExchange     string
	dlqRoutingKey   string
}

func (p *publisher) Publish(ctx context.Context, request_id string, body []byte, headers map[string]interface{}) error {
	newHeaders := make(map[string]interface{})
	for k, v := range headers {
		newHeaders[k] = v
	}
	newHeaders["request_id"] = request_id

	return p.sharedPublisher.PublishWithContext(
		ctx,
		body,
		p.routingKey,
		rabbitmq.WithPublishOptionsExchange(string(p.exchange)),
		rabbitmq.WithPublishOptionsHeaders(newHeaders),
		rabbitmq.WithPublishOptionsContentType("application/json"),
		rabbitmq.WithPublishOptionsPersistentDelivery,
	)
}

func (p *publisher) SafetyPublish(ctx context.Context, request_id string, body []byte, headers map[string]interface{}) error {
	newHeaders := make(map[string]interface{})
	for k, v := range headers {
		newHeaders[k] = v
	}
	newHeaders["request_id"] = request_id
	for attempt := 0; attempt < p.maxRetries; attempt++ {
		confirms, err := p.sharedPublisher.PublishWithDeferredConfirmWithContext(
			ctx,
			body,
			p.routingKey,
			rabbitmq.WithPublishOptionsExchange(string(p.exchange)),
			rabbitmq.WithPublishOptionsHeaders(newHeaders),
			rabbitmq.WithPublishOptionsContentType("application/json"),
			rabbitmq.WithPublishOptionsMandatory,
			rabbitmq.WithPublishOptionsPersistentDelivery,
		)

		if err != nil {
			p.logger.Error("Failed to publish message", request_id, zap.Error(err))
			if attempt < p.maxRetries-1 {
				time.Sleep(time.Duration(p.retryDelay) * time.Millisecond)
			}
			continue
		}

		if len(confirms) == 0 || confirms[0] == nil {
			p.logger.Error("No confirmation received for message", request_id)
			if attempt < p.maxRetries-1 {
				time.Sleep(time.Duration(p.retryDelay) * time.Millisecond)
			}
			continue
		}

		ok, waitErr := confirms[0].WaitContext(ctx)
		if waitErr != nil {
			p.logger.Error("Failed to wait for confirmation", request_id, zap.Error(waitErr))
			if attempt < p.maxRetries-1 {
				time.Sleep(time.Duration(p.retryDelay) * time.Millisecond)
			}
			continue
		}

		if !ok {
			p.logger.Error("Message was not confirmed by RabbitMQ", request_id)
			if attempt < p.maxRetries-1 {
				time.Sleep(time.Duration(p.retryDelay) * time.Millisecond)
			}
			continue
		}

		p.logger.Info("Message confirmed by RabbitMQ", request_id)
		return nil
	}

	p.logger.Error("Failed to publish message after retries, sending to DLQ", request_id)

	dlqErr := p.sharedPublisher.Publish(
		body,
		[]string{p.dlqRoutingKey},
		rabbitmq.WithPublishOptionsExchange(p.dlqExchange),
		rabbitmq.WithPublishOptionsHeaders(newHeaders),
		rabbitmq.WithPublishOptionsContentType("application/json"),
		rabbitmq.WithPublishOptionsPersistentDelivery,
	)
	if dlqErr != nil {
		p.logger.Error("Failed to send message to DLQ", request_id, zap.Error(dlqErr))
		return dlqErr
	}
	p.logger.Info("Message sent to DLQ", request_id)
	return fmt.Errorf("message was not confirmed after %d attempts, sent to DLQ", p.maxRetries)
}

func NewPublisher(
	connector *RabbitMQConnector,
	exchange ExchangeName,
	routingKey []string,

) Publisher {
	return &publisher{
		sharedPublisher: connector.publisher,
		exchange:        exchange,
		routingKey:      routingKey,
		logger:          connector.logger,
	}
}
