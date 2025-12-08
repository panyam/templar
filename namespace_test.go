package templar

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNamespace_BasicNamespacing(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "templar-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a component template
	componentContent := `{{ define "button" }}<button>{{ .Text }}</button>{{ end }}
{{ define "icon" }}<i class="icon"></i>{{ end }}`
	if err := os.WriteFile(filepath.Join(tmpDir, "component.html"), []byte(componentContent), 0644); err != nil {
		t.Fatalf("Failed to write component.html: %v", err)
	}

	// Create a page template that includes the component with namespace
	pageContent := `{{# namespace "UI" "component.html" #}}
{{ define "page" }}
<div>{{ template "UI:button" . }}</div>
{{ end }}`
	if err := os.WriteFile(filepath.Join(tmpDir, "page.html"), []byte(pageContent), 0644); err != nil {
		t.Fatalf("Failed to write page.html: %v", err)
	}

	group := NewTemplateGroup()
	group.Loader = &FileSystemLoader{
		Folders:    []string{tmpDir},
		Extensions: []string{".html"},
	}

	templates, err := group.Loader.Load("page.html", "")
	if err != nil {
		t.Fatalf("Failed to load page.html: %v", err)
	}

	var buf bytes.Buffer
	err = group.RenderHtmlTemplate(&buf, templates[0], "page", map[string]any{"Text": "Click Me"}, nil)
	if err != nil {
		t.Fatalf("Failed to render: %v", err)
	}

	result := buf.String()
	if !strings.Contains(result, "<button>Click Me</button>") {
		t.Errorf("Expected button with text, got: %s", result)
	}
}

func TestNamespace_CrossNamespaceReference(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "templar-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a shared utility template (no namespace)
	sharedContent := `{{ define "formatDate" }}2024-01-01{{ end }}`
	if err := os.WriteFile(filepath.Join(tmpDir, "shared.html"), []byte(sharedContent), 0644); err != nil {
		t.Fatalf("Failed to write shared.html: %v", err)
	}

	// Create a component that uses the shared utility with :: syntax
	componentContent := `{{ define "card" }}<div class="card">Date: {{ template "::formatDate" . }}</div>{{ end }}`
	if err := os.WriteFile(filepath.Join(tmpDir, "component.html"), []byte(componentContent), 0644); err != nil {
		t.Fatalf("Failed to write component.html: %v", err)
	}

	// Create a page that includes shared normally and component with namespace
	pageContent := `{{# include "shared.html" #}}
{{# namespace "Cards" "component.html" #}}
{{ define "page" }}{{ template "Cards:card" . }}{{ end }}`
	if err := os.WriteFile(filepath.Join(tmpDir, "page.html"), []byte(pageContent), 0644); err != nil {
		t.Fatalf("Failed to write page.html: %v", err)
	}

	group := NewTemplateGroup()
	group.Loader = &FileSystemLoader{
		Folders:    []string{tmpDir},
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
	if !strings.Contains(result, "Date: 2024-01-01") {
		t.Errorf("Expected date from global template, got: %s", result)
	}
}

func TestNamespace_DiamondIncludes(t *testing.T) {
	// Test the diamond problem:
	//      Page
	//     /    \
	//   LibA   LibB  (both include Shared with different namespaces)
	//     \    /
	//     Shared

	tmpDir, err := os.MkdirTemp("", "templar-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Shared component
	sharedContent := `{{ define "widget" }}[WIDGET]{{ end }}`
	if err := os.WriteFile(filepath.Join(tmpDir, "shared.html"), []byte(sharedContent), 0644); err != nil {
		t.Fatalf("Failed to write shared.html: %v", err)
	}

	// LibA includes shared as "A"
	libAContent := `{{# namespace "A" "shared.html" #}}
{{ define "libA" }}LibA uses {{ template "A:widget" . }}{{ end }}`
	if err := os.WriteFile(filepath.Join(tmpDir, "libA.html"), []byte(libAContent), 0644); err != nil {
		t.Fatalf("Failed to write libA.html: %v", err)
	}

	// LibB includes shared as "B"
	libBContent := `{{# namespace "B" "shared.html" #}}
{{ define "libB" }}LibB uses {{ template "B:widget" . }}{{ end }}`
	if err := os.WriteFile(filepath.Join(tmpDir, "libB.html"), []byte(libBContent), 0644); err != nil {
		t.Fatalf("Failed to write libB.html: %v", err)
	}

	// Page includes both libs
	pageContent := `{{# include "libA.html" #}}
{{# include "libB.html" #}}
{{ define "page" }}{{ template "libA" . }} AND {{ template "libB" . }}{{ end }}`
	if err := os.WriteFile(filepath.Join(tmpDir, "page.html"), []byte(pageContent), 0644); err != nil {
		t.Fatalf("Failed to write page.html: %v", err)
	}

	group := NewTemplateGroup()
	group.Loader = &FileSystemLoader{
		Folders:    []string{tmpDir},
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
	if !strings.Contains(result, "LibA uses [WIDGET]") {
		t.Errorf("Expected LibA widget, got: %s", result)
	}
	if !strings.Contains(result, "LibB uses [WIDGET]") {
		t.Errorf("Expected LibB widget, got: %s", result)
	}
}

func TestNamespace_EmptyNamespaceError(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "templar-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	componentContent := `{{ define "button" }}<button/>{{ end }}`
	if err := os.WriteFile(filepath.Join(tmpDir, "component.html"), []byte(componentContent), 0644); err != nil {
		t.Fatalf("Failed to write component.html: %v", err)
	}

	// Try to include with empty namespace - should error
	pageContent := `{{# namespace "" "component.html" #}}
{{ define "page" }}test{{ end }}`
	if err := os.WriteFile(filepath.Join(tmpDir, "page.html"), []byte(pageContent), 0644); err != nil {
		t.Fatalf("Failed to write page.html: %v", err)
	}

	group := NewTemplateGroup()
	group.Loader = &FileSystemLoader{
		Folders:    []string{tmpDir},
		Extensions: []string{".html"},
	}

	templates, err := group.Loader.Load("page.html", "")
	if err != nil {
		t.Fatalf("Failed to load page.html: %v", err)
	}

	var buf bytes.Buffer
	err = group.RenderHtmlTemplate(&buf, templates[0], "page", nil, nil)
	if err == nil {
		t.Error("Expected error for empty namespace, but got none")
	}
}

func TestNamespace_TreeShaking(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "templar-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a file with many templates, only some will be used
	componentContent := `{{ define "used1" }}USED1{{ end }}
{{ define "used2" }}USED2 calls {{ template "used3" . }}{{ end }}
{{ define "used3" }}USED3{{ end }}
{{ define "unused1" }}UNUSED1{{ end }}
{{ define "unused2" }}UNUSED2{{ end }}`
	if err := os.WriteFile(filepath.Join(tmpDir, "components.html"), []byte(componentContent), 0644); err != nil {
		t.Fatalf("Failed to write components.html: %v", err)
	}

	// Only include used1 and used2 (used3 should be included transitively)
	pageContent := `{{# namespace "C" "components.html" "used1" "used2" #}}
{{ define "page" }}{{ template "C:used1" . }} {{ template "C:used2" . }}{{ end }}`
	if err := os.WriteFile(filepath.Join(tmpDir, "page.html"), []byte(pageContent), 0644); err != nil {
		t.Fatalf("Failed to write page.html: %v", err)
	}

	group := NewTemplateGroup()
	group.Loader = &FileSystemLoader{
		Folders:    []string{tmpDir},
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
	if !strings.Contains(result, "USED1") {
		t.Errorf("Expected USED1, got: %s", result)
	}
	if !strings.Contains(result, "USED2 calls USED3") {
		t.Errorf("Expected USED2 with USED3, got: %s", result)
	}
}

func TestExtend_BasicExtension(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "templar-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a base layout
	baseContent := `{{ define "layout" }}
<html>
<head>{{ template "title" . }}</head>
<body>{{ template "content" . }}</body>
</html>
{{ end }}
{{ define "title" }}<title>Default Title</title>{{ end }}
{{ define "content" }}<p>Default content</p>{{ end }}`
	if err := os.WriteFile(filepath.Join(tmpDir, "base.html"), []byte(baseContent), 0644); err != nil {
		t.Fatalf("Failed to write base.html: %v", err)
	}

	// Create a page that extends the base
	pageContent := `{{# namespace "Base" "base.html" #}}
{{# extend "Base:layout" "MyLayout" "Base:title" "myTitle" "Base:content" "myContent" #}}

{{ define "myTitle" }}<title>My Custom Page</title>{{ end }}
{{ define "myContent" }}<main>Hello World!</main>{{ end }}

{{ template "MyLayout" . }}`
	if err := os.WriteFile(filepath.Join(tmpDir, "page.html"), []byte(pageContent), 0644); err != nil {
		t.Fatalf("Failed to write page.html: %v", err)
	}

	group := NewTemplateGroup()
	group.Loader = &FileSystemLoader{
		Folders:    []string{tmpDir},
		Extensions: []string{".html"},
	}

	templates, err := group.Loader.Load("page.html", "")
	if err != nil {
		t.Fatalf("Failed to load page.html: %v", err)
	}

	var buf bytes.Buffer
	err = group.RenderHtmlTemplate(&buf, templates[0], "", nil, nil)
	if err != nil {
		t.Fatalf("Failed to render: %v", err)
	}

	result := buf.String()

	if !strings.Contains(result, "<title>My Custom Page</title>") {
		t.Errorf("Expected custom title, got: %s", result)
	}
	if !strings.Contains(result, "<main>Hello World!</main>") {
		t.Errorf("Expected custom content, got: %s", result)
	}
	if !strings.Contains(result, "<html>") {
		t.Errorf("Expected HTML from base, got: %s", result)
	}
}

func TestExtend_PartialOverride(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "templar-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Base with multiple blocks
	baseContent := `{{ define "layout" }}
<header>{{ template "header" . }}</header>
<main>{{ template "content" . }}</main>
<footer>{{ template "footer" . }}</footer>
{{ end }}
{{ define "header" }}Default Header{{ end }}
{{ define "content" }}Default Content{{ end }}
{{ define "footer" }}Default Footer{{ end }}`
	if err := os.WriteFile(filepath.Join(tmpDir, "base.html"), []byte(baseContent), 0644); err != nil {
		t.Fatalf("Failed to write base.html: %v", err)
	}

	// Page that only overrides content (header and footer use defaults from base)
	pageContent := `{{# namespace "Base" "base.html" #}}
{{# extend "Base:layout" "MyLayout" "Base:content" "myContent" #}}

{{ define "myContent" }}Custom Content Only{{ end }}

{{ template "MyLayout" . }}`
	if err := os.WriteFile(filepath.Join(tmpDir, "page.html"), []byte(pageContent), 0644); err != nil {
		t.Fatalf("Failed to write page.html: %v", err)
	}

	group := NewTemplateGroup()
	group.Loader = &FileSystemLoader{
		Folders:    []string{tmpDir},
		Extensions: []string{".html"},
	}

	templates, err := group.Loader.Load("page.html", "")
	if err != nil {
		t.Fatalf("Failed to load page.html: %v", err)
	}

	var buf bytes.Buffer
	err = group.RenderHtmlTemplate(&buf, templates[0], "", nil, nil)
	if err != nil {
		t.Fatalf("Failed to render: %v", err)
	}

	result := buf.String()

	// Should have default header from base (via Base:header)
	if !strings.Contains(result, "Default Header") {
		t.Errorf("Expected default header, got: %s", result)
	}
	// Should have custom content
	if !strings.Contains(result, "Custom Content Only") {
		t.Errorf("Expected custom content, got: %s", result)
	}
	// Should have default footer from base
	if !strings.Contains(result, "Default Footer") {
		t.Errorf("Expected default footer, got: %s", result)
	}
}

func TestInclude_SelectiveInclude(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "templar-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// File with multiple templates
	componentContent := `{{ define "button" }}<button>Click</button>{{ end }}
{{ define "input" }}<input/>{{ end }}
{{ define "select" }}<select></select>{{ end }}`
	if err := os.WriteFile(filepath.Join(tmpDir, "forms.html"), []byte(componentContent), 0644); err != nil {
		t.Fatalf("Failed to write forms.html: %v", err)
	}

	// Only include button
	pageContent := `{{# include "forms.html" "button" #}}
{{ define "page" }}{{ template "button" . }}{{ end }}`
	if err := os.WriteFile(filepath.Join(tmpDir, "page.html"), []byte(pageContent), 0644); err != nil {
		t.Fatalf("Failed to write page.html: %v", err)
	}

	group := NewTemplateGroup()
	group.Loader = &FileSystemLoader{
		Folders:    []string{tmpDir},
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
