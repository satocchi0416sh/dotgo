package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/fatih/color"
	"gopkg.in/yaml.v3"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init [directory]",
	Short: "Initialize a new dotgo repository",
	Long: `Initialize a new dotgo repository in the current directory or specified directory.
This will create the necessary directory structure and configuration files for managing dotfiles with dotgo.

The init command will create:
  • .dotgo/ - Main configuration directory
  • .dotgo/config.yaml - Main configuration file
  • packages/ - Directory for package definitions
  • profiles/ - Directory for profile configurations

Examples:
  dotgo init                    # Initialize in current directory
  dotgo init ~/my-dotfiles      # Initialize in specific directory`,
	Args: cobra.MaximumNArgs(1),
	RunE: runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	// Determine target directory
	targetDir := "."
	if len(args) > 0 {
		targetDir = args[0]
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(targetDir)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	// Check if directory exists, create if not
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		if err := os.MkdirAll(absPath, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", absPath, err)
		}
		fmt.Printf("%s Created directory: %s\n", color.GreenString("✓"), absPath)
	}

	// Change to target directory
	if err := os.Chdir(absPath); err != nil {
		return fmt.Errorf("failed to change to directory %s: %w", absPath, err)
	}

	// Check if already initialized
	if _, err := os.Stat(".dotgo"); err == nil {
		return fmt.Errorf("dotgo repository already exists in %s", absPath)
	}

	// Create directory structure
	dirs := []string{
		".dotgo",
		"packages", 
		"profiles",
		"templates",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
		fmt.Printf("%s Created directory: %s\n", color.GreenString("✓"), dir)
	}

	// Create default configuration
	config := DefaultConfig{
		Version: "1.0",
		Repository: RepositoryConfig{
			Type: "local",
		},
		Profiles: map[string]ProfileConfig{
			"default": {
				Name:        "default",
				Description: "Default profile",
				Packages:    []string{},
				Variables:   map[string]interface{}{},
			},
		},
		Settings: SettingsConfig{
			DefaultProfile:  "default",
			BackupDir:      ".dotgo/backups",
			SymlinkMode:    "auto",
			ConflictMode:   "ask",
			PackagesDir:    "packages",
			ProfilesDir:    "profiles",
			TemplatesDir:   "templates",
		},
	}

	// Write configuration file
	configPath := filepath.Join(".dotgo", "config.yaml")
	if err := writeConfig(configPath, config); err != nil {
		return fmt.Errorf("failed to write configuration: %w", err)
	}
	fmt.Printf("%s Created configuration: %s\n", color.GreenString("✓"), configPath)

	// Create example package
	examplePackage := PackageConfig{
		Name:        "example",
		Description: "Example package configuration",
		Dependencies: []string{},
		Files: []FileMapping{
			{
				Source: "example/.example",
				Target: "~/.example",
			},
		},
		Commands: CommandsConfig{
			PreInstall:  []string{},
			PostInstall: []string{"echo 'Example package installed'"},
			PreRemove:   []string{},
			PostRemove:  []string{},
		},
	}

	// Create example package directory and config
	exampleDir := filepath.Join("packages", "example")
	if err := os.MkdirAll(exampleDir, 0755); err != nil {
		return fmt.Errorf("failed to create example package directory: %w", err)
	}

	exampleConfigPath := filepath.Join(exampleDir, "package.yaml")
	if err := writePackageConfig(exampleConfigPath, examplePackage); err != nil {
		return fmt.Errorf("failed to write example package config: %w", err)
	}
	fmt.Printf("%s Created example package: %s\n", color.GreenString("✓"), exampleConfigPath)

	// Create example file
	exampleFilePath := filepath.Join(exampleDir, ".example")
	exampleContent := "# This is an example dotfile managed by dotgo\nexport EXAMPLE_VAR=\"Hello from dotgo!\"\n"
	if err := os.WriteFile(exampleFilePath, []byte(exampleContent), 0644); err != nil {
		return fmt.Errorf("failed to write example file: %w", err)
	}
	fmt.Printf("%s Created example file: %s\n", color.GreenString("✓"), exampleFilePath)

	// Create .gitignore if it doesn't exist
	gitignorePath := ".gitignore"
	if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
		gitignoreContent := `# dotgo files
.dotgo/backups/
.dotgo/cache/

# OS files
.DS_Store
Thumbs.db

# Editor files
*.swp
*.swo
*~
.vscode/
.idea/
`
		if err := os.WriteFile(gitignorePath, []byte(gitignoreContent), 0644); err != nil {
			return fmt.Errorf("failed to write .gitignore: %w", err)
		}
		fmt.Printf("%s Created .gitignore\n", color.GreenString("✓"))
	}

	// Success message
	fmt.Println()
	fmt.Printf("%s dotgo repository initialized successfully in %s\n", color.GreenString("🎉"), absPath)
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("1. Add your dotfiles to packages/")
	fmt.Println("2. Configure packages in packages/*/package.yaml")
	fmt.Println("3. Run 'dotgo install' to apply your dotfiles")
	fmt.Println("4. Use 'dotgo status' to check your setup")

	return nil
}

// Configuration structures
type DefaultConfig struct {
	Version    string                    `yaml:"version"`
	Repository RepositoryConfig          `yaml:"repository"`
	Profiles   map[string]ProfileConfig  `yaml:"profiles"`
	Settings   SettingsConfig            `yaml:"settings"`
}

type RepositoryConfig struct {
	Type string `yaml:"type"`
	URL  string `yaml:"url,omitempty"`
}

type ProfileConfig struct {
	Name        string                 `yaml:"name"`
	Description string                 `yaml:"description"`
	Packages    []string               `yaml:"packages"`
	Variables   map[string]interface{} `yaml:"variables"`
}

type SettingsConfig struct {
	DefaultProfile string `yaml:"default_profile"`
	BackupDir     string `yaml:"backup_dir"`
	SymlinkMode   string `yaml:"symlink_mode"`
	ConflictMode  string `yaml:"conflict_mode"`
	PackagesDir   string `yaml:"packages_dir"`
	ProfilesDir   string `yaml:"profiles_dir"`
	TemplatesDir  string `yaml:"templates_dir"`
}

type PackageConfig struct {
	Name         string          `yaml:"name"`
	Description  string          `yaml:"description"`
	Dependencies []string        `yaml:"dependencies"`
	Files        []FileMapping   `yaml:"files"`
	Commands     CommandsConfig  `yaml:"commands"`
}

type FileMapping struct {
	Source string `yaml:"source"`
	Target string `yaml:"target"`
}

type CommandsConfig struct {
	PreInstall  []string `yaml:"pre_install"`
	PostInstall []string `yaml:"post_install"`
	PreRemove   []string `yaml:"pre_remove"`
	PostRemove  []string `yaml:"post_remove"`
}

func writeConfig(path string, config DefaultConfig) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func writePackageConfig(path string, config PackageConfig) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}