# GoLara Framework

A simple, Laravel-inspired web framework for Go that provides an elegant and developer-friendly experience with **MongoDB** as the primary database.

## Features

- 🚀 **Simple Routing** - Express-style routing with parameter support
- 🔧 **Middleware Support** - Easy middleware chain management
- 🏗️ **Dependency Injection** - Built-in service container with auto-registration
- ⚙️ **Configuration Management** - Environment-based configuration
- 🗄️ **MongoDB ODM** - Laravel-inspired query builder for MongoDB
- 🐰 **RabbitMQ Integration** - Simple message queue integration
- 🎨 **Template Engine** - Built-in view rendering with template functions
- 🛡️ **Built-in Middleware** - Logging, CORS, Recovery, and Auth middleware
- 🔄 **Graceful Shutdown** - Proper server shutdown handling
- ⚡ **Auto-Registration** - Core services registered automatically on bootstrap

## Quick Start

### Installation

```bash
go mod init your-project
go get github.com/taeyelor/golara
```

### Basic Usage

```go
package main

import (
    "github.com/taeyelor/golara/framework"
    "github.com/taeyelor/golara/framework/routing"
)

func main() {
    app := framework.NewApplication()
    
    app.GET("/", func(c *routing.Context) {
        c.JSON(200, map[string]string{
            "message": "Hello, GoLara!",
        })
    })
    
    app.Run(":8080")
}
```

## Routing

### Basic Routes

```go
app.GET("/users", listUsers)
app.POST("/users", createUser)
app.PUT("/users/{id}", updateUser)
app.DELETE("/users/{id}", deleteUser)
```

### Route Parameters

```go
app.GET("/users/{id}", func(c *routing.Context) {
    userID := c.Param("id")
    c.JSON(200, map[string]string{"user_id": userID})
})
```

### Query Parameters

```go
app.GET("/search", func(c *routing.Context) {
    query := c.Query("q")
    page := c.QueryDefault("page", "1")
    // Handle search...
})
```

### Route Groups

```go
api := app.Group("/api/v1")
api.GET("/users", listUsers)
api.POST("/users", createUser)

// With middleware
admin := app.Group("/admin", authMiddleware, adminMiddleware)
admin.GET("/dashboard", adminDashboard)
```

## Middleware

### Built-in Middleware

```go
import httpMW "github.com/taeyelor/golara/framework/http"

// Global middleware
app.Use(httpMW.LoggingMiddleware)
app.Use(httpMW.RecoveryMiddleware)
app.Use(httpMW.CORSMiddleware([]string{"*"}))

// Authentication middleware
app.Use(httpMW.AuthMiddleware(func(token string) bool {
    return token == "valid-token"
}))
```

### Custom Middleware

```go
func customMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Pre-processing
        next.ServeHTTP(w, r)
        // Post-processing
    })
}

app.Use(customMiddleware)
```

## RabbitMQ Integration

### Connection and Basic Usage

```go
import "github.com/taeyelor/golara/framework/rabbitmq"

// Register RabbitMQ using unified configuration
app := framework.NewApplication()
rabbitmq.RegisterRabbitMQFromEnv(app)

// Use RabbitMQ service
rabbit := rabbitmq.GetRabbitMQ(app)
```

### Simple Queue Operations

```go
// Push job to queue
err := rabbit.PushJob("emails", "send_welcome", map[string]interface{}{
    "email": "user@example.com",
    "name":  "John Doe",
})

// Listen for jobs with multiple workers
ctx := context.Background()
handlers := map[string]rabbitmq.MessageHandler{
    "send_welcome": func(delivery *rabbitmq.Delivery) error {
        var job rabbitmq.Job
        delivery.JSON(&job)
        log.Printf("Processing: %+v", job.Payload)
        return nil
    },
}

err = rabbit.ListenForJobs(ctx, "emails", handlers)
```

### Consumer with Middleware

```go
consumer, err := rabbit.CreateConsumer(&rabbitmq.ConsumerConfig{
    Queue:       "processing",
    Concurrency: 5,
})

// Add middleware
consumer.Use(rabbitmq.WithLogging())
consumer.Use(rabbitmq.WithRetry(3, 5*time.Second))
consumer.Use(rabbitmq.WithTimeout(30*time.Second))

// Handle messages
consumer.HandleAll(processMessage)
consumer.Start(ctx)
```

## MongoDB ODM

### Connection

```go
import "github.com/taeyelor/golara/framework/database"

db, err := database.Connect("mongodb://localhost:27017", "myapp")
if err != nil {
    log.Fatal(err)
}
```

### Query Builder

```go
import "go.mongodb.org/mongo-driver/bson"

// Select
var users []User
err := db.NewQueryBuilder().
    Collection("users").
    Where("active", "=", true).
    OrderBy("created_at", "DESC").
    Limit(10).
    Get(&users)

// Insert
userID, err := db.NewQueryBuilder().
    Collection("users").
    Insert(User{
        Name:  "John Doe",
        Email: "john@example.com",
    })

// Update
result, err := db.NewQueryBuilder().
    Collection("users").
    Where("_id", "=", objectID).
    UpdateOne(bson.M{"$set": bson.M{
        "name": "Jane Doe",
    }})

// Delete
result, err := db.NewQueryBuilder().
    Collection("users").
    Where("_id", "=", objectID).
    DeleteOne()

// Aggregation
pipeline := []bson.M{
    {"$match": bson.M{"status": "active"}},
    {"$group": bson.M{"_id": "$department", "count": bson.M{"$sum": 1}}},
}
var results []bson.M
err := db.NewQueryBuilder().
    Collection("users").
    Aggregate(pipeline, &results)
```

### MongoDB-specific Operations

```go
// MongoDB operators
db.NewQueryBuilder().
    Collection("users").
    Where("age", ">", 18).              // Greater than
    Where("name", "like", "John").       // Regex search
    WhereIn("status", []interface{}{"active", "pending"}).  // $in operator
    WhereExists("profile.avatar").       // Field exists
    Get(&users)

// Advanced queries
db.NewQueryBuilder().
    Collection("posts").
    Where("tags", "in", []interface{}{"go", "mongodb"}).
    Where("published", "=", true).
    OrderBy("views", "DESC").
    Skip(20).
    Limit(10).
    Get(&posts)
```

## Configuration

### Environment Variables

Create a `.env` file:

```env
# Application
APP_NAME=MyApp
APP_ENV=production
APP_PORT=:8080

# MongoDB
MONGODB_URI=mongodb://localhost:27017
MONGODB_DATABASE=myapp

# RabbitMQ
RABBITMQ_URL=amqp://guest:guest@localhost:5672/
RABBITMQ_AUTO_DECLARE_QUEUES=true
```

### Using Configuration

```go
app := framework.NewApplication()

// Configuration is automatically loaded from .env and defaults
appName := app.Config.GetString("app.name", "DefaultApp")
debug := app.Config.GetBool("app.debug", false)
port := app.Config.GetString("app.port", ":8080")

// Get configuration groups
appConfig := app.Config.GetAppConfig()
rabbitConfig := app.Config.GetRabbitMQConfig()
```

## Views

### Template Engine

```go
import "github.com/taeyelor/golara/framework/view"

engine := view.NewEngine("views")
engine.LoadTemplates()

// In your handler
func homeHandler(c *routing.Context) {
    data := view.ViewData{
        "title": "Home Page",
        "user":  getCurrentUser(),
    }
    
    html, err := engine.RenderString("home", data)
    if err != nil {
        c.Status(500)
        return
    }
    
    c.HTML(200, html)
}
```

### Template Example (`views/home.html`)

```html
<!DOCTYPE html>
<html>
<head>
    <title>{{ .title }}</title>
</head>
<body>
    <h1>Welcome, {{ .user.Name }}!</h1>
    <p>{{ "hello world" | title }}</p>
</body>
</html>
```

## Dependency Injection

### Service Registration

```go
// Bind a service
app.Bind("logger", func() interface{} {
    return log.New(os.Stdout, "[APP] ", log.LstdFlags)
})

// Register singleton
app.Singleton("database", func() interface{} {
    db, _ := database.Connect("mysql", getDSN())
    return db
})

// Register instance
app.Container.Instance("config", myConfig)
```

### Auto-Registration

GoLara automatically registers core services when calling `framework.NewApplication()`:

```go
app := framework.NewApplication()

// Automatically registered services:
config := app.Resolve("config")   // Configuration service
router := app.Resolve("router")   // Router service  
db := app.Resolve("db")           // MongoDB connection (if configured)
rabbit := app.Resolve("rabbitmq") // RabbitMQ placeholder (if enabled)
```

**Auto-registered services:**
- **config** - Unified configuration management
- **router** - HTTP router with middleware support
- **db** - MongoDB database connection (lazy-loaded)
- **rabbitmq** - RabbitMQ service placeholder (if `rabbitmq.enabled=true`)

For RabbitMQ, use the registration helper to initialize the actual service:
```go
import "github.com/taeyelor/golara/framework/rabbitmq"

if app.Config.Get("rabbitmq.enabled", false).(bool) {
    rabbitmq.RegisterRabbitMQ(app, nil) // Uses config from app.Config
}
```

### Service Resolution

```go
// Resolve from container
logger := app.Resolve("logger").(*log.Logger)
db := app.Resolve("database").(*database.DB)
```

## Response Types

```go
// JSON response
c.JSON(200, map[string]string{"status": "ok"})

// String response
c.String(200, "Hello, World!")

// HTML response
c.HTML(200, "<h1>Hello</h1>")

// Redirect
c.Redirect(302, "/login")

// Status code only
c.Status(204)
```

## Request Handling

```go
// Bind JSON request body
var user User
if err := c.Bind(&user); err != nil {
    c.JSON(400, map[string]string{"error": "Invalid JSON"})
    return
}

// Get headers
contentType := c.GetHeader("Content-Type")
userAgent := c.UserAgent()

// Set response headers
c.Header("X-Custom-Header", "value")
```

## Error Handling

The framework includes built-in error recovery:

```go
app.Use(httpMW.RecoveryMiddleware)

// This will be caught and return 500
func panicHandler(c *routing.Context) {
    panic("Something went wrong!")
}
```

## Example Application Structure

```
your-app/
├── main.go
├── controllers/
│   ├── user_controller.go
│   └── auth_controller.go
├── models/
│   └── user.go
├── middleware/
│   └── auth.go
├── views/
│   ├── layout.html
│   └── home.html
├── config/
│   └── database.go
└── .env
```

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Inspired by Laravel's elegant API design
- Built with Go's powerful standard library
- Designed for simplicity and developer productivity
