package templar

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// VendorLock represents the templar.lock file
type VendorLock struct {
	Version int                     `yaml:"version"`
	Sources map[string]LockedSource `yaml:"sources"`
}

// LockedSource represents a locked source in the lock file
type LockedSource struct {
	URL            string `yaml:"url"`
	Ref            string `yaml:"ref"`
	ResolvedCommit string `yaml:"resolved_commit"`
	FetchedAt      string `yaml:"fetched_at"`
}

// FetchResult contains the result of fetching a source
type FetchResult struct {
	SourceName     string
	URL            string
	Ref            string
	ResolvedCommit string
	DestDir        string
	FetchedAt      time.Time
}

// FetchSource fetches a single source from the config
func FetchSource(config *VendorConfig, sourceName string) (*FetchResult, error) {
	source, ok := config.Sources[sourceName]
	if !ok {
		return nil, fmt.Errorf("source '%s' not found in config", sourceName)
	}

	// Build destination directory: VendorDir/url
	destDir := filepath.Join(config.VendorDir, source.URL)

	// Clone or update the repository
	commit, err := gitCloneOrUpdate(source.URL, source.Ref, destDir)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch source '%s': %w", sourceName, err)
	}

	return &FetchResult{
		SourceName:     sourceName,
		URL:            source.URL,
		Ref:            source.Ref,
		ResolvedCommit: commit,
		DestDir:        destDir,
		FetchedAt:      time.Now(),
	}, nil
}

// FetchAllSources fetches all sources defined in the config
func FetchAllSources(config *VendorConfig) (map[string]*FetchResult, error) {
	results := make(map[string]*FetchResult)

	for name := range config.Sources {
		result, err := FetchSource(config, name)
		if err != nil {
			return results, fmt.Errorf("failed to fetch '%s': %w", name, err)
		}
		results[name] = result
	}

	return results, nil
}

// WriteLockFile writes a VendorLock to the specified path
func WriteLockFile(path string, lock *VendorLock) error {
	data, err := yaml.Marshal(lock)
	if err != nil {
		return fmt.Errorf("failed to marshal lock file: %w", err)
	}

	header := "# AUTO-GENERATED - Do not edit manually\n# Run 'templar get' to regenerate\n\n"
	content := header + string(data)

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write lock file: %w", err)
	}

	return nil
}

// LoadLockFile loads a VendorLock from the specified path
func LoadLockFile(path string) (*VendorLock, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read lock file: %w", err)
	}

	var lock VendorLock
	if err := yaml.Unmarshal(data, &lock); err != nil {
		return nil, fmt.Errorf("failed to parse lock file: %w", err)
	}

	return &lock, nil
}

// gitCloneOrUpdate clones a repository or updates it if it exists
func gitCloneOrUpdate(url, ref, destDir string) (string, error) {
	// Convert GitHub shorthand to full URL
	gitURL := url
	if strings.HasPrefix(url, "github.com/") {
		gitURL = "https://" + url + ".git"
	}

	// Check if directory already exists
	if _, err := os.Stat(destDir); err == nil {
		// Directory exists, fetch and checkout
		return gitFetchAndCheckout(destDir, ref)
	}

	// Create parent directories
	if err := os.MkdirAll(filepath.Dir(destDir), 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	// Clone the repository
	cmd := exec.Command("git", "clone", "--quiet", gitURL, destDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("git clone failed: %s: %w", string(output), err)
	}

	// Checkout the specific ref
	return gitCheckout(destDir, ref)
}

// gitFetchAndCheckout fetches updates and checks out a ref
func gitFetchAndCheckout(dir, ref string) (string, error) {
	// Fetch all refs
	cmd := exec.Command("git", "-C", dir, "fetch", "--all", "--quiet")
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("git fetch failed: %s: %w", string(output), err)
	}

	return gitCheckout(dir, ref)
}

// gitCheckout checks out a specific ref and returns the resolved commit
func gitCheckout(dir, ref string) (string, error) {
	// Try to checkout the ref
	cmd := exec.Command("git", "-C", dir, "checkout", "--quiet", ref)
	if output, err := cmd.CombinedOutput(); err != nil {
		// Try with origin/ prefix for branches
		cmd = exec.Command("git", "-C", dir, "checkout", "--quiet", "origin/"+ref)
		if output2, err2 := cmd.CombinedOutput(); err2 != nil {
			return "", fmt.Errorf("git checkout failed: %s / %s: %w", string(output), string(output2), err)
		}
	}

	// Get the resolved commit hash
	cmd = exec.Command("git", "-C", dir, "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get commit hash: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}
