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
		"app.name":                             "GoLara",
		"app.env":                              "local",
		"app.debug":                            true,
		"app.port":                             ":8080",
		"app.key":                              "",
		"database.default":                     "mysql",
		"database.connections.mysql.driver":    "mysql",
		"database.connections.mysql.host":      "127.0.0.1",
		"database.connections.mysql.port":      "3306",
		"database.connections.mysql.database":  "",
		"database.connections.mysql.username":  "",
		"database.connections.mysql.password":  "",
		"database.connections.sqlite.driver":   "sqlite3",
		"database.connections.sqlite.database": "database.sqlite",
	}

	for key, value := range defaults {
		c.Set(key, value)
	}
}

// loadFromEnv loads configuration from environment variables
func (c *Config) loadFromEnv() {
	envMappings := map[string]string{
		"APP_NAME":      "app.name",
		"APP_ENV":       "app.env",
		"APP_DEBUG":     "app.debug",
		"APP_PORT":      "app.port",
		"APP_KEY":       "app.key",
		"DB_CONNECTION": "database.default",
		"DB_HOST":       "database.connections.mysql.host",
		"DB_PORT":       "database.connections.mysql.port",
		"DB_DATABASE":   "database.connections.mysql.database",
		"DB_USERNAME":   "database.connections.mysql.username",
		"DB_PASSWORD":   "database.connections.mysql.password",
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
