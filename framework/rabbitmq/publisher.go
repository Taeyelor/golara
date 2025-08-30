package rabbitmq

import (
	"encoding/json"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Publisher handles message publishing to RabbitMQ
type Publisher struct {
	conn         *Connection
	exchange     string
	exchangeType string
	durable      bool
	autoDelete   bool
	internal     bool
	noWait       bool
	args         amqp.Table
}

// PublisherConfig holds publisher configuration
type PublisherConfig struct {
	Exchange     string
	ExchangeType string
	Durable      bool
	AutoDelete   bool
	Internal     bool
	NoWait       bool
	Args         amqp.Table
}

// Message represents a message to be published
type Message struct {
	Body        interface{}
	RoutingKey  string
	ContentType string
	Headers     amqp.Table
	Priority    uint8
	Expiration  string
	MessageID   string
	Timestamp   time.Time
	Type        string
	UserID      string
	AppID       string
	Persistent  bool
}

// NewPublisher creates a new publisher
func NewPublisher(conn *Connection, config *PublisherConfig) (*Publisher, error) {
	if config == nil {
		config = &PublisherConfig{
			Exchange:     "golara_default",
			ExchangeType: "direct",
			Durable:      true,
			AutoDelete:   false,
			Internal:     false,
			NoWait:       false,
			Args:         nil,
		}
	}

	publisher := &Publisher{
		conn:         conn,
		exchange:     config.Exchange,
		exchangeType: config.ExchangeType,
		durable:      config.Durable,
		autoDelete:   config.AutoDelete,
		internal:     config.Internal,
		noWait:       config.NoWait,
		args:         config.Args,
	}

	// Declare exchange if auto-declare is enabled
	if conn.config.AutoDeclareExchange {
		if err := publisher.declareExchange(); err != nil {
			return nil, fmt.Errorf("failed to declare exchange: %w", err)
		}
	}

	return publisher, nil
}

// declareExchange declares the exchange
func (p *Publisher) declareExchange() error {
	ch, err := p.conn.NewChannel()
	if err != nil {
		return err
	}
	defer ch.Close()

	return ch.ExchangeDeclare(
		p.exchange,     // name
		p.exchangeType, // type
		p.durable,      // durable
		p.autoDelete,   // auto-deleted
		p.internal,     // internal
		p.noWait,       // no-wait
		p.args,         // arguments
	)
}

// Publish publishes a message
func (p *Publisher) Publish(message *Message) error {
	ch, err := p.conn.NewChannel()
	if err != nil {
		return fmt.Errorf("failed to get channel: %w", err)
	}
	defer ch.Close()

	// Serialize message body
	var body []byte
	switch v := message.Body.(type) {
	case []byte:
		body = v
	case string:
		body = []byte(v)
	default:
		body, err = json.Marshal(v)
		if err != nil {
			return fmt.Errorf("failed to serialize message body: %w", err)
		}
		if message.ContentType == "" {
			message.ContentType = "application/json"
		}
	}

	// Set default content type
	if message.ContentType == "" {
		message.ContentType = "text/plain"
	}

	// Set timestamp if not provided
	if message.Timestamp.IsZero() {
		message.Timestamp = time.Now()
	}

	// Build publishing options
	publishing := amqp.Publishing{
		Headers:      message.Headers,
		ContentType:  message.ContentType,
		Body:         body,
		DeliveryMode: 1, // Non-persistent by default
		Priority:     message.Priority,
		Expiration:   message.Expiration,
		MessageId:    message.MessageID,
		Timestamp:    message.Timestamp,
		Type:         message.Type,
		UserId:       message.UserID,
		AppId:        message.AppID,
	}

	// Set persistent delivery if requested
	if message.Persistent {
		publishing.DeliveryMode = 2
	}

	// Publish the message
	return ch.Publish(
		p.exchange,         // exchange
		message.RoutingKey, // routing key
		false,              // mandatory
		false,              // immediate
		publishing,         // message
	)
}

// PublishJSON publishes a JSON message
func (p *Publisher) PublishJSON(routingKey string, data interface{}) error {
	message := &Message{
		Body:        data,
		RoutingKey:  routingKey,
		ContentType: "application/json",
		Persistent:  true,
	}
	return p.Publish(message)
}

// PublishString publishes a string message
func (p *Publisher) PublishString(routingKey, data string) error {
	message := &Message{
		Body:        data,
		RoutingKey:  routingKey,
		ContentType: "text/plain",
		Persistent:  true,
	}
	return p.Publish(message)
}

// PublishBytes publishes raw bytes
func (p *Publisher) PublishBytes(routingKey string, data []byte) error {
	message := &Message{
		Body:        data,
		RoutingKey:  routingKey,
		ContentType: "application/octet-stream",
		Persistent:  true,
	}
	return p.Publish(message)
}

// PublishWithHeaders publishes a message with custom headers
func (p *Publisher) PublishWithHeaders(routingKey string, data interface{}, headers amqp.Table) error {
	message := &Message{
		Body:        data,
		RoutingKey:  routingKey,
		ContentType: "application/json",
		Headers:     headers,
		Persistent:  true,
	}
	return p.Publish(message)
}

// PublishDelayed publishes a message with delay (requires rabbitmq-delayed-message-exchange plugin)
func (p *Publisher) PublishDelayed(routingKey string, data interface{}, delay time.Duration) error {
	headers := amqp.Table{
		"x-delay": int64(delay.Milliseconds()),
	}

	message := &Message{
		Body:        data,
		RoutingKey:  routingKey,
		ContentType: "application/json",
		Headers:     headers,
		Persistent:  true,
	}
	return p.Publish(message)
}

// Close closes the publisher (no-op for now, but kept for future use)
func (p *Publisher) Close() error {
	return nil
}
