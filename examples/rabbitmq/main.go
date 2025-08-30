package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/taeyelor/golara/framework"
	"github.com/taeyelor/golara/framework/rabbitmq"
	"github.com/taeyelor/golara/framework/routing"
)

type EmailJob struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

type NotificationJob struct {
	UserID  int    `json:"user_id"`
	Message string `json:"message"`
	Type    string `json:"type"`
}

func main() {
	// Create GoLara application
	app := framework.NewApplication()

	// Connect to RabbitMQ
	rabbitURL := app.Config.GetString("rabbitmq.url", "amqp://guest:guest@localhost:5672/")
	rabbitmq, err := rabbitmq.Connect(rabbitURL)
	if err != nil {
		log.Fatal("Failed to connect to RabbitMQ:", err)
	}
	defer rabbitmq.Close()

	// Register RabbitMQ in service container
	app.Singleton("rabbitmq", func() interface{} {
		return rabbitmq
	})

	// Setup routes for API endpoints
	setupRoutes(app)

	// Start background job processors
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start email processor
	go startEmailProcessor(ctx, rabbitmq)

	// Start notification processor
	go startNotificationProcessor(ctx, rabbitmq)

	// Start web server
	go func() {
		log.Println("Starting web server on :8080")
		if err := app.Run(":8080"); err != nil {
			log.Printf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	log.Println("Shutting down...")
	cancel()
}

func setupRoutes(app *framework.Application) {
	// API routes
	api := app.Group("/api")

	// Send email via queue
	api.POST("/send-email", func(c *routing.Context) {
		rabbit := app.Resolve("rabbitmq").(*rabbitmq.RabbitMQ)

		var emailData EmailJob
		if err := c.Bind(&emailData); err != nil {
			c.JSON(400, map[string]string{"error": "Invalid JSON"})
			return
		}

		// Push email job to queue
		if err := rabbit.PushJob("emails", "send_email", emailData); err != nil {
			c.JSON(500, map[string]string{"error": "Failed to queue email"})
			return
		}

		c.JSON(200, map[string]string{"message": "Email queued successfully"})
	})

	// Send notification via queue
	api.POST("/send-notification", func(c *routing.Context) {
		rabbit := app.Resolve("rabbitmq").(*rabbitmq.RabbitMQ)

		var notificationData NotificationJob
		if err := c.Bind(&notificationData); err != nil {
			c.JSON(400, map[string]string{"error": "Invalid JSON"})
			return
		}

		// Push notification job to queue
		if err := rabbit.PushJob("notifications", "send_notification", notificationData); err != nil {
			c.JSON(500, map[string]string{"error": "Failed to queue notification"})
			return
		}

		c.JSON(200, map[string]string{"message": "Notification queued successfully"})
	})

	// Queue stats
	api.GET("/queue-stats", func(c *routing.Context) {
		rabbit := app.Resolve("rabbitmq").(*rabbitmq.RabbitMQ)

		stats := rabbit.Stats()

		// Get queue info
		emailQueue, _ := rabbit.Queue("emails")
		emailInfo, _ := emailQueue.Inspect()

		notificationQueue, _ := rabbit.Queue("notifications")
		notificationInfo, _ := notificationQueue.Inspect()

		c.JSON(200, map[string]interface{}{
			"connection": stats,
			"queues": map[string]interface{}{
				"emails": map[string]interface{}{
					"messages":  emailInfo.Messages,
					"consumers": emailInfo.Consumers,
				},
				"notifications": map[string]interface{}{
					"messages":  notificationInfo.Messages,
					"consumers": notificationInfo.Consumers,
				},
			},
		})
	})

	// Health check
	api.GET("/health", func(c *routing.Context) {
		rabbit := app.Resolve("rabbitmq").(*rabbitmq.RabbitMQ)

		if err := rabbit.Health(); err != nil {
			c.JSON(503, map[string]interface{}{
				"status":   "error",
				"rabbitmq": "disconnected",
				"error":    err.Error(),
			})
			return
		}

		c.JSON(200, map[string]interface{}{
			"status":   "ok",
			"rabbitmq": "connected",
		})
	})
}

func startEmailProcessor(ctx context.Context, rabbit *rabbitmq.RabbitMQ) {
	log.Println("Starting email processor...")

	// Create consumer with middleware
	consumer, err := rabbit.CreateConsumer(&rabbitmq.ConsumerConfig{
		Queue:       "emails",
		Concurrency: 3, // Process 3 emails concurrently
		AutoAck:     false,
	})
	if err != nil {
		log.Printf("Failed to create email consumer: %v", err)
		return
	}

	// Add middleware
	consumer.Use(rabbitmq.WithLogging())
	consumer.Use(rabbitmq.WithRecovery())
	consumer.Use(rabbitmq.WithRetry(3, 5*time.Second))
	consumer.Use(rabbitmq.WithTimeout(30 * time.Second))

	// Define job handlers
	handlers := map[string]rabbitmq.MessageHandler{
		"send_email": handleSendEmail,
	}

	// Start processing jobs
	if err := rabbit.ListenForJobs(ctx, "emails", handlers); err != nil {
		log.Printf("Email processor error: %v", err)
	}
}

func startNotificationProcessor(ctx context.Context, rabbit *rabbitmq.RabbitMQ) {
	log.Println("Starting notification processor...")

	// Simple queue listener with workers
	err := rabbit.ListenWithWorkers(ctx, "notifications", 5, func(delivery *rabbitmq.Delivery) error {
		var job rabbitmq.Job
		if err := delivery.JSON(&job); err != nil {
			log.Printf("Failed to unmarshal notification job: %v", err)
			return err
		}

		switch job.Type {
		case "send_notification":
			return handleSendNotification(delivery)
		default:
			log.Printf("Unknown notification job type: %s", job.Type)
			return nil
		}
	})

	if err != nil {
		log.Printf("Notification processor error: %v", err)
	}
}

func handleSendEmail(delivery *rabbitmq.Delivery) error {
	var job rabbitmq.Job
	if err := delivery.JSON(&job); err != nil {
		return err
	}

	// Extract email data from job payload
	emailData, ok := job.Payload.(map[string]interface{})
	if !ok {
		log.Println("Invalid email job payload")
		return nil
	}

	to, _ := emailData["to"].(string)
	subject, _ := emailData["subject"].(string)
	body, _ := emailData["body"].(string)

	// Simulate email sending
	log.Printf("📧 Sending email to %s: %s", to, subject)
	log.Printf("📧 Email body preview: %.50s...", body)
	time.Sleep(2 * time.Second) // Simulate processing time

	// Here you would integrate with your actual email service
	// e.g., SendGrid, Mailgun, AWS SES, etc.

	log.Printf("✅ Email sent successfully to %s", to)
	return nil
}

func handleSendNotification(delivery *rabbitmq.Delivery) error {
	var job rabbitmq.Job
	if err := delivery.JSON(&job); err != nil {
		return err
	}

	// Extract notification data from job payload
	notificationData, ok := job.Payload.(map[string]interface{})
	if !ok {
		log.Println("Invalid notification job payload")
		return nil
	}

	userID, _ := notificationData["user_id"].(float64) // JSON numbers are float64
	message, _ := notificationData["message"].(string)
	notificationType, _ := notificationData["type"].(string)

	// Simulate notification sending
	log.Printf("🔔 Sending %s notification to user %d: %s", notificationType, int(userID), message)
	time.Sleep(1 * time.Second) // Simulate processing time

	// Here you would integrate with your actual notification service
	// e.g., Firebase, Pusher, WebSocket, etc.

	log.Printf("✅ Notification sent successfully to user %d", int(userID))
	return nil
}
