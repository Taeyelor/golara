package routing

import (
	"encoding/json"
	"net/http"
	"strconv"
)

// Context provides request context and response helpers
type Context struct {
	Writer  http.ResponseWriter
	Request *http.Request
	Params  map[string]string
}

// NewContext creates a new context instance
func NewContext(w http.ResponseWriter, r *http.Request, params map[string]string) *Context {
	return &Context{
		Writer:  w,
		Request: r,
		Params:  params,
	}
}

// Param gets a URL parameter by name
func (c *Context) Param(name string) string {
	return c.Params[name]
}

// ParamInt gets a URL parameter as integer
func (c *Context) ParamInt(name string) (int, error) {
	return strconv.Atoi(c.Params[name])
}

// Query gets a query parameter
func (c *Context) Query(name string) string {
	return c.Request.URL.Query().Get(name)
}

// QueryDefault gets a query parameter with default value
func (c *Context) QueryDefault(name, defaultValue string) string {
	value := c.Query(name)
	if value == "" {
		return defaultValue
	}
	return value
}

// JSON sends a JSON response
func (c *Context) JSON(statusCode int, data interface{}) error {
	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.WriteHeader(statusCode)
	return json.NewEncoder(c.Writer).Encode(data)
}

// String sends a plain text response
func (c *Context) String(statusCode int, message string) {
	c.Writer.Header().Set("Content-Type", "text/plain")
	c.Writer.WriteHeader(statusCode)
	c.Writer.Write([]byte(message))
}

// HTML sends an HTML response
func (c *Context) HTML(statusCode int, html string) {
	c.Writer.Header().Set("Content-Type", "text/html")
	c.Writer.WriteHeader(statusCode)
	c.Writer.Write([]byte(html))
}

// Status sets the HTTP status code
func (c *Context) Status(statusCode int) {
	c.Writer.WriteHeader(statusCode)
}

// Header sets a response header
func (c *Context) Header(key, value string) {
	c.Writer.Header().Set(key, value)
}

// GetHeader gets a request header
func (c *Context) GetHeader(key string) string {
	return c.Request.Header.Get(key)
}

// Bind binds request body to a struct (JSON)
func (c *Context) Bind(obj interface{}) error {
	return json.NewDecoder(c.Request.Body).Decode(obj)
}

// Redirect sends a redirect response
func (c *Context) Redirect(statusCode int, url string) {
	http.Redirect(c.Writer, c.Request, url, statusCode)
}

// Method returns the HTTP method
func (c *Context) Method() string {
	return c.Request.Method
}

// Path returns the request path
func (c *Context) Path() string {
	return c.Request.URL.Path
}

// UserAgent returns the user agent
func (c *Context) UserAgent() string {
	return c.Request.UserAgent()
}

// RemoteIP returns the client IP
func (c *Context) RemoteIP() string {
	return c.Request.RemoteAddr
}
