package templar

import (
	"testing"
	"text/template"
	"text/template/parse"
)

func TestTransformName(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		namespace string
		expected  string
	}{
		{
			name:      "local reference gets namespaced",
			input:     "button",
			namespace: "UI",
			expected:  "UI:button",
		},
		{
			name:      "explicit cross-namespace unchanged",
			input:     "Other:widget",
			namespace: "UI",
			expected:  "Other:widget",
		},
		{
			name:      "global reference strips ::",
			input:     "::formatDate",
			namespace: "UI",
			expected:  "formatDate",
		},
		{
			name:      "empty name",
			input:     "",
			namespace: "UI",
			expected:  "UI:",
		},
		{
			name:      "name with path-like structure",
			input:     "components/button",
			namespace: "UI",
			expected:  "UI:components/button",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TransformName(tt.input, tt.namespace)
			if result != tt.expected {
				t.Errorf("TransformName(%q, %q) = %q, want %q",
					tt.input, tt.namespace, result, tt.expected)
			}
		})
	}
}

func TestApplyNamespaceToTree(t *testing.T) {
	tests := []struct {
		name      string
		source    string
		namespace string
		wantNames []string // expected template names after transformation
	}{
		{
			name:      "single template call",
			source:    `{{ template "foo" . }}`,
			namespace: "NS",
			wantNames: []string{"NS:foo"},
		},
		{
			name:      "multiple template calls",
			source:    `{{ template "foo" . }}{{ template "bar" . }}`,
			namespace: "NS",
			wantNames: []string{"NS:foo", "NS:bar"},
		},
		{
			name:      "cross-namespace reference unchanged",
			source:    `{{ template "Other:widget" . }}`,
			namespace: "NS",
			wantNames: []string{"Other:widget"},
		},
		{
			name:      "global reference strips ::",
			source:    `{{ template "::global" . }}`,
			namespace: "NS",
			wantNames: []string{"global"},
		},
		{
			name:      "mixed references",
			source:    `{{ template "local" . }}{{ template "Other:cross" . }}{{ template "::global" . }}`,
			namespace: "NS",
			wantNames: []string{"NS:local", "Other:cross", "global"},
		},
		{
			name:      "template inside if",
			source:    `{{ if .Cond }}{{ template "inner" . }}{{ end }}`,
			namespace: "NS",
			wantNames: []string{"NS:inner"},
		},
		{
			name:      "template inside range",
			source:    `{{ range .Items }}{{ template "item" . }}{{ end }}`,
			namespace: "NS",
			wantNames: []string{"NS:item"},
		},
		{
			name:      "template in else branch",
			source:    `{{ if .Cond }}{{ template "then" . }}{{ else }}{{ template "else" . }}{{ end }}`,
			namespace: "NS",
			wantNames: []string{"NS:then", "NS:else"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the template
			tmpl, err := template.New("test").Parse(tt.source)
			if err != nil {
				t.Fatalf("Failed to parse template: %v", err)
			}

			// Apply namespace transformation
			ApplyNamespaceToTree(tmpl.Tree, tt.namespace)

			// Collect the transformed names
			names := CollectTemplateNames(tmpl.Tree)

			// Compare
			if len(names) != len(tt.wantNames) {
				t.Errorf("Got %d names, want %d", len(names), len(tt.wantNames))
				t.Errorf("Got: %v", names)
				t.Errorf("Want: %v", tt.wantNames)
				return
			}

			for i, got := range names {
				if got != tt.wantNames[i] {
					t.Errorf("names[%d] = %q, want %q", i, got, tt.wantNames[i])
				}
			}
		})
	}
}

func TestCopyTreeWithNamespace(t *testing.T) {
	source := `{{ define "foo" }}{{ template "bar" . }}{{ end }}`

	// Parse the template
	tmpl, err := template.New("original").Parse(source)
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}

	// Get the "foo" template's tree
	fooTmpl := tmpl.Lookup("foo")
	if fooTmpl == nil {
		t.Fatal("Could not find 'foo' template")
	}

	originalTree := fooTmpl.Tree
	originalName := originalTree.Name

	// Copy with namespace
	copiedTree := CopyTreeWithNamespace(originalTree, "NS")

	// Verify original is unchanged
	if originalTree.Name != originalName {
		t.Errorf("Original tree name changed from %q to %q", originalName, originalTree.Name)
	}

	originalNames := CollectTemplateNames(originalTree)
	if len(originalNames) > 0 && originalNames[0] != "bar" {
		t.Errorf("Original tree's template call changed to %q", originalNames[0])
	}

	// Verify copy has namespace applied
	if copiedTree.Name != "NS:foo" {
		t.Errorf("Copied tree name = %q, want %q", copiedTree.Name, "NS:foo")
	}

	copiedNames := CollectTemplateNames(copiedTree)
	if len(copiedNames) != 1 || copiedNames[0] != "NS:bar" {
		t.Errorf("Copied tree template calls = %v, want [NS:bar]", copiedNames)
	}
}

func TestWalkParseTree_NilHandling(t *testing.T) {
	// Should not panic on nil inputs
	WalkParseTree(nil, func(n *parse.TemplateNode) {
		t.Error("Visitor should not be called for nil tree")
	})

	ApplyNamespaceToTree(nil, "NS")
	// No panic = success

	result := CopyTreeWithNamespace(nil, "NS")
	if result != nil {
		t.Errorf("CopyTreeWithNamespace(nil) = %v, want nil", result)
	}

	names := CollectTemplateNames(nil)
	if names != nil {
		t.Errorf("CollectTemplateNames(nil) = %v, want nil", names)
	}
}

func TestIsLocalReference(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"plain name", "button", true},
		{"namespaced", "NS:button", false},
		{"global", "::button", false},
		{"empty", "", true},
		{"path-like", "components/button", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsLocalReference(tt.input)
			if result != tt.expected {
				t.Errorf("IsLocalReference(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestCollectLocalReferences(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		expected []string
	}{
		{
			name:     "all local",
			source:   `{{ template "foo" . }}{{ template "bar" . }}`,
			expected: []string{"foo", "bar"},
		},
		{
			name:     "mixed local and namespaced",
			source:   `{{ template "foo" . }}{{ template "NS:bar" . }}`,
			expected: []string{"foo"},
		},
		{
			name:     "mixed local and global",
			source:   `{{ template "foo" . }}{{ template "::bar" . }}`,
			expected: []string{"foo"},
		},
		{
			name:     "no local references",
			source:   `{{ template "NS:foo" . }}{{ template "::bar" . }}`,
			expected: []string{},
		},
		{
			name:     "deduplicated",
			source:   `{{ template "foo" . }}{{ template "foo" . }}`,
			expected: []string{"foo"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl, err := template.New("test").Parse(tt.source)
			if err != nil {
				t.Fatalf("Failed to parse template: %v", err)
			}

			result := CollectLocalReferences(tmpl.Tree)

			// Convert to set for comparison (order doesn't matter)
			resultSet := make(map[string]bool)
			for _, name := range result {
				resultSet[name] = true
			}
			expectedSet := make(map[string]bool)
			for _, name := range tt.expected {
				expectedSet[name] = true
			}

			if len(resultSet) != len(expectedSet) {
				t.Errorf("CollectLocalReferences() = %v, want %v", result, tt.expected)
				return
			}
			for name := range expectedSet {
				if !resultSet[name] {
					t.Errorf("CollectLocalReferences() missing %q", name)
				}
			}
		})
	}
}

func TestComputeReachableTemplates(t *testing.T) {
	// Create a set of templates:
	// layout -> header, content, footer
	// header -> icon
	// content -> sidebar
	// sidebar -> icon
	// footer -> (nothing)
	// icon -> (nothing)
	// unused1 -> unused2
	// unused2 -> (nothing)

	source := `
{{ define "layout" }}{{ template "header" . }}{{ template "content" . }}{{ template "footer" . }}{{ end }}
{{ define "header" }}{{ template "icon" . }}{{ end }}
{{ define "content" }}{{ template "sidebar" . }}{{ end }}
{{ define "sidebar" }}{{ template "icon" . }}{{ end }}
{{ define "footer" }}Footer{{ end }}
{{ define "icon" }}Icon{{ end }}
{{ define "unused1" }}{{ template "unused2" . }}{{ end }}
{{ define "unused2" }}Unused{{ end }}
`

	tmpl, err := template.New("test").Parse(source)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// Build templates map
	templates := make(map[string]*parse.Tree)
	for _, t := range tmpl.Templates() {
		if t.Name() != "test" && t.Tree != nil {
			templates[t.Name()] = t.Tree
		}
	}

	tests := []struct {
		name        string
		entryPoints []string
		expected    []string
	}{
		{
			name:        "from layout",
			entryPoints: []string{"layout"},
			expected:    []string{"layout", "header", "content", "footer", "sidebar", "icon"},
		},
		{
			name:        "from header only",
			entryPoints: []string{"header"},
			expected:    []string{"header", "icon"},
		},
		{
			name:        "from unused1",
			entryPoints: []string{"unused1"},
			expected:    []string{"unused1", "unused2"},
		},
		{
			name:        "multiple entry points",
			entryPoints: []string{"footer", "icon"},
			expected:    []string{"footer", "icon"},
		},
		{
			name:        "non-existent entry point",
			entryPoints: []string{"nonexistent"},
			expected:    []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ComputeReachableTemplates(templates, tt.entryPoints)

			// Check expected are all present
			for _, name := range tt.expected {
				if !result[name] {
					t.Errorf("Expected %q to be reachable, but it wasn't", name)
				}
			}

			// Check no extras
			if len(result) != len(tt.expected) {
				var got []string
				for name := range result {
					got = append(got, name)
				}
				t.Errorf("Got %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestCopyTreeWithRewrites(t *testing.T) {
	source := `{{ template "foo" . }}{{ template "bar" . }}{{ template "baz" . }}`

	tmpl, err := template.New("test").Parse(source)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	rewrites := map[string]string{
		"foo": "NS:foo",
		"bar": "Other:bar",
		// baz not in rewrites, should stay unchanged
	}

	copied := CopyTreeWithRewrites(tmpl.Tree, rewrites)

	// Verify original unchanged
	originalNames := CollectTemplateNames(tmpl.Tree)
	expectedOriginal := []string{"foo", "bar", "baz"}
	for i, name := range originalNames {
		if name != expectedOriginal[i] {
			t.Errorf("Original changed: got %v", originalNames)
			break
		}
	}

	// Verify copy has rewrites
	copiedNames := CollectTemplateNames(copied)
	expectedCopied := []string{"NS:foo", "Other:bar", "baz"}
	for i, name := range copiedNames {
		if name != expectedCopied[i] {
			t.Errorf("Copied names = %v, want %v", copiedNames, expectedCopied)
			break
		}
	}
}
