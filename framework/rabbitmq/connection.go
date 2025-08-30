package rabbitmq

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Connection represents a RabbitMQ connection with auto-reconnect
type Connection struct {
	url          string
	conn         *amqp.Connection
	channels     map[string]*amqp.Channel
	channelsMux  sync.RWMutex
	reconnectMux sync.Mutex
	isConnected  bool
	done         chan bool
	notifyClose  chan *amqp.Error
	notifyReady  chan bool
	config       *Config
}

// Config holds RabbitMQ connection configuration
type Config struct {
	URL                 string
	ReconnectDelay      time.Duration
	ReconnectAttempts   int
	EnableHeartbeat     bool
	HeartbeatInterval   time.Duration
	ChannelPoolSize     int
	AutoDeclareQueues   bool
	AutoDeclareExchange bool
}

// DefaultConfig returns default RabbitMQ configuration
func DefaultConfig() *Config {
	return &Config{
		URL:                 "amqp://guest:guest@localhost:5672/",
		ReconnectDelay:      5 * time.Second,
		ReconnectAttempts:   10,
		EnableHeartbeat:     true,
		HeartbeatInterval:   10 * time.Second,
		ChannelPoolSize:     10,
		AutoDeclareQueues:   true,
		AutoDeclareExchange: true,
	}
}

// NewConnection creates a new RabbitMQ connection
func NewConnection(url string, config *Config) (*Connection, error) {
	if config == nil {
		config = DefaultConfig()
	}
	if url != "" {
		config.URL = url
	}

	conn := &Connection{
		url:         config.URL,
		channels:    make(map[string]*amqp.Channel),
		done:        make(chan bool),
		notifyReady: make(chan bool, 1),
		config:      config,
	}

	go conn.handleReconnect()

	// Wait for initial connection
	select {
	case <-conn.notifyReady:
		log.Println("RabbitMQ: Initial connection established")
	case <-time.After(10 * time.Second):
		return nil, fmt.Errorf("failed to establish initial RabbitMQ connection within 10 seconds")
	}

	return conn, nil
}

// connect establishes connection to RabbitMQ
func (c *Connection) connect() error {
	var err error

	config := amqp.Config{
		Heartbeat: c.config.HeartbeatInterval,
		Locale:    "en_US",
	}

	c.conn, err = amqp.DialConfig(c.config.URL, config)
	if err != nil {
		return err
	}

	c.isConnected = true
	c.notifyClose = make(chan *amqp.Error)
	c.conn.NotifyClose(c.notifyClose)

	// Signal that connection is ready
	select {
	case c.notifyReady <- true:
	default:
	}

	log.Println("RabbitMQ: Connected successfully")
	return nil
}

// handleReconnect handles automatic reconnection
func (c *Connection) handleReconnect() {
	for {
		c.reconnectMux.Lock()
		err := c.connect()
		c.reconnectMux.Unlock()

		if err != nil {
			log.Printf("RabbitMQ: Failed to connect: %v. Retrying in %v", err, c.config.ReconnectDelay)
			time.Sleep(c.config.ReconnectDelay)
			continue
		}

		// Wait for connection to close
		select {
		case <-c.done:
			return
		case <-c.notifyClose:
			log.Println("RabbitMQ: Connection lost. Attempting to reconnect...")
			c.isConnected = false
			c.closeChannels()
		}
	}
}

// closeChannels closes all active channels
func (c *Connection) closeChannels() {
	c.channelsMux.Lock()
	defer c.channelsMux.Unlock()

	for name, ch := range c.channels {
		if ch != nil && !ch.IsClosed() {
			ch.Close()
		}
		delete(c.channels, name)
	}
}

// GetChannel returns a channel with the given name, creating it if necessary
func (c *Connection) GetChannel(name string) (*amqp.Channel, error) {
	c.channelsMux.RLock()
	if ch, exists := c.channels[name]; exists && ch != nil && !ch.IsClosed() {
		c.channelsMux.RUnlock()
		return ch, nil
	}
	c.channelsMux.RUnlock()

	c.channelsMux.Lock()
	defer c.channelsMux.Unlock()

	// Double-check after acquiring write lock
	if ch, exists := c.channels[name]; exists && ch != nil && !ch.IsClosed() {
		return ch, nil
	}

	if !c.isConnected {
		return nil, fmt.Errorf("RabbitMQ connection is not available")
	}

	ch, err := c.conn.Channel()
	if err != nil {
		return nil, err
	}

	c.channels[name] = ch
	return ch, nil
}

// NewChannel creates a new channel with auto-generated name
func (c *Connection) NewChannel() (*amqp.Channel, error) {
	name := fmt.Sprintf("channel_%d", time.Now().UnixNano())
	return c.GetChannel(name)
}

// CloseChannel closes a specific channel
func (c *Connection) CloseChannel(name string) error {
	c.channelsMux.Lock()
	defer c.channelsMux.Unlock()

	if ch, exists := c.channels[name]; exists {
		delete(c.channels, name)
		if ch != nil && !ch.IsClosed() {
			return ch.Close()
		}
	}
	return nil
}

// IsConnected returns true if connected to RabbitMQ
func (c *Connection) IsConnected() bool {
	return c.isConnected
}

// Close closes the connection and all channels
func (c *Connection) Close() error {
	if !c.isConnected {
		return nil
	}

	close(c.done)
	c.closeChannels()

	if c.conn != nil && !c.conn.IsClosed() {
		return c.conn.Close()
	}

	c.isConnected = false
	log.Println("RabbitMQ: Connection closed")
	return nil
}

// WaitForConnection waits until connection is established
func (c *Connection) WaitForConnection(timeout time.Duration) error {
	if c.isConnected {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	select {
	case <-c.notifyReady:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("timeout waiting for RabbitMQ connection")
	}
}
