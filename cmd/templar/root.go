package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "templar",
	Short: "Templar - A Go template loader with dependency management",
	Long: `Templar is a template loader library for Go that extends the standard
templating engine with:
  - Dependency management for templates
  - Template namespacing to avoid name collisions
  - Template extension/inheritance
  - Tree-shaking of dependencies
  - Multiple template loaders with fallback behavior

Configuration file locations (in order of precedence):
  1. --config flag
  2. .templar.yaml in current directory
  3. ~/.config/templar/config.yaml`,
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is .templar.yaml)")

	// Add subcommands
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(debugCmd)
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		// Look for config in current directory first
		viper.AddConfigPath(".")
		viper.SetConfigName(".templar")

		// Then check XDG config directory
		if home, err := os.UserHomeDir(); err == nil {
			viper.AddConfigPath(filepath.Join(home, ".config", "templar"))
			viper.SetConfigName("config")
		}
	}

	viper.SetConfigType("yaml")

	// Environment variable support (TEMPLAR_SERVE_ADDR, etc.)
	viper.SetEnvPrefix("TEMPLAR")
	viper.AutomaticEnv()

	// Read config file if it exists
	if err := viper.ReadInConfig(); err == nil {
		if viper.GetBool("verbose") {
			fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
		}
	}
}
