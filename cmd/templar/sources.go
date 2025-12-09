package main

import (
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/panyam/templar"
	"github.com/spf13/cobra"
)

var sourcesCmd = &cobra.Command{
	Use:   "sources",
	Short: "List configured template sources",
	Long: `List all external template sources defined in templar.yaml and their status.

Examples:
  # Show configured sources and their status
  templar sources`,
	RunE: runSources,
}

func init() {
	rootCmd.AddCommand(sourcesCmd)
}

func runSources(cmd *cobra.Command, args []string) error {
	// Find templar.yaml
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	configPath, err := templar.FindVendorConfig(cwd)
	if err != nil {
		return fmt.Errorf("no templar.yaml found: %w", err)
	}

	config, err := templar.LoadVendorConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Resolve paths relative to config file
	config.VendorDir = config.ResolveVendorDir()

	if len(config.Sources) == 0 {
		fmt.Println("No sources configured in templar.yaml")
		return nil
	}

	// Try to load lock file
	lockPath := filepath.Join(filepath.Dir(configPath), "templar.lock")
	lock, _ := templar.LoadLockFile(lockPath) // Ignore error if lock file doesn't exist

	// Print table
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "SOURCE\tURL\tREF\tSTATUS")

	for name, source := range config.Sources {
		status := "✗ not fetched"

		destDir := filepath.Join(config.VendorDir, source.URL)
		if _, err := os.Stat(destDir); err == nil {
			// Directory exists
			if lock != nil {
				if locked, ok := lock.Sources[name]; ok {
					status = fmt.Sprintf("✓ vendored (%s)", locked.ResolvedCommit[:7])
				} else {
					status = "✓ vendored (not locked)"
				}
			} else {
				status = "✓ vendored (no lock file)"
			}
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", name, source.URL, source.Ref, status)
	}

	w.Flush()
	return nil
}
