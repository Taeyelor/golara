package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"runtime"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Consumer handles message consumption from RabbitMQ
type Consumer struct {
	conn          *Connection
	queue         string
	exchange      string
	routingKey    string
	consumerTag   string
	durable       bool
	autoDelete    bool
	exclusive     bool
	noWait        bool
	args          amqp.Table
	concurrency   int
	prefetchCount int
	autoAck       bool
	handlers      map[string]MessageHandler
	middleware    []MiddlewareFunc
	isRunning     bool
	stopCh        chan struct{}
	wg            sync.WaitGroup
}

// ConsumerConfig holds consumer configuration
type ConsumerConfig struct {
	Queue         string
	Exchange      string
	RoutingKey    string
	ConsumerTag   string
	Durable       bool
	AutoDelete    bool
	Exclusive     bool
	NoWait        bool
	Args          amqp.Table
	Concurrency   int
	PrefetchCount int
	AutoAck       bool
}

// Delivery wraps amqp.Delivery with additional helper methods
type Delivery struct {
	*amqp.Delivery
	ctx context.Context
}

// MessageHandler defines the interface for message handlers
type MessageHandler func(*Delivery) error

// MiddlewareFunc defines middleware function signature
type MiddlewareFunc func(MessageHandler) MessageHandler

// NewConsumer creates a new consumer
func NewConsumer(conn *Connection, config *ConsumerConfig) (*Consumer, error) {
	if config == nil {
		config = &ConsumerConfig{
			Queue:         "golara_default_queue",
			Exchange:      "golara_default",
			RoutingKey:    "",
			ConsumerTag:   "",
			Durable:       true,
			AutoDelete:    false,
			Exclusive:     false,
			NoWait:        false,
			Args:          nil,
			Concurrency:   runtime.NumCPU(),
			PrefetchCount: 10,
			AutoAck:       false,
		}
	}

	// Set default concurrency
	if config.Concurrency <= 0 {
		config.Concurrency = runtime.NumCPU()
	}

	consumer := &Consumer{
		conn:          conn,
		queue:         config.Queue,
		exchange:      config.Exchange,
		routingKey:    config.RoutingKey,
		consumerTag:   config.ConsumerTag,
		durable:       config.Durable,
		autoDelete:    config.AutoDelete,
		exclusive:     config.Exclusive,
		noWait:        config.NoWait,
		args:          config.Args,
		concurrency:   config.Concurrency,
		prefetchCount: config.PrefetchCount,
		autoAck:       config.AutoAck,
		handlers:      make(map[string]MessageHandler),
		middleware:    make([]MiddlewareFunc, 0),
		stopCh:        make(chan struct{}),
	}

	// Declare queue if auto-declare is enabled
	if conn.config.AutoDeclareQueues {
		if err := consumer.declareQueue(); err != nil {
			return nil, fmt.Errorf("failed to declare queue: %w", err)
		}
	}

	return consumer, nil
}

// declareQueue declares the queue and binds it to exchange
func (c *Consumer) declareQueue() error {
	ch, err := c.conn.NewChannel()
	if err != nil {
		return err
	}
	defer ch.Close()

	// Declare queue
	_, err = ch.QueueDeclare(
		c.queue,      // name
		c.durable,    // durable
		c.autoDelete, // delete when unused
		c.exclusive,  // exclusive
		c.noWait,     // no-wait
		c.args,       // arguments
	)
	if err != nil {
		return err
	}

	// Bind queue to exchange if exchange is specified
	if c.exchange != "" {
		err = ch.QueueBind(
			c.queue,      // queue name
			c.routingKey, // routing key
			c.exchange,   // exchange
			c.noWait,     // no-wait
			nil,          // args
		)
		if err != nil {
			return err
		}
	}

	return nil
}

// Handle registers a message handler for a specific routing key
func (c *Consumer) Handle(routingKey string, handler MessageHandler) {
	c.handlers[routingKey] = handler
}

// HandleAll registers a default handler for all messages
func (c *Consumer) HandleAll(handler MessageHandler) {
	c.handlers["*"] = handler
}

// Use adds middleware to the consumer
func (c *Consumer) Use(middleware MiddlewareFunc) {
	c.middleware = append(c.middleware, middleware)
}

// Start starts consuming messages
func (c *Consumer) Start(ctx context.Context) error {
	if c.isRunning {
		return fmt.Errorf("consumer is already running")
	}

	c.isRunning = true
	log.Printf("RabbitMQ Consumer: Starting consumer for queue '%s' with %d workers", c.queue, c.concurrency)

	// Start workers
	for i := 0; i < c.concurrency; i++ {
		c.wg.Add(1)
		go c.worker(ctx, i)
	}

	// Wait for stop signal or context cancellation
	select {
	case <-ctx.Done():
		log.Println("RabbitMQ Consumer: Context cancelled, stopping...")
	case <-c.stopCh:
		log.Println("RabbitMQ Consumer: Stop signal received")
	}

	c.isRunning = false
	close(c.stopCh)
	c.wg.Wait()

	log.Println("RabbitMQ Consumer: All workers stopped")
	return nil
}

// Stop stops the consumer
func (c *Consumer) Stop() {
	if c.isRunning {
		close(c.stopCh)
	}
}

// worker processes messages in a separate goroutine
func (c *Consumer) worker(ctx context.Context, workerID int) {
	defer c.wg.Done()

	log.Printf("RabbitMQ Consumer: Worker %d started", workerID)

	for {
		select {
		case <-ctx.Done():
			log.Printf("RabbitMQ Consumer: Worker %d stopped (context cancelled)", workerID)
			return
		case <-c.stopCh:
			log.Printf("RabbitMQ Consumer: Worker %d stopped (stop signal)", workerID)
			return
		default:
			if err := c.processMessages(ctx, workerID); err != nil {
				log.Printf("RabbitMQ Consumer: Worker %d error: %v", workerID, err)
				// Add small delay before retrying
				select {
				case <-ctx.Done():
					return
				case <-c.stopCh:
					return
				default:
					// Continue processing
				}
			}
		}
	}
}

// processMessages handles the actual message processing
func (c *Consumer) processMessages(ctx context.Context, workerID int) error {
	ch, err := c.conn.NewChannel()
	if err != nil {
		return fmt.Errorf("failed to get channel: %w", err)
	}
	defer ch.Close()

	// Set QoS (prefetch count)
	if err := ch.Qos(c.prefetchCount, 0, false); err != nil {
		return fmt.Errorf("failed to set QoS: %w", err)
	}

	// Start consuming
	deliveries, err := ch.Consume(
		c.queue,       // queue
		c.consumerTag, // consumer
		c.autoAck,     // auto-ack
		c.exclusive,   // exclusive
		false,         // no-local
		c.noWait,      // no-wait
		c.args,        // args
	)
	if err != nil {
		return fmt.Errorf("failed to start consuming: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-c.stopCh:
			return nil
		case delivery, ok := <-deliveries:
			if !ok {
				return fmt.Errorf("delivery channel closed")
			}

			// Wrap delivery
			d := &Delivery{
				Delivery: &delivery,
				ctx:      ctx,
			}

			// Process message
			if err := c.handleMessage(d); err != nil {
				log.Printf("RabbitMQ Consumer: Error processing message: %v", err)
				if !c.autoAck {
					d.Nack(false, true) // Requeue the message
				}
			}
		}
	}
}

// handleMessage processes a single message
func (c *Consumer) handleMessage(delivery *Delivery) error {
	// Find appropriate handler
	handler := c.findHandler(delivery.RoutingKey)
	if handler == nil {
		log.Printf("RabbitMQ Consumer: No handler found for routing key: %s", delivery.RoutingKey)
		if !c.autoAck {
			delivery.Ack(false)
		}
		return nil
	}

	// Apply middleware
	finalHandler := handler
	for i := len(c.middleware) - 1; i >= 0; i-- {
		finalHandler = c.middleware[i](finalHandler)
	}

	// Execute handler
	if err := finalHandler(delivery); err != nil {
		return err
	}

	// Acknowledge message if not auto-ack
	if !c.autoAck {
		return delivery.Ack(false)
	}

	return nil
}

// findHandler finds the appropriate handler for a routing key
func (c *Consumer) findHandler(routingKey string) MessageHandler {
	// Try exact match first
	if handler, exists := c.handlers[routingKey]; exists {
		return handler
	}

	// Try wildcard handler
	if handler, exists := c.handlers["*"]; exists {
		return handler
	}

	return nil
}

// Helper methods for Delivery

// JSON unmarshals the message body as JSON
func (d *Delivery) JSON(v interface{}) error {
	return json.Unmarshal(d.Body, v)
}

// String returns the message body as string
func (d *Delivery) String() string {
	return string(d.Body)
}

// Bytes returns the message body as bytes
func (d *Delivery) Bytes() []byte {
	return d.Body
}

// Context returns the context associated with the delivery
func (d *Delivery) Context() context.Context {
	return d.ctx
}

// GetHeader gets a header value
func (d *Delivery) GetHeader(key string) (interface{}, bool) {
	val, exists := d.Headers[key]
	return val, exists
}

// GetStringHeader gets a header value as string
func (d *Delivery) GetStringHeader(key string) (string, bool) {
	if val, exists := d.Headers[key]; exists {
		if str, ok := val.(string); ok {
			return str, true
		}
	}
	return "", false
}
