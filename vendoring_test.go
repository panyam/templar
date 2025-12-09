package templar

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// VendorLock represents the templar.lock file
type VendorLock struct {
	Version int                     `yaml:"version"`
	Sources map[string]LockedSource `yaml:"sources"`
}

// LockedSource represents a locked source with resolved commit
type LockedSource struct {
	URL            string `yaml:"url"`
	Ref            string `yaml:"ref"`
	ResolvedCommit string `yaml:"resolved_commit"`
	FetchedAt      string `yaml:"fetched_at"`
}

// TestVendorConfig_Parse tests parsing of templar.yaml configuration
func TestVendorConfig_Parse(t *testing.T) {
	configYAML := `
sources:
  goapplib:
    url: github.com/panyam/goapplib
    path: templates
    ref: v1.2.0
  shared:
    url: github.com/myorg/shared-templates
    ref: main

vendor_dir: ./templar_modules

search_paths:
  - ./templates
  - ./templar_modules

require_lock: true
`

	var config VendorConfig
	err := yaml.Unmarshal([]byte(configYAML), &config)
	if err != nil {
		t.Fatalf("Failed to parse config YAML: %v", err)
	}

	// Check sources
	if len(config.Sources) != 2 {
		t.Errorf("Expected 2 sources, got %d", len(config.Sources))
	}

	goapplib, ok := config.Sources["goapplib"]
	if !ok {
		t.Error("Expected 'goapplib' source to exist")
	} else {
		if goapplib.URL != "github.com/panyam/goapplib" {
			t.Errorf("Expected URL 'github.com/panyam/goapplib', got '%s'", goapplib.URL)
		}
		if goapplib.Path != "templates" {
			t.Errorf("Expected path 'templates', got '%s'", goapplib.Path)
		}
		if goapplib.Ref != "v1.2.0" {
			t.Errorf("Expected ref 'v1.2.0', got '%s'", goapplib.Ref)
		}
	}

	// Check vendor_dir
	if config.VendorDir != "./templar_modules" {
		t.Errorf("Expected vendor_dir './templar_modules', got '%s'", config.VendorDir)
	}

	// Check search_paths
	if len(config.SearchPaths) != 2 {
		t.Errorf("Expected 2 search paths, got %d", len(config.SearchPaths))
	}

	// Check require_lock
	if !config.RequireLock {
		t.Error("Expected require_lock to be true")
	}
}

// TestVendorLock_Parse tests parsing of templar.lock file
func TestVendorLock_Parse(t *testing.T) {
	lockYAML := `
version: 1
sources:
  goapplib:
    url: github.com/panyam/goapplib
    ref: v1.2.0
    resolved_commit: abc123def456789
    fetched_at: "2024-12-08T10:30:00Z"
`

	var lock VendorLock
	err := yaml.Unmarshal([]byte(lockYAML), &lock)
	if err != nil {
		t.Fatalf("Failed to parse lock YAML: %v", err)
	}

	if lock.Version != 1 {
		t.Errorf("Expected version 1, got %d", lock.Version)
	}

	source, ok := lock.Sources["goapplib"]
	if !ok {
		t.Error("Expected 'goapplib' source in lock")
	} else {
		if source.ResolvedCommit != "abc123def456789" {
			t.Errorf("Expected resolved_commit 'abc123def456789', got '%s'", source.ResolvedCommit)
		}
	}
}

// TestSourceLoader_ResolveAtPrefix tests that @source paths are resolved correctly
func TestSourceLoader_ResolveAtPrefix(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "templar-vendor-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create directory structure simulating vendored files
	vendorDir := filepath.Join(tmpDir, "templar_modules", "github.com", "example", "uikit", "templates", "components")
	if err := os.MkdirAll(vendorDir, 0755); err != nil {
		t.Fatalf("Failed to create vendor dir: %v", err)
	}

	// Create a template in the vendored location
	cardContent := `{{ define "Card" }}<div class="card">{{ .Title }}</div>{{ end }}`
	if err := os.WriteFile(filepath.Join(vendorDir, "card.html"), []byte(cardContent), 0644); err != nil {
		t.Fatalf("Failed to write card.html: %v", err)
	}

	// Create local templates directory
	localTemplatesDir := filepath.Join(tmpDir, "templates")
	if err := os.MkdirAll(localTemplatesDir, 0755); err != nil {
		t.Fatalf("Failed to create local templates dir: %v", err)
	}

	// Create a page template that uses @uikit prefix
	pageContent := `{{# namespace "UI" "@uikit/components/card.html" #}}
{{ define "page" }}
{{ template "UI:Card" . }}
{{ end }}`
	if err := os.WriteFile(filepath.Join(localTemplatesDir, "page.html"), []byte(pageContent), 0644); err != nil {
		t.Fatalf("Failed to write page.html: %v", err)
	}

	// Create SourceLoader with config
	config := &VendorConfig{
		Sources: map[string]SourceConfig{
			"uikit": {
				URL:  "github.com/example/uikit",
				Path: "templates",
				Ref:  "v1.0.0",
			},
		},
		VendorDir:   filepath.Join(tmpDir, "templar_modules"),
		SearchPaths: []string{localTemplatesDir},
	}

	loader := NewSourceLoader(config)
	group := NewTemplateGroup()
	group.Loader = loader

	templates, err := group.Loader.Load("page.html", localTemplatesDir)
	if err != nil {
		t.Fatalf("Failed to load page.html: %v", err)
	}

	var buf bytes.Buffer
	err = group.RenderHtmlTemplate(&buf, templates[0], "page", map[string]any{"Title": "Hello"}, nil)
	if err != nil {
		t.Fatalf("Failed to render: %v", err)
	}

	result := buf.String()
	if !strings.Contains(result, "<div class=\"card\">Hello</div>") {
		t.Errorf("Expected card div, got: %s", result)
	}
}

// TestSourceLoader_LocalTemplatesTakePrecedence tests that local templates are found before vendored ones
func TestSourceLoader_LocalTemplatesTakePrecedence(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "templar-vendor-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create both local and vendored versions of the same template
	localTemplatesDir := filepath.Join(tmpDir, "templates")
	vendorDir := filepath.Join(tmpDir, "templar_modules", "github.com", "example", "lib")

	if err := os.MkdirAll(localTemplatesDir, 0755); err != nil {
		t.Fatalf("Failed to create local templates dir: %v", err)
	}
	if err := os.MkdirAll(vendorDir, 0755); err != nil {
		t.Fatalf("Failed to create vendor dir: %v", err)
	}

	// Local version
	localContent := `{{ define "shared" }}LOCAL{{ end }}`
	if err := os.WriteFile(filepath.Join(localTemplatesDir, "shared.html"), []byte(localContent), 0644); err != nil {
		t.Fatalf("Failed to write local shared.html: %v", err)
	}

	// Vendored version
	vendoredContent := `{{ define "shared" }}VENDORED{{ end }}`
	if err := os.WriteFile(filepath.Join(vendorDir, "shared.html"), []byte(vendoredContent), 0644); err != nil {
		t.Fatalf("Failed to write vendored shared.html: %v", err)
	}

	// Create page that uses non-@ path (should use local)
	localPageContent := `{{# include "shared.html" #}}
{{ define "page" }}{{ template "shared" . }}{{ end }}`
	if err := os.WriteFile(filepath.Join(localTemplatesDir, "local_page.html"), []byte(localPageContent), 0644); err != nil {
		t.Fatalf("Failed to write local_page.html: %v", err)
	}

	// Create page that uses @lib path (should use vendored)
	vendoredPageContent := `{{# include "@lib/shared.html" #}}
{{ define "page" }}{{ template "shared" . }}{{ end }}`
	if err := os.WriteFile(filepath.Join(localTemplatesDir, "vendored_page.html"), []byte(vendoredPageContent), 0644); err != nil {
		t.Fatalf("Failed to write vendored_page.html: %v", err)
	}

	config := &VendorConfig{
		Sources: map[string]SourceConfig{
			"lib": {
				URL: "github.com/example/lib",
				Ref: "v1.0.0",
			},
		},
		VendorDir:   filepath.Join(tmpDir, "templar_modules"),
		SearchPaths: []string{localTemplatesDir},
	}

	loader := NewSourceLoader(config)
	group := NewTemplateGroup()
	group.Loader = loader

	// Test 1: Non-@ path should load local version
	templates, err := group.Loader.Load("local_page.html", localTemplatesDir)
	if err != nil {
		t.Fatalf("Failed to load local_page.html: %v", err)
	}

	var buf bytes.Buffer
	err = group.RenderHtmlTemplate(&buf, templates[0], "page", nil, nil)
	if err != nil {
		t.Fatalf("Failed to render local page: %v", err)
	}

	result := buf.String()
	if !strings.Contains(result, "LOCAL") {
		t.Errorf("Expected LOCAL from local template, got: %s", result)
	}

	// Test 2: @ path should load vendored version
	templates2, err := group.Loader.Load("vendored_page.html", localTemplatesDir)
	if err != nil {
		t.Fatalf("Failed to load vendored_page.html: %v", err)
	}

	buf.Reset()
	err = group.RenderHtmlTemplate(&buf, templates2[0], "page", nil, nil)
	if err != nil {
		t.Fatalf("Failed to render vendored page: %v", err)
	}

	result2 := buf.String()
	if !strings.Contains(result2, "VENDORED") {
		t.Errorf("Expected VENDORED from vendored template, got: %s", result2)
	}
}

// TestSourceLoader_MissingSourceError tests that referencing undefined source gives clear error
func TestSourceLoader_MissingSourceError(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "templar-vendor-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	localTemplatesDir := filepath.Join(tmpDir, "templates")
	if err := os.MkdirAll(localTemplatesDir, 0755); err != nil {
		t.Fatalf("Failed to create local templates dir: %v", err)
	}

	config := &VendorConfig{
		Sources: map[string]SourceConfig{
			"uikit": {
				URL:  "github.com/example/uikit",
				Path: "templates",
				Ref:  "v1.0.0",
			},
		},
		VendorDir:   filepath.Join(tmpDir, "templar_modules"),
		SearchPaths: []string{localTemplatesDir},
	}

	loader := NewSourceLoader(config)

	// Try to load from undefined source
	_, err = loader.Load("@undefined/component.html", "")

	if err == nil {
		t.Error("Expected error for undefined source, but got none")
	}

	// Check error message mentions the source name
	if !strings.Contains(err.Error(), "undefined") {
		t.Errorf("Expected error to mention 'undefined', got: %v", err)
	}
}

// TestSourceLoader_CaseSensitiveSourceNames tests that source names are case-sensitive
func TestSourceLoader_CaseSensitiveSourceNames(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "templar-vendor-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create vendored files for the proper case
	vendorDir := filepath.Join(tmpDir, "templar_modules", "github.com", "example", "UIKit", "templates")
	if err := os.MkdirAll(vendorDir, 0755); err != nil {
		t.Fatalf("Failed to create vendor dir: %v", err)
	}

	cardContent := `{{ define "Card" }}<div>Card</div>{{ end }}`
	if err := os.WriteFile(filepath.Join(vendorDir, "card.html"), []byte(cardContent), 0644); err != nil {
		t.Fatalf("Failed to write card.html: %v", err)
	}

	config := &VendorConfig{
		Sources: map[string]SourceConfig{
			"UIKit": { // Note: specific casing
				URL:  "github.com/example/UIKit",
				Path: "templates",
				Ref:  "v1.0.0",
			},
		},
		VendorDir:   filepath.Join(tmpDir, "templar_modules"),
		SearchPaths: []string{},
	}

	loader := NewSourceLoader(config)

	// @UIKit should work (correct case)
	_, err = loader.Load("@UIKit/card.html", "")
	if err != nil {
		t.Errorf("Expected @UIKit to work, got error: %v", err)
	}

	// @uikit should fail (wrong case)
	_, err = loader.Load("@uikit/card.html", "")
	if err == nil {
		t.Error("Expected @uikit (lowercase) to fail, but it succeeded")
	}
}

// TestVendoredLoader_IntegrationWithNamespace tests that vendored templates work with namespace directive
func TestVendoredLoader_IntegrationWithNamespace(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "templar-vendor-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create vendored template structure
	vendorDir := filepath.Join(tmpDir, "templar_modules", "github.com", "example", "uikit", "templates", "components")
	if err := os.MkdirAll(vendorDir, 0755); err != nil {
		t.Fatalf("Failed to create vendor dir: %v", err)
	}

	// Create vendored templates with internal references
	cardGridContent := `{{ define "CardGrid" }}
<div class="grid">{{ range .Items }}{{ template "Card" . }}{{ end }}</div>
{{ end }}
{{ define "Card" }}
<div class="card">{{ template "CardContent" . }}</div>
{{ end }}
{{ define "CardContent" }}
<span>{{ .Name }}</span>
{{ end }}`
	if err := os.WriteFile(filepath.Join(vendorDir, "card.html"), []byte(cardGridContent), 0644); err != nil {
		t.Fatalf("Failed to write card.html: %v", err)
	}

	// Create local template that uses namespaced vendored template
	localTemplatesDir := filepath.Join(tmpDir, "templates")
	if err := os.MkdirAll(localTemplatesDir, 0755); err != nil {
		t.Fatalf("Failed to create local templates dir: %v", err)
	}

	pageContent := `{{# namespace "UI" "@uikit/components/card.html" #}}

{{ define "myCardContent" }}<strong>{{ .Name }}</strong>{{ end }}

{{# extend "UI:Card" "MyCard" "UI:CardContent" "myCardContent" #}}
{{# extend "UI:CardGrid" "MyCardGrid" "UI:Card" "MyCard" #}}

{{ define "page" }}
{{ template "MyCardGrid" . }}
{{ end }}`
	if err := os.WriteFile(filepath.Join(localTemplatesDir, "page.html"), []byte(pageContent), 0644); err != nil {
		t.Fatalf("Failed to write page.html: %v", err)
	}

	config := &VendorConfig{
		Sources: map[string]SourceConfig{
			"uikit": {
				URL:  "github.com/example/uikit",
				Path: "templates",
				Ref:  "v1.0.0",
			},
		},
		VendorDir:   filepath.Join(tmpDir, "templar_modules"),
		SearchPaths: []string{localTemplatesDir},
	}

	loader := NewSourceLoader(config)
	group := NewTemplateGroup()
	group.Loader = loader

	templates, err := group.Loader.Load("page.html", localTemplatesDir)
	if err != nil {
		t.Fatalf("Failed to load page.html: %v", err)
	}

	data := map[string]any{
		"Items": []map[string]any{
			{"Name": "Product A"},
			{"Name": "Product B"},
		},
	}

	var buf bytes.Buffer
	err = group.RenderHtmlTemplate(&buf, templates[0], "page", data, nil)
	if err != nil {
		t.Fatalf("Failed to render: %v", err)
	}

	result := buf.String()

	// Should use custom myCardContent (strong tags) not default (span tags)
	if !strings.Contains(result, "<strong>Product A</strong>") {
		t.Errorf("Expected custom card content with <strong>, got: %s", result)
	}
	if strings.Contains(result, "<span>Product A</span>") {
		t.Errorf("Should NOT have default span content, got: %s", result)
	}
}

// TestVendoredLoader_IntegrationWithExtend tests that vendored templates work with extend directive
func TestVendoredLoader_IntegrationWithExtend(t *testing.T) {
	// This test is covered by TestVendoredLoader_IntegrationWithNamespace which
	// already tests the extend directive with vendored templates
}

// TestSourceLoader_RelativePathsInVendoredTemplates tests that relative paths within vendored templates resolve correctly
func TestSourceLoader_RelativePathsInVendoredTemplates(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "templar-vendor-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create vendored template structure with relative includes
	vendorDir := filepath.Join(tmpDir, "templar_modules", "github.com", "example", "uikit", "templates")
	componentsDir := filepath.Join(vendorDir, "components")
	sharedDir := filepath.Join(vendorDir, "shared")
	if err := os.MkdirAll(componentsDir, 0755); err != nil {
		t.Fatalf("Failed to create components dir: %v", err)
	}
	if err := os.MkdirAll(sharedDir, 0755); err != nil {
		t.Fatalf("Failed to create shared dir: %v", err)
	}

	// Shared template
	sharedContent := `{{ define "icon" }}<i class="icon">★</i>{{ end }}`
	if err := os.WriteFile(filepath.Join(sharedDir, "icons.html"), []byte(sharedContent), 0644); err != nil {
		t.Fatalf("Failed to write icons.html: %v", err)
	}

	// Component that includes relative path to shared
	componentContent := `{{# include "../shared/icons.html" #}}
{{ define "button" }}<button>{{ template "icon" . }}Click</button>{{ end }}`
	if err := os.WriteFile(filepath.Join(componentsDir, "button.html"), []byte(componentContent), 0644); err != nil {
		t.Fatalf("Failed to write button.html: %v", err)
	}

	// Local template directory
	localTemplatesDir := filepath.Join(tmpDir, "templates")
	if err := os.MkdirAll(localTemplatesDir, 0755); err != nil {
		t.Fatalf("Failed to create local templates dir: %v", err)
	}

	// Page that uses the vendored button
	pageContent := `{{# include "@uikit/components/button.html" #}}
{{ define "page" }}{{ template "button" . }}{{ end }}`
	if err := os.WriteFile(filepath.Join(localTemplatesDir, "page.html"), []byte(pageContent), 0644); err != nil {
		t.Fatalf("Failed to write page.html: %v", err)
	}

	config := &VendorConfig{
		Sources: map[string]SourceConfig{
			"uikit": {
				URL:  "github.com/example/uikit",
				Path: "templates",
				Ref:  "v1.0.0",
			},
		},
		VendorDir:   filepath.Join(tmpDir, "templar_modules"),
		SearchPaths: []string{localTemplatesDir},
	}

	loader := NewSourceLoader(config)
	group := NewTemplateGroup()
	group.Loader = loader

	templates, err := group.Loader.Load("page.html", localTemplatesDir)
	if err != nil {
		t.Fatalf("Failed to load page.html: %v", err)
	}

	var buf bytes.Buffer
	err = group.RenderHtmlTemplate(&buf, templates[0], "page", nil, nil)
	if err != nil {
		t.Fatalf("Failed to render: %v", err)
	}

	result := buf.String()
	// Should have the icon from shared/icons.html
	if !strings.Contains(result, "<i class=\"icon\">★</i>") {
		t.Errorf("Expected icon from relative include, got: %s", result)
	}
	if !strings.Contains(result, "<button>") {
		t.Errorf("Expected button, got: %s", result)
	}
}

// TestLoadVendorConfig tests loading VendorConfig from a templar.yaml file
func TestLoadVendorConfig(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "templar-config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create templar.yaml
	configContent := `
sources:
  uikit:
    url: github.com/example/uikit
    path: templates
    ref: v1.0.0
  icons:
    url: github.com/example/icons
    ref: main

vendor_dir: ./templar_modules

search_paths:
  - ./templates
  - ./shared

require_lock: true
`
	configPath := filepath.Join(tmpDir, "templar.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write templar.yaml: %v", err)
	}

	config, err := LoadVendorConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Check sources
	if len(config.Sources) != 2 {
		t.Errorf("Expected 2 sources, got %d", len(config.Sources))
	}

	uikit, ok := config.Sources["uikit"]
	if !ok {
		t.Error("Expected 'uikit' source")
	} else {
		if uikit.URL != "github.com/example/uikit" {
			t.Errorf("Expected URL 'github.com/example/uikit', got '%s'", uikit.URL)
		}
		if uikit.Path != "templates" {
			t.Errorf("Expected path 'templates', got '%s'", uikit.Path)
		}
		if uikit.Ref != "v1.0.0" {
			t.Errorf("Expected ref 'v1.0.0', got '%s'", uikit.Ref)
		}
	}

	icons, ok := config.Sources["icons"]
	if !ok {
		t.Error("Expected 'icons' source")
	} else {
		if icons.Path != "" {
			t.Errorf("Expected empty path for icons, got '%s'", icons.Path)
		}
	}

	// Check vendor_dir
	if config.VendorDir != "./templar_modules" {
		t.Errorf("Expected vendor_dir './templar_modules', got '%s'", config.VendorDir)
	}

	// Check search_paths
	if len(config.SearchPaths) != 2 {
		t.Errorf("Expected 2 search paths, got %d", len(config.SearchPaths))
	}

	// Check require_lock
	if !config.RequireLock {
		t.Error("Expected require_lock to be true")
	}
}

// TestLoadVendorConfig_Defaults tests that missing fields get sensible defaults
func TestLoadVendorConfig_Defaults(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "templar-config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Minimal config - just sources
	configContent := `
sources:
  uikit:
    url: github.com/example/uikit
    ref: v1.0.0
`
	configPath := filepath.Join(tmpDir, "templar.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write templar.yaml: %v", err)
	}

	config, err := LoadVendorConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Should have default vendor_dir
	if config.VendorDir != "./templar_modules" {
		t.Errorf("Expected default vendor_dir './templar_modules', got '%s'", config.VendorDir)
	}

	// Should have default search_paths
	if len(config.SearchPaths) != 2 {
		t.Errorf("Expected 2 default search paths, got %d", len(config.SearchPaths))
	}
}

// TestLoadVendorConfig_NotFound tests error when config file doesn't exist
func TestLoadVendorConfig_NotFound(t *testing.T) {
	_, err := LoadVendorConfig("/nonexistent/templar.yaml")
	if err == nil {
		t.Error("Expected error for missing config file")
	}
}

// TestFindVendorConfig tests finding templar.yaml in current or parent directories
func TestFindVendorConfig(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "templar-config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create nested directory structure
	subDir := filepath.Join(tmpDir, "sub", "project")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirs: %v", err)
	}

	// Create templar.yaml in root
	configContent := `
sources:
  uikit:
    url: github.com/example/uikit
    ref: v1.0.0
`
	if err := os.WriteFile(filepath.Join(tmpDir, "templar.yaml"), []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write templar.yaml: %v", err)
	}

	// Should find config from subdirectory
	foundPath, err := FindVendorConfig(subDir)
	if err != nil {
		t.Fatalf("Failed to find config: %v", err)
	}

	expectedPath := filepath.Join(tmpDir, "templar.yaml")
	if foundPath != expectedPath {
		t.Errorf("Expected to find '%s', got '%s'", expectedPath, foundPath)
	}
}

// TestNewSourceLoaderFromConfig tests creating a SourceLoader from a config file
func TestNewSourceLoaderFromConfig(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "templar-config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create directory structure
	vendorDir := filepath.Join(tmpDir, "templar_modules", "github.com", "example", "uikit", "templates")
	if err := os.MkdirAll(vendorDir, 0755); err != nil {
		t.Fatalf("Failed to create vendor dir: %v", err)
	}

	templatesDir := filepath.Join(tmpDir, "templates")
	if err := os.MkdirAll(templatesDir, 0755); err != nil {
		t.Fatalf("Failed to create templates dir: %v", err)
	}

	// Create vendored template
	cardContent := `{{ define "Card" }}<div class="card">{{ .Title }}</div>{{ end }}`
	if err := os.WriteFile(filepath.Join(vendorDir, "card.html"), []byte(cardContent), 0644); err != nil {
		t.Fatalf("Failed to write card.html: %v", err)
	}

	// Create page template
	pageContent := `{{# namespace "UI" "@uikit/templates/card.html" #}}
{{ define "page" }}{{ template "UI:Card" . }}{{ end }}`
	if err := os.WriteFile(filepath.Join(templatesDir, "page.html"), []byte(pageContent), 0644); err != nil {
		t.Fatalf("Failed to write page.html: %v", err)
	}

	// Create templar.yaml
	configContent := `
sources:
  uikit:
    url: github.com/example/uikit
    ref: v1.0.0

vendor_dir: ./templar_modules

search_paths:
  - ./templates
`
	configPath := filepath.Join(tmpDir, "templar.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write templar.yaml: %v", err)
	}

	// Create loader from config
	loader, err := NewSourceLoaderFromConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to create loader from config: %v", err)
	}

	// Use the loader
	group := NewTemplateGroup()
	group.Loader = loader

	templates, err := group.Loader.Load("page.html", "")
	if err != nil {
		t.Fatalf("Failed to load page.html: %v", err)
	}

	var buf bytes.Buffer
	err = group.RenderHtmlTemplate(&buf, templates[0], "page", map[string]any{"Title": "Test"}, nil)
	if err != nil {
		t.Fatalf("Failed to render: %v", err)
	}

	result := buf.String()
	if !strings.Contains(result, "<div class=\"card\">Test</div>") {
		t.Errorf("Expected card div, got: %s", result)
	}
}

// TestNewSourceLoaderFromDir tests finding config and creating loader from a subdirectory
func TestNewSourceLoaderFromDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "templar-config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create nested directory structure
	subDir := filepath.Join(tmpDir, "src", "pages")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirs: %v", err)
	}

	// Create templar.yaml in root
	configContent := `
sources:
  uikit:
    url: github.com/example/uikit
    ref: v1.0.0
`
	if err := os.WriteFile(filepath.Join(tmpDir, "templar.yaml"), []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write templar.yaml: %v", err)
	}

	// Create loader from subdirectory
	loader, err := NewSourceLoaderFromDir(subDir)
	if err != nil {
		t.Fatalf("Failed to create loader from dir: %v", err)
	}

	// Verify the config was found and loaded
	if loader.config == nil {
		t.Error("Expected config to be loaded")
	}

	if _, ok := loader.config.Sources["uikit"]; !ok {
		t.Error("Expected 'uikit' source to be in config")
	}
}

// TestVendorLock_VerifyIntegrity tests that lock file can verify vendored files haven't changed
func TestVendorLock_VerifyIntegrity(t *testing.T) {
	t.Skip("VendorLock verification not yet implemented")

	// TODO: Test that we can detect when local vendored files don't match lock file
}

// TestSourceLoader_WithFileSystemLoaderFallback tests that SourceLoader works with existing FileSystemLoader
func TestSourceLoader_WithFileSystemLoaderFallback(t *testing.T) {
	// This test uses the existing FileSystemLoader infrastructure
	// to verify that non-@ paths still work as expected

	tmpDir, err := os.MkdirTemp("", "templar-vendor-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a local template
	localTemplatesDir := filepath.Join(tmpDir, "templates")
	if err := os.MkdirAll(localTemplatesDir, 0755); err != nil {
		t.Fatalf("Failed to create local templates dir: %v", err)
	}

	componentContent := `{{ define "button" }}<button>Click</button>{{ end }}`
	if err := os.WriteFile(filepath.Join(localTemplatesDir, "button.html"), []byte(componentContent), 0644); err != nil {
		t.Fatalf("Failed to write button.html: %v", err)
	}

	pageContent := `{{# include "button.html" #}}
{{ define "page" }}{{ template "button" . }}{{ end }}`
	if err := os.WriteFile(filepath.Join(localTemplatesDir, "page.html"), []byte(pageContent), 0644); err != nil {
		t.Fatalf("Failed to write page.html: %v", err)
	}

	// Use existing FileSystemLoader
	group := NewTemplateGroup()
	group.Loader = &FileSystemLoader{
		Folders:    []string{localTemplatesDir},
		Extensions: []string{".html"},
	}

	templates, err := group.Loader.Load("page.html", "")
	if err != nil {
		t.Fatalf("Failed to load page.html: %v", err)
	}

	var buf bytes.Buffer
	err = group.RenderHtmlTemplate(&buf, templates[0], "page", nil, nil)
	if err != nil {
		t.Fatalf("Failed to render: %v", err)
	}

	result := buf.String()
	if !strings.Contains(result, "<button>Click</button>") {
		t.Errorf("Expected button, got: %s", result)
	}
}
