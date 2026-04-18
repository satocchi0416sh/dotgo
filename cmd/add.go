package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"dotgo/pkg/add"
	"dotgo/pkg/config"
)

var (
	// packageName is the optional package name to use instead of auto-detection
	packageName string
	// targetPath is the optional target path override
	targetPath string
)

func init() {
	addCmd.Flags().StringVarP(&packageName, "package", "p", "", "Package name (auto-detected if not specified)")
	addCmd.Flags().StringVarP(&targetPath, "target", "t", "", "Target path override (uses source path relative to home if not specified)")
	rootCmd.AddCommand(addCmd)
}

// addCmd represents the add command
var addCmd = &cobra.Command{
	Use:   "add <file-path>",
	Short: "Add a file to dotfiles management",
	Long: `Add a file to dotfiles management by moving it to the packages directory
and creating appropriate symlinks and configuration.

This command will:
1. Analyze the source file path and infer package name
2. Move the file to packages/<package>/files/ directory
3. Create or update package configuration (dotgo.yaml)
4. Create symlinks from original location to package file
5. Handle conflicts by backing up existing files

Examples:
  dotgo add ~/.config/starship.toml           # Auto-detect package as 'starship'
  dotgo add ~/.vimrc --package vim            # Explicitly set package name
  dotgo add ~/.config/nvim/init.lua --target ~/.config/nvim/init.lua  # Override target path
  dotgo add ~/.zshrc --dry-run               # Show what would be done without executing`,
	Args: cobra.ExactArgs(1),
	RunE: runAdd,
}

// runAdd implements the add command logic
func runAdd(cmd *cobra.Command, args []string) error {
	sourcePath := args[0]

	// Get current working directory as dotfiles root
	rootDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Initialize configuration manager
	configMgr := config.NewManager(rootDir)
	if err := configMgr.Load(); err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Initialize path processor
	pathProcessor := add.NewPathProcessor()

	// Process and validate the source path
	processedPath, err := pathProcessor.ProcessPath(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to process source path: %w", err)
	}

	// Infer package name if not provided
	inferredPackage := packageName
	if inferredPackage == "" {
		inferredPackage = pathProcessor.InferPackageName(processedPath.Source)
		if viper.GetBool("verbose") {
			fmt.Printf("%s Inferred package name: %s\n",
				color.BlueString("🔍"), inferredPackage)
		}
	}

	// Check if this is a template file
	isTemplate := strings.HasSuffix(processedPath.Source, ".tmpl")
	if isTemplate && viper.GetBool("verbose") {
		fmt.Printf("%s Template file detected: %s\n",
			color.CyanString("📝"), filepath.Base(processedPath.Source))
		fmt.Println("  Template files will be processed during installation")
		fmt.Println("  Consider creating a .env file for any secrets needed")
	}

	// Determine target path
	finalTargetPath := targetPath
	if finalTargetPath == "" {
		finalTargetPath = processedPath.Source
	}

	// Initialize file manager
	fileManager := add.NewFileManager(
		configMgr,
		rootDir,
		viper.GetBool("verbose"),
		viper.GetBool("dry-run"),
	)

	// Create add operation
	operation := &add.AddOperation{
		SourcePath:  processedPath.Source,
		TargetPath:  finalTargetPath,
		PackageName: inferredPackage,
	}

	if viper.GetBool("dry-run") {
		fmt.Printf("%s Dry run mode - showing planned operations:\n",
			color.YellowString("🔍"))
	}

	fmt.Printf("%s Adding file to dotfiles management...\n",
		color.BlueString("📦"))
	fmt.Printf("  Source: %s\n", operation.SourcePath)
	fmt.Printf("  Package: %s\n", operation.PackageName)
	fmt.Printf("  Target: %s\n", operation.TargetPath)

	// Execute the add operation
	result, err := fileManager.AddFile(operation)
	if err != nil {
		// Attempt rollback on error
		if rollbackErr := fileManager.Rollback(); rollbackErr != nil {
			fmt.Printf("%s Rollback failed: %v\n",
				color.RedString("⚠️"), rollbackErr)
		}
		return fmt.Errorf("failed to add file: %w", err)
	}

	// Print success message
	if !viper.GetBool("dry-run") {
		fmt.Printf("\n%s File successfully added to dotfiles!\n",
			color.GreenString("✓"))
		fmt.Printf("  Package file: %s\n", result.PackageFilePath)
		fmt.Printf("  Config updated: %s\n", result.ConfigPath)

		if result.SymlinkCreated {
			fmt.Printf("  Symlink created: %s → %s\n",
				result.TargetPath, result.PackageFilePath)
		}

		// Provide next steps
		fmt.Println("\nNext steps:")
		if isTemplate {
			fmt.Println("  • The template file has been moved to packages directory")
			fmt.Println("  • Template will be processed during installation")
			fmt.Println("  • Create .env files in package directory for any required secrets")
			fmt.Println("  • Update package config with 'required_secrets' if needed")
		} else {
			fmt.Println("  • The original file has been moved to packages directory")
			fmt.Println("  • A symlink has been created at the original location")
		}
		fmt.Printf("  • Run 'dotgo install %s' to install this package on other systems\n",
			operation.PackageName)
	}

	return nil
}
