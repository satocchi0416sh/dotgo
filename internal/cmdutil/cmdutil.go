package cmdutil

import (
	"fmt"
	"os"
	"strings"

	"github.com/satocchi0416sh/dotgo/internal/engine"
)

// InitializeEngine creates and returns an initialized engine instance
func InitializeEngine(dryRun, verbose bool) (*engine.Engine, error) {
	rootDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}
	return engine.NewEngine(rootDir, dryRun, verbose), nil
}

// ProcessTags trims whitespace from all tags
func ProcessTags(rawTags []string) []string {
	if len(rawTags) == 0 {
		return nil
	}

	tags := make([]string, len(rawTags))
	for i, tag := range rawTags {
		tags[i] = strings.TrimSpace(tag)
	}
	return tags
}
