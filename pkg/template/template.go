package template

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"
	"time"

	"dotgo/pkg/config"
)

// Engine represents a template engine
type Engine struct {
	config     *config.TemplateSettings
	funcMap    template.FuncMap
	variables  map[string]any
}

// NewEngine creates a new template engine
func NewEngine(cfg *config.TemplateSettings, globalVars map[string]any) *Engine {
	engine := &Engine{
		config:    cfg,
		variables: make(map[string]any),
	}
	
	// Copy global variables
	for k, v := range globalVars {
		engine.variables[k] = v
	}
	
	// Setup function map
	engine.funcMap = template.FuncMap{
		"env":      os.Getenv,
		"default":  defaultFunc,
		"required": requiredFunc,
		"now":      time.Now,
		"hostname": getHostname,
		"username": getUsername,
		"homedir":  getHomeDir,
		"os":       func() string { return runtime.GOOS },
		"arch":     func() string { return runtime.GOARCH },
		"join":     strings.Join,
		"split":    strings.Split,
		"upper":    strings.ToUpper,
		"lower":    strings.ToLower,
		"title":    strings.Title,
		"trim":     strings.TrimSpace,
		"contains": strings.Contains,
		"hasPrefix": strings.HasPrefix,
		"hasSuffix": strings.HasSuffix,
		"replace":  strings.ReplaceAll,
		"abs":      filepath.Abs,
		"base":     filepath.Base,
		"dir":      filepath.Dir,
		"ext":      filepath.Ext,
		"clean":    filepath.Clean,
		"join_path": filepath.Join,
	}
	
	return engine
}

// ProcessTemplate processes a template string with variables
func (e *Engine) ProcessTemplate(templateStr string, vars map[string]any) (string, error) {
	if !e.config.Enabled {
		return templateStr, nil
	}
	
	// Merge variables
	allVars := make(map[string]any)
	for k, v := range e.variables {
		allVars[k] = v
	}
	for k, v := range vars {
		allVars[k] = v
	}
	
	// Add environment variables
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			allVars[parts[0]] = parts[1]
		}
	}
	
	// Create template
	tmpl := template.New("dotgo_template")
	
	// Set custom delimiters if configured
	if e.config.TemplateDelims.Left != "" && e.config.TemplateDelims.Right != "" {
		tmpl = tmpl.Delims(e.config.TemplateDelims.Left, e.config.TemplateDelims.Right)
	}
	
	// Add functions
	tmpl = tmpl.Funcs(e.funcMap)
	
	// Parse template
	tmpl, err := tmpl.Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}
	
	// Execute template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, allVars); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}
	
	return buf.String(), nil
}

// ProcessFile processes a template file
func (e *Engine) ProcessFile(templatePath string, vars map[string]any) (string, error) {
	content, err := os.ReadFile(templatePath)
	if err != nil {
		return "", fmt.Errorf("failed to read template file: %w", err)
	}
	
	return e.ProcessTemplate(string(content), vars)
}

// ProcessFileToFile processes a template file and writes to output file
func (e *Engine) ProcessFileToFile(templatePath, outputPath string, vars map[string]any) error {
	result, err := e.ProcessFile(templatePath, vars)
	if err != nil {
		return err
	}
	
	// Create output directory if it doesn't exist
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	
	// Write processed content
	return os.WriteFile(outputPath, []byte(result), 0644)
}

// AddVariable adds a variable to the engine
func (e *Engine) AddVariable(key string, value any) {
	e.variables[key] = value
}

// SetVariables sets multiple variables
func (e *Engine) SetVariables(vars map[string]any) {
	for k, v := range vars {
		e.variables[k] = v
	}
}

// GetVariables returns all variables
func (e *Engine) GetVariables() map[string]any {
	return e.variables
}

// AddFunction adds a custom function
func (e *Engine) AddFunction(name string, fn any) {
	e.funcMap[name] = fn
}

// Template helper functions

// defaultFunc provides a default value if the first argument is empty
func defaultFunc(defaultValue any, value any) any {
	if value == nil {
		return defaultValue
	}
	
	switch v := value.(type) {
	case string:
		if v == "" {
			return defaultValue
		}
	case []string:
		if len(v) == 0 {
			return defaultValue
		}
	case map[string]any:
		if len(v) == 0 {
			return defaultValue
		}
	}
	
	return value
}

// requiredFunc ensures a value is not empty
func requiredFunc(value any) (any, error) {
	if value == nil {
		return nil, fmt.Errorf("required value is nil")
	}
	
	switch v := value.(type) {
	case string:
		if v == "" {
			return nil, fmt.Errorf("required string value is empty")
		}
	case []string:
		if len(v) == 0 {
			return nil, fmt.Errorf("required slice value is empty")
		}
	case map[string]any:
		if len(v) == 0 {
			return nil, fmt.Errorf("required map value is empty")
		}
	}
	
	return value, nil
}

// getHostname returns the system hostname
func getHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}

// getUsername returns the current username
func getUsername() string {
	if user := os.Getenv("USER"); user != "" {
		return user
	}
	if user := os.Getenv("USERNAME"); user != "" {
		return user
	}
	return "unknown"
}

// getHomeDir returns the user's home directory
func getHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return os.Getenv("HOME")
	}
	return home
}

// IsTemplate checks if a string contains template syntax
func IsTemplate(content string) bool {
	return strings.Contains(content, "{{") && strings.Contains(content, "}}")
}

// DetectVariables extracts variable names from a template string
func DetectVariables(templateStr string) ([]string, error) {
	// This is a simplified detection - in practice you might want to parse the template
	// to extract all variable references
	var variables []string
	
	// Find all {{ .variableName }} patterns
	lines := strings.Split(templateStr, "\n")
	for _, line := range lines {
		if strings.Contains(line, "{{") && strings.Contains(line, "}}") {
			// Extract variable names (simplified regex-like approach)
			start := strings.Index(line, "{{")
			end := strings.Index(line, "}}")
			if start != -1 && end != -1 && end > start {
				content := strings.TrimSpace(line[start+2 : end])
				// Extract variable name (simplified)
				if strings.HasPrefix(content, ".") {
					varName := strings.Fields(content)[0][1:]
					variables = append(variables, varName)
				}
			}
		}
	}
	
	// Remove duplicates
	seen := make(map[string]bool)
	var unique []string
	for _, v := range variables {
		if !seen[v] {
			seen[v] = true
			unique = append(unique, v)
		}
	}
	
	return unique, nil
}

// ValidateTemplate checks if a template is syntactically valid
func (e *Engine) ValidateTemplate(templateStr string) error {
	tmpl := template.New("validation")
	
	if e.config.TemplateDelims.Left != "" && e.config.TemplateDelims.Right != "" {
		tmpl = tmpl.Delims(e.config.TemplateDelims.Left, e.config.TemplateDelims.Right)
	}
	
	tmpl = tmpl.Funcs(e.funcMap)
	
	_, err := tmpl.Parse(templateStr)
	return err
}