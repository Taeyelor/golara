package framework

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/taeyelor/golara/framework/config"
	"github.com/taeyelor/golara/framework/container"
	"github.com/taeyelor/golara/framework/database"
	"github.com/taeyelor/golara/framework/routing"
)

// Application is the main application structure
type Application struct {
	Router    *routing.Router
	Container *container.Container
	Config    *config.Config
	server    *http.Server
}

// NewApplication creates a new application instance
func NewApplication() *Application {
	app := &Application{
		Router:    routing.NewRouter(),
		Container: container.NewContainer(),
		Config:    config.NewConfig(),
	}

	// Register core services
	app.registerCoreServices()

	return app
}

// registerCoreServices registers the core framework services
func (app *Application) registerCoreServices() {
	// Register core framework services
	app.Container.Singleton("config", func() interface{} {
		return app.Config
	})

	app.Container.Singleton("router", func() interface{} {
		return app.Router
	})

	// Auto-register database service (MongoDB ODM)
	app.Container.Singleton("db", func() interface{} {
		// Get database config
		uri := app.Config.Get("database.connections.mongodb.uri", "mongodb://localhost:27017").(string)
		dbName := app.Config.Get("database.connections.mongodb.database", "golara").(string)

		db, err := database.Connect(uri, dbName)
		if err != nil {
			log.Printf("Failed to connect to database: %v", err)
			return nil
		}
		return db
	})

	// Auto-register RabbitMQ service if enabled
	if app.Config.Get("rabbitmq.enabled", false).(bool) {
		// Register RabbitMQ factory function that will be lazy-loaded
		app.Container.Singleton("rabbitmq", app.createRabbitMQFactory())
	}
}

// Run starts the application server
func (app *Application) Run(addr string) error {
	if addr == "" {
		addr = app.Config.Get("app.port", ":8080").(string)
	}

	app.server = &http.Server{
		Addr:    addr,
		Handler: app.Router,
	}

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		log.Println("Shutting down server...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := app.server.Shutdown(ctx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
	}()

	log.Printf("Server starting on %s", addr)
	return app.server.ListenAndServe()
}

// Bind registers a service in the container
func (app *Application) Bind(name string, resolver func() interface{}) {
	app.Container.Bind(name, resolver)
}

// Singleton registers a singleton service in the container
func (app *Application) Singleton(name string, resolver func() interface{}) {
	app.Container.Singleton(name, resolver)
}

// Resolve resolves a service from the container
func (app *Application) Resolve(name string) interface{} {
	return app.Container.Resolve(name)
}

// Group creates a route group with common middleware and prefix
func (app *Application) Group(prefix string, middleware ...func(http.Handler) http.Handler) *routing.Group {
	return app.Router.Group(prefix, middleware...)
}

// GET registers a GET route
func (app *Application) GET(path string, handler interface{}) {
	app.Router.GET(path, handler)
}

// POST registers a POST route
func (app *Application) POST(path string, handler interface{}) {
	app.Router.POST(path, handler)
}

// PUT registers a PUT route
func (app *Application) PUT(path string, handler interface{}) {
	app.Router.PUT(path, handler)
}

// DELETE registers a DELETE route
func (app *Application) DELETE(path string, handler interface{}) {
	app.Router.DELETE(path, handler)
}

// PATCH registers a PATCH route
func (app *Application) PATCH(path string, handler interface{}) {
	app.Router.PATCH(path, handler)
}

// Use registers global middleware
func (app *Application) Use(middleware func(http.Handler) http.Handler) {
	app.Router.Use(middleware)
}

// createRabbitMQFactory creates a factory function for RabbitMQ service
// This avoids import cycles by using reflection and dynamic loading
func (app *Application) createRabbitMQFactory() func() interface{} {
	return func() interface{} {
		// Note: This is a placeholder implementation
		// The actual RabbitMQ instance should be created by the developer
		// using the provided registration helpers in the rabbitmq package
		log.Println("RabbitMQ service placeholder registered. Use rabbitmq.RegisterRabbitMQ() to initialize.")
		return map[string]interface{}{
			"type":    "placeholder",
			"message": "Use rabbitmq.RegisterRabbitMQ() to initialize RabbitMQ service",
		}
	}
}
