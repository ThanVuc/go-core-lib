package eventbus

type ExchangeType string

const (
	ExchangeTypeDirect  ExchangeType = "direct"
	ExchangeTypeFanout  ExchangeType = "fanout"
	ExchangeTypeTopic   ExchangeType = "topic"
	ExchangeTypeHeaders ExchangeType = "headers"
	ExchangeTypeDefault ExchangeType = "default"
)

type ExchangeName string

const (
	SyncDatabaseExchange ExchangeName = "sync_database"
)
