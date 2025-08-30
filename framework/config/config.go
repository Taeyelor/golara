package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
)

// Config provides configuration management
type Config struct {
	data  map[string]interface{}
	mutex sync.RWMutex
}

// NewConfig creates a new config instance
func NewConfig() *Config {
	config := &Config{
		data: make(map[string]interface{}),
	}

	// Load default configuration
	config.loadDefaults()

	// Load environment variables
	config.loadFromEnv()

	return config
}

// loadDefaults sets default configuration values
func (c *Config) loadDefaults() {
	defaults := map[string]interface{}{
		"app.name":                              "GoLara",
		"app.env":                               "local",
		"app.debug":                             true,
		"app.port":                              ":8080",
		"app.key":                               "",
		"database.default":                      "mongodb",
		"database.connections.mongodb.uri":      "mongodb://localhost:27017",
		"database.connections.mongodb.database": "golara",
		"database.connections.mongodb.options": map[string]interface{}{
			"maxPoolSize": 10,
			"timeout":     "5s",
		},
		"rabbitmq.url":                   "amqp://guest:guest@localhost:5672/",
		"rabbitmq.reconnect_delay":       "5s",
		"rabbitmq.reconnect_attempts":    10,
		"rabbitmq.enable_heartbeat":      true,
		"rabbitmq.heartbeat_interval":    "10s",
		"rabbitmq.channel_pool_size":     10,
		"rabbitmq.auto_declare_queues":   true,
		"rabbitmq.auto_declare_exchange": true,
	}

	for key, value := range defaults {
		c.Set(key, value)
	}
}

// loadFromEnv loads configuration from environment variables
func (c *Config) loadFromEnv() {
	envMappings := map[string]string{
		// App configuration
		"APP_NAME":  "app.name",
		"APP_ENV":   "app.env",
		"APP_DEBUG": "app.debug",
		"APP_PORT":  "app.port",
		"APP_KEY":   "app.key",

		// Database configuration
		"DB_CONNECTION":    "database.default",
		"MONGODB_URI":      "database.connections.mongodb.uri",
		"DB_DATABASE":      "database.connections.mongodb.database",
		"MONGODB_DATABASE": "database.connections.mongodb.database",

		// RabbitMQ configuration
		"RABBITMQ_URL":                   "rabbitmq.url",
		"RABBITMQ_RECONNECT_DELAY":       "rabbitmq.reconnect_delay",
		"RABBITMQ_RECONNECT_ATTEMPTS":    "rabbitmq.reconnect_attempts",
		"RABBITMQ_ENABLE_HEARTBEAT":      "rabbitmq.enable_heartbeat",
		"RABBITMQ_HEARTBEAT_INTERVAL":    "rabbitmq.heartbeat_interval",
		"RABBITMQ_CHANNEL_POOL_SIZE":     "rabbitmq.channel_pool_size",
		"RABBITMQ_AUTO_DECLARE_QUEUES":   "rabbitmq.auto_declare_queues",
		"RABBITMQ_AUTO_DECLARE_EXCHANGE": "rabbitmq.auto_declare_exchange",
	}

	for envKey, configKey := range envMappings {
		if value := os.Getenv(envKey); value != "" {
			c.Set(configKey, c.parseEnvValue(value))
		}
	}
}

// parseEnvValue parses environment variable value to appropriate type
func (c *Config) parseEnvValue(value string) interface{} {
	// Try to parse as boolean
	if strings.ToLower(value) == "true" {
		return true
	}
	if strings.ToLower(value) == "false" {
		return false
	}

	// Try to parse as integer
	if intValue, err := strconv.Atoi(value); err == nil {
		return intValue
	}

	// Try to parse as float
	if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
		return floatValue
	}

	// Return as string
	return value
}

// Get retrieves a configuration value by key
func (c *Config) Get(key string, defaultValue ...interface{}) interface{} {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	value := c.getNestedValue(key)
	if value == nil && len(defaultValue) > 0 {
		return defaultValue[0]
	}

	return value
}

// GetString gets a string configuration value
func (c *Config) GetString(key string, defaultValue ...string) string {
	value := c.Get(key)
	if value == nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return ""
	}

	if str, ok := value.(string); ok {
		return str
	}

	return fmt.Sprintf("%v", value)
}

// GetInt gets an integer configuration value
func (c *Config) GetInt(key string, defaultValue ...int) int {
	value := c.Get(key)
	if value == nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}

	if intVal, ok := value.(int); ok {
		return intVal
	}

	if str, ok := value.(string); ok {
		if intVal, err := strconv.Atoi(str); err == nil {
			return intVal
		}
	}

	if len(defaultValue) > 0 {
		return defaultValue[0]
	}

	return 0
}

// GetBool gets a boolean configuration value
func (c *Config) GetBool(key string, defaultValue ...bool) bool {
	value := c.Get(key)
	if value == nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return false
	}

	if boolVal, ok := value.(bool); ok {
		return boolVal
	}

	if str, ok := value.(string); ok {
		return strings.ToLower(str) == "true"
	}

	if len(defaultValue) > 0 {
		return defaultValue[0]
	}

	return false
}

// Set sets a configuration value
func (c *Config) Set(key string, value interface{}) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.setNestedValue(key, value)
}

// getNestedValue retrieves a nested configuration value
func (c *Config) getNestedValue(key string) interface{} {
	keys := strings.Split(key, ".")
	current := c.data

	for i, k := range keys {
		if i == len(keys)-1 {
			return current[k]
		}

		if next, ok := current[k].(map[string]interface{}); ok {
			current = next
		} else {
			return nil
		}
	}

	return nil
}

// setNestedValue sets a nested configuration value
func (c *Config) setNestedValue(key string, value interface{}) {
	keys := strings.Split(key, ".")
	current := c.data

	for i, k := range keys {
		if i == len(keys)-1 {
			current[k] = value
			return
		}

		if _, exists := current[k]; !exists {
			current[k] = make(map[string]interface{})
		}

		if next, ok := current[k].(map[string]interface{}); ok {
			current = next
		} else {
			// Overwrite non-map value with map
			current[k] = make(map[string]interface{})
			current = current[k].(map[string]interface{})
		}
	}
}

// LoadFromFile loads configuration from a JSON file
func (c *Config) LoadFromFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	var data map[string]interface{}
	if err := json.NewDecoder(file).Decode(&data); err != nil {
		return err
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.mergeData(data)
	return nil
}

// mergeData merges new data into existing configuration
func (c *Config) mergeData(data map[string]interface{}) {
	for key, value := range data {
		if existing, exists := c.data[key]; exists {
			if existingMap, ok := existing.(map[string]interface{}); ok {
				if valueMap, ok := value.(map[string]interface{}); ok {
					c.mergeMap(existingMap, valueMap)
					continue
				}
			}
		}
		c.data[key] = value
	}
}

// mergeMap recursively merges two maps
func (c *Config) mergeMap(existing, new map[string]interface{}) {
	for key, value := range new {
		if existingValue, exists := existing[key]; exists {
			if existingMap, ok := existingValue.(map[string]interface{}); ok {
				if valueMap, ok := value.(map[string]interface{}); ok {
					c.mergeMap(existingMap, valueMap)
					continue
				}
			}
		}
		existing[key] = value
	}
}

// All returns all configuration data
func (c *Config) All() map[string]interface{} {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	// Create a deep copy
	result := make(map[string]interface{})
	c.copyMap(c.data, result)
	return result
}

// copyMap creates a deep copy of a map
func (c *Config) copyMap(src, dst map[string]interface{}) {
	for key, value := range src {
		if valueMap, ok := value.(map[string]interface{}); ok {
			dst[key] = make(map[string]interface{})
			c.copyMap(valueMap, dst[key].(map[string]interface{}))
		} else {
			dst[key] = value
		}
	}
}

// GetDatabaseConfig returns database configuration
func (c *Config) GetDatabaseConfig() map[string]interface{} {
	return map[string]interface{}{
		"default": c.GetString("database.default"),
		"mongodb": map[string]interface{}{
			"uri":      c.GetString("database.connections.mongodb.uri"),
			"database": c.GetString("database.connections.mongodb.database"),
			"options":  c.Get("database.connections.mongodb.options"),
		},
	}
}

// GetRabbitMQConfig returns RabbitMQ configuration
func (c *Config) GetRabbitMQConfig() map[string]interface{} {
	return map[string]interface{}{
		"url":                   c.GetString("rabbitmq.url"),
		"reconnect_delay":       c.GetString("rabbitmq.reconnect_delay"),
		"reconnect_attempts":    c.GetInt("rabbitmq.reconnect_attempts"),
		"enable_heartbeat":      c.GetBool("rabbitmq.enable_heartbeat"),
		"heartbeat_interval":    c.GetString("rabbitmq.heartbeat_interval"),
		"channel_pool_size":     c.GetInt("rabbitmq.channel_pool_size"),
		"auto_declare_queues":   c.GetBool("rabbitmq.auto_declare_queues"),
		"auto_declare_exchange": c.GetBool("rabbitmq.auto_declare_exchange"),
	}
}

// GetAppConfig returns application configuration
func (c *Config) GetAppConfig() map[string]interface{} {
	return map[string]interface{}{
		"name":  c.GetString("app.name"),
		"env":   c.GetString("app.env"),
		"debug": c.GetBool("app.debug"),
		"port":  c.GetString("app.port"),
		"key":   c.GetString("app.key"),
	}
}
