package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		showUsage()
		return
	}

	command := os.Args[1]

	switch command {
	case "new":
		if len(os.Args) < 3 {
			fmt.Println("Usage: golara new <project-name>")
			return
		}
		createProject(os.Args[2])
	case "make:controller":
		if len(os.Args) < 3 {
			fmt.Println("Usage: golara make:controller <controller-name>")
			return
		}
		createController(os.Args[2])
	case "make:model":
		if len(os.Args) < 3 {
			fmt.Println("Usage: golara make:model <model-name>")
			return
		}
		createModel(os.Args[2])
	case "serve":
		serveApp()
	default:
		showUsage()
	}
}

func showUsage() {
	fmt.Println("GoLara CLI Tool (MongoDB Edition)")
	fmt.Println("Usage:")
	fmt.Println("  golara new <project-name>        Create a new GoLara project")
	fmt.Println("  golara make:controller <name>    Create a new controller")
	fmt.Println("  golara make:model <name>         Create a new MongoDB model")
	fmt.Println("  golara serve                     Start the development server")
}

func createProject(name string) {
	fmt.Printf("Creating new GoLara project with MongoDB: %s\n", name)

	// Create project directory
	if err := os.MkdirAll(name, 0755); err != nil {
		fmt.Printf("Error creating directory: %v\n", err)
		return
	}

	// Create project structure
	dirs := []string{
		"controllers",
		"models",
		"middleware",
		"views",
		"config",
		"routes",
		"public",
		"storage/logs",
	}

	for _, dir := range dirs {
		fullPath := filepath.Join(name, dir)
		if err := os.MkdirAll(fullPath, 0755); err != nil {
			fmt.Printf("Error creating directory %s: %v\n", dir, err)
			return
		}
	}

	// Create main.go with MongoDB support
	mainContent := fmt.Sprintf(`package main

import (
	"log"

	"github.com/taeyelor/golara/framework"
	"github.com/taeyelor/golara/framework/database"
	httpMW "github.com/taeyelor/golara/framework/http"
	"github.com/taeyelor/golara/framework/routing"
)

func main() {
	app := framework.NewApplication()

	// Global middleware
	app.Use(httpMW.LoggingMiddleware)
	app.Use(httpMW.RecoveryMiddleware)
	app.Use(httpMW.CORSMiddleware([]string{"*"}))

	// Connect to MongoDB
	mongoURI := app.Config.GetString("database.connections.mongodb.uri", "mongodb://localhost:27017")
	dbName := app.Config.GetString("database.connections.mongodb.database", "%s")
	
	db, err := database.Connect(mongoURI, dbName)
	if err != nil {
		log.Fatal("Failed to connect to MongoDB:", err)
	}
	defer db.Disconnect()

	// Register database in service container
	app.Singleton("database", func() interface{} {
		return db
	})

	// Routes
	app.GET("/", func(c *routing.Context) {
		c.JSON(200, map[string]interface{}{
			"message": "Welcome to %s!",
			"framework": "GoLara",
			"database": "MongoDB",
		})
	})

	// Health check
	app.GET("/health", func(c *routing.Context) {
		if err := db.Ping(); err != nil {
			c.JSON(503, map[string]string{"status": "error", "database": "disconnected"})
			return
		}
		c.JSON(200, map[string]string{"status": "ok", "database": "connected"})
	})

	// Start server
	port := app.Config.GetString("app.port", ":8080")
	log.Printf("Starting %s on %%s with MongoDB", port)
	
	if err := app.Run(port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}`, name, name, name)

	writeFile(filepath.Join(name, "main.go"), mainContent)

	// Create go.mod
	modContent := fmt.Sprintf("module %s\n\ngo 1.21\n\nrequire (\n\tgithub.com/taeyelor/golara v0.1.0\n\tgo.mongodb.org/mongo-driver v1.12.1\n)\n", name)
	writeFile(filepath.Join(name, "go.mod"), modContent)

	// Create .env file with MongoDB configuration
	envContent := fmt.Sprintf(`APP_NAME=%s
APP_ENV=local
APP_DEBUG=true
APP_PORT=:8080

DB_CONNECTION=mongodb
MONGODB_URI=mongodb://localhost:27017
MONGODB_DATABASE=%s
`, name, name)
	writeFile(filepath.Join(name, ".env"), envContent)

	// Create .gitignore
	gitignoreContent := `.env
*.log
tmp/
dist/
vendor/
.DS_Store
`
	writeFile(filepath.Join(name, ".gitignore"), gitignoreContent)

	// Create docker-compose.yml for MongoDB
	dockerComposeContent := `version: '3.8'
services:
  mongodb:
    image: mongo:6.0
    container_name: golara_mongodb
    restart: always
    ports:
      - "27017:27017"
    environment:
      MONGO_INITDB_ROOT_USERNAME: admin
      MONGO_INITDB_ROOT_PASSWORD: password
    volumes:
      - mongodb_data:/data/db

volumes:
  mongodb_data:
`
	writeFile(filepath.Join(name, "docker-compose.yml"), dockerComposeContent)

	fmt.Printf("✅ Project %s created successfully with MongoDB support!\n", name)
	fmt.Printf("📁 cd %s\n", name)
	fmt.Printf("🐳 docker-compose up -d  # Start MongoDB\n")
	fmt.Printf("🚀 go run main.go\n")
}

func createController(name string) {
	if !strings.HasSuffix(name, "Controller") {
		name += "Controller"
	}

	content := fmt.Sprintf(`package controllers

import (
	"github.com/taeyelor/golara/framework/database"
	"github.com/taeyelor/golara/framework/routing"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type %s struct {
	db *database.DB
}

func New%s(db *database.DB) *%s {
	return &%s{db: db}
}

func (ctrl *%s) Index(c *routing.Context) {
	// Get all documents from collection
	var results []bson.M
	err := ctrl.db.NewQueryBuilder().
		Collection("%s").
		OrderBy("created_at", "DESC").
		Get(&results)
	
	if err != nil {
		c.JSON(500, map[string]string{"error": "Failed to fetch data"})
		return
	}
	
	c.JSON(200, map[string]interface{}{
		"message": "%s index",
		"data":    results,
	})
}

func (ctrl *%s) Show(c *routing.Context) {
	id := c.Param("id")
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		c.JSON(400, map[string]string{"error": "Invalid ID"})
		return
	}
	
	var result bson.M
	err = ctrl.db.NewQueryBuilder().
		Collection("%s").
		Where("_id", "=", objectID).
		First(&result)
	
	if err != nil {
		c.JSON(404, map[string]string{"error": "Document not found"})
		return
	}
	
	c.JSON(200, map[string]interface{}{
		"message": "%s show",
		"data":    result,
	})
}

func (ctrl *%s) Store(c *routing.Context) {
	var data bson.M
	if err := c.Bind(&data); err != nil {
		c.JSON(400, map[string]string{"error": "Invalid JSON"})
		return
	}
	
	id, err := ctrl.db.NewQueryBuilder().
		Collection("%s").
		Insert(data)
	
	if err != nil {
		c.JSON(500, map[string]string{"error": "Failed to create document"})
		return
	}
	
	c.JSON(201, map[string]interface{}{
		"message": "%s created",
		"id":      id,
	})
}

func (ctrl *%s) Update(c *routing.Context) {
	id := c.Param("id")
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		c.JSON(400, map[string]string{"error": "Invalid ID"})
		return
	}
	
	var updateData bson.M
	if err := c.Bind(&updateData); err != nil {
		c.JSON(400, map[string]string{"error": "Invalid JSON"})
		return
	}
	
	result, err := ctrl.db.NewQueryBuilder().
		Collection("%s").
		Where("_id", "=", objectID).
		UpdateOne(bson.M{"$set": updateData})
	
	if err != nil {
		c.JSON(500, map[string]string{"error": "Failed to update document"})
		return
	}
	
	if result.MatchedCount == 0 {
		c.JSON(404, map[string]string{"error": "Document not found"})
		return
	}
	
	c.JSON(200, map[string]interface{}{
		"message": "%s updated",
		"id":      id,
	})
}

func (ctrl *%s) Delete(c *routing.Context) {
	id := c.Param("id")
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		c.JSON(400, map[string]string{"error": "Invalid ID"})
		return
	}
	
	result, err := ctrl.db.NewQueryBuilder().
		Collection("%s").
		Where("_id", "=", objectID).
		DeleteOne()
	
	if err != nil {
		c.JSON(500, map[string]string{"error": "Failed to delete document"})
		return
	}
	
	if result.DeletedCount == 0 {
		c.JSON(404, map[string]string{"error": "Document not found"})
		return
	}
	
	c.JSON(200, map[string]interface{}{
		"message": "%s deleted",
		"id":      id,
	})
}
`, name, name, name, name, name, strings.ToLower(strings.TrimSuffix(name, "Controller"))+"s", name, name, strings.ToLower(strings.TrimSuffix(name, "Controller"))+"s", name, name, strings.ToLower(strings.TrimSuffix(name, "Controller"))+"s", name, name, strings.ToLower(strings.TrimSuffix(name, "Controller"))+"s", name, name, strings.ToLower(strings.TrimSuffix(name, "Controller"))+"s", name)

	filename := fmt.Sprintf("controllers/%s.go", strings.ToLower(strings.TrimSuffix(name, "Controller")))
	writeFile(filename, content)
	fmt.Printf("✅ Controller %s created at %s\n", name, filename)
}

func createModel(name string) {
	content := fmt.Sprintf(`package models

import (
	"context"
	"time"
	"github.com/taeyelor/golara/framework/database"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type %s struct {
	database.Model `+"`bson:\",inline\"`"+`
	Name           string    `+"`json:\"name\" bson:\"name\"`"+`
	Email          string    `+"`json:\"email\" bson:\"email\"`"+`
	// Add your fields here
}

func New%s() *%s {
	return &%s{}
}

func (m *%s) CollectionName() string {
	return "%s"
}

func (m *%s) FindByID(db *database.DB, id primitive.ObjectID) (*%s, error) {
	var model %s
	err := db.NewQueryBuilder().
		Collection(m.CollectionName()).
		Where("_id", "=", id).
		First(&model)
	
	if err != nil {
		return nil, err
	}
	
	return &model, nil
}

func (m *%s) All(db *database.DB) ([]%s, error) {
	var models []%s
	err := db.NewQueryBuilder().
		Collection(m.CollectionName()).
		OrderBy("created_at", "DESC").
		Get(&models)
	
	return models, err
}

func (m *%s) Save(db *database.DB) error {
	if m.ID.IsZero() {
		// Insert
		m.SetTimestamps()
		
		id, err := db.NewQueryBuilder().
			Collection(m.CollectionName()).
			Insert(m)
		
		if err != nil {
			return err
		}
		
		m.ID = *id
	} else {
		// Update
		m.BeforeUpdate()
		
		_, err := db.NewQueryBuilder().
			Collection(m.CollectionName()).
			Where("_id", "=", m.ID).
			UpdateOne(bson.M{"$set": bson.M{
				"name":       m.Name,
				"email":      m.Email,
				"updated_at": m.UpdatedAt,
			}})
		
		return err
	}
	
	return nil
}

func (m *%s) Delete(db *database.DB) error {
	_, err := db.NewQueryBuilder().
		Collection(m.CollectionName()).
		Where("_id", "=", m.ID).
		DeleteOne()
	
	return err
}

// Static methods for querying
func Find%sByEmail(db *database.DB, email string) (*%s, error) {
	var model %s
	err := db.NewQueryBuilder().
		Collection("%s").
		Where("email", "=", email).
		First(&model)
	
	if err != nil {
		return nil, err
	}
	
	return &model, nil
}

func Count%s(db *database.DB) (int64, error) {
	return db.NewQueryBuilder().
		Collection("%s").
		Count()
}

func (m *%s) BeforeInsert() {
	m.SetTimestamps()
}

func (m *%s) BeforeUpdate() {
	m.UpdatedAt = time.Now()
}
`, name, name, name, name, name, strings.ToLower(name)+"s", name, name, name, name, name, name, name, name, name, name, name, strings.ToLower(name)+"s", name, name, strings.ToLower(name)+"s", name, strings.ToLower(name)+"s", name, name)

	filename := fmt.Sprintf("models/%s.go", strings.ToLower(name))
	writeFile(filename, content)
	fmt.Printf("✅ Model %s created at %s (MongoDB ODM)\n", name, filename)
}

func serveApp() {
	fmt.Println("🚀 Starting development server...")
	cmd := exec.Command("go", "run", "main.go")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}

func writeFile(filename, content string) {
	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		fmt.Printf("Error writing file %s: %v\n", filename, err)
	}
}
