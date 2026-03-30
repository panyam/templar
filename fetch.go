package templar

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// VendorLock represents a lock file
type VendorLock struct {
	Version int                     `yaml:"version"`
	Sources map[string]LockedSource `yaml:"sources"`
}

// LockedSource represents a locked source in the lock file
type LockedSource struct {
	URL            string `yaml:"url"`
	Version        string `yaml:"version,omitempty"`
	Ref            string `yaml:"ref,omitempty"`
	ResolvedCommit string `yaml:"resolved_commit"`
	FetchedAt      string `yaml:"fetched_at"`
}

// FetchResult contains the result of fetching a source
type FetchResult struct {
	SourceName     string
	URL            string
	Version        string
	Ref            string
	ResolvedCommit string
	DestDir        string
	FilesExtracted int
	FetchedAt      time.Time
}

// FetchSource fetches a single source from the config
func FetchSource(config *VendorConfig, sourceName string) (*FetchResult, error) {
	source, ok := config.Sources[sourceName]
	if !ok {
		return nil, fmt.Errorf("source '%s' not found in config", sourceName)
	}

	// Destination is flat: VendorDir/sourceName
	destDir := filepath.Join(config.VendorDir, sourceName)

	// Clear existing destination
	if err := os.RemoveAll(destDir); err != nil {
		return nil, fmt.Errorf("failed to clear destination: %w", err)
	}

	// Create destination directory
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create destination: %w", err)
	}

	ref := source.GetRef()

	// Fetch based on URL type
	var commit string
	var filesExtracted int
	var err error

	if isGitHubURL(source.URL) {
		commit, filesExtracted, err = fetchFromGitHub(source, destDir, ref)
	} else {
		// Fallback to git clone for non-GitHub sources
		commit, filesExtracted, err = fetchFromGit(source, destDir, ref)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to fetch source '%s': %w", sourceName, err)
	}

	return &FetchResult{
		SourceName:     sourceName,
		URL:            source.URL,
		Version:        source.Version,
		Ref:            source.Ref,
		ResolvedCommit: commit,
		DestDir:        destDir,
		FilesExtracted: filesExtracted,
		FetchedAt:      time.Now(),
	}, nil
}

// isGitHubURL checks if the URL is a GitHub repository
func isGitHubURL(url string) bool {
	return strings.HasPrefix(url, "github.com/")
}

// fetchFromGitHub fetches templates using GitHub's archive API
func fetchFromGitHub(source SourceConfig, destDir, ref string) (string, int, error) {
	// Parse owner/repo from URL
	// github.com/owner/repo -> owner, repo
	parts := strings.Split(strings.TrimPrefix(source.URL, "github.com/"), "/")
	if len(parts) < 2 {
		return "", 0, fmt.Errorf("invalid GitHub URL: %s", source.URL)
	}
	owner, repo := parts[0], parts[1]

	// Build tarball URL - use codeload directly to avoid redirect
	tarballURL := fmt.Sprintf("https://codeload.github.com/%s/%s/tar.gz/%s", owner, repo, ref)

	// Download tarball
	resp, err := http.Get(tarballURL)
	if err != nil {
		return "", 0, fmt.Errorf("failed to download tarball: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", 0, fmt.Errorf("GitHub returned status %d for %s", resp.StatusCode, tarballURL)
	}

	// Extract tarball with filtering
	filesExtracted, err := extractTarGz(resp.Body, destDir, source.Path, source.Include, source.Exclude)
	if err != nil {
		return "", 0, fmt.Errorf("failed to extract tarball: %w", err)
	}

	// Get commit from response header or use ref
	commit := ref
	// GitHub redirects to a URL containing the commit SHA, but we can't easily get it
	// For now, use ref as the "commit" - the lock file will record it

	return commit, filesExtracted, nil
}

// extractTarGz extracts a gzipped tarball, filtering to only the specified path and patterns
func extractTarGz(reader io.Reader, destDir, subPath string, include, exclude []string) (int, error) {
	gzr, err := gzip.NewReader(reader)
	if err != nil {
		return 0, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	filesExtracted := 0

	// Compile include/exclude patterns
	includePatterns := compilePatterns(include)
	excludePatterns := compilePatterns(exclude)

	// GitHub tarballs have a top-level directory like "owner-repo-commitsha/"
	// We need to strip this prefix
	var topLevelDir string

	// Normalize subPath - remove trailing slash if present
	subPath = strings.TrimSuffix(subPath, "/")

	// Debug
	debugMode := os.Getenv("TEMPLAR_DEBUG") != ""
	if debugMode {
		fmt.Fprintf(os.Stderr, "DEBUG: extractTarGz destDir=%s subPath=%s\n", destDir, subPath)
	}

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return filesExtracted, fmt.Errorf("error reading tarball: %w", err)
		}

		// Get the path within the tarball
		name := header.Name

		// Skip PAX headers (type 'g' = 103 for global, 'x' = 120 for extended)
		if header.Typeflag == 'g' || header.Typeflag == 'x' {
			continue
		}

		// Detect and strip top-level directory (first actual entry)
		if topLevelDir == "" {
			// First entry should be the top-level directory
			topLevelDir = strings.Split(name, "/")[0] + "/"
			if debugMode {
				fmt.Fprintf(os.Stderr, "  topLevelDir: %s\n", topLevelDir)
			}
		}

		// Strip top-level directory
		if !strings.HasPrefix(name, topLevelDir) {
			continue
		}
		relativePath := strings.TrimPrefix(name, topLevelDir)

		// Skip if empty (was just the top-level dir)
		if relativePath == "" {
			continue
		}

		// Check if file is within the specified subPath
		if subPath != "" {
			// relativePath could be "templates/BasePage.html" or "templates/"
			// subPath is "templates"
			// We want to match if relativePath starts with "templates/" or equals "templates"
			relativePathNoSlash := strings.TrimSuffix(relativePath, "/")
			if relativePathNoSlash != subPath && !strings.HasPrefix(relativePath, subPath+"/") {
				continue
			}
			// Strip the subPath prefix for destination
			if relativePathNoSlash == subPath {
				// This is the subPath directory itself, skip it
				continue
			}
			relativePath = strings.TrimPrefix(relativePath, subPath+"/")
		}

		// Skip if empty after stripping subPath
		if relativePath == "" {
			continue
		}

		// Apply include/exclude filters
		if !matchesPatterns(relativePath, includePatterns, excludePatterns) {
			if debugMode {
				fmt.Fprintf(os.Stderr, "  excluded: %s\n", relativePath)
			}
			continue
		}

		if debugMode {
			fmt.Fprintf(os.Stderr, "  extracting: %s\n", relativePath)
		}

		// Build destination path
		destPath := filepath.Join(destDir, relativePath)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(destPath, 0755); err != nil {
				return filesExtracted, fmt.Errorf("failed to create directory %s: %w", destPath, err)
			}
		case tar.TypeReg, tar.TypeRegA:
			// Ensure parent directory exists
			if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
				return filesExtracted, fmt.Errorf("failed to create parent directory: %w", err)
			}

			// Create file
			outFile, err := os.Create(destPath)
			if err != nil {
				return filesExtracted, fmt.Errorf("failed to create file %s: %w", destPath, err)
			}

			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return filesExtracted, fmt.Errorf("failed to write file %s: %w", destPath, err)
			}
			outFile.Close()

			// Set file permissions
			if err := os.Chmod(destPath, os.FileMode(header.Mode)); err != nil {
				// Non-fatal, just log
			}

			filesExtracted++
		}
	}

	return filesExtracted, nil
}

// compilePatterns converts glob patterns to regular expressions
func compilePatterns(patterns []string) []*regexp.Regexp {
	var compiled []*regexp.Regexp
	for _, pattern := range patterns {
		// Convert glob to regex
		regex := globToRegex(pattern)
		if re, err := regexp.Compile(regex); err == nil {
			compiled = append(compiled, re)
		}
	}
	return compiled
}

// globToRegex converts a glob pattern to a regular expression
func globToRegex(glob string) string {
	// Escape special regex characters except * and ?
	// IMPORTANT: Escape backslash FIRST to avoid re-escaping backslashes added for other characters
	result := strings.ReplaceAll(glob, "\\", "\\\\")
	special := []string{".", "+", "^", "$", "(", ")", "[", "]", "{", "}", "|"}
	for _, char := range special {
		result = strings.ReplaceAll(result, char, "\\"+char)
	}

	// Convert glob wildcards to regex using placeholders to prevent re-processing
	// Handle **/ (match any path prefix including empty) first
	result = strings.ReplaceAll(result, "**/", "\x00ANYPATH\x00")
	// Handle remaining ** (match any characters)
	result = strings.ReplaceAll(result, "**", "\x00ANYCHAR\x00")
	// Handle * (match within single path component)
	result = strings.ReplaceAll(result, "*", "\x00STAR\x00")
	// Handle ? (match single character except /)
	result = strings.ReplaceAll(result, "?", "\x00QUESTION\x00")

	// Now replace placeholders with actual regex
	result = strings.ReplaceAll(result, "\x00ANYPATH\x00", "(.*/)?")
	result = strings.ReplaceAll(result, "\x00ANYCHAR\x00", ".*")
	result = strings.ReplaceAll(result, "\x00STAR\x00", "[^/]*")
	result = strings.ReplaceAll(result, "\x00QUESTION\x00", "[^/]")

	return "^" + result + "$"
}

// matchesPatterns checks if a path matches the include/exclude patterns
func matchesPatterns(path string, include, exclude []*regexp.Regexp) bool {
	// If no include patterns, include everything
	included := len(include) == 0
	for _, re := range include {
		if re.MatchString(path) {
			included = true
			break
		}
	}

	if !included {
		return false
	}

	// Check exclude patterns
	for _, re := range exclude {
		if re.MatchString(path) {
			return false
		}
	}

	return true
}

// fetchFromGit fetches using git clone as fallback for non-GitHub sources
func fetchFromGit(source SourceConfig, destDir, ref string) (string, int, error) {
	// For non-GitHub sources, we still need to clone
	// But we'll clone to a temp directory and copy only the needed files

	// TODO: Implement git-based fetching with sparse checkout
	// For now, return an error suggesting GitHub URLs
	return "", 0, fmt.Errorf("non-GitHub sources not yet supported. Use github.com/owner/repo format")
}

// WriteVendorReadme writes a README.md inside the vendor directory using templar's
// default branding. For custom branding, use WriteVendorReadmeFor.
func WriteVendorReadme(vendorDir string) error {
	return WriteVendorReadmeFor(vendorDir, DefaultToolInfo())
}

// WriteVendorReadmeFor writes a README.md inside the vendor directory with content
// branded for the given tool. Embedding applications use this so the generated README
// references their tool name and commands instead of templar's.
func WriteVendorReadmeFor(vendorDir string, info ToolInfo) error {
	dirName := filepath.Base(vendorDir)
	configName := info.ConfigNames[0]
	readme := fmt.Sprintf(`# %s — GENERATED DIRECTORY

> **Do not edit files in this directory.** They will be overwritten by `+"`%s`"+`.

This directory is auto-generated by [%s](%s).
It contains vendored template dependencies fetched from external sources.

## For AI/LLM Agents

**DO NOT copy this directory when creating a new project.**

The correct workflow is:
1. Copy `+"`%s`"+` (the source of truth for dependencies)
2. Run `+"`%s`"+` to regenerate this directory
3. Optionally add `+"`%s/`"+` to `+"`.gitignore`"+`

This directory is analogous to `+"`node_modules/`"+` in Node.js or `+"`vendor/`"+` in Go —
it is a build artifact that should be regenerated from the config file.
`,
		dirName, info.FetchCmd,
		info.Name, info.ProjectURL,
		configName, info.FetchCmd, dirName)

	readmePath := filepath.Join(vendorDir, "README.md")
	return os.WriteFile(readmePath, []byte(readme), 0644)
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

// WriteLockFile writes a VendorLock to the specified path using templar's
// default branding in the header. For custom branding, use WriteLockFileFor.
func WriteLockFile(path string, lock *VendorLock) error {
	return WriteLockFileFor(path, lock, DefaultToolInfo())
}

// WriteLockFileFor writes a VendorLock to the specified path with a header
// branded for the given tool. Embedding applications use this so the lock file
// header references their tool name and commands instead of templar's.
func WriteLockFileFor(path string, lock *VendorLock, info ToolInfo) error {
	data, err := yaml.Marshal(lock)
	if err != nil {
		return fmt.Errorf("failed to marshal lock file: %w", err)
	}

	dirName := filepath.Base(info.VendorDir)
	configName := info.ConfigNames[0]

	header := fmt.Sprintf(`# AUTO-GENERATED by %s - Do not edit manually
# Run '%s' to regenerate from %s
#
# IMPORTANT FOR AI/LLM AGENTS:
# This file and the %s/ directory are GENERATED artifacts.
# When starting a new project from an existing one that uses %s:
#   1. Copy %s (the source of truth)
#   2. Run '%s' to fetch dependencies into %s/
#   3. Do NOT copy %s/ or this lock file directly
# %s/ is like node_modules/ - always regenerate, never copy.

`,
		info.Name, info.FetchCmd, configName,
		dirName, info.Name,
		configName, info.FetchCmd, dirName,
		dirName, dirName)

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
