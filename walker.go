package templar

import (
	"bytes"
	"fmt"
	"log/slog"
	"path/filepath"
	ttmpl "text/template"
)

// Walker provides a mechanism for walking through templates and their dependencies
// in a customizable way, applying visitor patterns as templates are processed.
// Unlike the WalkTemplate method which uses post-order traversal, Walker implements
// in-order traversal, processing includes immediately when encountered.
type Walker struct {
	// Buffer stores the processed template content
	Buffer *bytes.Buffer

	// Loader is used to resolve and load template dependencies
	Loader TemplateLoader

	// FoundInclude is called when an include directive is encountered.
	// If it returns true, the include is skipped and not processed.
	FoundInclude func(included string) bool

	// Called before a template is preprocessed.  This is an opportunity
	// for the handler to control entering/preprocessing etc.  For example
	// This could be a place for the handler to skip processing a template
	EnteringTemplate func(template *Template) (skip bool, err error)

	// ProcessedTemplate is called after a template and all its children
	// have been processed. This allows for custom post-processing.
	ProcessedTemplate func(template *Template) error

	// visited tracks templates that have already been processed to prevent cycles or filtering out duplicate definitions
	visited map[string]bool
}

// Walk processes a template and its dependencies using in-order traversal.
// This means includes are processed as soon as they are encountered in the template.
// After processing, the template's ParsedSource will contain the processed content.
// If ProcessedTemplate is defined, it will be called on each processed template.
func (w *Walker) Walk(root *Template) (err error) {
	if w.Buffer == nil {
		w.Buffer = bytes.NewBufferString("")
	}
	// An Inorder walk of of a template.  Unlike WalkTemplate which applies a PostOrder traversal (first collects all
	// includes, processes them and then the root template), here we will process an included template as soon as it is
	// encountered.
	cwd := root.Path
	if cwd != "" {
		cwd = filepath.Dir(cwd)
	}

	if w.EnteringTemplate != nil {
		skip, err := w.EnteringTemplate(root)
		if skip || err != nil {
			return err
		}
	}

	// parse the template and render it
	fm := ttmpl.FuncMap{
		"include": func(glob string) (string, error) {
			// TODO - avoid duplicates
			skipped, err := w.processInclude(root, glob, cwd)
			if skipped {
				return fmt.Sprintf("{{/* Skipping: '%s' */}}", glob), err
			} else {
				return fmt.Sprintf("{{/* Finished Including: '%s' */}}", glob), err
			}
		},
	}

	templ, err := ttmpl.New("").Funcs(fm).Delims("{{#", "#}}").Parse(string(root.RawSource))
	if err != nil {
		slog.Error("error preprocessing template: ", "path", root.Path, "error", err)
		return panicOrError(err)
	}
	if err := templ.Execute(w.Buffer, nil); err != nil {
		slog.Error("error preprocessing template: ", "path", root.Path, "error", err)
		root.Error = err
		return panicOrError(err)
	} else {
		root.ParsedSource = w.Buffer.String()
	}

	// No handle this template
	if w.ProcessedTemplate != nil {
		return w.ProcessedTemplate(root)
	}
	return nil
}

// processInclude handles the inclusion of another template within the current template.
// If FoundInclude returns true, the include is skipped. Otherwise, the included template
// and its dependencies are loaded and processed.
// Returns a boolean indicating if the include was skipped, and any error encountered.
func (w *Walker) processInclude(root *Template, included string, cwd string) (skipped bool, err error) {
	skipped = w.FoundInclude != nil && w.FoundInclude(included)
	if skipped {
		return
	}

	children, err := w.Loader.Load(included, cwd)
	if err != nil {
		slog.Error("error loading include: ", "included", included, "error", err)
		return false, panicOrError(err)
	}
	for _, child := range children {
		if child.Path != "" {
			if !root.AddDependency(child) {
				slog.Error(fmt.Sprintf("found cyclical dependency: %s -> %s", child.Path, root.Path), "from", child.Path, "to", root.Path)
				continue
			}
		}
		err = w.Walk(child)
		if err != nil {
			slog.Error("error walking", "included", included, "error", err)
			root.Error = err
			return false, panicOrError(err)
		}
	}
	return
}
