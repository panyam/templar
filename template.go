package gotl

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"path/filepath"
	ttmpl "text/template"
)

var TemplateNotFound = errors.New("template not found")

// The template is the most basic rendered at its core.
type Template struct {
	Name string

	// Actual template content
	RawSource []byte

	// Source after the template has been parsed
	ParsedSource string

	// File name for this template (if any - ie if it was loaded from a file)
	Path string

	// Whether the contents have been loaded and parsed or not
	Status int

	// Whether to return a text or a html template (later ensures extra escaping)
	AsHtml bool

	// Other template included in this template
	includes []*Template

	// Whether there are any errors in this template
	Error error

	// Any metadata we want to extract from the template (eg FrontMatter etc)
	Metadata map[string]any
}

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

func (t *Template) Dependencies() []*Template {
	return t.includes
}

// Interface for the loader which loads the content of a template from its name.
type TemplateLoader interface {
	Load(pattern string, cwd string) (template []*Template, err error)
}

// Walks a template starting from the root template loading
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
