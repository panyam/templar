package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	initForce bool
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new templar.yaml configuration",
	Long: `Initialize a new templar.yaml configuration file in the current directory.

This creates a minimal configuration file with example sources and
sensible defaults for vendor_dir and search_paths.

Examples:
  # Create templar.yaml in current directory
  templar init

  # Overwrite existing templar.yaml
  templar init --force`,
	RunE: runInit,
}

func init() {
	initCmd.Flags().BoolVarP(&initForce, "force", "f", false, "Overwrite existing templar.yaml")

	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	configPath := "templar.yaml"

	// Check if file already exists
	if _, err := os.Stat(configPath); err == nil && !initForce {
		return fmt.Errorf("templar.yaml already exists. Use --force to overwrite")
	}

	// Create templates directory if it doesn't exist
	if err := os.MkdirAll("templates", 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not create templates directory: %v\n", err)
	}

	// Write default config
	content := `# Templar Configuration
# See https://github.com/panyam/templar/blob/main/docs/vendoring.md

# External template sources
# Add sources here and run 'templar get' to fetch them
sources:
  # Example: UI component library
  # uikit:
  #   url: github.com/example/uikit
  #   path: templates    # subdirectory within repo (optional)
  #   ref: v1.0.0        # tag, branch, or commit

# Where vendored templates are stored
vendor_dir: ./templar_modules

# Template search paths (in order of priority)
search_paths:
  - ./templates           # Local templates first
  - ./templar_modules     # Then vendored dependencies
`

	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write templar.yaml: %w", err)
	}

	absPath, _ := filepath.Abs(configPath)
	fmt.Printf("Created %s\n", absPath)
	fmt.Println("\nNext steps:")
	fmt.Println("  1. Add sources to templar.yaml")
	fmt.Println("  2. Run 'templar get' to fetch them")
	fmt.Println("  3. Reference templates with @sourcename/path syntax")

	return nil
}
