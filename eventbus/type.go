package eventbus

type TrackedMessage struct {
	RequestID  string                 `json:"request_id"`
	Body       []byte                 `json:"body"`
	Exchange   string                 `json:"exchange"`
	RoutingKey string                 `json:"routing_key"`
	Queue      string                 `json:"queue"`
	Headers    map[string]interface{} `json:"headers"`
	RetryCount int                    `json:"retry_count"`
	MaxRetries int                    `json:"max_retries"`
	RetryDelay int                    `json:"retry_delay"`
}
