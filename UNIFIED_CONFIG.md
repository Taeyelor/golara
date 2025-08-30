# GoLara Unified Configuration System

GoLara provides a centralized configuration system that manages all framework components including App, Database (MongoDB), and RabbitMQ settings from a single place.

## Configuration Sources

GoLara loads configuration in the following order (later sources override earlier ones):

1. **Default values** (defined in `framework/config/config.go`)
2. **Environment variables** (from `.env` file or system environment)
3. **Configuration files** (JSON files loaded with `LoadFromFile()`)
4. **Runtime settings** (set with `app.Config.Set()`)

## Configuration Structure

```go
// Application configuration
app.name                    // APP_NAME
app.env                     // APP_ENV  
app.debug                   // APP_DEBUG
app.port                    // APP_PORT
app.key                     // APP_KEY

// Database configuration (MongoDB)
database.default                           // DB_CONNECTION
database.connections.mongodb.uri          // MONGODB_URI
database.connections.mongodb.database     // MONGODB_DATABASE
database.connections.mongodb.options      // (nested object)

// RabbitMQ configuration
rabbitmq.url                      // RABBITMQ_URL
rabbitmq.reconnect_delay          // RABBITMQ_RECONNECT_DELAY
rabbitmq.reconnect_attempts       // RABBITMQ_RECONNECT_ATTEMPTS
rabbitmq.enable_heartbeat         // RABBITMQ_ENABLE_HEARTBEAT
rabbitmq.heartbeat_interval       // RABBITMQ_HEARTBEAT_INTERVAL
rabbitmq.channel_pool_size        // RABBITMQ_CHANNEL_POOL_SIZE
rabbitmq.auto_declare_queues      // RABBITMQ_AUTO_DECLARE_QUEUES
rabbitmq.auto_declare_exchange    // RABBITMQ_AUTO_DECLARE_EXCHANGE
```

## Environment Variables (.env)

Create a `.env` file in your project root:

```env
# Application Settings
APP_NAME=MyGoLaraApp
APP_ENV=local
APP_DEBUG=true
APP_PORT=:8080
APP_KEY=your-secret-key

# Database (MongoDB)
MONGODB_URI=mongodb://localhost:27017
MONGODB_DATABASE=myapp

# RabbitMQ
RABBITMQ_URL=amqp://guest:guest@localhost:5672/
RABBITMQ_RECONNECT_DELAY=5s
RABBITMQ_RECONNECT_ATTEMPTS=10
RABBITMQ_ENABLE_HEARTBEAT=true
RABBITMQ_HEARTBEAT_INTERVAL=10s
RABBITMQ_CHANNEL_POOL_SIZE=10
RABBITMQ_AUTO_DECLARE_QUEUES=true
RABBITMQ_AUTO_DECLARE_EXCHANGE=true
```

## Usage Examples

### Basic Setup

```go
package main

import (
    "github.com/taeyelor/golara/framework"
    "github.com/taeyelor/golara/framework/rabbitmq"
)

func main() {
    // Create application - automatically loads .env
    app := framework.NewApplication()
    
    // Register services using unified config
    rabbitmq.RegisterRabbitMQFromEnv(app)
    
    // Configuration is automatically loaded!
    app.Run(":8080")
}
```

### Accessing Configuration

```go
// Get individual values
appName := app.Config.GetString("app.name")
debug := app.Config.GetBool("app.debug")
rabbitURL := app.Config.GetString("rabbitmq.url")

// Get configuration groups
appConfig := app.Config.GetAppConfig()
dbConfig := app.Config.GetDatabaseConfig()
rabbitConfig := app.Config.GetRabbitMQConfig()

// Set values at runtime
app.Config.Set("app.name", "New Name")
app.Config.Set("rabbitmq.channel_pool_size", 20)
```

### Service Registration

```go
// RabbitMQ with unified config
rabbitmq.RegisterRabbitMQFromEnv(app)  // Uses app.Config

// Or with custom config
customConfig := &rabbitmq.RabbitMQConfig{
    URL: "amqp://custom:password@server:5672/",
    ChannelPoolSize: 20,
}
rabbitmq.RegisterRabbitMQ(app, customConfig)

// Get services
rabbit := rabbitmq.GetRabbitMQ(app)
```

### Configuration Helpers

```go
// Built-in helper methods
appConfig := app.Config.GetAppConfig()
// Returns:
// {
//   "name": "MyApp",
//   "env": "local", 
//   "debug": true,
//   "port": ":8080",
//   "key": "secret"
// }

rabbitConfig := app.Config.GetRabbitMQConfig()
// Returns:
// {
//   "url": "amqp://guest:guest@localhost:5672/",
//   "reconnect_delay": "5s",
//   "reconnect_attempts": 10,
//   "enable_heartbeat": true,
//   "heartbeat_interval": "10s",
//   "channel_pool_size": 10,
//   "auto_declare_queues": true,
//   "auto_declare_exchange": true
// }
```

### Loading from Files

```go
// Load additional config from JSON file
err := app.Config.LoadFromFile("config/production.json")

// Example production.json:
// {
//   "app": {
//     "debug": false,
//     "env": "production"
//   },
//   "rabbitmq": {
//     "url": "amqp://prod:password@rabbitmq.example.com:5672/",
//     "channel_pool_size": 50
//   }
// }
```

## Configuration Best Practices

### 1. Environment-Specific Configuration

```env
# .env.local
APP_ENV=local
APP_DEBUG=true
RABBITMQ_URL=amqp://guest:guest@localhost:5672/

# .env.production  
APP_ENV=production
APP_DEBUG=false
RABBITMQ_URL=amqp://prod:password@rabbitmq.prod.com:5672/
```

### 2. Service Registration

```go
func RegisterServices(app *framework.Application) {
    // RabbitMQ
    rabbitmq.RegisterRabbitMQFromEnv(app)
    
    // Database (if you have a database service)
    // database.RegisterFromEnv(app)
    
    // Other services...
}

func main() {
    app := framework.NewApplication()
    RegisterServices(app)
    
    // Start app...
}
```

### 3. Health Checks

```go
app.GET("/health", func(c *routing.Context) {
    health := map[string]interface{}{
        "app": map[string]interface{}{
            "name": app.Config.GetString("app.name"),
            "env":  app.Config.GetString("app.env"),
        },
        "rabbitmq": rabbitmq.QueueHealthCheck(app),
    }
    
    c.JSON(200, health)
})
```

### 4. Configuration Validation

```go
func validateConfig(app *framework.Application) {
    required := []string{
        "app.name",
        "app.key", 
        "rabbitmq.url",
        "database.connections.mongodb.uri",
    }
    
    for _, key := range required {
        if app.Config.GetString(key) == "" {
            log.Fatalf("Required configuration missing: %s", key)
        }
    }
}
```

## Migration from Separate Configs

If you were using separate configuration before:

### Before (Separate)

```go
// Old way
rabbitConfig := &rabbitmq.RabbitMQConfig{
    URL: "amqp://localhost:5672/",
    ChannelPoolSize: 10,
}
rabbit, err := rabbitmq.New(rabbitConfig)
```

### After (Unified)

```go
// New way - just set in .env:
// RABBITMQ_URL=amqp://localhost:5672/
// RABBITMQ_CHANNEL_POOL_SIZE=10

rabbitmq.RegisterRabbitMQFromEnv(app)
rabbit := rabbitmq.GetRabbitMQ(app)
```

## Default Values

All configuration keys have sensible defaults defined in `framework/config/config.go`:

```go
defaults := map[string]interface{}{
    // App defaults
    "app.name":  "GoLara",
    "app.env":   "local", 
    "app.debug": true,
    "app.port":  ":8080",
    
    // RabbitMQ defaults
    "rabbitmq.url":                   "amqp://guest:guest@localhost:5672/",
    "rabbitmq.reconnect_delay":       "5s",
    "rabbitmq.reconnect_attempts":    10,
    "rabbitmq.enable_heartbeat":      true,
    "rabbitmq.heartbeat_interval":    "10s", 
    "rabbitmq.channel_pool_size":     10,
    "rabbitmq.auto_declare_queues":   true,
    "rabbitmq.auto_declare_exchange": true,
    
    // MongoDB defaults
    "database.connections.mongodb.uri":      "mongodb://localhost:27017",
    "database.connections.mongodb.database": "golara",
}
```

This means your application will work out of the box with just:

```go
app := framework.NewApplication()
rabbitmq.RegisterRabbitMQFromEnv(app)
app.Run(":8080")
```

The unified configuration system makes GoLara applications easier to configure, deploy, and maintain across different environments.
