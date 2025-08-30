# GoLara Framework

A simple, Laravel-inspired web framework for Go that provides an elegant and developer-friendly experience.

## Features

- 🚀 **Simple Routing** - Express-style routing with parameter support
- 🔧 **Middleware Support** - Easy middleware chain management
- 🏗️ **Dependency Injection** - Built-in service container
- ⚙️ **Configuration Management** - Environment-based configuration
- 🗄️ **Simple ORM** - Laravel-inspired database query builder
- 🎨 **Template Engine** - Built-in view rendering with template functions
- 🛡️ **Built-in Middleware** - Logging, CORS, Recovery, and Auth middleware
- 🔄 **Graceful Shutdown** - Proper server shutdown handling

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

## Database

### Connection

```go
import "github.com/taeyelor/golara/framework/database"

db, err := database.Connect("mysql", "user:password@/dbname")
if err != nil {
    log.Fatal(err)
}
```

### Query Builder

```go
// Select
var users []User
err := db.NewQueryBuilder().
    Table("users").
    Where("active", "=", true).
    OrderBy("created_at", "DESC").
    Limit(10).
    Get(&users)

// Insert
id, err := db.NewQueryBuilder().
    Table("users").
    Insert(map[string]interface{}{
        "name":  "John Doe",
        "email": "john@example.com",
    })

// Update
affected, err := db.NewQueryBuilder().
    Table("users").
    Where("id", "=", 1).
    Update(map[string]interface{}{
        "name": "Jane Doe",
    })

// Delete
affected, err := db.NewQueryBuilder().
    Table("users").
    Where("id", "=", 1).
    Delete()
```

## Configuration

### Environment Variables

Create a `.env` file or set environment variables:

```env
APP_NAME=MyApp
APP_ENV=production
APP_PORT=:8080
DB_CONNECTION=mysql
DB_HOST=127.0.0.1
DB_DATABASE=myapp
DB_USERNAME=user
DB_PASSWORD=password
```

### Using Configuration

```go
app := framework.NewApplication()

// Get configuration values
appName := app.Config.GetString("app.name", "DefaultApp")
debug := app.Config.GetBool("app.debug", false)
port := app.Config.GetString("app.port", ":8080")
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
