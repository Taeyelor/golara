package rabbitmq

import (
	"context"
	"fmt"
	"log"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Manager is the main RabbitMQ manager that provides a simple interface
type Manager struct {
	conn       *Connection
	publishers map[string]*Publisher
	consumers  map[string]*Consumer
	queues     map[string]*Queue
	mutex      sync.RWMutex
}

// ManagerConfig holds the configuration for the RabbitMQ manager
type ManagerConfig struct {
	URL                 string
	ReconnectDelay      string
	ReconnectAttempts   int
	EnableHeartbeat     bool
	HeartbeatInterval   string
	ChannelPoolSize     int
	AutoDeclareQueues   bool
	AutoDeclareExchange bool
}

// Exchange configuration
type ExchangeConfig struct {
	Name       string
	Type       string
	Durable    bool
	AutoDelete bool
	Internal   bool
	NoWait     bool
	Args       amqp.Table
}

// Job represents a simple job structure
type Job struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// NewManager creates a new RabbitMQ manager
func NewManager(url string, config *Config) (*Manager, error) {
	conn, err := NewConnection(url, config)
	if err != nil {
		return nil, err
	}

	return &Manager{
		conn:       conn,
		publishers: make(map[string]*Publisher),
		consumers:  make(map[string]*Consumer),
		queues:     make(map[string]*Queue),
	}, nil
}

// NewManagerFromConfig creates a new manager from configuration
func NewManagerFromConfig(config *ManagerConfig) (*Manager, error) {
	rabbitConfig := DefaultConfig()
	rabbitConfig.URL = config.URL

	if config.ReconnectAttempts > 0 {
		rabbitConfig.ReconnectAttempts = config.ReconnectAttempts
	}
	if config.ChannelPoolSize > 0 {
		rabbitConfig.ChannelPoolSize = config.ChannelPoolSize
	}

	rabbitConfig.AutoDeclareQueues = config.AutoDeclareQueues
	rabbitConfig.AutoDeclareExchange = config.AutoDeclareExchange

	return NewManager(config.URL, rabbitConfig)
}

// Connection returns the underlying connection
func (m *Manager) Connection() *Connection {
	return m.conn
}

// DeclareExchange declares an exchange
func (m *Manager) DeclareExchange(config *ExchangeConfig) error {
	ch, err := m.conn.NewChannel()
	if err != nil {
		return err
	}
	defer ch.Close()

	err = ch.ExchangeDeclare(
		config.Name,       // name
		config.Type,       // type
		config.Durable,    // durable
		config.AutoDelete, // auto-deleted
		config.Internal,   // internal
		config.NoWait,     // no-wait
		config.Args,       // arguments
	)

	if err == nil {
		log.Printf("RabbitMQ Manager: Declared exchange '%s' of type '%s'", config.Name, config.Type)
	}

	return err
}

// Queue gets or creates a queue
func (m *Manager) Queue(name string, config *QueueConfig) (*Queue, error) {
	m.mutex.RLock()
	if queue, exists := m.queues[name]; exists {
		m.mutex.RUnlock()
		return queue, nil
	}
	m.mutex.RUnlock()

	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Double-check after acquiring write lock
	if queue, exists := m.queues[name]; exists {
		return queue, nil
	}

	if config == nil {
		config = &QueueConfig{
			Name:    name,
			Durable: true,
		}
	} else if config.Name == "" {
		config.Name = name
	}

	queue, err := NewQueue(m.conn, config)
	if err != nil {
		return nil, err
	}

	m.queues[name] = queue
	return queue, nil
}

// Publisher gets or creates a publisher
func (m *Manager) Publisher(exchange string, config *PublisherConfig) (*Publisher, error) {
	m.mutex.RLock()
	if publisher, exists := m.publishers[exchange]; exists {
		m.mutex.RUnlock()
		return publisher, nil
	}
	m.mutex.RUnlock()

	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Double-check after acquiring write lock
	if publisher, exists := m.publishers[exchange]; exists {
		return publisher, nil
	}

	if config == nil {
		config = &PublisherConfig{
			Exchange:     exchange,
			ExchangeType: "direct",
			Durable:      true,
		}
	} else if config.Exchange == "" {
		config.Exchange = exchange
	}

	publisher, err := NewPublisher(m.conn, config)
	if err != nil {
		return nil, err
	}

	m.publishers[exchange] = publisher
	return publisher, nil
}

// Consumer gets or creates a consumer
func (m *Manager) Consumer(queue string, config *ConsumerConfig) (*Consumer, error) {
	m.mutex.RLock()
	if consumer, exists := m.consumers[queue]; exists {
		m.mutex.RUnlock()
		return consumer, nil
	}
	m.mutex.RUnlock()

	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Double-check after acquiring write lock
	if consumer, exists := m.consumers[queue]; exists {
		return consumer, nil
	}

	if config == nil {
		config = &ConsumerConfig{
			Queue:   queue,
			Durable: true,
		}
	} else if config.Queue == "" {
		config.Queue = queue
	}

	consumer, err := NewConsumer(m.conn, config)
	if err != nil {
		return nil, err
	}

	m.consumers[queue] = consumer
	return consumer, nil
}

// Publish publishes a message to an exchange
func (m *Manager) Publish(exchange, routingKey string, data interface{}) error {
	publisher, err := m.Publisher(exchange, nil)
	if err != nil {
		return err
	}

	return publisher.PublishJSON(routingKey, data)
}

// PublishToQueue publishes a message directly to a queue (using default exchange)
func (m *Manager) PublishToQueue(queueName string, data interface{}) error {
	publisher, err := m.Publisher("", &PublisherConfig{
		Exchange:     "", // Default exchange
		ExchangeType: "direct",
		Durable:      true,
	})
	if err != nil {
		return err
	}

	return publisher.PublishJSON(queueName, data)
}

// Consume starts consuming messages from a queue
func (m *Manager) Consume(ctx context.Context, queueName string, handler MessageHandler) error {
	consumer, err := m.Consumer(queueName, nil)
	if err != nil {
		return err
	}

	consumer.HandleAll(handler)
	return consumer.Start(ctx)
}

// ConsumeWithConfig starts consuming with custom configuration
func (m *Manager) ConsumeWithConfig(ctx context.Context, config *ConsumerConfig, handler MessageHandler) error {
	consumer, err := m.Consumer(config.Queue, config)
	if err != nil {
		return err
	}

	consumer.HandleAll(handler)
	return consumer.Start(ctx)
}

// Job processing methods

// PublishJob publishes a job to a queue
func (m *Manager) PublishJob(queueName, jobType string, payload interface{}) error {
	job := &Job{
		Type:    jobType,
		Payload: payload,
	}
	return m.PublishToQueue(queueName, job)
}

// ConsumeJobs starts consuming jobs from a queue
func (m *Manager) ConsumeJobs(ctx context.Context, queueName string, handlers map[string]MessageHandler) error {
	consumer, err := m.Consumer(queueName, nil)
	if err != nil {
		return err
	}

	// Set up job router
	consumer.HandleAll(func(delivery *Delivery) error {
		var job Job
		if err := delivery.JSON(&job); err != nil {
			log.Printf("RabbitMQ Manager: Failed to unmarshal job: %v", err)
			return err
		}

		handler, exists := handlers[job.Type]
		if !exists {
			log.Printf("RabbitMQ Manager: No handler found for job type: %s", job.Type)
			return nil // Acknowledge message but don't process
		}

		return handler(delivery)
	})

	return consumer.Start(ctx)
}

// Utility methods

// IsConnected checks if the connection is active
func (m *Manager) IsConnected() bool {
	return m.conn.IsConnected()
}

// Close closes all connections and cleans up resources
func (m *Manager) Close() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Close all consumers
	for name, consumer := range m.consumers {
		consumer.Stop()
		log.Printf("RabbitMQ Manager: Stopped consumer for queue '%s'", name)
	}

	// Close all publishers
	for name, publisher := range m.publishers {
		publisher.Close()
		log.Printf("RabbitMQ Manager: Closed publisher for exchange '%s'", name)
	}

	// Close connection
	if err := m.conn.Close(); err != nil {
		return err
	}

	log.Println("RabbitMQ Manager: All resources closed")
	return nil
}

// Health checks the health of the RabbitMQ connection
func (m *Manager) Health() error {
	if !m.conn.IsConnected() {
		return fmt.Errorf("RabbitMQ connection is not active")
	}

	// Try to create a temporary channel to test the connection
	ch, err := m.conn.NewChannel()
	if err != nil {
		return fmt.Errorf("failed to create test channel: %w", err)
	}
	defer ch.Close()

	return nil
}

// Stats returns statistics about the manager
func (m *Manager) Stats() map[string]interface{} {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return map[string]interface{}{
		"connected":        m.conn.IsConnected(),
		"total_publishers": len(m.publishers),
		"total_consumers":  len(m.consumers),
		"total_queues":     len(m.queues),
	}
}
