package templar

import (
	"testing"
)


// createTestTarGzWithPAX creates an in-memory gzipped tarball with PAX headers








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
