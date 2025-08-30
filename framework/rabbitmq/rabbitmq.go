// Package rabbitmq provides a simple, Laravel-inspired RabbitMQ integration for GoLara
package rabbitmq

import (
	"context"
	"time"
)

// RabbitMQ provides a simple interface for common RabbitMQ operations
type RabbitMQ struct {
	manager *Manager
}

// Config represents the configuration for RabbitMQ
type RabbitMQConfig struct {
	URL                 string `json:"url" yaml:"url"`
	ReconnectDelay      string `json:"reconnect_delay" yaml:"reconnect_delay"`
	ReconnectAttempts   int    `json:"reconnect_attempts" yaml:"reconnect_attempts"`
	EnableHeartbeat     bool   `json:"enable_heartbeat" yaml:"enable_heartbeat"`
	HeartbeatInterval   string `json:"heartbeat_interval" yaml:"heartbeat_interval"`
	ChannelPoolSize     int    `json:"channel_pool_size" yaml:"channel_pool_size"`
	AutoDeclareQueues   bool   `json:"auto_declare_queues" yaml:"auto_declare_queues"`
	AutoDeclareExchange bool   `json:"auto_declare_exchange" yaml:"auto_declare_exchange"`
}

// DefaultRabbitMQConfig returns the default configuration
func DefaultRabbitMQConfig() *RabbitMQConfig {
	return &RabbitMQConfig{
		URL:                 "amqp://guest:guest@localhost:5672/",
		ReconnectDelay:      "5s",
		ReconnectAttempts:   10,
		EnableHeartbeat:     true,
		HeartbeatInterval:   "10s",
		ChannelPoolSize:     10,
		AutoDeclareQueues:   true,
		AutoDeclareExchange: true,
	}
}

// New creates a new RabbitMQ instance
func New(config *RabbitMQConfig) (*RabbitMQ, error) {
	if config == nil {
		config = DefaultRabbitMQConfig()
	}

	// Convert to manager config
	managerConfig := &ManagerConfig{
		URL:                 config.URL,
		ReconnectDelay:      config.ReconnectDelay,
		ReconnectAttempts:   config.ReconnectAttempts,
		EnableHeartbeat:     config.EnableHeartbeat,
		HeartbeatInterval:   config.HeartbeatInterval,
		ChannelPoolSize:     config.ChannelPoolSize,
		AutoDeclareQueues:   config.AutoDeclareQueues,
		AutoDeclareExchange: config.AutoDeclareExchange,
	}

	manager, err := NewManagerFromConfig(managerConfig)
	if err != nil {
		return nil, err
	}

	return &RabbitMQ{
		manager: manager,
	}, nil
}

// Connect creates a new RabbitMQ connection with simple URL
func Connect(url string) (*RabbitMQ, error) {
	config := DefaultRabbitMQConfig()
	config.URL = url
	return New(config)
}

// Queue operations

// Queue gets or creates a queue
func (r *RabbitMQ) Queue(name string) (*Queue, error) {
	return r.manager.Queue(name, nil)
}

// QueueWithConfig gets or creates a queue with custom configuration
func (r *RabbitMQ) QueueWithConfig(name string, config *QueueConfig) (*Queue, error) {
	return r.manager.Queue(name, config)
}

// Push pushes data to a queue
func (r *RabbitMQ) Push(queueName string, data interface{}) error {
	return r.manager.PublishToQueue(queueName, data)
}

// PushJob pushes a job to a queue
func (r *RabbitMQ) PushJob(queueName, jobType string, payload interface{}) error {
	return r.manager.PublishJob(queueName, jobType, payload)
}

// Pop pops a message from a queue
func (r *RabbitMQ) Pop(queueName string) (*Delivery, error) {
	queue, err := r.Queue(queueName)
	if err != nil {
		return nil, err
	}
	return queue.Pop(false)
}

// Publishing operations

// Publish publishes a message to an exchange
func (r *RabbitMQ) Publish(exchange, routingKey string, data interface{}) error {
	return r.manager.Publish(exchange, routingKey, data)
}

// PublishString publishes a string message
func (r *RabbitMQ) PublishString(exchange, routingKey, data string) error {
	publisher, err := r.manager.Publisher(exchange, nil)
	if err != nil {
		return err
	}
	return publisher.PublishString(routingKey, data)
}

// PublishBytes publishes raw bytes
func (r *RabbitMQ) PublishBytes(exchange, routingKey string, data []byte) error {
	publisher, err := r.manager.Publisher(exchange, nil)
	if err != nil {
		return err
	}
	return publisher.PublishBytes(routingKey, data)
}

// Consumer operations

// Listen starts listening to a queue with a simple callback
func (r *RabbitMQ) Listen(ctx context.Context, queueName string, handler func(*Delivery) error) error {
	return r.manager.Consume(ctx, queueName, handler)
}

// ListenWithWorkers starts listening with multiple workers
func (r *RabbitMQ) ListenWithWorkers(ctx context.Context, queueName string, workers int, handler func(*Delivery) error) error {
	config := &ConsumerConfig{
		Queue:       queueName,
		Concurrency: workers,
		AutoAck:     false,
	}
	return r.manager.ConsumeWithConfig(ctx, config, handler)
}

// ListenForJobs starts listening for jobs with type-based routing
func (r *RabbitMQ) ListenForJobs(ctx context.Context, queueName string, handlers map[string]MessageHandler) error {
	return r.manager.ConsumeJobs(ctx, queueName, handlers)
}

// Advanced operations

// CreateConsumer creates a consumer with advanced configuration
func (r *RabbitMQ) CreateConsumer(config *ConsumerConfig) (*Consumer, error) {
	return r.manager.Consumer(config.Queue, config)
}

// CreatePublisher creates a publisher with advanced configuration
func (r *RabbitMQ) CreatePublisher(config *PublisherConfig) (*Publisher, error) {
	return r.manager.Publisher(config.Exchange, config)
}

// DeclareExchange declares an exchange
func (r *RabbitMQ) DeclareExchange(name, exchangeType string, durable bool) error {
	config := &ExchangeConfig{
		Name:       name,
		Type:       exchangeType,
		Durable:    durable,
		AutoDelete: false,
		Internal:   false,
		NoWait:     false,
	}
	return r.manager.DeclareExchange(config)
}

// Utility methods

// IsConnected checks if the connection is active
func (r *RabbitMQ) IsConnected() bool {
	return r.manager.IsConnected()
}

// Health checks the health of the connection
func (r *RabbitMQ) Health() error {
	return r.manager.Health()
}

// Stats returns connection statistics
func (r *RabbitMQ) Stats() map[string]interface{} {
	return r.manager.Stats()
}

// Close closes all connections and resources
func (r *RabbitMQ) Close() error {
	return r.manager.Close()
}

// Manager returns the underlying manager for advanced operations
func (r *RabbitMQ) Manager() *Manager {
	return r.manager
}

// Helper functions for creating middleware

// WithLogging adds logging middleware
func WithLogging() MiddlewareFunc {
	return LoggingMiddleware
}

// WithRetry adds retry middleware
func WithRetry(maxRetries int, delay time.Duration) MiddlewareFunc {
	return RetryMiddleware(maxRetries, delay)
}

// WithTimeout adds timeout middleware
func WithTimeout(timeout time.Duration) MiddlewareFunc {
	return TimeoutMiddleware(timeout)
}

// WithRecovery adds panic recovery middleware
func WithRecovery() MiddlewareFunc {
	return RecoveryMiddleware
}

// WithRateLimit adds rate limiting middleware
func WithRateLimit(requestsPerSecond int) MiddlewareFunc {
	return RateLimitMiddleware(requestsPerSecond)
}

// WithValidation adds validation middleware
func WithValidation(validator func(*Delivery) error) MiddlewareFunc {
	return ValidationMiddleware(validator)
}

// WithDeduplication adds deduplication middleware
func WithDeduplication(ttl time.Duration) MiddlewareFunc {
	store := NewInMemoryMessageStore(ttl)
	return DeduplicationMiddleware(store)
}
