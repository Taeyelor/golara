package main

import (
	"log"
	"net/http"

	"github.com/taeyelor/golara/framework"
	httpMW "github.com/taeyelor/golara/framework/http"
	"github.com/taeyelor/golara/framework/routing"
)

func main() {
	// Create new application
	app := framework.NewApplication()

	// Add global middleware
	app.Use(httpMW.LoggingMiddleware)
	app.Use(httpMW.RecoveryMiddleware)
	app.Use(httpMW.CORSMiddleware([]string{"*"}))

	// Define routes
	app.GET("/", homeHandler)
	app.GET("/users/{id}", getUserHandler)
	app.POST("/users", createUserHandler)

	// API group with common prefix
	api := app.Group("/api/v1")
	api.GET("/health", healthCheckHandler)
	api.GET("/users", listUsersHandler)

	// Start server
	log.Println("Starting GoLara application...")
	if err := app.Run(":8080"); err != nil && err != http.ErrServerClosed {
		log.Fatal("Server failed to start:", err)
	}
}

// Handlers
func homeHandler(c *routing.Context) {
	c.JSON(200, map[string]interface{}{
		"message": "Welcome to GoLara!",
		"version": "1.0.0",
	})
}

func getUserHandler(c *routing.Context) {
	userID := c.Param("id")
	c.JSON(200, map[string]interface{}{
		"user_id": userID,
		"name":    "John Doe",
		"email":   "john@example.com",
	})
}

func createUserHandler(c *routing.Context) {
	var user map[string]interface{}
	if err := c.Bind(&user); err != nil {
		c.JSON(400, map[string]string{"error": "Invalid JSON"})
		return
	}

	// Simulate user creation
	user["id"] = 123
	c.JSON(201, user)
}

func healthCheckHandler(c *routing.Context) {
	c.JSON(200, map[string]string{
		"status":  "ok",
		"service": "golara-api",
	})
}

func listUsersHandler(c *routing.Context) {
	page := c.QueryDefault("page", "1")
	limit := c.QueryDefault("limit", "10")

	c.JSON(200, map[string]interface{}{
		"users": []map[string]interface{}{
			{"id": 1, "name": "John Doe", "email": "john@example.com"},
			{"id": 2, "name": "Jane Smith", "email": "jane@example.com"},
		},
		"pagination": map[string]string{
			"page":  page,
			"limit": limit,
		},
	})
}
