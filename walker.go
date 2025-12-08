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

	// inProgress tracks templates currently being processed to detect cycles (infinite recursion)
	inProgress map[string]bool
}

// Walk processes a template and its dependencies using in-order traversal.
// This means includes are processed as soon as they are encountered in the template.
// After processing, the template's ParsedSource will contain the processed content.
// If ProcessedTemplate is defined, it will be called on each processed template.
func (w *Walker) Walk(root *Template) (err error) {
	if w.Buffer == nil {
		w.Buffer = bytes.NewBufferString("")
	}
	if w.inProgress == nil {
		w.inProgress = make(map[string]bool)
	}

	// Check if this template is currently being processed (cycle detection)
	if root.Path != "" {
		if w.inProgress[root.Path] {
			slog.Warn("cycle detected, skipping template already in progress", "path", root.Path)
			return nil
		}
		w.inProgress[root.Path] = true
		defer func() { w.inProgress[root.Path] = false }()
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
		"include": func(args ...string) (string, error) {
			// Syntax: include "file.html" ["template1" "template2" ...]
			// If no templates specified, includes all templates from the file.
			// If templates specified, includes only those (and their dependencies).
			if len(args) < 1 {
				return "", fmt.Errorf("include requires at least a file path")
			}
			glob := args[0]
			var entryPoints []string
			if len(args) > 1 {
				entryPoints = args[1:]
			}
			skipped, err := w.processInclude(root, glob, entryPoints, cwd)
			if skipped {
				return fmt.Sprintf("{{/* Skipping: '%s' */}}", glob), err
			} else {
				return fmt.Sprintf("{{/* Finished Including: '%s' */}}", glob), err
			}
		},
		"namespace": func(args ...string) (string, error) {
			// Syntax: namespace "NS" "file.html" ["template1" "template2" ...]
			// Loads templates into namespace NS with tree-shaking.
			if len(args) < 2 {
				return "", fmt.Errorf("namespace requires: namespace file [templates...]")
			}
			namespace, glob := args[0], args[1]
			if namespace == "" {
				return "", fmt.Errorf("namespace requires a non-empty namespace name")
			}
			var entryPoints []string
			if len(args) > 2 {
				entryPoints = args[2:]
			}
			skipped, err := w.processNamespace(root, namespace, glob, entryPoints, cwd)
			if skipped {
				return fmt.Sprintf("{{/* Skipping namespace '%s' from '%s' */}}", namespace, glob), err
			} else {
				return fmt.Sprintf("{{/* Loaded namespace '%s' from '%s' */}}", namespace, glob), err
			}
		},
		"extend": func(args ...string) (string, error) {
			// Syntax: extend "SourceTemplate" "DestTemplate" "block1" "override1" ...
			// Creates DestTemplate as a copy of SourceTemplate with references rewired.
			// SourceTemplate must already exist (from a prior include/namespace).
			if len(args) < 2 {
				return "", fmt.Errorf("extend requires at least: sourceTemplate destTemplate")
			}
			if len(args)%2 != 0 {
				return "", fmt.Errorf("extend requires pairs of block/override after destTemplate")
			}
			source, dest := args[0], args[1]
			if dest == "" {
				return "", fmt.Errorf("extend requires a non-empty destination template name")
			}

			// Parse block/override pairs
			rewrites := make(map[string]string)
			for i := 2; i < len(args); i += 2 {
				block, override := args[i], args[i+1]
				rewrites[block] = override
			}

			w.processExtend(root, source, dest, rewrites)
			return fmt.Sprintf("{{/* Extended '%s' as '%s' */}}", source, dest), nil
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
//
// If entryPoints is non-empty, only those templates (and their dependencies) are included.
// Returns a boolean indicating if the include was skipped, and any error encountered.
func (w *Walker) processInclude(root *Template, included string, entryPoints []string, cwd string) (skipped bool, err error) {
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
		// Inherit namespace from parent template
		if root.Namespace != "" {
			child.Namespace = root.Namespace
		}

		// Set entry points for selective inclusion (tree-shaking)
		if len(entryPoints) > 0 {
			child.NamespaceEntryPoints = entryPoints
		}

		if child.Path != "" {
			if !root.AddDependency(child) {
				slog.Error(fmt.Sprintf("found cyclical dependency: %s -> %s", child.Path, root.Path), "from", child.Path, "to", root.Path)
				continue
			}
		}

		// If the child has a namespace (inherited or otherwise), use a fresh walker
		// with its own buffer. This ensures the child's ParsedSource contains only
		// its own content, not contaminated with the parent's partial buffer content.
		if child.Namespace != "" {
			childWalker := &Walker{
				Loader:            w.Loader,
				FoundInclude:      w.FoundInclude,
				EnteringTemplate:  w.EnteringTemplate,
				ProcessedTemplate: w.ProcessedTemplate,
				inProgress:        w.inProgress, // Share inProgress map for cycle detection
			}
			err = childWalker.Walk(child)
		} else {
			err = w.Walk(child)
		}
		if err != nil {
			slog.Error("error walking", "included", included, "error", err)
			root.Error = err
			return false, panicOrError(err)
		}
	}
	return
}

// processNamespace handles the inclusion of templates into a namespace.
// Templates are loaded from the file and will be registered with the given namespace prefix.
// If entryPoints is non-empty, only those templates (and their dependencies) are included.
func (w *Walker) processNamespace(root *Template, namespace string, included string, entryPoints []string, cwd string) (skipped bool, err error) {
	skipped = w.FoundInclude != nil && w.FoundInclude(included)
	if skipped {
		return
	}

	children, err := w.Loader.Load(included, cwd)
	if err != nil {
		slog.Error("error loading namespace: ", "included", included, "error", err)
		return false, panicOrError(err)
	}
	for _, child := range children {
		// Set the namespace and entry points on the child template
		child.Namespace = namespace
		if len(entryPoints) > 0 {
			child.NamespaceEntryPoints = entryPoints
		}

		if child.Path != "" {
			if !root.AddDependency(child) {
				slog.Error(fmt.Sprintf("found cyclical dependency: %s -> %s", child.Path, root.Path), "from", child.Path, "to", root.Path)
				continue
			}
		}

		// Use a fresh walker with its own buffer for namespaced includes.
		// This ensures the child's ParsedSource contains only its own content,
		// avoiding conflicts when the same template is included multiple times
		// with different namespaces.
		// IMPORTANT: Share the inProgress map to detect cycles (infinite recursion).
		childWalker := &Walker{
			Loader:            w.Loader,
			FoundInclude:      w.FoundInclude,
			EnteringTemplate:  w.EnteringTemplate,
			ProcessedTemplate: w.ProcessedTemplate,
			inProgress:        w.inProgress, // Share inProgress map for cycle detection
		}
		err = childWalker.Walk(child)
		if err != nil {
			slog.Error("error walking namespace", "included", included, "error", err)
			root.Error = err
			return false, panicOrError(err)
		}
	}
	return
}

// processExtend records an extend directive on the root template.
// The actual extension (copying and rewiring) is performed later in group.go
// after all templates have been parsed.
func (w *Walker) processExtend(root *Template, source string, dest string, rewrites map[string]string) {
	root.Extensions = append(root.Extensions, Extension{
		SourceTemplate: source,
		DestTemplate:   dest,
		Rewrites:       rewrites,
	})
}
