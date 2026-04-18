package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"dotgo/pkg/config"
	"dotgo/pkg/packages"
	"dotgo/pkg/symlink"
	"dotgo/pkg/template"
)

var (
	installProfile  string
	installPackages []string
	forceInstall    bool
	secretsFile     string
)

// installCmd represents the install command
var installCmd = &cobra.Command{
	Use:   "install [packages...]",
	Short: "Install dotfiles packages",
	Long: `Install dotfiles packages by creating symlinks to the target locations.

This command will:
1. Load the specified profile (or default profile)
2. Validate required secrets for template files
3. Install all packages in the profile, or specific packages if provided
4. Process template files using available secrets
5. Create symlinks from package files to their target locations
6. Handle conflicts by backing up existing files
7. Run pre/post-install commands

Examples:
  dotgo install                                    # Install all packages from default profile
  dotgo install --profile dev                     # Install packages from 'dev' profile
  dotgo install zsh git                           # Install only 'zsh' and 'git' packages
  dotgo install --secrets-file ~/.dotgo-secrets   # Use custom secrets file for templates
  dotgo install --force                           # Force reinstall even if already installed
  dotgo install --dry-run                         # Show what would be installed without doing it`,
	RunE: runInstall,
}

func init() {
	rootCmd.AddCommand(installCmd)

	installCmd.Flags().StringVarP(&installProfile, "profile", "p", "", "profile to install (default: use config default)")
	installCmd.Flags().StringSliceVar(&installPackages, "packages", []string{}, "specific packages to install")
	installCmd.Flags().BoolVarP(&forceInstall, "force", "f", false, "force reinstall even if already installed")
	installCmd.Flags().StringVar(&secretsFile, "secrets-file", "", "path to .env file containing secrets for template processing")
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

	// Validate required secrets for template files
	if err := validateRequiredSecrets(packageMgr, resolvedPackages, cfg); err != nil {
		return err
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

// validateRequiredSecrets checks if all required secrets are available for template files
func validateRequiredSecrets(packageMgr *packages.Manager, packageNames []string, cfg *config.Config) error {
	templateEngine := template.NewEngine(&cfg.Settings.TemplateEngine, cfg.Settings.TemplateEngine.GlobalVariables)
	
	// Load secrets from specified file if provided
	if secretsFile != "" {
		if err := templateEngine.LoadEnvWithHierarchy(secretsFile); err != nil {
			return fmt.Errorf("failed to load secrets from %s: %w", secretsFile, err)
		}
	}
	
	var allMissingSecrets []string
	var packagesWithMissingSecrets []string
	
	for _, packageName := range packageNames {
		pkg, err := packageMgr.LoadPackage(packageName)
		if err != nil {
			continue // Skip packages that can't be loaded
		}
		
		var packageMissingSecrets []string
		
		// Check each file for required secrets
		for _, file := range pkg.Files {
			if file.Template && len(file.RequiredSecrets) > 0 {
				missingSecrets := templateEngine.ValidateRequiredSecrets(file.RequiredSecrets)
				if len(missingSecrets) > 0 {
					packageMissingSecrets = append(packageMissingSecrets, missingSecrets...)
					allMissingSecrets = append(allMissingSecrets, missingSecrets...)
				}
			}
		}
		
		if len(packageMissingSecrets) > 0 {
			packagesWithMissingSecrets = append(packagesWithMissingSecrets, packageName)
		}
	}
	
	if len(allMissingSecrets) > 0 {
		// Remove duplicates
		uniqueSecrets := make(map[string]bool)
		var uniqueSecretsList []string
		for _, secret := range allMissingSecrets {
			if !uniqueSecrets[secret] {
				uniqueSecrets[secret] = true
				uniqueSecretsList = append(uniqueSecretsList, secret)
			}
		}
		
		fmt.Printf("%s Missing required secrets for template processing:\n", color.RedString("❌"))
		for _, secret := range uniqueSecretsList {
			fmt.Printf("  • %s\n", secret)
		}
		
		fmt.Printf("\n%s Affected packages: %s\n", color.YellowString("⚠️"), strings.Join(packagesWithMissingSecrets, ", "))
		
		fmt.Println("\nTo fix this:")
		fmt.Println("  • Create ~/.config/dotgo/secrets/.env with the required secrets")
		fmt.Println("  • Or use --secrets-file to specify a custom .env file")
		fmt.Println("  • Example .env format:")
		fmt.Println("    GIT_USER_EMAIL=user@example.com")
		fmt.Println("    GIT_SIGNING_KEY=ABC123")
		
		return fmt.Errorf("missing required secrets: %s", strings.Join(uniqueSecretsList, ", "))
	}
	
	return nil
}
