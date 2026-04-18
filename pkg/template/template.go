package template

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"text/template"
	"time"

	"dotgo/pkg/config"
)

// Engine represents a template engine
type Engine struct {
	config    *config.TemplateSettings
	funcMap   template.FuncMap
	variables map[string]any
	envFiles  map[string]map[string]string // cached environment files
}

// NewEngine creates a new template engine
func NewEngine(cfg *config.TemplateSettings, globalVars map[string]any) *Engine {
	engine := &Engine{
		config:    cfg,
		variables: make(map[string]any),
		envFiles:  make(map[string]map[string]string),
	}

	// Copy global variables
	for k, v := range globalVars {
		engine.variables[k] = v
	}

	// Setup function map
	engine.funcMap = template.FuncMap{
		"env":       os.Getenv,
		"default":   defaultFunc,
		"required":  requiredFunc,
		"now":       time.Now,
		"hostname":  getHostname,
		"username":  getUsername,
		"homedir":   getHomeDir,
		"os":        func() string { return runtime.GOOS },
		"arch":      func() string { return runtime.GOARCH },
		"join":      strings.Join,
		"split":     strings.Split,
		"upper":     strings.ToUpper,
		"lower":     strings.ToLower,
		"title":     strings.Title,
		"trim":      strings.TrimSpace,
		"contains":  strings.Contains,
		"hasPrefix": strings.HasPrefix,
		"hasSuffix": strings.HasSuffix,
		"replace":   strings.ReplaceAll,
		"abs":       filepath.Abs,
		"base":      filepath.Base,
		"dir":       filepath.Dir,
		"ext":       filepath.Ext,
		"clean":     filepath.Clean,
		"join_path": filepath.Join,
		// New template functions for secure handling
		"envFile":    engine.envFileFunc,
		"secret":     engine.secretFunc,
		"hasSecret":  engine.hasSecretFunc,
		"gitInclude": engine.gitIncludeFunc,
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

// New template functions for secure handling

// envFileFunc loads environment variables from a file with hierarchy support
func (e *Engine) envFileFunc(filename string) map[string]string {
	// Check cache first
	if envVars, exists := e.envFiles[filename]; exists {
		return envVars
	}

	envVars := e.loadEnvFile(filename)
	e.envFiles[filename] = envVars
	return envVars
}

// secretFunc gets a secret with optional default value
func (e *Engine) secretFunc(key string, defaultValue ...string) string {
	// First check environment variables
	if value := os.Getenv(key); value != "" {
		return value
	}

	// Check secrets directory (~/.config/dotgo/secrets/)
	homeDir, err := os.UserHomeDir()
	if err == nil {
		secretsDir := filepath.Join(homeDir, ".config", "dotgo", "secrets")

		// Try loading from .env files in hierarchy
		envFiles := []string{
			filepath.Join(secretsDir, ".env.local"),
			filepath.Join(secretsDir, ".env"),
		}

		for _, envFile := range envFiles {
			if envVars := e.loadEnvFile(envFile); envVars[key] != "" {
				return envVars[key]
			}
		}
	}

	// Return default value if provided
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}

	return ""
}

// hasSecretFunc checks if a secret exists
func (e *Engine) hasSecretFunc(key string) bool {
	return e.secretFunc(key) != ""
}

// gitIncludeFunc generates a Git include directive
func (e *Engine) gitIncludeFunc(path string) string {
	// Expand path if it contains ~
	expandedPath := path
	if strings.HasPrefix(path, "~/") {
		if homeDir, err := os.UserHomeDir(); err == nil {
			expandedPath = filepath.Join(homeDir, path[2:])
		}
	}

	return fmt.Sprintf("[include]\n\tpath = %s", expandedPath)
}

// loadEnvFile loads environment variables from a .env file
func (e *Engine) loadEnvFile(filename string) map[string]string {
	envVars := make(map[string]string)

	file, err := os.Open(filename)
	if err != nil {
		return envVars // Return empty map if file doesn't exist
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse KEY=VALUE format
		if idx := strings.Index(line, "="); idx > 0 {
			key := strings.TrimSpace(line[:idx])
			value := strings.TrimSpace(line[idx+1:])

			// Remove quotes if present
			if len(value) >= 2 {
				if (value[0] == '"' && value[len(value)-1] == '"') ||
					(value[0] == '\'' && value[len(value)-1] == '\'') {
					value = value[1 : len(value)-1]
				}
			}

			envVars[key] = value
		}
	}

	return envVars
}

// LoadEnvWithHierarchy loads environment variables with hierarchy support
func (e *Engine) LoadEnvWithHierarchy(basePath string) error {
	// Load env files in order of precedence: .env.local > .env > system env
	envFiles := []string{
		filepath.Join(basePath, ".env"),
		filepath.Join(basePath, ".env.local"),
	}

	for _, envFile := range envFiles {
		if envVars := e.loadEnvFile(envFile); len(envVars) > 0 {
			// Add to variables (higher precedence files override lower ones)
			for key, value := range envVars {
				e.variables[key] = value
			}
		}
	}

	return nil
}

// ValidateRequiredSecrets checks if all required secrets are available
func (e *Engine) ValidateRequiredSecrets(requiredSecrets []string) []string {
	var missing []string

	for _, secret := range requiredSecrets {
		if !e.hasSecretFunc(secret) {
			missing = append(missing, secret)
		}
	}

	sort.Strings(missing) // Sort for consistent output
	return missing

}
