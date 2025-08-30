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
	fmt.Println("GoLara CLI Tool")
	fmt.Println("Usage:")
	fmt.Println("  golara new <project-name>        Create a new GoLara project")
	fmt.Println("  golara make:controller <name>    Create a new controller")
	fmt.Println("  golara make:model <name>         Create a new model")
	fmt.Println("  golara serve                     Start the development server")
}

func createProject(name string) {
	fmt.Printf("Creating new GoLara project: %s\n", name)

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

	// Create main.go
	mainContent := fmt.Sprintf(`package main

import (
	"log"

	"github.com/taeyelor/golara/framework"
	httpMW "github.com/taeyelor/golara/framework/http"
	"github.com/taeyelor/golara/framework/routing"
)

func main() {
	app := framework.NewApplication()

	// Global middleware
	app.Use(httpMW.LoggingMiddleware)
	app.Use(httpMW.RecoveryMiddleware)
	app.Use(httpMW.CORSMiddleware([]string{"*"}))

	// Routes
	app.GET("/", func(c *routing.Context) {
		c.JSON(200, map[string]interface{}{
			"message": "Welcome to %s!",
			"framework": "GoLara",
		})
	})

	// Start server
	port := app.Config.GetString("app.port", ":8080")
	log.Printf("Starting %s on %%s", port)
	
	if err := app.Run(port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}`, name, name)

	writeFile(filepath.Join(name, "main.go"), mainContent)

	// Create go.mod
	modContent := fmt.Sprintf("module %s\n\ngo 1.21\n\nrequire github.com/taeyelor/golara v0.1.0\n", name)
	writeFile(filepath.Join(name, "go.mod"), modContent)

	// Create .env file
	envContent := fmt.Sprintf(`APP_NAME=%s
APP_ENV=local
APP_DEBUG=true
APP_PORT=:8080

DB_CONNECTION=mysql
DB_HOST=127.0.0.1
DB_PORT=3306
DB_DATABASE=%s
DB_USERNAME=
DB_PASSWORD=
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

	fmt.Printf("✅ Project %s created successfully!\n", name)
	fmt.Printf("📁 cd %s\n", name)
	fmt.Printf("🚀 go run main.go\n")
}

func createController(name string) {
	if !strings.HasSuffix(name, "Controller") {
		name += "Controller"
	}

	content := fmt.Sprintf(`package controllers

import "github.com/taeyelor/golara/framework/routing"

type %s struct{}

func New%s() *%s {
	return &%s{}
}

func (ctrl *%s) Index(c *routing.Context) {
	c.JSON(200, map[string]interface{}{
		"message": "%s index",
	})
}

func (ctrl *%s) Show(c *routing.Context) {
	id := c.Param("id")
	c.JSON(200, map[string]interface{}{
		"id": id,
		"message": "%s show",
	})
}

func (ctrl *%s) Store(c *routing.Context) {
	c.JSON(201, map[string]interface{}{
		"message": "%s created",
	})
}

func (ctrl *%s) Update(c *routing.Context) {
	id := c.Param("id")
	c.JSON(200, map[string]interface{}{
		"id": id,
		"message": "%s updated",
	})
}

func (ctrl *%s) Delete(c *routing.Context) {
	id := c.Param("id")
	c.JSON(200, map[string]interface{}{
		"id": id,
		"message": "%s deleted",
	})
}
`, name, name, name, name, name, name, name, name, name, name, name, name, name, name)

	filename := fmt.Sprintf("controllers/%s.go", strings.ToLower(strings.TrimSuffix(name, "Controller")))
	writeFile(filename, content)
	fmt.Printf("✅ Controller %s created at %s\n", name, filename)
}

func createModel(name string) {
	content := fmt.Sprintf(`package models

import (
	"time"
	"github.com/taeyelor/golara/framework/database"
)

type %s struct {
	database.Model
	Name        string    `+"`json:\"name\" db:\"name\"`"+`
	Email       string    `+"`json:\"email\" db:\"email\"`"+`
	// Add your fields here
}

func New%s() *%s {
	return &%s{}
}

func (m *%s) TableName() string {
	return "%s"
}

func (m *%s) FindByID(db *database.DB, id int) (*%s, error) {
	var model %s
	err := db.NewQueryBuilder().
		Table(m.TableName()).
		Where("id", "=", id).
		First(&model)
	
	if err != nil {
		return nil, err
	}
	
	return &model, nil
}

func (m *%s) All(db *database.DB) ([]%s, error) {
	var models []%s
	err := db.NewQueryBuilder().
		Table(m.TableName()).
		OrderBy("created_at", "DESC").
		Get(&models)
	
	return models, err
}

func (m *%s) Save(db *database.DB) error {
	if m.ID == 0 {
		// Insert
		m.CreatedAt = time.Now()
		m.UpdatedAt = time.Now()
		
		id, err := db.NewQueryBuilder().
			Table(m.TableName()).
			Insert(map[string]interface{}{
				"name":       m.Name,
				"email":      m.Email,
				"created_at": m.CreatedAt,
				"updated_at": m.UpdatedAt,
			})
		
		if err != nil {
			return err
		}
		
		m.ID = uint(id)
	} else {
		// Update
		m.UpdatedAt = time.Now()
		
		_, err := db.NewQueryBuilder().
			Table(m.TableName()).
			Where("id", "=", m.ID).
			Update(map[string]interface{}{
				"name":       m.Name,
				"email":      m.Email,
				"updated_at": m.UpdatedAt,
			})
		
		return err
	}
	
	return nil
}

func (m *%s) Delete(db *database.DB) error {
	_, err := db.NewQueryBuilder().
		Table(m.TableName()).
		Where("id", "=", m.ID).
		Delete()
	
	return err
}
`, name, name, name, name, name, strings.ToLower(name)+"s", name, name, name, name, name, name, name, name)

	filename := fmt.Sprintf("models/%s.go", strings.ToLower(name))
	writeFile(filename, content)
	fmt.Printf("✅ Model %s created at %s\n", name, filename)
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
