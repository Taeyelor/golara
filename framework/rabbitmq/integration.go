package rabbitmq

import (
	"log"

	"github.com/taeyelor/golara/framework"
)

// RegisterRabbitMQ registers RabbitMQ service in the GoLara application container
func RegisterRabbitMQ(app *framework.Application, config *RabbitMQConfig) {
	app.Singleton("rabbitmq", func() interface{} {
		if config == nil {
			// Try to load from application config
			config = &RabbitMQConfig{
				URL:                 app.Config.GetString("rabbitmq.url", "amqp://guest:guest@localhost:5672/"),
				ReconnectDelay:      app.Config.GetString("rabbitmq.reconnect_delay", "5s"),
				ReconnectAttempts:   app.Config.GetInt("rabbitmq.reconnect_attempts", 10),
				EnableHeartbeat:     app.Config.GetBool("rabbitmq.enable_heartbeat", true),
				HeartbeatInterval:   app.Config.GetString("rabbitmq.heartbeat_interval", "10s"),
				ChannelPoolSize:     app.Config.GetInt("rabbitmq.channel_pool_size", 10),
				AutoDeclareQueues:   app.Config.GetBool("rabbitmq.auto_declare_queues", true),
				AutoDeclareExchange: app.Config.GetBool("rabbitmq.auto_declare_exchange", true),
			}
		}

		rabbit, err := New(config)
		if err != nil {
			log.Printf("Warning: Failed to connect to RabbitMQ: %v", err)
			return nil
		}

		log.Println("RabbitMQ: Service registered successfully")
		return rabbit
	})
}

// GetRabbitMQ retrieves RabbitMQ from the application container
func GetRabbitMQ(app *framework.Application) *RabbitMQ {
	service := app.Resolve("rabbitmq")
	if service == nil {
		return nil
	}

	if rabbit, ok := service.(*RabbitMQ); ok {
		return rabbit
	}

	return nil
}

// RegisterRabbitMQFromEnv registers RabbitMQ using environment variables
func RegisterRabbitMQFromEnv(app *framework.Application) {
	RegisterRabbitMQ(app, nil) // Will load from config
}

// MustRegisterRabbitMQ registers RabbitMQ and panics if connection fails
func MustRegisterRabbitMQ(app *framework.Application, config *RabbitMQConfig) {
	app.Singleton("rabbitmq", func() interface{} {
		if config == nil {
			config = &RabbitMQConfig{
				URL:                 app.Config.GetString("rabbitmq.url", "amqp://guest:guest@localhost:5672/"),
				ReconnectDelay:      app.Config.GetString("rabbitmq.reconnect_delay", "5s"),
				ReconnectAttempts:   app.Config.GetInt("rabbitmq.reconnect_attempts", 10),
				EnableHeartbeat:     app.Config.GetBool("rabbitmq.enable_heartbeat", true),
				HeartbeatInterval:   app.Config.GetString("rabbitmq.heartbeat_interval", "10s"),
				ChannelPoolSize:     app.Config.GetInt("rabbitmq.channel_pool_size", 10),
				AutoDeclareQueues:   app.Config.GetBool("rabbitmq.auto_declare_queues", true),
				AutoDeclareExchange: app.Config.GetBool("rabbitmq.auto_declare_exchange", true),
			}
		}

		rabbit, err := New(config)
		if err != nil {
			log.Fatalf("Failed to connect to RabbitMQ: %v", err)
		}

		log.Println("RabbitMQ: Service registered successfully")
		return rabbit
	})
}

// QueueHealthCheck provides a health check endpoint for RabbitMQ
func QueueHealthCheck(app *framework.Application) map[string]interface{} {
	rabbit := GetRabbitMQ(app)
	if rabbit == nil {
		return map[string]interface{}{
			"status": "error",
			"error":  "RabbitMQ service not available",
		}
	}

	if err := rabbit.Health(); err != nil {
		return map[string]interface{}{
			"status": "error",
			"error":  err.Error(),
		}
	}

	stats := rabbit.Stats()
	return map[string]interface{}{
		"status": "ok",
		"stats":  stats,
	}
}
