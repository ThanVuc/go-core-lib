package eventbus

type ExchangeType string

const (
	ExchangeTypeDirect  ExchangeType = "direct"
	ExchangeTypeFanout  ExchangeType = "fanout"
	ExchangeTypeTopic   ExchangeType = "topic"
	ExchangeTypeHeaders ExchangeType = "headers"
	ExchangeTypeDefault ExchangeType = "default"
)

const (
	CheckHealthExchange     ExchangeName = "check_health"
	SyncDatabaseExchange    ExchangeName = "sync_database"
	DLQCheckHealthExchange  ExchangeName = "dlq_check_health"
	DLQSyncDatabaseExchange ExchangeName = "dlq_sync_database"
)
