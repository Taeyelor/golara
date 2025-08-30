# GoLara RabbitMQ - Quick Start Guide

This guide shows you how to add RabbitMQ support to your GoLara application in just a few steps.

## 1. Setup RabbitMQ

### Using Docker (Recommended)

```bash
docker run -d --name rabbitmq \
  -p 5672:5672 \
  -p 15672:15672 \
  -e RABBITMQ_DEFAULT_USER=admin \
  -e RABBITMQ_DEFAULT_PASS=password \
  rabbitmq:3-management
```

Access management UI at: http://localhost:15672 (admin/password)

### Using Package Manager

```bash
# macOS
brew install rabbitmq

# Ubuntu/Debian
sudo apt-get install rabbitmq-server

# CentOS/RHEL
sudo yum install rabbitmq-server
```

## 2. Add to Your GoLara Application

### Basic Setup

```go
package main

import (
    "context"
    "log"
    
    "github.com/taeyelor/golara/framework"
    "github.com/taeyelor/golara/framework/rabbitmq"
    "github.com/taeyelor/golara/framework/routing"
)

func main() {
    app := framework.NewApplication()
    
    // Register RabbitMQ using unified configuration
    // This automatically loads from .env and defaults
    rabbitmq.RegisterRabbitMQFromEnv(app)
    
    // Your routes
    app.POST("/send-email", sendEmailHandler)
    app.GET("/queue-stats", queueStatsHandler)
    
    // Start background worker
    go startEmailWorker(app)
    
    app.Run(":8080")
}
```

### Environment Configuration (.env)

```env
# Application
APP_NAME=MyGoLaraApp
APP_PORT=:8080

# RabbitMQ (all optional - has sensible defaults)
RABBITMQ_URL=amqp://admin:password@localhost:5672/
RABBITMQ_AUTO_DECLARE_QUEUES=true
RABBITMQ_AUTO_DECLARE_EXCHANGE=true
RABBITMQ_CHANNEL_POOL_SIZE=10
```

## 3. Queue Jobs in Controllers

```go
func sendEmailHandler(c *routing.Context) {
    rabbit := rabbitmq.GetRabbitMQ(app)
    if rabbit == nil {
        c.JSON(503, map[string]string{"error": "RabbitMQ not available"})
        return
    }
    
    var emailData struct {
        To      string `json:"to"`
        Subject string `json:"subject"`
        Body    string `json:"body"`
    }
    
    if err := c.Bind(&emailData); err != nil {
        c.JSON(400, map[string]string{"error": "Invalid data"})
        return
    }
    
    // Queue the email job
    err := rabbit.PushJob("emails", "send_email", emailData)
    if err != nil {
        c.JSON(500, map[string]string{"error": "Failed to queue email"})
        return
    }
    
    c.JSON(200, map[string]string{"message": "Email queued successfully"})
}
```

## 4. Process Jobs in Background

```go
func startEmailWorker(app *framework.Application) {
    rabbit := rabbitmq.GetRabbitMQ(app)
    if rabbit == nil {
        log.Println("RabbitMQ not available, skipping email worker")
        return
    }
    
    ctx := context.Background()
    
    // Define job handlers
    handlers := map[string]rabbitmq.MessageHandler{
        "send_email": handleSendEmail,
    }
    
    log.Println("Starting email worker...")
    if err := rabbit.ListenForJobs(ctx, "emails", handlers); err != nil {
        log.Printf("Email worker error: %v", err)
    }
}

func handleSendEmail(delivery *rabbitmq.Delivery) error {
    var job rabbitmq.Job
    if err := delivery.JSON(&job); err != nil {
        return err
    }
    
    // Extract email data
    emailData, ok := job.Payload.(map[string]interface{})
    if !ok {
        log.Println("Invalid email job payload")
        return nil
    }
    
    to, _ := emailData["to"].(string)
    subject, _ := emailData["subject"].(string)
    body, _ := emailData["body"].(string)
    
    // Send the email (integrate with your email service)
    log.Printf("📧 Sending email to %s: %s", to, subject)
    
    // Simulate email sending
    time.Sleep(2 * time.Second)
    
    log.Printf("✅ Email sent successfully to %s", to)
    return nil
}
```

## 5. Monitor Queue Health

```go
func queueStatsHandler(c *routing.Context) {
    // Use the built-in health check
    health := rabbitmq.QueueHealthCheck(c.App)
    
    if health["status"] == "error" {
        c.JSON(503, health)
        return
    }
    
    c.JSON(200, health)
}
```

## 6. Test Your Implementation

### Send an email via API

```bash
curl -X POST http://localhost:8080/send-email \
  -H "Content-Type: application/json" \
  -d '{
    "to": "user@example.com",
    "subject": "Welcome!",
    "body": "Welcome to our service!"
  }'
```

### Check queue stats

```bash
curl http://localhost:8080/queue-stats
```

## Advanced Usage

### Custom Consumer with Middleware

```go
func startAdvancedWorker(app *framework.Application) {
    rabbit := rabbitmq.GetRabbitMQ(app)
    
    // Create consumer with custom config
    consumer, err := rabbit.CreateConsumer(&rabbitmq.ConsumerConfig{
        Queue:       "processing",
        Concurrency: 5,
        AutoAck:     false,
    })
    if err != nil {
        log.Printf("Failed to create consumer: %v", err)
        return
    }
    
    // Add middleware
    consumer.Use(rabbitmq.WithLogging())
    consumer.Use(rabbitmq.WithRecovery())
    consumer.Use(rabbitmq.WithRetry(3, 5*time.Second))
    consumer.Use(rabbitmq.WithTimeout(30*time.Second))
    
    // Handle messages
    consumer.HandleAll(func(delivery *rabbitmq.Delivery) error {
        log.Printf("Processing: %s", delivery.String())
        return nil
    })
    
    ctx := context.Background()
    consumer.Start(ctx)
}
```

### Direct Queue Operations

```go
// Simple queue operations
rabbit := rabbitmq.GetRabbitMQ(app)

// Push to queue
err := rabbit.Push("notifications", map[string]string{
    "message": "Hello World",
})

// Pop from queue
delivery, err := rabbit.Pop("notifications")
if delivery != nil {
    var data map[string]string
    delivery.JSON(&data)
    log.Printf("Received: %+v", data)
    delivery.Ack(false)
}
```

## Common Patterns

### 1. Email Service

```go
type EmailService struct {
    rabbit *rabbitmq.RabbitMQ
}

func (s *EmailService) SendWelcomeEmail(email, name string) error {
    return s.rabbit.PushJob("emails", "send_welcome", map[string]string{
        "email": email,
        "name":  name,
    })
}

func (s *EmailService) SendPasswordReset(email, token string) error {
    return s.rabbit.PushJob("emails", "password_reset", map[string]string{
        "email": email,
        "token": token,
    })
}
```

### 2. Image Processing

```go
func uploadImageHandler(c *routing.Context) {
    // Handle file upload...
    
    rabbit := rabbitmq.GetRabbitMQ(c.App)
    
    // Queue image processing job
    err := rabbit.PushJob("images", "process_image", map[string]interface{}{
        "image_path": imagePath,
        "user_id":    userID,
        "sizes":      []string{"thumbnail", "medium", "large"},
    })
    
    if err != nil {
        c.JSON(500, map[string]string{"error": "Failed to queue processing"})
        return
    }
    
    c.JSON(200, map[string]string{"message": "Image uploaded and queued for processing"})
}
```

### 3. Notification System

```go
func handleUserRegistration(delivery *rabbitmq.Delivery) error {
    var job rabbitmq.Job
    delivery.JSON(&job)
    
    userData := job.Payload.(map[string]interface{})
    
    // Send multiple notifications
    rabbit := getRabbitFromContext(delivery)
    
    // Send welcome email
    rabbit.PushJob("emails", "send_welcome", userData)
    
    // Send SMS verification
    rabbit.PushJob("sms", "send_verification", userData)
    
    // Send push notification
    rabbit.PushJob("push", "welcome_notification", userData)
    
    return nil
}
```

That's it! You now have a powerful, Laravel-inspired queue system in your GoLara application. The RabbitMQ integration handles connection management, auto-reconnection, and provides a simple API for common queue operations.
