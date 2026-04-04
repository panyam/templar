package templar

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// FetchSourceFS fetches a single source and writes the extracted files to the
// given WritableFS. The destination is {sourceName}/ within the FS.
// All bytes flow from network → gzip → tar → WritableFS. No temp files.
func FetchSourceFS(fsys WritableFS, config *VendorConfig, sourceName string) (*FetchResult, error) {
	source, ok := config.Sources[sourceName]
	if !ok {
		return nil, fmt.Errorf("source '%s' not found in config", sourceName)
	}

	destPath := sourceName
	ref := source.GetRef()

	// Clear existing destination by removing known files
	// (WritableFS doesn't have RemoveAll, but we can recreate the dir)
	fsys.MkdirAll(destPath, 0750)

	var commit string
	var filesExtracted int
	var err error

	if isGitHubURL(source.URL) {
		commit, filesExtracted, err = fetchFromGitHubFS(fsys, source, destPath, ref)
	} else {
		return nil, fmt.Errorf("non-GitHub sources not yet supported with WritableFS")
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
		DestDir:        destPath,
		FilesExtracted: filesExtracted,
		FetchedAt:      time.Now(),
	}, nil
}

// FetchAllSourcesFS fetches all sources and writes to the WritableFS.
func FetchAllSourcesFS(fsys WritableFS, config *VendorConfig) (map[string]*FetchResult, error) {
	results := make(map[string]*FetchResult)
	for name := range config.Sources {
		result, err := FetchSourceFS(fsys, config, name)
		if err != nil {
			return results, fmt.Errorf("failed to fetch '%s': %w", name, err)
		}
		results[name] = result
	}
	return results, nil
}

// fetchFromGitHubFS downloads a GitHub tarball and extracts it to WritableFS.
// Pure streaming: HTTP response → gzip → tar → FS writes.
func fetchFromGitHubFS(fsys WritableFS, source SourceConfig, destPath, ref string) (string, int, error) {
	parts := strings.Split(strings.TrimPrefix(source.URL, "github.com/"), "/")
	if len(parts) < 2 {
		return "", 0, fmt.Errorf("invalid GitHub URL: %s", source.URL)
	}
	owner, repo := parts[0], parts[1]

	tarballURL := fmt.Sprintf("https://codeload.github.com/%s/%s/tar.gz/%s", owner, repo, ref)

	resp, err := http.Get(tarballURL) // #nosec G107
	if err != nil {
		return "", 0, fmt.Errorf("failed to download tarball: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", 0, fmt.Errorf("GitHub returned status %d for %s", resp.StatusCode, tarballURL)
	}

	filesExtracted, err := extractTarGzFS(fsys, resp.Body, destPath, source.Path, source.Include, source.Exclude)
	if err != nil {
		return "", 0, fmt.Errorf("failed to extract tarball: %w", err)
	}

	return ref, filesExtracted, nil
}

// extractTarGzFS extracts a gzipped tarball to WritableFS.
// All data flows through io.Reader — no temp files.
func extractTarGzFS(fsys WritableFS, reader io.Reader, destBase, subPath string, include, exclude []string) (int, error) {
	gzr, err := gzip.NewReader(reader)
	if err != nil {
		return 0, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	filesExtracted := 0

	includePatterns := compilePatterns(include)
	excludePatterns := compilePatterns(exclude)

	var topLevelDir string
	subPath = strings.TrimSuffix(subPath, "/")

	debugMode := os.Getenv("TEMPLAR_DEBUG") != ""

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return filesExtracted, fmt.Errorf("error reading tarball: %w", err)
		}

		name := header.Name

		// Skip PAX headers
		if header.Typeflag == 'g' || header.Typeflag == 'x' {
			continue
		}

		// Detect top-level dir
		if topLevelDir == "" {
			topLevelDir = strings.Split(name, "/")[0] + "/"
		}

		if !strings.HasPrefix(name, topLevelDir) {
			continue
		}
		relativePath := strings.TrimPrefix(name, topLevelDir)
		if relativePath == "" {
			continue
		}

		// subPath filtering
		if subPath != "" {
			relNoSlash := strings.TrimSuffix(relativePath, "/")
			if relNoSlash != subPath && !strings.HasPrefix(relativePath, subPath+"/") {
				continue
			}
			if relNoSlash == subPath {
				continue
			}
			relativePath = strings.TrimPrefix(relativePath, subPath+"/")
		}

		if relativePath == "" {
			continue
		}

		// Include/exclude
		if !matchesPatterns(relativePath, includePatterns, excludePatterns) {
			if debugMode {
				fmt.Fprintf(os.Stderr, "  excluded: %s\n", relativePath)
			}
			continue
		}

		// Build destination path within FS
		destPath := destBase + "/" + relativePath

		switch header.Typeflag {
		case tar.TypeDir:
			fsys.MkdirAll(destPath, 0750)
		case tar.TypeReg, tar.TypeRegA:
			// Ensure parent directory exists
			if dir := path.Dir(destPath); dir != "." {
				fsys.MkdirAll(dir, 0750)
			}

			// Read file content with size limit (256MB)
			const maxFileSize = 256 << 20
			data, err := io.ReadAll(io.LimitReader(tr, maxFileSize))
			if err != nil {
				return filesExtracted, fmt.Errorf("failed to read file %s: %w", destPath, err)
			}

			perm := fs.FileMode(header.Mode) & fs.ModePerm
			if perm == 0 {
				perm = 0644
			}

			if err := fsys.WriteFile(destPath, data, perm); err != nil {
				return filesExtracted, fmt.Errorf("failed to write file %s: %w", destPath, err)
			}

			filesExtracted++
		}
	}

	return filesExtracted, nil
}

// WriteLockFileFS writes a VendorLock to the WritableFS.
func WriteLockFileFS(fsys WritableFS, name string, lock *VendorLock, info ToolInfo) error {
	data, err := yaml.Marshal(lock)
	if err != nil {
		return fmt.Errorf("failed to marshal lock file: %w", err)
	}

	dirName := info.VendorDir
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
	return fsys.WriteFile(name, []byte(content), 0600)
}

// WriteVendorReadmeFS writes a README.md to the WritableFS.
func WriteVendorReadmeFS(fsys WritableFS, dirName string, info ToolInfo) error {
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

	return fsys.WriteFile(dirName+"/README.md", []byte(readme), 0600)
}

// LoadLockFileFS loads a VendorLock from a WritableFS.
func LoadLockFileFS(fsys WritableFS, name string) (*VendorLock, error) {
	data, err := fsys.ReadFile(name)
	if err != nil {
		return nil, fmt.Errorf("failed to read lock file: %w", err)
	}
	var lock VendorLock
	if err := yaml.Unmarshal(data, &lock); err != nil {
		return nil, fmt.Errorf("failed to parse lock file: %w", err)
	}
	return &lock, nil
}

// Note: compilePatterns, matchesPatterns, globToRegex, isGitHubURL
// are defined in fetch.go and shared by both path-based and FS-based functions.
