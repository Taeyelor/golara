package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/taeyelor/golara/framework"
	"github.com/taeyelor/golara/framework/database"
	"github.com/taeyelor/golara/framework/rabbitmq"
)

func main() {
	// Create a new GoLara application - this automatically registers services
	app := framework.NewApplication()

	// The following services are automatically registered:
	// - "config": Configuration service
	// - "router": Router service
	// - "db": MongoDB database connection (if configured)
	// - "rabbitmq": RabbitMQ placeholder (if enabled in config)

	// For RabbitMQ, you need to manually call the registration to avoid import cycles
	if app.Config.Get("rabbitmq.enabled", false).(bool) {
		rabbitmq.RegisterRabbitMQ(app, nil) // Uses config from app.Config
	}

	// Example route to show database service
	app.GET("/db-status", func(w http.ResponseWriter, r *http.Request) {
		db := app.Resolve("db")
		if db == nil {
			http.Error(w, "Database not connected", http.StatusServiceUnavailable)
			return
		}

		// Type assert to database.DB
		dbInstance, ok := db.(*database.DB)
		if !ok {
			http.Error(w, "Invalid database service", http.StatusInternalServerError)
			return
		}

		// Check database health
		err := dbInstance.Ping()
		if err != nil {
			http.Error(w, "Database ping failed: "+err.Error(), http.StatusServiceUnavailable)
			return
		}

		response := map[string]interface{}{
			"status":   "ok",
			"database": dbInstance.Name,
			"message":  "Database connection is healthy",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	// Example route to show RabbitMQ service
	app.GET("/rabbitmq-status", func(w http.ResponseWriter, r *http.Request) {
		rabbit := app.Resolve("rabbitmq")
		if rabbit == nil {
			http.Error(w, "RabbitMQ not available", http.StatusServiceUnavailable)
			return
		}

		// Check if it's the placeholder or actual service
		if placeholder, ok := rabbit.(map[string]interface{}); ok {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(placeholder)
			return
		}

		// If it's the actual RabbitMQ service
		if rabbitService, ok := rabbit.(*rabbitmq.RabbitMQ); ok {
			err := rabbitService.Health()
			status := "ok"
			message := "RabbitMQ connection is healthy"

			if err != nil {
				status = "error"
				message = err.Error()
			}

			response := map[string]interface{}{
				"status":  status,
				"message": message,
				"stats":   rabbitService.Stats(),
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}

		http.Error(w, "Unknown RabbitMQ service type", http.StatusInternalServerError)
	})

	// Example route to show all registered services
	app.GET("/services", func(w http.ResponseWriter, r *http.Request) {
		services := []string{"config", "router", "db"}

		if app.Config.Get("rabbitmq.enabled", false).(bool) {
			services = append(services, "rabbitmq")
		}

		response := map[string]interface{}{
			"auto_registered_services": services,
			"message":                  "These services are automatically registered when calling framework.NewApplication()",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	log.Println("Auto-registration example starting...")
	log.Println("Available endpoints:")
	log.Println("  GET /services       - List auto-registered services")
	log.Println("  GET /db-status      - Check database connection")
	log.Println("  GET /rabbitmq-status - Check RabbitMQ connection")

	log.Fatal(app.Run(":8080"))
}
