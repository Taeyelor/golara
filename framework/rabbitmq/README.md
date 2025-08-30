# GoLara RabbitMQ Integration

Simple and elegant RabbitMQ integration for GoLara framework, inspired by Laravel's queue system.

## Features

- 🚀 **Simple API** - Laravel-inspired queue operations
- 🔄 **Auto-Reconnection** - Automatic connection recovery
- 🛡️ **Middleware Support** - Logging, retry, timeout, validation
- ⚡ **Concurrent Processing** - Multi-worker message processing
- 🎯 **Job-based Queues** - Type-based job routing
- 📊 **Health Monitoring** - Connection and queue statistics
- 🔧 **Flexible Configuration** - Environment-based configuration

## Quick Start

### 1. Installation

```bash
go get github.com/rabbitmq/amqp091-go
```

### 2. Basic Usage

```go
package main

import (
    "context"
    "log"
    
    "github.com/taeyelor/golara/framework"
    "github.com/taeyelor/golara/framework/rabbitmq"
)

func main() {
    app := framework.NewApplication()
    
    // Connect to RabbitMQ
    rabbit, err := rabbitmq.Connect("amqp://guest:guest@localhost:5672/")
    if err != nil {
        log.Fatal("Failed to connect to RabbitMQ:", err)
    }
    defer rabbit.Close()
    
    // Register in service container
    app.Singleton("rabbitmq", func() interface{} {
        return rabbit
    })
    
    // Push a job to queue
    err = rabbit.PushJob("emails", "send_welcome", map[string]interface{}{
        "email": "user@example.com",
        "name":  "John Doe",
    })
    
    // Listen for jobs
    ctx := context.Background()
    handlers := map[string]rabbitmq.MessageHandler{
        "send_welcome": handleWelcomeEmail,
    }
    
    rabbit.ListenForJobs(ctx, "emails", handlers)
}

func handleWelcomeEmail(delivery *rabbitmq.Delivery) error {
    var job rabbitmq.Job
    if err := delivery.JSON(&job); err != nil {
        return err
    }
    
    // Process the email...
    log.Printf("Sending welcome email: %+v", job.Payload)
    return nil
}
```

## Configuration

### Environment Variables

```env
# RabbitMQ Configuration
RABBITMQ_URL=amqp://guest:guest@localhost:5672/
RABBITMQ_RECONNECT_DELAY=5s
RABBITMQ_RECONNECT_ATTEMPTS=10
RABBITMQ_ENABLE_HEARTBEAT=true
RABBITMQ_HEARTBEAT_INTERVAL=10s
RABBITMQ_CHANNEL_POOL_SIZE=10
RABBITMQ_AUTO_DECLARE_QUEUES=true
RABBITMQ_AUTO_DECLARE_EXCHANGE=true
```

### Programmatic Configuration

```go
config := &rabbitmq.RabbitMQConfig{
    URL:                 "amqp://guest:guest@localhost:5672/",
    ReconnectDelay:      "5s",
    ReconnectAttempts:   10,
    EnableHeartbeat:     true,
    HeartbeatInterval:   "10s",
    ChannelPoolSize:     10,
    AutoDeclareQueues:   true,
    AutoDeclareExchange: true,
}

rabbit, err := rabbitmq.New(config)
```

## Queue Operations

### Simple Queue Operations

```go
// Push data to queue
err := rabbit.Push("my_queue", map[string]string{
    "message": "Hello, World!",
})

// Push different data types
err = rabbit.PushJob("jobs", "process_image", imageData)

// Pop message from queue
delivery, err := rabbit.Pop("my_queue")
if delivery != nil {
    var data map[string]string
    delivery.JSON(&data)
    log.Printf("Received: %+v", data)
    delivery.Ack(false) // Acknowledge the message
}
```

### Queue Management

```go
// Get queue instance
queue, err := rabbit.Queue("my_queue")

// Queue operations
count, err := queue.Count()        // Get message count
empty, err := queue.IsEmpty()      // Check if empty
info, err := queue.Inspect()       // Get queue info
_, err = queue.Purge()             // Remove all messages
```

## Publishing Messages

### Basic Publishing

```go
// Publish to exchange
err := rabbit.Publish("my_exchange", "routing.key", data)

// Publish different types
err = rabbit.PublishString("logs", "info", "Application started")
err = rabbit.PublishBytes("binary", "data", binaryData)
```

### Advanced Publishing

```go
// Create publisher with configuration
publisher, err := rabbit.CreatePublisher(&rabbitmq.PublisherConfig{
    Exchange:     "my_exchange",
    ExchangeType: "topic",
    Durable:      true,
})

// Publish with custom message
message := &rabbitmq.Message{
    Body:        data,
    RoutingKey:  "user.created",
    ContentType: "application/json",
    Persistent:  true,
    Headers: amqp.Table{
        "source": "user-service",
    },
}

err = publisher.Publish(message)
```

## Consuming Messages

### Simple Consumers

```go
// Listen with single worker
ctx := context.Background()
err := rabbit.Listen(ctx, "my_queue", func(delivery *rabbitmq.Delivery) error {
    log.Printf("Received: %s", delivery.String())
    return nil
})

// Listen with multiple workers
err = rabbit.ListenWithWorkers(ctx, "my_queue", 5, handler)
```

### Advanced Consumers

```go
// Create consumer with configuration
consumer, err := rabbit.CreateConsumer(&rabbitmq.ConsumerConfig{
    Queue:         "my_queue",
    Exchange:      "my_exchange",
    RoutingKey:    "*.important",
    Concurrency:   3,
    PrefetchCount: 10,
    AutoAck:       false,
})

// Add middleware
consumer.Use(rabbitmq.WithLogging())
consumer.Use(rabbitmq.WithRetry(3, 5*time.Second))
consumer.Use(rabbitmq.WithTimeout(30*time.Second))

// Handle specific routing keys
consumer.Handle("user.created", handleUserCreated)
consumer.Handle("user.updated", handleUserUpdated)
consumer.HandleAll(handleDefault) // Fallback handler

// Start consuming
err = consumer.Start(ctx)
```

## Job-Based Queues

### Job Structure

```go
type EmailJob struct {
    To      string `json:"to"`
    Subject string `json:"subject"`
    Body    string `json:"body"`
}

// Push job
job := EmailJob{
    To:      "user@example.com",
    Subject: "Welcome!",
    Body:    "Welcome to our service!",
}

err := rabbit.PushJob("emails", "send_email", job)
```

### Job Handlers

```go
// Define job handlers
handlers := map[string]rabbitmq.MessageHandler{
    "send_email":    handleSendEmail,
    "send_sms":      handleSendSMS,
    "process_image": handleImageProcessing,
}

// Start job processor
err := rabbit.ListenForJobs(ctx, "jobs", handlers)

func handleSendEmail(delivery *rabbitmq.Delivery) error {
    var job rabbitmq.Job
    if err := delivery.JSON(&job); err != nil {
        return err
    }
    
    // Type assertion to get email data
    emailData := job.Payload.(EmailJob)
    
    // Send email...
    log.Printf("Sending email to %s", emailData.To)
    return nil
}
```

## Middleware

### Built-in Middleware

```go
consumer.Use(rabbitmq.WithLogging())                    // Request logging
consumer.Use(rabbitmq.WithRecovery())                   // Panic recovery
consumer.Use(rabbitmq.WithRetry(3, 5*time.Second))      // Retry failed messages
consumer.Use(rabbitmq.WithTimeout(30*time.Second))      // Processing timeout
consumer.Use(rabbitmq.WithRateLimit(10))                // Rate limiting
consumer.Use(rabbitmq.WithDeduplication(time.Hour))     // Deduplication
```

### Custom Middleware

```go
func customMiddleware(next rabbitmq.MessageHandler) rabbitmq.MessageHandler {
    return func(delivery *rabbitmq.Delivery) error {
        start := time.Now()
        
        // Pre-processing
        log.Printf("Processing message: %s", delivery.MessageId)
        
        // Execute handler
        err := next(delivery)
        
        // Post-processing
        duration := time.Since(start)
        log.Printf("Message processed in %v", duration)
        
        return err
    }
}

consumer.Use(customMiddleware)
```

## Exchange Management

```go
// Declare exchanges
err := rabbit.DeclareExchange("logs", "topic", true)
err = rabbit.DeclareExchange("events", "fanout", true)

// Bind queue to exchange
queue, _ := rabbit.Queue("error_logs")
err = queue.Bind("logs", "error.*", nil)
```

## Health Monitoring

```go
// Check connection health
err := rabbit.Health()
if err != nil {
    log.Printf("RabbitMQ health check failed: %v", err)
}

// Get statistics
stats := rabbit.Stats()
log.Printf("Connected: %v", stats["connected"])
log.Printf("Publishers: %v", stats["total_publishers"])
log.Printf("Consumers: %v", stats["total_consumers"])

// Queue information
queue, _ := rabbit.Queue("my_queue")
info, _ := queue.Inspect()
log.Printf("Queue '%s' has %d messages and %d consumers", 
    info.Name, info.Messages, info.Consumers)
```

## Error Handling

### Error Types

```go
// Check error types
if rabbitmq.IsConnectionError(err) {
    log.Println("Connection error, will retry...")
}

if rabbitmq.IsRetryableError(err) {
    log.Println("Retryable error")
}

if rabbitmq.IsTemporaryError(err) {
    log.Println("Temporary error")
}
```

### Custom Error Handling

```go
consumer.HandleAll(func(delivery *rabbitmq.Delivery) error {
    err := processMessage(delivery)
    if err != nil {
        if rabbitmq.IsRetryableError(err) {
            return err // Will be retried by retry middleware
        }
        
        // Log error and acknowledge message to prevent requeue
        log.Printf("Non-retryable error: %v", err)
        return nil
    }
    
    return nil
})
```

## Integration with GoLara

### Service Registration

```go
// In your application bootstrap
func RegisterServices(app *framework.Application) {
    // Register RabbitMQ
    app.Singleton("rabbitmq", func() interface{} {
        config := &rabbitmq.RabbitMQConfig{
            URL: app.Config.GetString("rabbitmq.url", "amqp://localhost:5672/"),
        }
        
        rabbit, err := rabbitmq.New(config)
        if err != nil {
            log.Fatal("Failed to connect to RabbitMQ:", err)
        }
        
        return rabbit
    })
}
```

### Controller Usage

```go
func SendEmailController(c *routing.Context) {
    rabbit := c.App.Resolve("rabbitmq").(*rabbitmq.RabbitMQ)
    
    var emailData EmailJob
    if err := c.Bind(&emailData); err != nil {
        c.JSON(400, map[string]string{"error": "Invalid data"})
        return
    }
    
    err := rabbit.PushJob("emails", "send_email", emailData)
    if err != nil {
        c.JSON(500, map[string]string{"error": "Failed to queue email"})
        return
    }
    
    c.JSON(200, map[string]string{"message": "Email queued successfully"})
}
```

## Production Deployment

### Docker Compose

```yaml
version: '3.8'
services:
  rabbitmq:
    image: rabbitmq:3-management
    container_name: rabbitmq
    ports:
      - "5672:5672"
      - "15672:15672"
    environment:
      RABBITMQ_DEFAULT_USER: admin
      RABBITMQ_DEFAULT_PASS: password
    volumes:
      - rabbitmq_data:/var/lib/rabbitmq

  app:
    build: .
    environment:
      RABBITMQ_URL: amqp://admin:password@rabbitmq:5672/
    depends_on:
      - rabbitmq

volumes:
  rabbitmq_data:
```

### Monitoring and Logging

```go
// Add comprehensive logging
consumer.Use(func(next rabbitmq.MessageHandler) rabbitmq.MessageHandler {
    return func(delivery *rabbitmq.Delivery) error {
        start := time.Now()
        messageId := delivery.MessageId
        routingKey := delivery.RoutingKey
        
        log.Printf("[RABBITMQ] Processing message %s from %s", messageId, routingKey)
        
        err := next(delivery)
        
        duration := time.Since(start)
        if err != nil {
            log.Printf("[RABBITMQ] Message %s failed after %v: %v", messageId, duration, err)
        } else {
            log.Printf("[RABBITMQ] Message %s processed successfully in %v", messageId, duration)
        }
        
        return err
    }
})
```

## Best Practices

1. **Always use middleware** for logging, recovery, and retry
2. **Set appropriate timeouts** for message processing
3. **Use job-based queues** for complex workflows
4. **Monitor queue health** and message counts
5. **Handle errors gracefully** with proper acknowledgments
6. **Use connection pooling** for high-throughput applications
7. **Set up proper monitoring** and alerting
8. **Test failure scenarios** with network interruptions

## Example Application

See `/examples/rabbitmq/main.go` for a complete example showing:

- Web API endpoints that queue jobs
- Background job processors
- Middleware usage
- Health monitoring
- Error handling

This integration makes RabbitMQ usage in GoLara as simple and elegant as Laravel's queue system while maintaining the performance benefits of Go.
