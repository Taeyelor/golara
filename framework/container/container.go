package container

import (
	"fmt"
	"sync"
)

// Container provides dependency injection capabilities
type Container struct {
	bindings   map[string]binding
	singletons map[string]interface{}
	mutex      sync.RWMutex
}

// binding represents a service binding
type binding struct {
	resolver  func() interface{}
	singleton bool
}

// NewContainer creates a new container instance
func NewContainer() *Container {
	return &Container{
		bindings:   make(map[string]binding),
		singletons: make(map[string]interface{}),
	}
}

// Bind registers a service resolver
func (c *Container) Bind(name string, resolver func() interface{}) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.bindings[name] = binding{
		resolver:  resolver,
		singleton: false,
	}
}

// Singleton registers a singleton service resolver
func (c *Container) Singleton(name string, resolver func() interface{}) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.bindings[name] = binding{
		resolver:  resolver,
		singleton: true,
	}
}

// Resolve resolves a service from the container
func (c *Container) Resolve(name string) interface{} {
	c.mutex.RLock()

	// Check if singleton instance exists
	if instance, exists := c.singletons[name]; exists {
		c.mutex.RUnlock()
		return instance
	}

	// Check if binding exists
	binding, exists := c.bindings[name]
	if !exists {
		c.mutex.RUnlock()
		panic(fmt.Sprintf("Service '%s' not found in container", name))
	}

	c.mutex.RUnlock()

	// Resolve the service
	instance := binding.resolver()

	// Store singleton instance
	if binding.singleton {
		c.mutex.Lock()
		c.singletons[name] = instance
		c.mutex.Unlock()
	}

	return instance
}

// Has checks if a service is registered
func (c *Container) Has(name string) bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	_, exists := c.bindings[name]
	return exists
}

// Remove removes a service binding
func (c *Container) Remove(name string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	delete(c.bindings, name)
	delete(c.singletons, name)
}

// Clear removes all bindings
func (c *Container) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.bindings = make(map[string]binding)
	c.singletons = make(map[string]interface{})
}

// Instance registers an existing instance as a singleton
func (c *Container) Instance(name string, instance interface{}) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.singletons[name] = instance
	c.bindings[name] = binding{
		resolver: func() interface{} {
			return instance
		},
		singleton: true,
	}
}
