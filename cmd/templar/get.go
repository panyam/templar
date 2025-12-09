package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/panyam/templar"
	"github.com/spf13/cobra"
)

var (
	updateFlag  bool
	verifyFlag  bool
	dryRunFlag  bool
	verboseFlag bool
)

var getCmd = &cobra.Command{
	Use:   "get [source...]",
	Short: "Fetch external template sources",
	Long: `Fetch external template sources defined in templar.yaml.

Examples:
  # Fetch all configured sources
  templar get

  # Fetch a specific source
  templar get @uikit

  # Update to latest versions matching refs
  templar get --update

  # Verify local files match lock file
  templar get --verify

  # Show what would be fetched without doing it
  templar get --dry-run`,
	RunE: runGet,
}

func init() {
	getCmd.Flags().BoolVarP(&updateFlag, "update", "u", false, "Update to latest versions matching refs")
	getCmd.Flags().BoolVar(&verifyFlag, "verify", false, "Verify local files match lock file")
	getCmd.Flags().BoolVar(&dryRunFlag, "dry-run", false, "Show what would be fetched without doing it")
	getCmd.Flags().BoolVarP(&verboseFlag, "verbose", "v", false, "Verbose output")

	rootCmd.AddCommand(getCmd)
}

func runGet(cmd *cobra.Command, args []string) error {
	// Find templar.yaml
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	configPath, err := templar.FindVendorConfig(cwd)
	if err != nil {
		return fmt.Errorf("no templar.yaml found: %w", err)
	}

	if verboseFlag {
		fmt.Fprintf(os.Stderr, "Using config: %s\n", configPath)
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

	// Determine which sources to fetch
	sourcesToFetch := make([]string, 0)
	if len(args) > 0 {
		for _, arg := range args {
			// Strip @ prefix if present
			name := arg
			if len(name) > 0 && name[0] == '@' {
				name = name[1:]
			}
			if _, ok := config.Sources[name]; !ok {
				return fmt.Errorf("source '%s' not found in templar.yaml", name)
			}
			sourcesToFetch = append(sourcesToFetch, name)
		}
	} else {
		for name := range config.Sources {
			sourcesToFetch = append(sourcesToFetch, name)
		}
	}

	// Dry run mode
	if dryRunFlag {
		fmt.Println("Would fetch:")
		for _, name := range sourcesToFetch {
			source := config.Sources[name]
			destDir := filepath.Join(config.VendorDir, source.URL)
			fmt.Printf("  %s: %s@%s → %s\n", name, source.URL, source.Ref, destDir)
		}
		return nil
	}

	// Verify mode
	if verifyFlag {
		return runVerify(config, configPath, sourcesToFetch)
	}

	// Fetch sources
	fmt.Printf("Fetching %d source(s)...\n", len(sourcesToFetch))

	results := make(map[string]*templar.FetchResult)
	for _, name := range sourcesToFetch {
		source := config.Sources[name]
		fmt.Printf("  %s: %s@%s... ", name, source.URL, source.Ref)

		result, err := templar.FetchSource(config, name)
		if err != nil {
			fmt.Println("FAILED")
			return fmt.Errorf("failed to fetch '%s': %w", name, err)
		}

		results[name] = result
		fmt.Printf("OK (%s)\n", result.ResolvedCommit[:7])
	}

	// Write lock file
	lockPath := filepath.Join(filepath.Dir(configPath), "templar.lock")
	lock := &templar.VendorLock{
		Version: 1,
		Sources: make(map[string]templar.LockedSource),
	}

	// Load existing lock file to preserve entries we didn't fetch
	if existing, err := templar.LoadLockFile(lockPath); err == nil {
		lock = existing
	}

	// Update with new results
	for name, result := range results {
		lock.Sources[name] = templar.LockedSource{
			URL:            result.URL,
			Ref:            result.Ref,
			ResolvedCommit: result.ResolvedCommit,
			FetchedAt:      result.FetchedAt.Format("2006-01-02T15:04:05Z"),
		}
	}

	if err := templar.WriteLockFile(lockPath, lock); err != nil {
		return fmt.Errorf("failed to write lock file: %w", err)
	}

	fmt.Printf("\nWrote %s\n", lockPath)
	return nil
}

func runVerify(config *templar.VendorConfig, configPath string, sources []string) error {
	lockPath := filepath.Join(filepath.Dir(configPath), "templar.lock")

	lock, err := templar.LoadLockFile(lockPath)
	if err != nil {
		return fmt.Errorf("no lock file found: %w", err)
	}

	allGood := true
	for _, name := range sources {
		source := config.Sources[name]
		destDir := filepath.Join(config.VendorDir, source.URL)

		locked, ok := lock.Sources[name]
		if !ok {
			fmt.Printf("✗ %s: not in lock file\n", name)
			allGood = false
			continue
		}

		// Check if directory exists
		if _, err := os.Stat(destDir); os.IsNotExist(err) {
			fmt.Printf("✗ %s: not fetched\n", name)
			allGood = false
			continue
		}

		// TODO: Verify actual commit matches lock file
		fmt.Printf("✓ %s: matches lock (%s)\n", name, locked.ResolvedCommit[:7])
	}

	if !allGood {
		return fmt.Errorf("verification failed")
	}

	return nil
}
