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

// statusCmd represents the status command
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current dotfiles status",
	Long: `Show the current status of your dotfiles installation.

This command displays:
• Repository information
• Profile configuration
• Package installation status  
• Broken or missing symlinks
• Overall health of the dotfiles setup

The status command helps you quickly identify any issues with your
dotfiles configuration and provides recommendations for fixes.`,
	RunE: runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
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

	// Print header
	fmt.Printf("%s dotgo Status Report\n", color.BlueString("📊"))
	fmt.Printf("Repository: %s\n\n", rootDir)

	// Repository information
	fmt.Printf("%s Repository Information:\n", color.CyanString("📁"))
	fmt.Printf("  Type: %s\n", cfg.Repository.Type)
	if cfg.Repository.URL != "" {
		fmt.Printf("  URL: %s\n", cfg.Repository.URL)
	}
	fmt.Printf("  Version: %s\n", cfg.Version)
	fmt.Println()

	// Current profile
	profileName := cfg.Settings.DefaultProfile
	profile, err := configMgr.GetProfile(profileName)
	if err != nil {
		return fmt.Errorf("failed to get profile '%s': %w", profileName, err)
	}

	fmt.Printf("%s Current Profile: %s\n", color.CyanString("👤"), color.YellowString(profileName))
	if profile.Description != "" {
		fmt.Printf("  Description: %s\n", profile.Description)
	}
	if len(profile.Packages) > 0 {
		fmt.Printf("  Packages: %s\n", strings.Join(profile.Packages, ", "))
	} else {
		fmt.Printf("  Packages: %s\n", color.YellowString("none"))
	}
	if len(profile.Variables) > 0 {
		fmt.Printf("  Variables: %d defined\n", len(profile.Variables))
	}
	fmt.Println()

	// Initialize managers for package status
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

	// Get all packages
	allPackages, err := packageMgr.List()
	if err != nil {
		return fmt.Errorf("failed to list packages: %w", err)
	}

	// Count installed packages
	var installedCount int
	var totalFiles int
	var installedFiles int
	var brokenLinks []string
	var missingFiles []string

	for _, pkg := range allPackages {
		if pkg.Installed {
			installedCount++
		}

		// Check each file
		for _, file := range pkg.Files {
			totalFiles++

			if file.LinkInfo == nil {
				continue
			}

			if file.LinkInfo.Exists {
				if file.LinkInfo.IsSymlink {
					if file.LinkInfo.IsValid {
						installedFiles++
					} else {
						brokenLinks = append(brokenLinks, file.Target)
					}
				} else {
					// File exists but is not a symlink
					missingFiles = append(missingFiles, fmt.Sprintf("%s (not a symlink)", file.Target))
				}
			} else {
				missingFiles = append(missingFiles, file.Target)
			}
		}
	}

	// Package status summary
	fmt.Printf("%s Package Summary:\n", color.CyanString("📦"))
	fmt.Printf("  Available: %d\n", len(allPackages))
	fmt.Printf("  Installed: %s/%d\n", getStatusColor(installedCount, len(allPackages), installedCount), len(allPackages))
	fmt.Printf("  Files: %s/%d linked\n", getStatusColor(installedFiles, totalFiles, installedFiles), totalFiles)
	fmt.Println()

	// Health status
	var healthIssues []string
	
	if len(brokenLinks) > 0 {
		healthIssues = append(healthIssues, fmt.Sprintf("%d broken symlinks", len(brokenLinks)))
	}
	
	if len(missingFiles) > 0 {
		healthIssues = append(healthIssues, fmt.Sprintf("%d missing files", len(missingFiles)))
	}

	fmt.Printf("%s Health Status: ", color.CyanString("🏥"))
	if len(healthIssues) == 0 {
		fmt.Printf("%s\n", color.GreenString("Healthy"))
	} else {
		fmt.Printf("%s (%s)\n", color.RedString("Issues found"), strings.Join(healthIssues, ", "))
	}
	fmt.Println()

	// Show detailed package status
	if viper.GetBool("verbose") || len(healthIssues) > 0 {
		fmt.Printf("%s Detailed Package Status:\n", color.CyanString("📋"))
		
		for _, pkg := range allPackages {
			status := color.RedString("✗")
			statusText := "Not installed"
			
			if pkg.Installed {
				status = color.GreenString("✓")
				statusText = "Installed"
			}

			fmt.Printf("  %s %s - %s\n", status, color.CyanString(pkg.Name), statusText)
			
			// Show file details if verbose or if there are issues
			if viper.GetBool("verbose") || !pkg.Installed {
				for _, file := range pkg.Files {
					if file.LinkInfo == nil {
						continue
					}

					fileStatus := getFileStatusIcon(file.LinkInfo)
					fmt.Printf("    %s %s\n", fileStatus, file.Target)
				}
			}
		}
		fmt.Println()
	}

	// Show broken links if any
	if len(brokenLinks) > 0 {
		fmt.Printf("%s Broken Symlinks:\n", color.RedString("💔"))
		for _, link := range brokenLinks {
			fmt.Printf("  %s %s\n", color.RedString("✗"), link)
		}
		fmt.Println()
	}

	// Show missing files if any
	if len(missingFiles) > 0 {
		fmt.Printf("%s Missing Files:\n", color.YellowString("❓"))
		for _, file := range missingFiles {
			fmt.Printf("  %s %s\n", color.YellowString("?"), file)
		}
		fmt.Println()
	}

	// Recommendations
	if len(healthIssues) > 0 {
		fmt.Printf("%s Recommendations:\n", color.BlueString("💡"))
		
		if len(brokenLinks) > 0 {
			fmt.Printf("  • Run '%s' to fix broken symlinks\n", color.GreenString("dotgo install --force"))
		}
		
		if len(missingFiles) > 0 {
			fmt.Printf("  • Run '%s' to install missing packages\n", color.GreenString("dotgo install"))
		}
		
		fmt.Printf("  • Use '%s' for detailed package information\n", color.GreenString("dotgo packages status"))
		fmt.Println()
	}

	// Configuration summary
	fmt.Printf("%s Configuration:\n", color.CyanString("⚙️"))
	fmt.Printf("  Config file: %s/.dotgo/config.yaml\n", rootDir)
	fmt.Printf("  Backup directory: %s\n", configMgr.GetBackupDir())
	fmt.Printf("  Symlink mode: %s\n", cfg.Settings.SymlinkMode)
	fmt.Printf("  Conflict mode: %s\n", cfg.Settings.ConflictMode)

	return nil
}

// getStatusColor returns a colored string based on current vs total values
func getStatusColor(current, total int, value int) string {
	if current == total && total > 0 {
		return color.GreenString("%d", value)
	} else if current > 0 {
		return color.YellowString("%d", value)
	} else {
		return color.RedString("%d", value)
	}
}

// getFileStatusIcon returns an appropriate icon for file status
func getFileStatusIcon(linkInfo *symlink.LinkInfo) string {
	if !linkInfo.Exists {
		return color.RedString("✗")
	} else if linkInfo.IsSymlink {
		if linkInfo.IsValid {
			return color.GreenString("✓")
		} else {
			return color.RedString("💔")
		}
	} else {
		return color.YellowString("❓")
	}
}