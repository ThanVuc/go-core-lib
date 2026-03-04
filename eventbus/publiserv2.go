package eventbus

import (
	"context"
	"fmt"
	"time"

	"github.com/thanvuc/go-core-lib/log"
	"github.com/wagslane/go-rabbitmq"
	"go.uber.org/zap"
)

type PublisherV2 interface {
	Publish(
		ctx context.Context,
		routingKey []string,
		body []byte,
		opts ...PublishOption,
	) error

	SafetyPublish(
		ctx context.Context,
		routingKey []string,
		body []byte,
		opts ...PublishOption,
	) error
}

type PublishOption func(*publishOptions)

type publishOptions struct {
	requestID     string
	headers       map[string]interface{}
	dlqExchange   *ExchangeName
	dlqRoutingKey *string
}

// Options helper functions
func WithRequestID(requestID string) PublishOption {
	return func(o *publishOptions) {
		o.requestID = requestID
	}
}

func WithHeaders(headers map[string]interface{}) PublishOption {
	return func(o *publishOptions) {
		if o.headers == nil {
			o.headers = make(map[string]interface{})
		}
		for k, v := range headers {
			o.headers[k] = v
		}
	}
}

func WithDLQ(exchange ExchangeName, routingKey string) PublishOption {
	return func(o *publishOptions) {
		o.dlqExchange = &exchange
		o.dlqRoutingKey = &routingKey
	}
}

type publisherV2 struct {
	publisher  *rabbitmq.Publisher
	exchange   ExchangeName
	logger     log.LoggerV2
	maxRetries *int
	retryDelay *int
}

// buildHeaders constructs the headers for the publish operation based on the provided options.
func (p *publisherV2) buildHeaders(opts ...PublishOption) publishOptions {
	options := publishOptions{
		headers: make(map[string]interface{}),
	}

	for _, opt := range opts {
		opt(&options)
	}

	if options.requestID != "" {
		options.headers["request_id"] = options.requestID
	}

	return options
}

func (p *publisherV2) Publish(
	ctx context.Context,
	routingKey []string,
	body []byte,
	opts ...PublishOption,
) error {

	options := p.buildHeaders(opts...)

	return p.publisher.PublishWithContext(
		ctx,
		body,
		routingKey,
		rabbitmq.WithPublishOptionsExchange(string(p.exchange)),
		rabbitmq.WithPublishOptionsHeaders(options.headers),
		rabbitmq.WithPublishOptionsContentType("application/json"),
		rabbitmq.WithPublishOptionsPersistentDelivery,
	)
}

func (p *publisherV2) SafetyPublish(
	ctx context.Context,
	routingKey []string,
	body []byte,
	opts ...PublishOption,
) error {
	options := p.buildHeaders(opts...)

	if p.maxRetries == nil || p.retryDelay == nil {
		return fmt.Errorf("retry configuration is missing")
	}

	for attempt := 0; attempt < *p.maxRetries; attempt++ {

		confirms, err := p.publisher.PublishWithDeferredConfirmWithContext(
			ctx,
			body,
			routingKey,
			rabbitmq.WithPublishOptionsExchange(string(p.exchange)),
			rabbitmq.WithPublishOptionsHeaders(options.headers),
			rabbitmq.WithPublishOptionsContentType("application/json"),
			rabbitmq.WithPublishOptionsMandatory,
			rabbitmq.WithPublishOptionsPersistentDelivery,
		)

		if err != nil {
			p.logger.Error("Failed to publish message",
				log.WithRequestID(options.requestID),
				log.WithFields(zap.Error(err)),
			)
			time.Sleep(time.Duration(*p.retryDelay) * time.Millisecond)
			continue
		}

		if len(confirms) == 0 || confirms[0] == nil {
			p.logger.Error("No confirmation received",
				log.WithRequestID(options.requestID),
			)
			time.Sleep(time.Duration(*p.retryDelay) * time.Millisecond)
			continue
		}

		ok, waitErr := confirms[0].WaitContext(ctx)
		if waitErr != nil || !ok {
			p.logger.Error("Message confirmation failed",
				log.WithRequestID(options.requestID),
				log.WithFields(zap.Error(waitErr)),
			)
			time.Sleep(time.Duration(*p.retryDelay) * time.Millisecond)
			continue
		}

		return nil
	}

	// DLQ handling
	if options.dlqExchange == nil || options.dlqRoutingKey == nil {
		return fmt.Errorf("message failed after retries and no DLQ configured")
	}

	p.logger.Warn("Sending message to DLQ",
		log.WithRequestID(options.requestID),
	)

	dlqErr := p.publisher.Publish(
		body,
		[]string{*options.dlqRoutingKey},
		rabbitmq.WithPublishOptionsExchange(string(*options.dlqExchange)),
		rabbitmq.WithPublishOptionsHeaders(options.headers),
		rabbitmq.WithPublishOptionsContentType("application/json"),
		rabbitmq.WithPublishOptionsPersistentDelivery,
	)

	if dlqErr != nil {
		p.logger.Error("Failed to send to DLQ",
			log.WithRequestID(options.requestID),
			log.WithFields(zap.Error(dlqErr)),
		)
		return dlqErr
	}

	return fmt.Errorf("message failed after %d retries and was sent to DLQ", *p.maxRetries)
}
