package main

import (
	"log"
	"net/http"

	"github.com/taeyelor/golara/framework"
	"github.com/taeyelor/golara/framework/database"
	httpMW "github.com/taeyelor/golara/framework/http"
	"github.com/taeyelor/golara/framework/routing"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	database.Model `bson:",inline"`
	Name           string `json:"name" bson:"name"`
	Email          string `json:"email" bson:"email"`
	Age            int    `json:"age" bson:"age"`
}

func main() {
	// Create new application
	app := framework.NewApplication()

	// Add global middleware
	app.Use(httpMW.LoggingMiddleware)
	app.Use(httpMW.RecoveryMiddleware)
	app.Use(httpMW.CORSMiddleware([]string{"*"}))

	// Connect to MongoDB
	mongoURI := app.Config.GetString("database.connections.mongodb.uri", "mongodb://localhost:27017")
	dbName := app.Config.GetString("database.connections.mongodb.database", "golara")

	db, err := database.Connect(mongoURI, dbName)
	if err != nil {
		log.Fatal("Failed to connect to MongoDB:", err)
	}
	defer db.Disconnect()

	// Register database in service container
	app.Singleton("database", func() interface{} {
		return db
	})

	// Define routes
	app.GET("/", homeHandler)
	app.GET("/users/{id}", getUserHandler)
	app.POST("/users", createUserHandler)
	app.GET("/users", listUsersHandler)
	app.PUT("/users/{id}", updateUserHandler)
	app.DELETE("/users/{id}", deleteUserHandler)

	// API group with common prefix
	api := app.Group("/api/v1")
	api.GET("/health", healthCheckHandler)
	api.GET("/stats", statsHandler)

	// Start server
	log.Println("Starting GoLara application with MongoDB...")
	if err := app.Run(":8080"); err != nil && err != http.ErrServerClosed {
		log.Fatal("Server failed to start:", err)
	}
}

// Handlers
func homeHandler(c *routing.Context) {
	c.JSON(200, map[string]interface{}{
		"message":  "Welcome to GoLara with MongoDB!",
		"version":  "1.0.0",
		"database": "MongoDB",
	})
}

func getUserHandler(c *routing.Context) {
	// Get database from context (you'd typically inject this)
	db := getDB() // helper function

	userID := c.Param("id")
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		c.JSON(400, map[string]string{"error": "Invalid user ID"})
		return
	}

	var user User
	err = db.NewQueryBuilder().
		Collection("users").
		Where("_id", "=", objectID).
		First(&user)

	if err != nil {
		c.JSON(404, map[string]string{"error": "User not found"})
		return
	}

	c.JSON(200, user)
}

func createUserHandler(c *routing.Context) {
	db := getDB()

	var user User
	if err := c.Bind(&user); err != nil {
		c.JSON(400, map[string]string{"error": "Invalid JSON"})
		return
	}

	// Insert user
	userID, err := db.NewQueryBuilder().
		Collection("users").
		Insert(user)

	if err != nil {
		c.JSON(500, map[string]string{"error": "Failed to create user"})
		return
	}

	user.ID = *userID
	c.JSON(201, user)
}

func listUsersHandler(c *routing.Context) {
	db := getDB()

	page := c.QueryDefault("page", "1")
	limit := c.QueryDefault("limit", "10")

	var users []User
	err := db.NewQueryBuilder().
		Collection("users").
		OrderBy("created_at", "DESC").
		Limit(10).
		Get(&users)

	if err != nil {
		c.JSON(500, map[string]string{"error": "Failed to fetch users"})
		return
	}

	// Count total users
	total, _ := db.NewQueryBuilder().
		Collection("users").
		Count()

	c.JSON(200, map[string]interface{}{
		"users": users,
		"pagination": map[string]interface{}{
			"page":  page,
			"limit": limit,
			"total": total,
		},
	})
}

func updateUserHandler(c *routing.Context) {
	db := getDB()

	userID := c.Param("id")
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		c.JSON(400, map[string]string{"error": "Invalid user ID"})
		return
	}

	var updateData map[string]interface{}
	if err := c.Bind(&updateData); err != nil {
		c.JSON(400, map[string]string{"error": "Invalid JSON"})
		return
	}

	// Update user
	result, err := db.NewQueryBuilder().
		Collection("users").
		Where("_id", "=", objectID).
		UpdateOne(bson.M{"$set": updateData})

	if err != nil {
		c.JSON(500, map[string]string{"error": "Failed to update user"})
		return
	}

	if result.MatchedCount == 0 {
		c.JSON(404, map[string]string{"error": "User not found"})
		return
	}

	c.JSON(200, map[string]string{"message": "User updated successfully"})
}

func deleteUserHandler(c *routing.Context) {
	db := getDB()

	userID := c.Param("id")
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		c.JSON(400, map[string]string{"error": "Invalid user ID"})
		return
	}

	// Delete user
	result, err := db.NewQueryBuilder().
		Collection("users").
		Where("_id", "=", objectID).
		DeleteOne()

	if err != nil {
		c.JSON(500, map[string]string{"error": "Failed to delete user"})
		return
	}

	if result.DeletedCount == 0 {
		c.JSON(404, map[string]string{"error": "User not found"})
		return
	}

	c.JSON(200, map[string]string{"message": "User deleted successfully"})
}

func healthCheckHandler(c *routing.Context) {
	db := getDB()

	// Check MongoDB connection
	if err := db.Ping(); err != nil {
		c.JSON(503, map[string]string{
			"status":   "error",
			"database": "disconnected",
		})
		return
	}

	c.JSON(200, map[string]string{
		"status":   "ok",
		"service":  "golara-api",
		"database": "connected",
	})
}

func statsHandler(c *routing.Context) {
	db := getDB()

	// Get user count
	userCount, _ := db.NewQueryBuilder().
		Collection("users").
		Count()

	c.JSON(200, map[string]interface{}{
		"total_users": userCount,
		"database":    "MongoDB",
		"collections": []string{"users"},
	})
}

// Helper function to get database (in real app, this would be injected)
func getDB() *database.DB {
	// This is a simplified example - in a real app you'd get this from the service container
	db, _ := database.Connect("mongodb://localhost:27017", "golara")
	return db
}
