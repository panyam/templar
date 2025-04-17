package templar

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"path/filepath"
	ttmpl "text/template"
)

// TemplateNotFound is returned when a template could not be found by a loader.
var TemplateNotFound = errors.New("template not found")

// Template is the basic unit of rendering that manages content and dependencies.
type Template struct {
	// Name is an identifier for this template.
	Name string

	// RawSource contains the original, unprocessed template content.
	RawSource []byte

	// ParsedSource contains the template content after preprocessing.
	ParsedSource string

	// Path is the file path for this template if it was loaded from a file.
	Path string

	// Status indicates whether the template has been loaded and parsed.
	Status int

	// AsHtml determines whether the content should be treated as HTML (with escaping)
	// or as plain text.
	AsHtml bool

	// includes contains other templates that this template depends on.
	includes []*Template

	// Error contains any error encountered during template processing.
	Error error

	// Metadata stores extracted information from the template (e.g., FrontMatter).
	Metadata map[string]any
}

// AddDependency adds another template as a dependency of this template.
// It returns false if the dependency would create a cycle, true otherwise.
func (t *Template) AddDependency(another *Template) bool {
	if t.Path != "" {
		for _, child := range t.includes {
			// TODO - check full cycles
			if child.Path == another.Path {
				return false
			}
		}
		t.includes = append(t.includes, another)
	}
	return true
}

// Dependencies returns all templates that this template directly depends on.
func (t *Template) Dependencies() []*Template {
	return t.includes
}

// TemplateLoader defines an interface for loading template content by name or pattern.
type TemplateLoader interface {
	// Load attempts to load templates matching the given pattern.
	// If cwd is not empty, it's used as the base directory for relative paths.
	// Returns matching templates or an error if no templates were found.
	Load(pattern string, cwd string) (template []*Template, err error)
}

func (root *Template) WalkTemplate(loader TemplateLoader, handler func(template *Template) error) (err error) {
	// An Inorder walk of of a template.  Unlike WalkTemplate which applies a PostOrder traversal (first collects all
	// includes, processes them and then the root template), here we will process an included template as soon as it is
	// encountered.
	cwd := root.Path
	if cwd != "" {
		cwd = filepath.Dir(cwd)
	}

	log.Println("Coming from : ", root.Name)
	defer log.Println("Finished with: ", root.Name, root.Path)
	var includes []string
	fm := ttmpl.FuncMap{
		"include": func(glob string) string {
			log.Println("Coming to: ", glob)
			// TODO - avoid duplicates
			includes = append(includes, glob)
			return fmt.Sprintf("{{/* Including: '%s' */}}", glob)
		},
	}

	// First parse the macro template
	templ, err := ttmpl.New("").Funcs(fm).Delims("{{#", "#}}").Parse(string(root.RawSource))
	if err != nil {
		slog.Error("error template: ", "path", root.Path, "error", err)
		return panicOrError(err)
	}

	// New execute it so that all includes are evaluated
	buff := bytes.NewBufferString("")
	if err := templ.Execute(buff, nil); err != nil {
		slog.Error("error preprocessing template: ", "path", root.Path, "error", err)
		root.Error = err
		return panicOrError(err)
	} else {
		root.ParsedSource = buff.String()
	}

	// Resolve the includes - for now non-wildcards are only allowed
	for _, included := range includes {
		children, err := loader.Load(included, cwd)
		if err != nil {
			slog.Error("error loading include: ", "included", included, "error", err)
			return panicOrError(err)
		}
		for _, child := range children {
			if child.Path != "" {
				if !root.AddDependency(child) {
					slog.Error(fmt.Sprintf("found cyclical dependency: %s -> %s", child.Path, root.Path), "from", child.Path, "to", root.Path)
					continue
				}
			}
			err = child.WalkTemplate(loader, handler)
			if err != nil {
				slog.Error("error walking", "included", included, "error", err)
				root.Error = err
				return panicOrError(err)
			}
		}
	}

	// No handle this template
	return handler(root)
}
