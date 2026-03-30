package templar

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// SourceConfig represents a single external template source configuration
type SourceConfig struct {
	URL     string   `yaml:"url"`               // Repository URL (e.g., github.com/user/repo)
	Path    string   `yaml:"path"`              // Directory within repo to fetch (e.g., templates)
	Version string   `yaml:"version,omitempty"` // Semantic version tag (e.g., v1.2.0)
	Ref     string   `yaml:"ref,omitempty"`     // Git ref - branch or commit (fallback if no version)
	Include []string `yaml:"include,omitempty"` // Glob patterns to include (e.g., ["**/*.html"])
	Exclude []string `yaml:"exclude,omitempty"` // Glob patterns to exclude (e.g., ["*_test.*"])
}

// GetRef returns the effective git ref (version takes precedence over ref)
func (s *SourceConfig) GetRef() string {
	if s.Version != "" {
		return s.Version
	}
	if s.Ref != "" {
		return s.Ref
	}
	return "main" // Default to main branch
}

// VendorConfig represents the templar.yaml configuration
type VendorConfig struct {
	Sources     map[string]SourceConfig `yaml:"sources"`
	VendorDir   string                  `yaml:"vendor_dir"`
	SearchPaths []string                `yaml:"search_paths"`
	RequireLock bool                    `yaml:"require_lock"`

	// configDir is the directory containing the config file (for resolving relative paths)
	configDir string
}

// LoadVendorConfig loads a VendorConfig from a config file, applying templar's
// standard defaults. For custom defaults, use LoadVendorConfigWithDefaults.
func LoadVendorConfig(path string) (*VendorConfig, error) {
	return LoadVendorConfigWithDefaults(path, DefaultToolInfo())
}

// LoadVendorConfigWithDefaults loads a VendorConfig from a config file, applying
// defaults from the given ToolInfo. Embedding applications use this to set their
// own default vendor directory when the config file doesn't specify one.
func LoadVendorConfigWithDefaults(path string, info ToolInfo) (*VendorConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config VendorConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Store the config directory for resolving relative paths
	config.configDir = filepath.Dir(path)

	// Apply defaults from ToolInfo
	if config.VendorDir == "" {
		config.VendorDir = info.VendorDir
	}

	if len(config.SearchPaths) == 0 {
		config.SearchPaths = []string{"./templates", config.VendorDir}
	}

	return &config, nil
}

// FindVendorConfig searches for templar.yaml starting from the given directory
// and walking up to parent directories until found or root is reached.
// For custom config file names, use FindVendorConfigWithNames.
func FindVendorConfig(startDir string) (string, error) {
	return FindVendorConfigWithNames(startDir, DefaultConfigNames)
}

// FindVendorConfigWithNames searches for a config file with any of the given names,
// starting from startDir and walking up to parent directories. Embedding applications
// use this to search for their own config file names (e.g. ".slyds.yaml").
func FindVendorConfigWithNames(startDir string, configNames []string) (string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", err
	}

	for {
		for _, name := range configNames {
			configPath := filepath.Join(dir, name)
			if _, err := os.Stat(configPath); err == nil {
				return configPath, nil
			}
		}

		// Move to parent directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root
			return "", fmt.Errorf("config file not found in %s or any parent directory (searched for: %s)",
				startDir, strings.Join(configNames, ", "))
		}
		dir = parent
	}
}

// ResolveVendorDir returns the absolute path to the vendor directory
func (c *VendorConfig) ResolveVendorDir() string {
	if filepath.IsAbs(c.VendorDir) {
		return c.VendorDir
	}
	return filepath.Join(c.configDir, c.VendorDir)
}

// ResolveSearchPaths returns absolute paths for all search paths
func (c *VendorConfig) ResolveSearchPaths() []string {
	resolved := make([]string, len(c.SearchPaths))
	for i, p := range c.SearchPaths {
		if filepath.IsAbs(p) {
			resolved[i] = p
		} else {
			resolved[i] = filepath.Join(c.configDir, p)
		}
	}
	return resolved
}

// NewSourceLoaderFromConfig creates a SourceLoader from a config file path.
// It loads the config, resolves all paths relative to the config file location,
// and creates the appropriate loader.
func NewSourceLoaderFromConfig(configPath string) (*SourceLoader, error) {
	config, err := LoadVendorConfig(configPath)
	if err != nil {
		return nil, err
	}

	// Resolve paths relative to config file
	config.VendorDir = config.ResolveVendorDir()
	config.SearchPaths = config.ResolveSearchPaths()

	return NewSourceLoader(config), nil
}

// NewSourceLoaderFromDir finds templar.yaml starting from the given directory
// and creates a SourceLoader from it.
func NewSourceLoaderFromDir(dir string) (*SourceLoader, error) {
	configPath, err := FindVendorConfig(dir)
	if err != nil {
		return nil, err
	}
	return NewSourceLoaderFromConfig(configPath)
}

// SourceLoader is a template loader that resolves @source prefixed paths
// to vendored template locations, while falling back to a FileSystemLoader
// for regular paths.
type SourceLoader struct {
	config     *VendorConfig
	fsLoader   *FileSystemLoader
	extensions []string
}

// NewSourceLoader creates a new SourceLoader with the given configuration.
func NewSourceLoader(config *VendorConfig) *SourceLoader {
	// Build file system loader from search paths
	fsLoader := &FileSystemLoader{
		Folders:    config.SearchPaths,
		Extensions: []string{"tmpl", "tmplus", "html"},
	}

	return &SourceLoader{
		config:     config,
		fsLoader:   fsLoader,
		extensions: []string{"tmpl", "tmplus", "html"},
	}
}

// Load attempts to load templates matching the given pattern.
// If the pattern starts with @sourcename/, it resolves to the vendored location.
// Otherwise, it delegates to the underlying FileSystemLoader.
func (s *SourceLoader) Load(pattern string, cwd string) ([]*Template, error) {
	// Check if pattern starts with @
	if strings.HasPrefix(pattern, "@") {
		return s.loadFromSource(pattern, cwd)
	}

	// Fall back to file system loader
	return s.fsLoader.Load(pattern, cwd)
}

// loadFromSource resolves @sourcename/path to the vendored location
func (s *SourceLoader) loadFromSource(pattern string, cwd string) ([]*Template, error) {
	// Pattern is @sourcename/path/to/file.html
	// Extract source name and path
	withoutAt := pattern[1:] // Remove @
	slashIdx := strings.Index(withoutAt, "/")
	if slashIdx == -1 {
		return nil, fmt.Errorf("invalid source pattern '%s': expected @sourcename/path", pattern)
	}

	sourceName := withoutAt[:slashIdx]
	sourcePath := withoutAt[slashIdx+1:]

	// Look up source in config
	_, ok := s.config.Sources[sourceName]
	if !ok {
		return nil, fmt.Errorf("source '%s' not defined in config (pattern: %s)", sourceName, pattern)
	}

	// Build the vendored path using flat structure
	// VendorDir/sourceName/sourcePath
	// e.g., templar_modules/goapplib/components/EntityListing.html
	vendoredPath := filepath.Join(
		s.config.VendorDir,
		sourceName,
		sourcePath,
	)

	// Create a temporary FileSystemLoader to load from this specific path
	vendorLoader := &FileSystemLoader{
		Folders:    []string{filepath.Dir(vendoredPath)},
		Extensions: s.extensions,
	}

	return vendorLoader.Load(filepath.Base(vendoredPath), "")
}
