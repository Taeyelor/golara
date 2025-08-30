# Auto-Registration Example

This example demonstrates how GoLara automatically registers core services when calling `framework.NewApplication()`.

## Automatically Registered Services

When you call `framework.NewApplication()`, the following services are automatically registered in the service container:

1. **config** - Configuration service with unified config management
2. **router** - Router service for handling HTTP routes
3. **db** - MongoDB database connection (if configured)
4. **rabbitmq** - RabbitMQ service placeholder (if enabled in config)

## Configuration

The auto-registration is based on your configuration settings. Copy `.env.example` to `.env` and modify as needed:

```bash
cp .env.example .env
```

### Database Auto-Registration

The database service is automatically registered using these config keys:
- `database.connections.mongodb.uri` (default: "mongodb://localhost:27017")
- `database.connections.mongodb.database` (default: "golara")

### RabbitMQ Auto-Registration

RabbitMQ is automatically registered as a placeholder if:
- `rabbitmq.enabled` is set to `true`

To initialize the actual RabbitMQ service, call:
```go
rabbitmq.RegisterRabbitMQ(app, nil) // Uses config from app.Config
```

## Usage

### Basic Auto-Registration

```go
package main

import (
    "github.com/taeyelor/golara/framework"
    "github.com/taeyelor/golara/framework/rabbitmq"
)

func main() {
    // Auto-registers: config, router, db, rabbitmq (placeholder)
    app := framework.NewApplication()
    
    // Initialize RabbitMQ if enabled
    if app.Config.Get("rabbitmq.enabled", false).(bool) {
        rabbitmq.RegisterRabbitMQ(app, nil)
    }
    
    // Use registered services
    db := app.Resolve("db")           // Database connection
    config := app.Resolve("config")   // Configuration service
    router := app.Resolve("router")   // Router service
    rabbit := app.Resolve("rabbitmq") // RabbitMQ service
    
    app.Run(":8080")
}
```

### Service Usage Examples

```go
// Using the database service
app.GET("/users", func(w http.ResponseWriter, r *http.Request) {
    db := app.Resolve("db").(*database.DB)
    
    // Use Laravel-style query builder
    users := db.NewQueryBuilder().
        Collection("users").
        Where("active", true).
        Get()
    
    // Return response...
})

// Using RabbitMQ service
app.POST("/send-message", func(w http.ResponseWriter, r *http.Request) {
    rabbit := app.Resolve("rabbitmq").(*rabbitmq.RabbitMQ)
    
    err := rabbit.Publish("user.notifications", map[string]interface{}{
        "message": "Welcome to GoLara!",
        "user_id": 123,
    })
    
    // Handle response...
})
```

## Running the Example

1. Make sure MongoDB is running (if testing database features)
2. Make sure RabbitMQ is running (if testing RabbitMQ features)
3. Copy and configure the environment file:
   ```bash
   cp .env.example .env
   ```
4. Run the example:
   ```bash
   go mod tidy
   go run main.go
   ```

## Available Endpoints

- `GET /services` - List all auto-registered services
- `GET /db-status` - Check database connection status
- `GET /rabbitmq-status` - Check RabbitMQ connection status

## Benefits of Auto-Registration

1. **Simplified Bootstrap**: No need to manually register core services
2. **Consistent Configuration**: All services use the unified config system
3. **Lazy Loading**: Services are only initialized when first accessed
4. **Error Handling**: Built-in error handling for failed connections
5. **Laravel-like Experience**: Similar to Laravel's service providers

## Manual Registration (Alternative)

If you prefer manual control, you can still register services manually:

```go
app := framework.NewApplication()

// Manual database registration
app.Singleton("db", func() interface{} {
    db, err := database.Connect("mongodb://localhost:27017", "mydb")
    if err != nil {
        log.Fatal(err)
    }
    return db
})

// Manual RabbitMQ registration
app.Singleton("rabbitmq", func() interface{} {
    config := &rabbitmq.RabbitMQConfig{
        URL: "amqp://localhost:5672",
    }
    rabbit, err := rabbitmq.New(config)
    if err != nil {
        log.Fatal(err)
    }
    return rabbit
})
```

The auto-registration feature provides sensible defaults while maintaining the flexibility for manual configuration when needed.
