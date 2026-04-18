package cmd

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"dotgo/pkg/config"
	"dotgo/pkg/packages"
	"dotgo/pkg/symlink"
)

var (
	installProfile  string
	installPackages []string
	forceInstall    bool
)

// installCmd represents the install command
var installCmd = &cobra.Command{
	Use:   "install [packages...]",
	Short: "Install dotfiles packages",
	Long: `Install dotfiles packages by creating symlinks to the target locations.

This command will:
1. Load the specified profile (or default profile)
2. Install all packages in the profile, or specific packages if provided
3. Create symlinks from package files to their target locations
4. Handle conflicts by backing up existing files
5. Run pre/post-install commands

Examples:
  dotgo install                     # Install all packages from default profile
  dotgo install --profile dev       # Install packages from 'dev' profile
  dotgo install zsh git             # Install only 'zsh' and 'git' packages
  dotgo install --force             # Force reinstall even if already installed
  dotgo install --dry-run           # Show what would be installed without doing it`,
	RunE: runInstall,
}

func init() {
	rootCmd.AddCommand(installCmd)

	installCmd.Flags().StringVarP(&installProfile, "profile", "p", "", "profile to install (default: use config default)")
	installCmd.Flags().StringSliceVar(&installPackages, "packages", []string{}, "specific packages to install")
	installCmd.Flags().BoolVarP(&forceInstall, "force", "f", false, "force reinstall even if already installed")
}

func runInstall(cmd *cobra.Command, args []string) error {
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

	// Get configuration
	cfg := configMgr.GetConfig()

	// Determine profile to use
	profileName := installProfile
	if profileName == "" {
		profileName = cfg.Settings.DefaultProfile
	}

	// Get profile
	profile, err := configMgr.GetProfile(profileName)
	if err != nil {
		return fmt.Errorf("failed to get profile '%s': %w", profileName, err)
	}

	// Determine packages to install
	var packagesToInstall []string
	if len(args) > 0 {
		packagesToInstall = args
	} else if len(installPackages) > 0 {
		packagesToInstall = installPackages
	} else {
		packagesToInstall = profile.Packages
	}

	if len(packagesToInstall) == 0 {
		fmt.Printf("%s No packages to install in profile '%s'\n",
			color.YellowString("⚠️"), profileName)
		return nil
	}

	// Initialize managers
	symlinkMgr := symlink.NewManager(
		configMgr.GetBackupDir(),
		viper.GetBool("dry-run"),
		viper.GetBool("verbose"),
	)

	packageMgr := packages.NewManager(
		configMgr,
		symlinkMgr,
		rootDir,
		viper.GetBool("verbose"),
		viper.GetBool("dry-run"),
	)

	// Resolve dependencies
	fmt.Printf("%s Resolving dependencies...\n", color.BlueString("🔍"))
	resolvedPackages, err := packageMgr.ResolveDependencies(packagesToInstall)
	if err != nil {
		return fmt.Errorf("failed to resolve dependencies: %w", err)
	}

	if viper.GetBool("verbose") {
		fmt.Printf("Install order: %s\n", resolvedPackages)
	}

	// Install packages
	fmt.Printf("%s Installing packages from profile '%s'...\n",
		color.BlueString("📦"), profileName)

	var totalFilesLinked int
	var installedPackages []string
	var errors []string

	for _, packageName := range resolvedPackages {
		fmt.Printf("\n%s Installing package: %s\n",
			color.CyanString("→"), packageName)

		// Check if package exists
		_, err := packageMgr.LoadPackage(packageName)
		if err != nil {
			errorMsg := fmt.Sprintf("Package '%s': %v", packageName, err)
			errors = append(errors, errorMsg)
			fmt.Printf("%s %s\n", color.RedString("✗"), errorMsg)
			continue
		}

		// Install package
		result, err := packageMgr.Install(packageName)
		if err != nil {
			errorMsg := fmt.Sprintf("Package '%s': %v", packageName, err)
			errors = append(errors, errorMsg)
			fmt.Printf("%s %s\n", color.RedString("✗"), errorMsg)
			continue
		}

		if result.Success {
			installedPackages = append(installedPackages, packageName)
			totalFilesLinked += result.FilesLinked

			if result.FilesLinked > 0 {
				fmt.Printf("%s Package '%s' installed (%d files linked)\n",
					color.GreenString("✓"), packageName, result.FilesLinked)
			} else {
				fmt.Printf("%s Package '%s' already installed\n",
					color.YellowString("⏭️"), packageName)
			}
		}
	}

	// Print summary
	fmt.Println()
	if len(installedPackages) > 0 {
		fmt.Printf("%s Installation completed!\n", color.GreenString("🎉"))
		fmt.Printf("  Packages installed: %d\n", len(installedPackages))
		fmt.Printf("  Files linked: %d\n", totalFilesLinked)

		if viper.GetBool("verbose") {
			fmt.Printf("  Installed packages: %s\n", installedPackages)
		}
	}

	if len(errors) > 0 {
		fmt.Printf("\n%s Installation completed with errors:\n", color.YellowString("⚠️"))
		for _, err := range errors {
			fmt.Printf("  %s %s\n", color.RedString("✗"), err)
		}
		return fmt.Errorf("installation completed with %d errors", len(errors))
	}

	// Provide next steps
	if len(installedPackages) > 0 {
		fmt.Println()
		fmt.Println("Next steps:")
		fmt.Println("  • Restart your shell or source your dotfiles")
		fmt.Println("  • Run 'dotgo status' to verify installation")
		fmt.Println("  • Use 'dotgo packages list' to see all available packages")
	}

	return nil
}
