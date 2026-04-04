package templar

import (
	"bytes"
	"strings"
	"testing"
)

// loadAndRender creates a MemFS from the given files, loads the entry template,
// and renders it. Returns the rendered output string.
func loadAndRender(t *testing.T, files map[string]string, entry, templateName string, data any) string {
	t.Helper()
	mfs := NewMemFS()
	for name, content := range files {
		mfs.SetFile(name, []byte(content))
	}

	group := NewTemplateGroup()
	group.Loader = &FileSystemLoader{
		Folders:    []FSFolder{{FS: mfs, Path: "."}},
		Extensions: []string{"html"},
	}

	templates, err := group.Loader.Load(entry, "")
	if err != nil {
		t.Fatalf("Failed to load %s: %v", entry, err)
	}

	var buf bytes.Buffer
	err = group.RenderHtmlTemplate(&buf, templates[0], templateName, data, nil)
	if err != nil {
		t.Fatalf("Failed to render: %v", err)
	}
	return buf.String()
}

func TestNamespace_BasicNamespacing(t *testing.T) {
	result := loadAndRender(t, map[string]string{
		"component.html": `{{ define "button" }}<button>{{ .Text }}</button>{{ end }}
{{ define "icon" }}<i class="icon"></i>{{ end }}`,
		"page.html": `{{# namespace "UI" "component.html" #}}
{{ define "page" }}
<div>{{ template "UI:button" . }}</div>
{{ end }}`,
	}, "page.html", "page", map[string]any{"Text": "Click Me"})

	if !strings.Contains(result, "<button>Click Me</button>") {
		t.Errorf("Expected button with text, got: %s", result)
	}
}

func TestNamespace_CrossNamespaceReference(t *testing.T) {
	result := loadAndRender(t, map[string]string{
		"shared.html":    `{{ define "formatDate" }}2024-01-01{{ end }}`,
		"component.html": `{{ define "card" }}<div class="card">Date: {{ template "::formatDate" . }}</div>{{ end }}`,
		"page.html": `{{# include "shared.html" #}}
{{# namespace "Cards" "component.html" #}}
{{ define "page" }}{{ template "Cards:card" . }}{{ end }}`,
	}, "page.html", "page", nil)

	if !strings.Contains(result, "Date: 2024-01-01") {
		t.Errorf("Expected date from global template, got: %s", result)
	}
}

func TestNamespace_DiamondIncludes(t *testing.T) {
	result := loadAndRender(t, map[string]string{
		"shared.html": `{{ define "widget" }}[WIDGET]{{ end }}`,
		"libA.html": `{{# namespace "A" "shared.html" #}}
{{ define "libA" }}LibA uses {{ template "A:widget" . }}{{ end }}`,
		"libB.html": `{{# namespace "B" "shared.html" #}}
{{ define "libB" }}LibB uses {{ template "B:widget" . }}{{ end }}`,
		"page.html": `{{# include "libA.html" #}}
{{# include "libB.html" #}}
{{ define "page" }}{{ template "libA" . }} AND {{ template "libB" . }}{{ end }}`,
	}, "page.html", "page", nil)

	if !strings.Contains(result, "LibA uses [WIDGET]") {
		t.Errorf("Expected LibA widget, got: %s", result)
	}
	if !strings.Contains(result, "LibB uses [WIDGET]") {
		t.Errorf("Expected LibB widget, got: %s", result)
	}
}

func TestNamespace_EmptyNamespaceError(t *testing.T) {
	mfs := NewMemFS()
	mfs.SetFile("component.html", []byte(`{{ define "button" }}<button/>{{ end }}`))
	mfs.SetFile("page.html", []byte(`{{# namespace "" "component.html" #}}
{{ define "page" }}test{{ end }}`))

	group := NewTemplateGroup()
	group.Loader = &FileSystemLoader{
		Folders:    []FSFolder{{FS: mfs, Path: "."}},
		Extensions: []string{"html"},
	}

	templates, err := group.Loader.Load("page.html", "")
	if err != nil {
		t.Fatalf("Failed to load: %v", err)
	}

	var buf bytes.Buffer
	err = group.RenderHtmlTemplate(&buf, templates[0], "page", nil, nil)
	if err == nil {
		t.Error("Expected error for empty namespace, but got none")
	}
}

func TestNamespace_TreeShaking(t *testing.T) {
	result := loadAndRender(t, map[string]string{
		"components.html": `{{ define "used1" }}USED1{{ end }}
{{ define "used2" }}USED2 calls {{ template "used3" . }}{{ end }}
{{ define "used3" }}USED3{{ end }}
{{ define "unused1" }}UNUSED1{{ end }}
{{ define "unused2" }}UNUSED2{{ end }}`,
		"page.html": `{{# namespace "C" "components.html" "used1" "used2" #}}
{{ define "page" }}{{ template "C:used1" . }} {{ template "C:used2" . }}{{ end }}`,
	}, "page.html", "page", nil)

	if !strings.Contains(result, "USED1") {
		t.Errorf("Expected USED1, got: %s", result)
	}
	if !strings.Contains(result, "USED2 calls USED3") {
		t.Errorf("Expected USED2 with USED3, got: %s", result)
	}
}

func TestExtend_BasicExtension(t *testing.T) {
	result := loadAndRender(t, map[string]string{
		"base.html": `{{ define "layout" }}
<html>
<head>{{ template "title" . }}</head>
<body>{{ template "content" . }}</body>
</html>
{{ end }}
{{ define "title" }}<title>Default Title</title>{{ end }}
{{ define "content" }}<p>Default content</p>{{ end }}`,
		"page.html": `{{# namespace "Base" "base.html" #}}
{{# extend "Base:layout" "MyLayout" "Base:title" "myTitle" "Base:content" "myContent" #}}

{{ define "myTitle" }}<title>My Custom Page</title>{{ end }}
{{ define "myContent" }}<main>Hello World!</main>{{ end }}

{{ template "MyLayout" . }}`,
	}, "page.html", "", nil)

	if !strings.Contains(result, "<title>My Custom Page</title>") {
		t.Errorf("Expected custom title, got: %s", result)
	}
	if !strings.Contains(result, "<main>Hello World!</main>") {
		t.Errorf("Expected custom content, got: %s", result)
	}
}

func TestExtend_PartialOverride(t *testing.T) {
	result := loadAndRender(t, map[string]string{
		"base.html": `{{ define "layout" }}
<header>{{ template "header" . }}</header>
<main>{{ template "content" . }}</main>
<footer>{{ template "footer" . }}</footer>
{{ end }}
{{ define "header" }}Default Header{{ end }}
{{ define "content" }}Default Content{{ end }}
{{ define "footer" }}Default Footer{{ end }}`,
		"page.html": `{{# namespace "Base" "base.html" #}}
{{# extend "Base:layout" "MyLayout" "Base:content" "myContent" #}}

{{ define "myContent" }}Custom Content Only{{ end }}

{{ template "MyLayout" . }}`,
	}, "page.html", "", nil)

	if !strings.Contains(result, "Default Header") {
		t.Errorf("Expected default header, got: %s", result)
	}
	if !strings.Contains(result, "Custom Content Only") {
		t.Errorf("Expected custom content, got: %s", result)
	}
	if !strings.Contains(result, "Default Footer") {
		t.Errorf("Expected default footer, got: %s", result)
	}
}

func TestInclude_SelectiveInclude(t *testing.T) {
	result := loadAndRender(t, map[string]string{
		"forms.html": `{{ define "button" }}<button>Click</button>{{ end }}
{{ define "input" }}<input/>{{ end }}
{{ define "select" }}<select></select>{{ end }}`,
		"page.html": `{{# include "forms.html" "button" #}}
{{ define "page" }}{{ template "button" . }}{{ end }}`,
	}, "page.html", "page", nil)

	if !strings.Contains(result, "<button>Click</button>") {
		t.Errorf("Expected button, got: %s", result)
	}
}
