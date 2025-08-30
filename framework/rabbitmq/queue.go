package rabbitmq

import (
	"context"
	"fmt"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Queue represents a simple queue interface for common operations
type Queue struct {
	conn       *Connection
	name       string
	durable    bool
	autoDelete bool
	exclusive  bool
	noWait     bool
	args       amqp.Table
}

// QueueConfig holds queue configuration
type QueueConfig struct {
	Name       string
	Durable    bool
	AutoDelete bool
	Exclusive  bool
	NoWait     bool
	Args       amqp.Table
}

// QueueInfo holds information about a queue
type QueueInfo struct {
	Name      string
	Messages  int
	Consumers int
}

// NewQueue creates a new queue manager
func NewQueue(conn *Connection, config *QueueConfig) (*Queue, error) {
	if config == nil {
		config = &QueueConfig{
			Name:       "golara_queue",
			Durable:    true,
			AutoDelete: false,
			Exclusive:  false,
			NoWait:     false,
			Args:       nil,
		}
	}

	queue := &Queue{
		conn:       conn,
		name:       config.Name,
		durable:    config.Durable,
		autoDelete: config.AutoDelete,
		exclusive:  config.Exclusive,
		noWait:     config.NoWait,
		args:       config.Args,
	}

	// Declare queue if auto-declare is enabled
	if conn.config.AutoDeclareQueues {
		if err := queue.Declare(); err != nil {
			return nil, fmt.Errorf("failed to declare queue: %w", err)
		}
	}

	return queue, nil
}

// Declare declares the queue
func (q *Queue) Declare() error {
	ch, err := q.conn.NewChannel()
	if err != nil {
		return err
	}
	defer ch.Close()

	_, err = ch.QueueDeclare(
		q.name,       // name
		q.durable,    // durable
		q.autoDelete, // delete when unused
		q.exclusive,  // exclusive
		q.noWait,     // no-wait
		q.args,       // arguments
	)

	if err == nil {
		log.Printf("RabbitMQ Queue: Declared queue '%s'", q.name)
	}

	return err
}

// Purge removes all messages from the queue
func (q *Queue) Purge() (int, error) {
	ch, err := q.conn.NewChannel()
	if err != nil {
		return 0, err
	}
	defer ch.Close()

	count, err := ch.QueuePurge(q.name, false)
	if err == nil {
		log.Printf("RabbitMQ Queue: Purged %d messages from queue '%s'", count, q.name)
	}

	return count, err
}

// Delete deletes the queue
func (q *Queue) Delete(ifUnused, ifEmpty bool) (int, error) {
	ch, err := q.conn.NewChannel()
	if err != nil {
		return 0, err
	}
	defer ch.Close()

	count, err := ch.QueueDelete(q.name, ifUnused, ifEmpty, false)
	if err == nil {
		log.Printf("RabbitMQ Queue: Deleted queue '%s' with %d messages", q.name, count)
	}

	return count, err
}

// Inspect returns information about the queue
func (q *Queue) Inspect() (*QueueInfo, error) {
	ch, err := q.conn.NewChannel()
	if err != nil {
		return nil, err
	}
	defer ch.Close()

	inspection, err := ch.QueueInspect(q.name)
	if err != nil {
		return nil, err
	}

	return &QueueInfo{
		Name:      inspection.Name,
		Messages:  inspection.Messages,
		Consumers: inspection.Consumers,
	}, nil
}

// Bind binds the queue to an exchange
func (q *Queue) Bind(exchange, routingKey string, args amqp.Table) error {
	ch, err := q.conn.NewChannel()
	if err != nil {
		return err
	}
	defer ch.Close()

	err = ch.QueueBind(
		q.name,     // queue name
		routingKey, // routing key
		exchange,   // exchange
		false,      // no-wait
		args,       // args
	)

	if err == nil {
		log.Printf("RabbitMQ Queue: Bound queue '%s' to exchange '%s' with routing key '%s'", q.name, exchange, routingKey)
	}

	return err
}

// Unbind unbinds the queue from an exchange
func (q *Queue) Unbind(exchange, routingKey string, args amqp.Table) error {
	ch, err := q.conn.NewChannel()
	if err != nil {
		return err
	}
	defer ch.Close()

	err = ch.QueueUnbind(
		q.name,     // queue name
		routingKey, // routing key
		exchange,   // exchange
		args,       // args
	)

	if err == nil {
		log.Printf("RabbitMQ Queue: Unbound queue '%s' from exchange '%s' with routing key '%s'", q.name, exchange, routingKey)
	}

	return err
}

// Push pushes a simple message to the queue (publishes to default exchange)
func (q *Queue) Push(data interface{}) error {
	publisher, err := NewPublisher(q.conn, &PublisherConfig{
		Exchange:     "", // Default exchange
		ExchangeType: "direct",
		Durable:      true,
	})
	if err != nil {
		return err
	}

	return publisher.PublishJSON(q.name, data)
}

// PushString pushes a string message to the queue
func (q *Queue) PushString(data string) error {
	publisher, err := NewPublisher(q.conn, &PublisherConfig{
		Exchange:     "", // Default exchange
		ExchangeType: "direct",
		Durable:      true,
	})
	if err != nil {
		return err
	}

	return publisher.PublishString(q.name, data)
}

// PushDelayed pushes a delayed message to the queue (requires rabbitmq-delayed-message-exchange plugin)
func (q *Queue) PushDelayed(data interface{}, delay time.Duration) error {
	publisher, err := NewPublisher(q.conn, &PublisherConfig{
		Exchange:     "golara_delayed", // Delayed exchange
		ExchangeType: "x-delayed-message",
		Durable:      true,
	})
	if err != nil {
		return err
	}

	return publisher.PublishDelayed(q.name, data, delay)
}

// Pop pops a single message from the queue
func (q *Queue) Pop(autoAck bool) (*Delivery, error) {
	ch, err := q.conn.NewChannel()
	if err != nil {
		return nil, err
	}

	// Don't defer close here as the delivery might need the channel

	delivery, ok, err := ch.Get(q.name, autoAck)
	if err != nil {
		ch.Close()
		return nil, err
	}

	if !ok {
		ch.Close()
		return nil, nil // No message available
	}

	return &Delivery{
		Delivery: &delivery,
		ctx:      context.Background(),
	}, nil
}

// Listen starts listening for messages with a simple callback
func (q *Queue) Listen(ctx context.Context, handler func(*Delivery) error) error {
	consumer, err := NewConsumer(q.conn, &ConsumerConfig{
		Queue:       q.name,
		Concurrency: 1,
		AutoAck:     false,
	})
	if err != nil {
		return err
	}

	consumer.HandleAll(handler)
	return consumer.Start(ctx)
}

// ListenWithWorkers starts listening with multiple workers
func (q *Queue) ListenWithWorkers(ctx context.Context, workers int, handler func(*Delivery) error) error {
	consumer, err := NewConsumer(q.conn, &ConsumerConfig{
		Queue:       q.name,
		Concurrency: workers,
		AutoAck:     false,
	})
	if err != nil {
		return err
	}

	consumer.HandleAll(handler)
	return consumer.Start(ctx)
}

// Name returns the queue name
func (q *Queue) Name() string {
	return q.name
}

// IsEmpty checks if the queue is empty
func (q *Queue) IsEmpty() (bool, error) {
	info, err := q.Inspect()
	if err != nil {
		return false, err
	}
	return info.Messages == 0, nil
}

// Count returns the number of messages in the queue
func (q *Queue) Count() (int, error) {
	info, err := q.Inspect()
	if err != nil {
		return 0, err
	}
	return info.Messages, nil
}
