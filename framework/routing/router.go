package routing

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

// Router handles HTTP routing
type Router struct {
	routes      []*Route
	middlewares []func(http.Handler) http.Handler
}

// Route represents a single route
type Route struct {
	Method      string
	Pattern     string
	Handler     interface{}
	Middlewares []func(http.Handler) http.Handler
	regex       *regexp.Regexp
	paramNames  []string
}

// Group represents a route group
type Group struct {
	router      *Router
	prefix      string
	middlewares []func(http.Handler) http.Handler
}

// NewRouter creates a new router instance
func NewRouter() *Router {
	return &Router{
		routes:      make([]*Route, 0),
		middlewares: make([]func(http.Handler) http.Handler, 0),
	}
}

// ServeHTTP implements the http.Handler interface
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// Find matching route
	route, params := r.findRoute(req.Method, req.URL.Path)
	if route == nil {
		http.NotFound(w, req)
		return
	}

	// Create context with parameters
	ctx := NewContext(w, req, params)

	// Build middleware chain
	handler := r.buildHandler(route.Handler, ctx)

	// Apply route-specific middleware
	for i := len(route.Middlewares) - 1; i >= 0; i-- {
		handler = route.Middlewares[i](handler)
	}

	// Apply global middleware
	for i := len(r.middlewares) - 1; i >= 0; i-- {
		handler = r.middlewares[i](handler)
	}

	handler.ServeHTTP(w, req)
}

// findRoute finds a matching route for the given method and path
func (r *Router) findRoute(method, path string) (*Route, map[string]string) {
	for _, route := range r.routes {
		if route.Method != method {
			continue
		}

		if route.regex != nil {
			matches := route.regex.FindStringSubmatch(path)
			if matches != nil {
				params := make(map[string]string)
				for i, name := range route.paramNames {
					if i+1 < len(matches) {
						params[name] = matches[i+1]
					}
				}
				return route, params
			}
		} else if route.Pattern == path {
			return route, make(map[string]string)
		}
	}
	return nil, nil
}

// buildHandler creates an http.Handler from various handler types
func (r *Router) buildHandler(handler interface{}, ctx *Context) http.Handler {
	switch h := handler.(type) {
	case func(*Context):
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			h(ctx)
		})
	case func(http.ResponseWriter, *http.Request):
		return http.HandlerFunc(h)
	case http.Handler:
		return h
	case http.HandlerFunc:
		return h
	default:
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			http.Error(w, "Invalid handler type", http.StatusInternalServerError)
		})
	}
}

// addRoute adds a new route to the router
func (r *Router) addRoute(method, pattern string, handler interface{}) {
	route := &Route{
		Method:      method,
		Pattern:     pattern,
		Handler:     handler,
		Middlewares: make([]func(http.Handler) http.Handler, 0),
	}

	// Compile regex for parameterized routes
	if strings.Contains(pattern, "{") {
		route.regex, route.paramNames = r.compilePattern(pattern)
	}

	r.routes = append(r.routes, route)
}

// compilePattern compiles a route pattern with parameters into a regex
func (r *Router) compilePattern(pattern string) (*regexp.Regexp, []string) {
	var paramNames []string
	regexPattern := pattern

	// Find all parameters in the pattern
	paramRegex := regexp.MustCompile(`\{([^}]+)\}`)
	matches := paramRegex.FindAllStringSubmatch(pattern, -1)

	for _, match := range matches {
		paramName := match[1]
		paramNames = append(paramNames, paramName)

		// Replace {param} with capturing group
		regexPattern = strings.Replace(regexPattern, match[0], `([^/]+)`, 1)
	}

	regex, err := regexp.Compile("^" + regexPattern + "$")
	if err != nil {
		panic(fmt.Sprintf("Invalid route pattern: %s", pattern))
	}

	return regex, paramNames
}

// HTTP method methods
func (r *Router) GET(path string, handler interface{}) {
	r.addRoute("GET", path, handler)
}

func (r *Router) POST(path string, handler interface{}) {
	r.addRoute("POST", path, handler)
}

func (r *Router) PUT(path string, handler interface{}) {
	r.addRoute("PUT", path, handler)
}

func (r *Router) DELETE(path string, handler interface{}) {
	r.addRoute("DELETE", path, handler)
}

func (r *Router) PATCH(path string, handler interface{}) {
	r.addRoute("PATCH", path, handler)
}

// Use adds global middleware
func (r *Router) Use(middleware func(http.Handler) http.Handler) {
	r.middlewares = append(r.middlewares, middleware)
}

// Group creates a new route group
func (r *Router) Group(prefix string, middlewares ...func(http.Handler) http.Handler) *Group {
	return &Group{
		router:      r,
		prefix:      strings.TrimSuffix(prefix, "/"),
		middlewares: middlewares,
	}
}

// Group methods
func (g *Group) GET(path string, handler interface{}) {
	g.addRoute("GET", path, handler)
}

func (g *Group) POST(path string, handler interface{}) {
	g.addRoute("POST", path, handler)
}

func (g *Group) PUT(path string, handler interface{}) {
	g.addRoute("PUT", path, handler)
}

func (g *Group) DELETE(path string, handler interface{}) {
	g.addRoute("DELETE", path, handler)
}

func (g *Group) PATCH(path string, handler interface{}) {
	g.addRoute("PATCH", path, handler)
}

func (g *Group) addRoute(method, path string, handler interface{}) {
	fullPath := g.prefix + path
	route := &Route{
		Method:      method,
		Pattern:     fullPath,
		Handler:     handler,
		Middlewares: g.middlewares,
	}

	// Compile regex for parameterized routes
	if strings.Contains(fullPath, "{") {
		route.regex, route.paramNames = g.router.compilePattern(fullPath)
	}

	g.router.routes = append(g.router.routes, route)
}
