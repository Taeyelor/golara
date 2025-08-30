package main

import (
	"context"
	"log"
	"time"

	"github.com/taeyelor/golara/framework"
	"github.com/taeyelor/golara/framework/rabbitmq"
	"github.com/taeyelor/golara/framework/routing"
)

func main() {
	// Create GoLara application - automatically loads config from .env
	app := framework.NewApplication()

	// Show current configuration
	showConfiguration(app)

	// Register RabbitMQ using unified configuration system
	// This will read from app.Config which loads from:
	// 1. Default values in config.go
	// 2. Environment variables (.env file or system env)
	rabbitmq.RegisterRabbitMQFromEnv(app)

	// Setup simple routes
	setupSimpleRoutes(app)

	// Test RabbitMQ functionality
	testRabbitMQ(app)

	// Start server
	log.Println("Server starting on", app.Config.GetString("app.port"))
	app.Run(app.Config.GetString("app.port"))
}

func showConfiguration(app *framework.Application) {
	log.Println("=== GoLara Configuration ===")

	// App config
	appConfig := app.Config.GetAppConfig()
	log.Printf("App: %+v", appConfig)

	// Database config
	dbConfig := app.Config.GetDatabaseConfig()
	log.Printf("Database: %+v", dbConfig)

	// RabbitMQ config
	rabbitConfig := app.Config.GetRabbitMQConfig()
	log.Printf("RabbitMQ: %+v", rabbitConfig)

	log.Println("===========================")
}

func setupSimpleRoutes(app *framework.Application) {
	// Root route
	app.GET("/", func(c *routing.Context) {
		c.JSON(200, map[string]interface{}{
			"message": "GoLara with Unified Configuration",
			"config": map[string]interface{}{
				"app":      app.Config.GetAppConfig(),
				"rabbitmq": app.Config.GetRabbitMQConfig(),
			},
		})
	})

	// Test queue route
	app.POST("/test-queue", func(c *routing.Context) {
		rabbit := rabbitmq.GetRabbitMQ(app)
		if rabbit == nil {
			c.JSON(503, map[string]string{"error": "RabbitMQ not available"})
			return
		}

		var message struct {
			Text string `json:"text"`
		}

		if err := c.Bind(&message); err != nil {
			c.JSON(400, map[string]string{"error": "Invalid JSON"})
			return
		}

		// Push to test queue
		err := rabbit.Push("test", map[string]interface{}{
			"message":   message.Text,
			"timestamp": time.Now(),
		})

		if err != nil {
			c.JSON(500, map[string]string{"error": err.Error()})
			return
		}

		c.JSON(200, map[string]string{"status": "Message queued successfully"})
	})

	// Health check route
	app.GET("/health", func(c *routing.Context) {
		health := rabbitmq.QueueHealthCheck(app)

		status := 200
		if health["status"] == "error" {
			status = 503
		}

		c.JSON(status, health)
	})
}

func testRabbitMQ(app *framework.Application) {
	rabbit := rabbitmq.GetRabbitMQ(app)
	if rabbit == nil {
		log.Println("RabbitMQ not available - skipping test")
		return
	}

	log.Println("Testing RabbitMQ connection...")

	// Test push message
	err := rabbit.Push("test", map[string]interface{}{
		"message": "Hello from GoLara unified config!",
		"time":    time.Now(),
	})

	if err != nil {
		log.Printf("Failed to push test message: %v", err)
		return
	}

	log.Println("✅ RabbitMQ test message sent successfully")

	// Start a simple consumer for testing
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		log.Println("Starting test message consumer...")

		err := rabbit.Listen(ctx, "test", func(delivery *rabbitmq.Delivery) error {
			var data map[string]interface{}
			if err := delivery.JSON(&data); err != nil {
				return err
			}

			log.Printf("📨 Received test message: %+v", data)
			return nil
		})

		if err != nil {
			log.Printf("Consumer error: %v", err)
		}
	}()
}
