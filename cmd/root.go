package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	verbose bool
	dryRun  bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "dotgo",
	Short: "Simple dotfiles management tool",
	Long: `dotgo is a simple dotfiles management tool written in Go.
It provides tag-based organization and symlink management for dotfiles.

Features:
  • Simple YAML-based configuration (dotgo.yaml)
  • Tag-based file organization
  • Cross-platform symlink management
  • Automatic backup and restore
  • Dry-run support for safe operations

Commands:
  • init   - Initialize a new dotgo repository
  • add    - Add files to dotfiles management
  • apply  - Create symlinks for managed files  
  • rm     - Remove files from management
  • status - Show current status`,
	Version: "0.2.0",
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./dotgo.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "show what would be done without making changes")

	// Bind flags to viper
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
	viper.BindPFlag("dry-run", rootCmd.PersistentFlags().Lookup("dry-run"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".dotgo" (without extension).
		dotgoDir := filepath.Join(home, ".dotgo")
		viper.AddConfigPath(dotgoDir)
		viper.AddConfigPath(".")
		viper.SetConfigType("yaml")
		viper.SetConfigName("config")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil && verbose {
		fmt.Fprintln(os.Stderr, color.GreenString("Using config file: %s", viper.ConfigFileUsed()))
	}
}
