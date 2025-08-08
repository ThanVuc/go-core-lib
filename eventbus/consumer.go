package eventbus

import "github.com/thanvuc/go-core-lib/log"

type Consumer interface{}

type consumer struct {
	exchange     ExchangeName
	exchangeType ExchangeType
	routingKey   []string
	queueName    string
	logger       log.Logger
	connector    *RabbitMQConnector
}

func NewConsumer(
	exchange ExchangeName,
	exchangeType ExchangeType,
	routingKey []string,
	connector *RabbitMQConnector,
	queueName string,
) Consumer {
	return &consumer{
		exchange:     exchange,
		exchangeType: exchangeType,
		routingKey:   routingKey,
		logger:       connector.logger,
		connector:    connector,
		queueName:    queueName,
	}
}
