package view

import (
	"fmt"
	"html/template"
	"io"
	"path/filepath"
	"strings"
	"sync"
)

// Engine represents the view engine
type Engine struct {
	templates map[string]*template.Template
	viewsDir  string
	extension string
	funcMap   template.FuncMap
	mutex     sync.RWMutex
	debug     bool
}

// ViewData represents data passed to views
type ViewData map[string]interface{}

// NewEngine creates a new view engine
func NewEngine(viewsDir string) *Engine {
	return &Engine{
		templates: make(map[string]*template.Template),
		viewsDir:  viewsDir,
		extension: ".html",
		funcMap:   make(template.FuncMap),
		debug:     false,
	}
}

// SetExtension sets the template file extension
func (e *Engine) SetExtension(ext string) {
	e.extension = ext
}

// SetDebug enables/disables debug mode (recompiles templates on each render)
func (e *Engine) SetDebug(debug bool) {
	e.debug = debug
}

// AddFunc adds a template function
func (e *Engine) AddFunc(name string, fn interface{}) {
	e.funcMap[name] = fn
}

// LoadTemplates loads all templates from the views directory
func (e *Engine) LoadTemplates() error {
	pattern := filepath.Join(e.viewsDir, "**/*"+e.extension)
	files, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}

	// Add default functions
	e.addDefaultFunctions()

	for _, file := range files {
		if err := e.loadTemplate(file); err != nil {
			return err
		}
	}

	return nil
}

// loadTemplate loads a single template file
func (e *Engine) loadTemplate(file string) error {
	// Get template name relative to views directory
	relPath, err := filepath.Rel(e.viewsDir, file)
	if err != nil {
		return err
	}

	// Remove extension and normalize path separators
	name := strings.TrimSuffix(relPath, e.extension)
	name = filepath.ToSlash(name)

	tmpl, err := template.New(name).Funcs(e.funcMap).ParseFiles(file)
	if err != nil {
		return err
	}

	e.mutex.Lock()
	e.templates[name] = tmpl
	e.mutex.Unlock()

	return nil
}

// Render renders a template to the given writer
func (e *Engine) Render(w io.Writer, name string, data ViewData) error {
	var tmpl *template.Template
	var exists bool

	if e.debug {
		// In debug mode, reload template
		file := filepath.Join(e.viewsDir, name+e.extension)
		if err := e.loadTemplate(file); err != nil {
			return err
		}
	}

	e.mutex.RLock()
	tmpl, exists = e.templates[name]
	e.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("template '%s' not found", name)
	}

	return tmpl.Execute(w, data)
}

// RenderString renders a template and returns the result as a string
func (e *Engine) RenderString(name string, data ViewData) (string, error) {
	var buf strings.Builder
	err := e.Render(&buf, name, data)
	return buf.String(), err
}

// Exists checks if a template exists
func (e *Engine) Exists(name string) bool {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	_, exists := e.templates[name]
	return exists
}

// addDefaultFunctions adds default template functions
func (e *Engine) addDefaultFunctions() {
	e.funcMap["upper"] = strings.ToUpper
	e.funcMap["lower"] = strings.ToLower
	e.funcMap["title"] = strings.Title
	e.funcMap["trim"] = strings.TrimSpace

	// URL helper
	e.funcMap["url"] = func(path string) string {
		if strings.HasPrefix(path, "/") {
			return path
		}
		return "/" + path
	}

	// Asset helper
	e.funcMap["asset"] = func(path string) string {
		return "/assets/" + strings.TrimPrefix(path, "/")
	}

	// Safe HTML
	e.funcMap["safe"] = func(html string) template.HTML {
		return template.HTML(html)
	}

	// Loop utilities
	e.funcMap["loop"] = func(n int) []int {
		result := make([]int, n)
		for i := range result {
			result[i] = i
		}
		return result
	}

	// Default value
	e.funcMap["default"] = func(defaultVal, val interface{}) interface{} {
		if val == nil || val == "" {
			return defaultVal
		}
		return val
	}
}

// ParseString parses a template string and returns a template
func (e *Engine) ParseString(name, content string) (*template.Template, error) {
	return template.New(name).Funcs(e.funcMap).Parse(content)
}

// RenderStringTemplate renders a template string with data
func (e *Engine) RenderStringTemplate(content string, data ViewData) (string, error) {
	tmpl, err := e.ParseString("string_template", content)
	if err != nil {
		return "", err
	}

	var buf strings.Builder
	err = tmpl.Execute(&buf, data)
	return buf.String(), err
}
