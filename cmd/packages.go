package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/fatih/color"

	"dotgo/pkg/config"
	"dotgo/pkg/packages"
	"dotgo/pkg/symlink"
)

// packagesCmd represents the packages command
var packagesCmd = &cobra.Command{
	Use:   "packages",
	Short: "Manage dotfiles packages",
	Long: `Manage dotfiles packages including listing, installing, removing, and getting status information.

Subcommands:
  list        List all available packages
  status      Show installation status of packages
  install     Install specific packages
  remove      Remove specific packages
  info        Show detailed information about a package`,
}

// packagesListCmd lists all available packages
var packagesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available packages",
	Long: `List all available packages in the repository.

Shows package name, installation status, and description.`,
	RunE: runPackagesList,
}

// packagesStatusCmd shows package installation status
var packagesStatusCmd = &cobra.Command{
	Use:   "status [package...]",
	Short: "Show installation status of packages",
	Long: `Show detailed installation status of packages including:
- Whether the package is installed
- Status of individual files/symlinks
- Any broken or missing links

Examples:
  dotgo packages status           # Show status of all packages
  dotgo packages status zsh git   # Show status of specific packages`,
	RunE: runPackagesStatus,
}

// packagesInstallCmd installs specific packages
var packagesInstallCmd = &cobra.Command{
	Use:   "install <package...>",
	Short: "Install specific packages",
	Long: `Install one or more specific packages.

This is equivalent to 'dotgo install <package...>' but provides a more explicit interface.

Examples:
  dotgo packages install zsh      # Install zsh package
  dotgo packages install zsh git  # Install multiple packages`,
	Args: cobra.MinimumNArgs(1),
	RunE: runPackagesInstall,
}

// packagesRemoveCmd removes specific packages
var packagesRemoveCmd = &cobra.Command{
	Use:   "remove <package...>",
	Short: "Remove specific packages",
	Long: `Remove one or more packages by removing their symlinks.

Options:
  --restore-backup    Restore backed up files when removing symlinks

Examples:
  dotgo packages remove zsh               # Remove zsh package
  dotgo packages remove zsh --restore-backup  # Remove and restore backups`,
	Args: cobra.MinimumNArgs(1),
	RunE: runPackagesRemove,
}

// packagesInfoCmd shows detailed package information
var packagesInfoCmd = &cobra.Command{
	Use:   "info <package>",
	Short: "Show detailed information about a package",
	Long: `Show detailed information about a package including:
- Package metadata (name, description, version)
- Dependencies
- Files and their target locations
- Installation status
- Commands that will be executed

Examples:
  dotgo packages info zsh        # Show info for zsh package`,
	Args: cobra.ExactArgs(1),
	RunE: runPackagesInfo,
}

var (
	restoreBackup bool
)

func init() {
	rootCmd.AddCommand(packagesCmd)
	
	packagesCmd.AddCommand(packagesListCmd)
	packagesCmd.AddCommand(packagesStatusCmd)
	packagesCmd.AddCommand(packagesInstallCmd)
	packagesCmd.AddCommand(packagesRemoveCmd)
	packagesCmd.AddCommand(packagesInfoCmd)

	// Flags for remove command
	packagesRemoveCmd.Flags().BoolVar(&restoreBackup, "restore-backup", false, "restore backed up files")
}

func runPackagesList(cmd *cobra.Command, args []string) error {
	// Get current working directory as dotfiles root
	rootDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Initialize managers
	_, packageMgr, err := initializeManagers(rootDir)
	if err != nil {
		return err
	}

	// Get all packages
	packages, err := packageMgr.List()
	if err != nil {
		return fmt.Errorf("failed to list packages: %w", err)
	}

	if len(packages) == 0 {
		fmt.Printf("%s No packages found\n", color.YellowString("ℹ️"))
		return nil
	}

	// Print packages
	fmt.Printf("%s Available packages:\n\n", color.BlueString("📦"))
	
	for _, pkg := range packages {
		status := color.RedString("✗ Not installed")
		if pkg.Installed {
			status = color.GreenString("✓ Installed")
		}

		fmt.Printf("  %s %s\n", status, color.CyanString(pkg.Name))
		if pkg.Description != "" {
			fmt.Printf("    %s\n", pkg.Description)
		}
		if pkg.Version != "" {
			fmt.Printf("    Version: %s\n", pkg.Version)
		}
		if len(pkg.Dependencies) > 0 {
			fmt.Printf("    Dependencies: %s\n", strings.Join(pkg.Dependencies, ", "))
		}
		fmt.Println()
	}

	return nil
}

func runPackagesStatus(cmd *cobra.Command, args []string) error {
	// Get current working directory as dotfiles root
	rootDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Initialize managers
	_, packageMgr, err := initializeManagers(rootDir)
	if err != nil {
		return err
	}

	// Get packages to check
	var packagesToCheck []string
	if len(args) > 0 {
		packagesToCheck = args
	} else {
		// Get all packages
		allPackages, err := packageMgr.List()
		if err != nil {
			return fmt.Errorf("failed to list packages: %w", err)
		}
		for _, pkg := range allPackages {
			packagesToCheck = append(packagesToCheck, pkg.Name)
		}
	}

	// Check status of each package
	fmt.Printf("%s Package Status:\n\n", color.BlueString("📊"))

	for _, packageName := range packagesToCheck {
		pkg, err := packageMgr.LoadPackage(packageName)
		if err != nil {
			fmt.Printf("%s %s: %v\n", color.RedString("✗"), packageName, err)
			continue
		}

		status := color.RedString("Not installed")
		if pkg.Installed {
			status = color.GreenString("Installed")
		}

		fmt.Printf("  %s %s\n", status, color.CyanString(pkg.Name))
		
		// Show file status if verbose or if there are issues
		if viper.GetBool("verbose") || !pkg.Installed {
			for _, file := range pkg.Files {
				if file.LinkInfo == nil {
					continue
				}

				fileStatus := ""
				if !file.LinkInfo.Exists {
					fileStatus = color.RedString("Missing")
				} else if file.LinkInfo.IsSymlink {
					if file.LinkInfo.IsValid {
						fileStatus = color.GreenString("OK")
					} else {
						fileStatus = color.RedString("Broken")
					}
				} else {
					fileStatus = color.YellowString("Not a symlink")
				}

				fmt.Printf("    %s %s -> %s\n", fileStatus, file.Target, file.Source)
			}
		}
		fmt.Println()
	}

	return nil
}

func runPackagesInstall(cmd *cobra.Command, args []string) error {
	// Get current working directory as dotfiles root
	rootDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Initialize managers
	_, packageMgr, err := initializeManagers(rootDir)
	if err != nil {
		return err
	}

	// Install each package
	var errors []string
	var installedCount int

	for _, packageName := range args {
		fmt.Printf("%s Installing package: %s\n", 
			color.CyanString("→"), packageName)

		result, err := packageMgr.Install(packageName)
		if err != nil {
			errorMsg := fmt.Sprintf("Failed to install '%s': %v", packageName, err)
			errors = append(errors, errorMsg)
			fmt.Printf("%s %s\n", color.RedString("✗"), errorMsg)
			continue
		}

		if result.Success {
			installedCount++
			fmt.Printf("%s Package '%s' installed (%d files linked)\n", 
				color.GreenString("✓"), packageName, result.FilesLinked)
		}
	}

	// Print summary
	fmt.Printf("\n%s Installation summary:\n", color.BlueString("📋"))
	fmt.Printf("  Packages installed: %d/%d\n", installedCount, len(args))

	if len(errors) > 0 {
		fmt.Printf("  Errors: %d\n", len(errors))
		for _, err := range errors {
			fmt.Printf("    %s %s\n", color.RedString("✗"), err)
		}
		return fmt.Errorf("installation completed with %d errors", len(errors))
	}

	return nil
}

func runPackagesRemove(cmd *cobra.Command, args []string) error {
	// Get current working directory as dotfiles root
	rootDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Initialize managers
	_, packageMgr, err := initializeManagers(rootDir)
	if err != nil {
		return err
	}

	// Remove each package
	var errors []string
	var removedCount int

	for _, packageName := range args {
		fmt.Printf("%s Removing package: %s\n", 
			color.YellowString("→"), packageName)

		err := packageMgr.Remove(packageName, restoreBackup)
		if err != nil {
			errorMsg := fmt.Sprintf("Failed to remove '%s': %v", packageName, err)
			errors = append(errors, errorMsg)
			fmt.Printf("%s %s\n", color.RedString("✗"), errorMsg)
			continue
		}

		removedCount++
		fmt.Printf("%s Package '%s' removed\n", 
			color.GreenString("✓"), packageName)
	}

	// Print summary
	fmt.Printf("\n%s Removal summary:\n", color.BlueString("📋"))
	fmt.Printf("  Packages removed: %d/%d\n", removedCount, len(args))

	if len(errors) > 0 {
		fmt.Printf("  Errors: %d\n", len(errors))
		for _, err := range errors {
			fmt.Printf("    %s %s\n", color.RedString("✗"), err)
		}
		return fmt.Errorf("removal completed with %d errors", len(errors))
	}

	return nil
}

func runPackagesInfo(cmd *cobra.Command, args []string) error {
	packageName := args[0]

	// Get current working directory as dotfiles root
	rootDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Initialize managers
	_, packageMgr, err := initializeManagers(rootDir)
	if err != nil {
		return err
	}

	// Load package
	pkg, err := packageMgr.LoadPackage(packageName)
	if err != nil {
		return fmt.Errorf("failed to load package '%s': %w", packageName, err)
	}

	// Print package information
	fmt.Printf("%s Package Information: %s\n\n", color.BlueString("📋"), color.CyanString(pkg.Name))

	fmt.Printf("Name: %s\n", pkg.Name)
	if pkg.Description != "" {
		fmt.Printf("Description: %s\n", pkg.Description)
	}
	if pkg.Version != "" {
		fmt.Printf("Version: %s\n", pkg.Version)
	}
	fmt.Printf("Path: %s\n", pkg.Path)

	status := color.RedString("Not installed")
	if pkg.Installed {
		status = color.GreenString("Installed")
	}
	fmt.Printf("Status: %s\n", status)

	// Dependencies
	if len(pkg.Dependencies) > 0 {
		fmt.Printf("\nDependencies:\n")
		for _, dep := range pkg.Dependencies {
			fmt.Printf("  - %s\n", dep)
		}
	}

	// Files
	if len(pkg.Files) > 0 {
		fmt.Printf("\nFiles:\n")
		for _, file := range pkg.Files {
			fileStatus := ""
			if file.LinkInfo != nil {
				if !file.LinkInfo.Exists {
					fileStatus = color.RedString(" (missing)")
				} else if file.LinkInfo.IsSymlink {
					if file.LinkInfo.IsValid {
						fileStatus = color.GreenString(" (linked)")
					} else {
						fileStatus = color.RedString(" (broken)")
					}
				} else {
					fileStatus = color.YellowString(" (not linked)")
				}
			}

			fmt.Printf("  %s -> %s%s\n", file.Source, file.Target, fileStatus)
			
			if file.Executable {
				fmt.Printf("    (executable)\n")
			}
			if file.Template {
				fmt.Printf("    (template)\n")
			}
			if file.Condition != "" {
				fmt.Printf("    (condition: %s)\n", file.Condition)
			}
		}
	}

	// Commands
	if pkg.Config != nil {
		commands := pkg.Config.Commands
		if len(commands.PreInstall) > 0 || len(commands.PostInstall) > 0 || 
		   len(commands.PreRemove) > 0 || len(commands.PostRemove) > 0 {
			fmt.Printf("\nCommands:\n")
			
			if len(commands.PreInstall) > 0 {
				fmt.Printf("  Pre-install:\n")
				for _, cmd := range commands.PreInstall {
					fmt.Printf("    %s\n", cmd)
				}
			}
			
			if len(commands.PostInstall) > 0 {
				fmt.Printf("  Post-install:\n")
				for _, cmd := range commands.PostInstall {
					fmt.Printf("    %s\n", cmd)
				}
			}
			
			if len(commands.PreRemove) > 0 {
				fmt.Printf("  Pre-remove:\n")
				for _, cmd := range commands.PreRemove {
					fmt.Printf("    %s\n", cmd)
				}
			}
			
			if len(commands.PostRemove) > 0 {
				fmt.Printf("  Post-remove:\n")
				for _, cmd := range commands.PostRemove {
					fmt.Printf("    %s\n", cmd)
				}
			}
		}
	}

	return nil
}

// initializeManagers creates and initializes the configuration and package managers
func initializeManagers(rootDir string) (*config.Manager, *packages.Manager, error) {
	// Initialize configuration manager
	configMgr := config.NewManager(rootDir)
	if err := configMgr.Load(); err != nil {
		return nil, nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Initialize symlink manager
	symlinkMgr := symlink.NewManager(
		configMgr.GetBackupDir(),
		viper.GetBool("dry-run"),
		viper.GetBool("verbose"),
	)

	// Initialize package manager
	packageMgr := packages.NewManager(
		configMgr,
		symlinkMgr,
		rootDir,
		viper.GetBool("verbose"),
		viper.GetBool("dry-run"),
	)

	return configMgr, packageMgr, nil
}