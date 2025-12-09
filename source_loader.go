package templar

import (
	"fmt"
	"path/filepath"
	"strings"
)

// SourceConfig represents a single external template source configuration
type SourceConfig struct {
	URL  string `yaml:"url"`
	Path string `yaml:"path"`
	Ref  string `yaml:"ref"`
}

// VendorConfig represents the templar.yaml configuration
type VendorConfig struct {
	Sources     map[string]SourceConfig `yaml:"sources"`
	VendorDir   string                  `yaml:"vendor_dir"`
	SearchPaths []string                `yaml:"search_paths"`
	RequireLock bool                    `yaml:"require_lock"`
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
	source, ok := s.config.Sources[sourceName]
	if !ok {
		return nil, fmt.Errorf("source '%s' not defined in templar.yaml (pattern: %s)", sourceName, pattern)
	}

	// Build the vendored path
	// VendorDir/url/path/sourcePath
	// e.g., templar_modules/github.com/panyam/goapplib/templates/components/EntityListing.html
	vendoredPath := filepath.Join(
		s.config.VendorDir,
		source.URL,
		source.Path,
		sourcePath,
	)

	// Create a temporary FileSystemLoader to load from this specific path
	vendorLoader := &FileSystemLoader{
		Folders:    []string{filepath.Dir(vendoredPath)},
		Extensions: s.extensions,
	}

	return vendorLoader.Load(filepath.Base(vendoredPath), "")
}
