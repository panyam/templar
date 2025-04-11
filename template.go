package gotl

import (
	"bytes"
	"errors"
	"fmt"
	"log"
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

// WalkTemplate processes a template and its dependencies recursively.
// It starts from the root template, processes all includes with the {{# include "..." #}} directive,
// and calls the provided handler function on each template in the dependency tree.
// The loader is used to resolve and load included templates.
func (root *Template) WalkTemplate(loader TemplateLoader, handler func(template *Template) error) (err error) {
	// Now get a list of all the includes - It doesnt matter *where* the include is - they are all collected first
	// Given how go templates treat definitions (by name) their order doesnt matter and we dont want to change this
	// here
	var includes []string
	fm := ttmpl.FuncMap{
		"include": func(glob string) string {
			// TODO - avoid duplicates
			includes = append(includes, glob)
			return fmt.Sprintf("{{/* Including: '%s' */}}", glob)
		},
	}

	// First parse the macro template
	templ, err := ttmpl.New("").Funcs(fm).Delims("{{#", "#}}").Parse(string(root.RawSource))
	if err != nil {
		log.Println("Error loading template: ", err, root.Path)
		return panicOrError(err)
	}

	// New execute it so that all includes are evaluated
	buff := bytes.NewBufferString("")
	if err := templ.Execute(buff, nil); err != nil {
		log.Println("Pre Processor Error: ", err, root.Path)
		root.Error = err
		return panicOrError(err)
	} else {
		root.ParsedSource = buff.String()
	}

	// Resolve the includes - for now non-wildcards are only allowed
	cwd := root.Path
	if cwd != "" {
		cwd = filepath.Dir(cwd)
	}
	for _, included := range includes {
		// log.Printf("Trying to load '%s' from '%s", included, root.Path)
		children, err := loader.Load(included, cwd)
		if err != nil {
			log.Println("error loading: ", included, err)
			return panicOrError(err)
		}
		for _, child := range children {
			if child.Path != "" {
				if !root.AddDependency(child) {
					log.Printf("Found cyclical dependency: %s -> %s", child.Path, root.Path)
					continue
				}
			}
			err = child.WalkTemplate(loader, handler)
			if err != nil {
				log.Println("error walking: ", included, err)
				root.Error = err
				return panicOrError(err)
			}
		}
	}

	// No handle this template
	return handler(root)
}
