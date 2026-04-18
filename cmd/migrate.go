package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/fatih/color"

	"dotgo/pkg/migration"
)

var (
	preserveOriginal bool
	analyzeOnly      bool
)

// migrateCmd represents the migrate command
var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate existing dotfiles to dotgo structure",
	Long: `Migrate existing dotfiles to the dotgo package-based structure.

This command analyzes your existing dotfiles and converts them into
dotgo packages, making it easy to manage them with the new system.

The migration process:
1. Analyzes existing dotfiles structure
2. Creates package suggestions based on file types
3. Generates package configurations  
4. Preserves your original install scripts for reference
5. Creates a new dotgo configuration

Options:
  --analyze-only      Only analyze and show what would be migrated
  --preserve          Keep original files (copy instead of move)

Examples:
  dotgo migrate --analyze-only    # Show migration analysis only
  dotgo migrate                   # Migrate files to packages  
  dotgo migrate --preserve        # Migrate while keeping originals`,
	RunE: runMigrate,
}

func init() {
	rootCmd.AddCommand(migrateCmd)

	migrateCmd.Flags().BoolVar(&preserveOriginal, "preserve", false, "preserve original files (copy instead of move)")
	migrateCmd.Flags().BoolVar(&analyzeOnly, "analyze-only", false, "only analyze existing structure without migrating")
}

func runMigrate(cmd *cobra.Command, args []string) error {
	// Get current working directory as dotfiles root
	rootDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Check if already initialized as dotgo repository
	if _, err := os.Stat(".dotgo/config.yaml"); err == nil {
		return fmt.Errorf("directory is already initialized as a dotgo repository")
	}

	// Initialize migrator
	migrator := migration.NewMigrator(
		rootDir,
		viper.GetBool("verbose"),
		viper.GetBool("dry-run"),
	)

	// Analyze existing structure
	fmt.Printf("%s Analyzing existing dotfiles structure...\n", color.BlueString("🔍"))
	
	analysis, err := migrator.AnalyzeExisting()
	if err != nil {
		return fmt.Errorf("failed to analyze existing structure: %w", err)
	}

	// Print analysis results
	printAnalysisResults(analysis)

	// If analyze-only flag is set, stop here
	if analyzeOnly {
		fmt.Printf("\n%s Analysis complete. Use 'dotgo migrate' to perform the migration.\n", 
			color.BlueString("ℹ️"))
		return nil
	}

	// Confirm migration if not in dry-run mode
	if !viper.GetBool("dry-run") {
		fmt.Printf("\n%s This will convert your dotfiles to dotgo structure.\n", 
			color.YellowString("⚠️"))
		
		if !preserveOriginal {
			fmt.Printf("%s Original files will be moved to packages (use --preserve to copy instead).\n", 
				color.YellowString("⚠️"))
		}
		
		fmt.Print("Continue? [y/N]: ")
		var response string
		fmt.Scanln(&response)
		
		if response != "y" && response != "Y" && response != "yes" {
			fmt.Println("Migration cancelled.")
			return nil
		}
	}

	// Perform migration
	if err := migrator.Migrate(analysis, preserveOriginal); err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	return nil
}

func printAnalysisResults(analysis *migration.AnalysisResult) {
	fmt.Printf("\n%s Analysis Results:\n\n", color.CyanString("📊"))

	// Summary
	fmt.Printf("Repository: %s\n", analysis.RootDir)
	fmt.Printf("Found:\n")
	fmt.Printf("  • %d dotfiles\n", len(analysis.DotFiles))
	fmt.Printf("  • %d config directories\n", len(analysis.ConfigDirs))
	fmt.Printf("  • %d scripts\n", len(analysis.Scripts))
	
	if analysis.InstallScript != "" {
		fmt.Printf("  • Install script: %s\n", analysis.InstallScript)
	}
	
	if analysis.BrewFile != "" {
		fmt.Printf("  • Brewfile: %s\n", analysis.BrewFile)
	}

	// Suggested packages
	if len(analysis.SuggestedPackages) > 0 {
		fmt.Printf("\n%s Suggested Packages:\n", color.GreenString("📦"))
		
		for _, pkg := range analysis.SuggestedPackages {
			fmt.Printf("\n  %s %s\n", color.CyanString("→"), color.YellowString(pkg.Name))
			fmt.Printf("    Description: %s\n", pkg.Description)
			fmt.Printf("    Files: %d\n", len(pkg.Files))
			
			if viper.GetBool("verbose") {
				for _, file := range pkg.Files {
					fmt.Printf("      %s -> %s\n", file.Source, file.Target)
				}
			}
			
			if len(pkg.Commands) > 0 {
				fmt.Printf("    Commands: %d\n", len(pkg.Commands))
				if viper.GetBool("verbose") {
					for _, cmd := range pkg.Commands {
						fmt.Printf("      %s\n", cmd)
					}
				}
			}
		}
	} else {
		fmt.Printf("\n%s No packages to migrate found.\n", color.YellowString("⚠️"))
	}

	// Files that need backup
	if len(analysis.BackupNeeded) > 0 {
		fmt.Printf("\n%s Files that may need backup:\n", color.YellowString("💾"))
		for _, file := range analysis.BackupNeeded {
			fmt.Printf("  • %s\n", file)
		}
	}

	// Migration preview
	fmt.Printf("\n%s Migration Preview:\n", color.BlueString("🔄"))
	fmt.Printf("This migration will:\n")
	fmt.Printf("  ✓ Create dotgo directory structure\n")
	fmt.Printf("  ✓ Generate %d package(s)\n", len(analysis.SuggestedPackages))
	fmt.Printf("  ✓ Create package configurations\n")
	fmt.Printf("  ✓ Update main dotgo configuration\n")
	
	if analysis.InstallScript != "" {
		fmt.Printf("  ✓ Preserve original install script as legacy-install.sh\n")
	}
}