package templar

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

// createTestTarGz creates an in-memory gzipped tarball for testing
func createTestTarGz(t *testing.T, files map[string]string) *bytes.Buffer {
	t.Helper()

	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)

	for name, content := range files {
		// Determine if this is a directory
		isDir := len(content) == 0 && name[len(name)-1] == '/'

		var hdr *tar.Header
		if isDir {
			hdr = &tar.Header{
				Name:     name,
				Mode:     0755,
				Typeflag: tar.TypeDir,
			}
		} else {
			hdr = &tar.Header{
				Name:     name,
				Mode:     0644,
				Size:     int64(len(content)),
				Typeflag: tar.TypeReg,
			}
		}

		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("Failed to write tar header for %s: %v", name, err)
		}

		if !isDir {
			if _, err := tw.Write([]byte(content)); err != nil {
				t.Fatalf("Failed to write tar content for %s: %v", name, err)
			}
		}
	}

	if err := tw.Close(); err != nil {
		t.Fatalf("Failed to close tar writer: %v", err)
	}
	if err := gzw.Close(); err != nil {
		t.Fatalf("Failed to close gzip writer: %v", err)
	}

	return &buf
}

// createTestTarGzWithPAX creates an in-memory gzipped tarball with PAX headers
func createTestTarGzWithPAX(t *testing.T, files map[string]string) *bytes.Buffer {
	t.Helper()

	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)

	// Write a PAX global header first (type 'g')
	paxHeader := &tar.Header{
		Name:     "pax_global_header",
		Typeflag: tar.TypeXGlobalHeader,
	}
	if err := tw.WriteHeader(paxHeader); err != nil {
		t.Fatalf("Failed to write PAX header: %v", err)
	}

	for name, content := range files {
		isDir := len(content) == 0 && name[len(name)-1] == '/'

		var hdr *tar.Header
		if isDir {
			hdr = &tar.Header{
				Name:     name,
				Mode:     0755,
				Typeflag: tar.TypeDir,
			}
		} else {
			hdr = &tar.Header{
				Name:     name,
				Mode:     0644,
				Size:     int64(len(content)),
				Typeflag: tar.TypeReg,
			}
		}

		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("Failed to write tar header for %s: %v", name, err)
		}

		if !isDir {
			if _, err := tw.Write([]byte(content)); err != nil {
				t.Fatalf("Failed to write tar content for %s: %v", name, err)
			}
		}
	}

	if err := tw.Close(); err != nil {
		t.Fatalf("Failed to close tar writer: %v", err)
	}
	if err := gzw.Close(); err != nil {
		t.Fatalf("Failed to close gzip writer: %v", err)
	}

	return &buf
}

func TestExtractTarGz_Basic(t *testing.T) {
	// Create a tarball simulating GitHub structure
	tarball := createTestTarGz(t, map[string]string{
		"repo-main/":           "",
		"repo-main/README.md":  "# Readme",
		"repo-main/file1.txt":  "content1",
		"repo-main/file2.html": "<html>test</html>",
	})

	// Create temp directory
	destDir := t.TempDir()

	// Extract
	count, err := extractTarGz(tarball, destDir, "", nil, nil)
	if err != nil {
		t.Fatalf("extractTarGz failed: %v", err)
	}

	if count != 3 {
		t.Errorf("Expected 3 files extracted, got %d", count)
	}

	// Verify files exist
	for _, name := range []string{"README.md", "file1.txt", "file2.html"} {
		path := filepath.Join(destDir, name)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected file %s to exist", name)
		}
	}
}

func TestExtractTarGz_WithPAXHeader(t *testing.T) {
	// Create a tarball with PAX global header (like GitHub produces)
	tarball := createTestTarGzWithPAX(t, map[string]string{
		"repo-main/":          "",
		"repo-main/file1.txt": "content1",
		"repo-main/file2.txt": "content2",
	})

	destDir := t.TempDir()

	count, err := extractTarGz(tarball, destDir, "", nil, nil)
	if err != nil {
		t.Fatalf("extractTarGz failed: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected 2 files extracted, got %d", count)
	}

	// Verify files exist (PAX header should be skipped)
	for _, name := range []string{"file1.txt", "file2.txt"} {
		path := filepath.Join(destDir, name)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected file %s to exist", name)
		}
	}

	// Verify PAX header was not extracted as a file
	paxPath := filepath.Join(destDir, "pax_global_header")
	if _, err := os.Stat(paxPath); !os.IsNotExist(err) {
		t.Error("PAX header should not be extracted as a file")
	}
}

func TestExtractTarGz_SubPath(t *testing.T) {
	// Create a tarball with subdirectories
	tarball := createTestTarGz(t, map[string]string{
		"repo-main/":                       "",
		"repo-main/README.md":              "# Readme",
		"repo-main/src/":                   "",
		"repo-main/src/main.go":            "package main",
		"repo-main/templates/":             "",
		"repo-main/templates/base.html":    "<html>base</html>",
		"repo-main/templates/page.html":    "<html>page</html>",
		"repo-main/templates/components/":  "",
		"repo-main/templates/components/btn.html": "<button>btn</button>",
	})

	destDir := t.TempDir()

	// Extract only the templates subdirectory
	count, err := extractTarGz(tarball, destDir, "templates", nil, nil)
	if err != nil {
		t.Fatalf("extractTarGz failed: %v", err)
	}

	if count != 3 {
		t.Errorf("Expected 3 files extracted, got %d", count)
	}

	// Verify template files exist at root (subPath stripped)
	expectedFiles := []string{"base.html", "page.html", "components/btn.html"}
	for _, name := range expectedFiles {
		path := filepath.Join(destDir, name)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected file %s to exist", name)
		}
	}

	// Verify non-template files were NOT extracted
	unexpectedFiles := []string{"README.md", "src/main.go"}
	for _, name := range unexpectedFiles {
		path := filepath.Join(destDir, name)
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("File %s should NOT exist (not in subPath)", name)
		}
	}
}

func TestExtractTarGz_IncludePattern(t *testing.T) {
	tarball := createTestTarGz(t, map[string]string{
		"repo-main/":           "",
		"repo-main/file1.html": "<html>1</html>",
		"repo-main/file2.html": "<html>2</html>",
		"repo-main/file3.txt":  "text file",
		"repo-main/style.css":  "body {}",
	})

	destDir := t.TempDir()

	// Extract only .html files
	count, err := extractTarGz(tarball, destDir, "", []string{"*.html"}, nil)
	if err != nil {
		t.Fatalf("extractTarGz failed: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected 2 files extracted, got %d", count)
	}

	// Verify .html files exist
	for _, name := range []string{"file1.html", "file2.html"} {
		path := filepath.Join(destDir, name)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected file %s to exist", name)
		}
	}

	// Verify non-.html files were excluded
	for _, name := range []string{"file3.txt", "style.css"} {
		path := filepath.Join(destDir, name)
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("File %s should NOT exist (not matching include pattern)", name)
		}
	}
}

func TestExtractTarGz_ExcludePattern(t *testing.T) {
	tarball := createTestTarGz(t, map[string]string{
		"repo-main/":              "",
		"repo-main/main.go":       "package main",
		"repo-main/main_test.go":  "package main // test",
		"repo-main/utils.go":      "package main",
		"repo-main/utils_test.go": "package main // test",
	})

	destDir := t.TempDir()

	// Exclude test files
	count, err := extractTarGz(tarball, destDir, "", nil, []string{"*_test.go"})
	if err != nil {
		t.Fatalf("extractTarGz failed: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected 2 files extracted, got %d", count)
	}

	// Verify non-test files exist
	for _, name := range []string{"main.go", "utils.go"} {
		path := filepath.Join(destDir, name)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected file %s to exist", name)
		}
	}

	// Verify test files were excluded
	for _, name := range []string{"main_test.go", "utils_test.go"} {
		path := filepath.Join(destDir, name)
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("File %s should NOT exist (matching exclude pattern)", name)
		}
	}
}

func TestExtractTarGz_IncludeAndExclude(t *testing.T) {
	tarball := createTestTarGz(t, map[string]string{
		"repo-main/":                   "",
		"repo-main/page.html":          "<html>page</html>",
		"repo-main/page_test.html":     "<html>test</html>",
		"repo-main/style.css":          "body {}",
		"repo-main/components/":        "",
		"repo-main/components/btn.html": "<button>",
		"repo-main/components/btn_test.html": "<button>test",
	})

	destDir := t.TempDir()

	// Include .html but exclude *_test.html in any directory
	count, err := extractTarGz(tarball, destDir, "", []string{"**/*.html"}, []string{"**/*_test.html"})
	if err != nil {
		t.Fatalf("extractTarGz failed: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected 2 files extracted, got %d", count)
	}

	// Verify correct files exist
	for _, name := range []string{"page.html", "components/btn.html"} {
		path := filepath.Join(destDir, name)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected file %s to exist", name)
		}
	}

	// Verify excluded files don't exist
	for _, name := range []string{"page_test.html", "style.css", "components/btn_test.html"} {
		path := filepath.Join(destDir, name)
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("File %s should NOT exist", name)
		}
	}
}

func TestExtractTarGz_NestedDirectories(t *testing.T) {
	tarball := createTestTarGz(t, map[string]string{
		"repo-main/":                     "",
		"repo-main/a/":                   "",
		"repo-main/a/b/":                 "",
		"repo-main/a/b/c/":               "",
		"repo-main/a/b/c/deep.txt":       "deep content",
		"repo-main/a/file.txt":           "a content",
	})

	destDir := t.TempDir()

	count, err := extractTarGz(tarball, destDir, "", nil, nil)
	if err != nil {
		t.Fatalf("extractTarGz failed: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected 2 files extracted, got %d", count)
	}

	// Verify nested file exists
	deepPath := filepath.Join(destDir, "a", "b", "c", "deep.txt")
	if _, err := os.Stat(deepPath); os.IsNotExist(err) {
		t.Error("Expected deep nested file to exist")
	}

	// Verify content
	content, err := os.ReadFile(deepPath)
	if err != nil {
		t.Fatalf("Failed to read deep file: %v", err)
	}
	if string(content) != "deep content" {
		t.Errorf("Expected 'deep content', got '%s'", string(content))
	}
}

func TestGlobToRegex(t *testing.T) {
	tests := []struct {
		glob     string
		input    string
		expected bool
	}{
		// Simple wildcards
		{"*.html", "file.html", true},
		{"*.html", "file.txt", false},
		{"*.html", "path/file.html", false}, // * doesn't match /

		// Double star (matches any path)
		{"**/*.html", "file.html", true},
		{"**/*.html", "path/file.html", true},
		{"**/*.html", "a/b/c/file.html", true},
		{"**/*.html", "file.txt", false},

		// Question mark
		{"file?.txt", "file1.txt", true},
		{"file?.txt", "file12.txt", false},
		{"file?.txt", "file.txt", false},

		// Complex patterns
		{"*_test.go", "main_test.go", true},
		{"*_test.go", "utils_test.go", true},
		{"*_test.go", "main.go", false},

		// Escape special regex chars
		{"file.txt", "file.txt", true},
		{"file.txt", "fileatxt", false}, // . should be literal, not regex any
	}

	for _, tt := range tests {
		t.Run(tt.glob+"_"+tt.input, func(t *testing.T) {
			regex := globToRegex(tt.glob)
			patterns := compilePatterns([]string{tt.glob})
			result := matchesPatterns(tt.input, patterns, nil)
			if result != tt.expected {
				t.Errorf("globToRegex(%q) matching %q: expected %v, got %v (regex: %s)",
					tt.glob, tt.input, tt.expected, result, regex)
			}
		})
	}
}

func TestMatchesPatterns(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		include  []string
		exclude  []string
		expected bool
	}{
		{
			name:     "no patterns matches all",
			path:     "anything.txt",
			include:  nil,
			exclude:  nil,
			expected: true,
		},
		{
			name:     "include pattern matches",
			path:     "file.html",
			include:  []string{"*.html"},
			exclude:  nil,
			expected: true,
		},
		{
			name:     "include pattern doesn't match",
			path:     "file.txt",
			include:  []string{"*.html"},
			exclude:  nil,
			expected: false,
		},
		{
			name:     "exclude pattern matches",
			path:     "test_file.go",
			include:  nil,
			exclude:  []string{"test_*"},
			expected: false,
		},
		{
			name:     "include matches but exclude also matches",
			path:     "page_test.html",
			include:  []string{"*.html"},
			exclude:  []string{"*_test.*"},
			expected: false,
		},
		{
			name:     "multiple include patterns",
			path:     "style.css",
			include:  []string{"*.html", "*.css"},
			exclude:  nil,
			expected: true,
		},
		{
			name:     "multiple exclude patterns",
			path:     "vendor/lib.js",
			include:  nil,
			exclude:  []string{"vendor/*", "node_modules/*"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			includePatterns := compilePatterns(tt.include)
			excludePatterns := compilePatterns(tt.exclude)
			result := matchesPatterns(tt.path, includePatterns, excludePatterns)
			if result != tt.expected {
				t.Errorf("matchesPatterns(%q, %v, %v): expected %v, got %v",
					tt.path, tt.include, tt.exclude, tt.expected, result)
			}
		})
	}
}

func TestIsGitHubURL(t *testing.T) {
	tests := []struct {
		url      string
		expected bool
	}{
		{"github.com/user/repo", true},
		{"github.com/org/project", true},
		{"gitlab.com/user/repo", false},
		{"bitbucket.org/user/repo", false},
		{"https://github.com/user/repo", false}, // We expect without https://
		{"git@github.com:user/repo.git", false},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			result := isGitHubURL(tt.url)
			if result != tt.expected {
				t.Errorf("isGitHubURL(%q): expected %v, got %v", tt.url, tt.expected, result)
			}
		})
	}
}

func TestSourceConfig_GetRef(t *testing.T) {
	tests := []struct {
		name     string
		source   SourceConfig
		expected string
	}{
		{
			name:     "version takes precedence",
			source:   SourceConfig{Version: "v1.2.0", Ref: "main"},
			expected: "v1.2.0",
		},
		{
			name:     "ref when no version",
			source:   SourceConfig{Ref: "develop"},
			expected: "develop",
		},
		{
			name:     "default to main",
			source:   SourceConfig{},
			expected: "main",
		},
		{
			name:     "commit hash as ref",
			source:   SourceConfig{Ref: "abc123def"},
			expected: "abc123def",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.source.GetRef()
			if result != tt.expected {
				t.Errorf("GetRef(): expected %q, got %q", tt.expected, result)
			}
		})
	}
}
