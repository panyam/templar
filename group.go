package templar

import (
	"fmt"
	htmpl "html/template"
	"io"
	"log/slog"
	"maps"
	"path/filepath"
	ttmpl "text/template"
	"text/template/parse"
)

// TemplateGroup manages a collection of templates and their dependencies,
// providing methods to process and render them.
type TemplateGroup struct {
	templates map[string]*Template
	// Underlying html and text template that map to given names (NOT PATHS)

	// Funcs contains template functions available to all templates in this group.
	Funcs map[string]any

	// Loader is used to resolve and load template dependencies.
	Loader TemplateLoader

	htmlTemplates map[string]*htmpl.Template
	textTemplates map[string]*ttmpl.Template
	dependencies  map[string]map[string]bool
}

// NewTemplateGroup creates a new empty template group with initialized internals.
func NewTemplateGroup() *TemplateGroup {
	return &TemplateGroup{
		Funcs:         make(map[string]any),
		htmlTemplates: make(map[string]*htmpl.Template),
		textTemplates: make(map[string]*ttmpl.Template),
		templates:     make(map[string]*Template),
		dependencies:  make(map[string]map[string]bool),
	}
}

// Calls the underlying Loader to load templates matching a pattern and optional using a cwd for relative paths.
// Panics if an error is encountered.
// Returns matching templates or an error if no templates were found.
func (t *TemplateGroup) MustLoad(pattern string, cwd string) []*Template {
	out, err := t.Loader.Load(pattern, cwd)
	if err != nil {
		panic(err)
	}
	return out
}

// AddFuncs adds template functions to this group, making them available
// to all templates. Returns the template group for method chaining.
func (t *TemplateGroup) AddFuncs(funcs map[string]any) *TemplateGroup {
	maps.Copy(t.Funcs, funcs)
	return t
}

// NewHtmlTemplate creates a new HTML template with the given name.
// The template will have access to the group's functions and any additional
// functions provided.
func (t *TemplateGroup) NewHtmlTemplate(name string, funcs map[string]any) (out *htmpl.Template) {
	out = htmpl.New(name).Funcs(t.Funcs)
	if funcs != nil {
		out = out.Funcs(funcs)
	}
	return out
}

// NewTextTemplate creates a new TEXT template with the given name.
// The template will have access to the group's functions and any additional
// functions provided.
func (t *TemplateGroup) NewTextTemplate(name string, funcs map[string]any) (out *ttmpl.Template) {
	out = ttmpl.New(name).Funcs(t.Funcs)
	if funcs != nil {
		out = out.Funcs(funcs)
	}
	return out
}

// PreProcessTextTemplate processes a template and its dependencies, creating a text/template
// that can be used for rendering. It handles template dependencies recursively.
// Returns the processed template and any error encountered.
func (t *TemplateGroup) PreProcessTextTemplate(root *Template, funcs ttmpl.FuncMap) (out *ttmpl.Template, err error) {
	name := root.Name
	if name == "" {
		name = root.Path
	}
	if name != "" {
		out = t.textTemplates[name]
	}
	if true || out == nil {
		// try and load it
		out = t.NewTextTemplate(name, funcs)
		err = root.WalkTemplate(t.Loader, func(t *Template) error {
			if t.Path == "" {
				out, err = out.Parse(t.ParsedSource)
				return panicOrError(err)
			} else {
				x, err := out.Parse(t.ParsedSource)
				if err != nil {
					return panicOrError(err)
				}
				// TODO - is this really necessary to add the parsed source back to out
				// Should the parsing already do that for "out" anyway?
				base := filepath.Base(t.Path)
				out, err = out.AddParseTree(base, x.Tree)
				return panicOrError(err)
			}
		})
		if err == nil && name != "" {
			t.textTemplates[name] = out
		}
	}
	return out, err
}

// PreProcessHtmlTemplate processes a HTML template and its dependencies, creating an html/template
// that can be used for rendering. It handles template dependencies recursively.
// Returns the processed template and any error encountered.
func (t *TemplateGroup) PreProcessHtmlTemplate(root *Template, funcs htmpl.FuncMap) (out *htmpl.Template, err error) {
	name := root.Name
	if name == "" {
		name = root.Path
	}
	if name != "" {
		out = t.htmlTemplates[name]
	}
	if true || out == nil {
		// try and load it
		out = htmpl.New(name).Funcs(t.Funcs)
		if funcs != nil {
			out = out.Funcs(funcs)
		}

		// Collect all extensions from all processed templates
		var allExtensions []Extension

		w := Walker{Loader: t.Loader,
			ProcessedTemplate: func(curr *Template) error {
				// Collect extensions from this template
				allExtensions = append(allExtensions, curr.Extensions...)

				// Skip non-root templates that don't have a namespace and no entry points
				// (they will be processed via normal include mechanism)
				if curr != root && curr.Namespace == "" && len(curr.NamespaceEntryPoints) == 0 {
					return nil
				}

				if curr.Path == "" {
					out, err = out.Parse(curr.ParsedSource)
					return panicOrError(err)
				}

				// If namespace is set, parse into a temporary template and apply namespacing
				if curr.Namespace != "" {
					return t.processNamespacedTemplate(curr, out, funcs)
				}

				// If entry points are set (selective include), apply tree-shaking
				if len(curr.NamespaceEntryPoints) > 0 {
					return t.processSelectiveInclude(curr, out, funcs)
				}

				// Normal case: parse and add with original name
				base := filepath.Base(curr.Path)
				x, err := out.Parse(curr.ParsedSource)
				if err != nil {
					return panicOrError(err)
				}
				out, err = out.AddParseTree(base, x.Tree)
				return panicOrError(err)
			}}
		err = w.Walk(root)
		if err != nil {
			return out, err
		}

		// Process all collected extensions after all templates are parsed
		err = t.processExtensionsList(allExtensions, out)
		if err != nil {
			return out, err
		}

		if name != "" {
			t.htmlTemplates[name] = out
		}
	}
	return out, err
}

// processNamespacedTemplate handles templates that should be added to a namespace.
// It parses the template, applies tree-shaking if entry points are specified,
// and adds all reachable templates with namespaced names.
func (t *TemplateGroup) processNamespacedTemplate(curr *Template, out *htmpl.Template, funcs htmpl.FuncMap) error {
	slog.Debug("processNamespacedTemplate", "path", curr.Path, "namespace", curr.Namespace)

	// Parse into a fresh temporary template to avoid name collisions
	temp := htmpl.New("temp").Funcs(t.Funcs)
	if funcs != nil {
		temp = temp.Funcs(funcs)
	}
	temp, err := temp.Parse(curr.ParsedSource)
	if err != nil {
		return panicOrError(err)
	}

	// Build map of all templates for tree-shaking
	allTemplates := make(map[string]*htmpl.Template)
	var allNames []string
	for _, tmpl := range temp.Templates() {
		if tmpl.Tree != nil && tmpl.Name() != "temp" {
			allTemplates[tmpl.Name()] = tmpl
			allNames = append(allNames, tmpl.Name())
		}
	}
	slog.Debug("processNamespacedTemplate: found templates", "path", curr.Path, "templates", allNames)

	// Determine which templates to include
	var templatesToInclude map[string]bool
	if len(curr.NamespaceEntryPoints) > 0 {
		// Tree-shaking: only include reachable templates
		treesMap := make(map[string]*parse.Tree)
		for name, tmpl := range allTemplates {
			treesMap[name] = tmpl.Tree
		}
		templatesToInclude = ComputeReachableTemplates(treesMap, curr.NamespaceEntryPoints)
	} else {
		// Include all templates
		templatesToInclude = make(map[string]bool)
		for _, name := range allNames {
			templatesToInclude[name] = true
		}
	}

	// Build rewrite map for all templates being included
	rewrites := make(map[string]string)
	for name := range templatesToInclude {
		rewrites[name] = TransformName(name, curr.Namespace)
	}

	// Add namespaced templates to output
	var createdNames []string
	for name := range templatesToInclude {
		tmpl := allTemplates[name]
		if tmpl == nil || tmpl.Tree == nil {
			continue
		}

		// Copy tree and apply namespace rewrites
		copiedTree := tmpl.Tree.Copy()
		WalkParseTree(copiedTree.Root, func(node *parse.TemplateNode) {
			// Apply full namespace transformation rules
			node.Name = TransformName(node.Name, curr.Namespace)
		})

		namespacedName := rewrites[name]
		copiedTree.Name = namespacedName
		out, err = out.AddParseTree(namespacedName, copiedTree)
		if err != nil {
			return panicOrError(err)
		}
		createdNames = append(createdNames, namespacedName)
	}
	slog.Debug("processNamespacedTemplate: created templates", "path", curr.Path, "created", createdNames)

	return nil
}

// processSelectiveInclude handles templates with entry points but no namespace.
// It applies tree-shaking to only include the specified templates and their dependencies.
func (t *TemplateGroup) processSelectiveInclude(curr *Template, out *htmpl.Template, funcs htmpl.FuncMap) error {
	// Parse into a fresh temporary template
	temp := htmpl.New("temp").Funcs(t.Funcs)
	if funcs != nil {
		temp = temp.Funcs(funcs)
	}
	temp, err := temp.Parse(curr.ParsedSource)
	if err != nil {
		return panicOrError(err)
	}

	// Build map of all templates for tree-shaking
	treesMap := make(map[string]*parse.Tree)
	templatesMap := make(map[string]*htmpl.Template)
	for _, tmpl := range temp.Templates() {
		if tmpl.Tree != nil && tmpl.Name() != "temp" {
			treesMap[tmpl.Name()] = tmpl.Tree
			templatesMap[tmpl.Name()] = tmpl
		}
	}

	// Compute reachable templates
	templatesToInclude := ComputeReachableTemplates(treesMap, curr.NamespaceEntryPoints)

	// Add only reachable templates to output
	for name := range templatesToInclude {
		tmpl := templatesMap[name]
		if tmpl == nil || tmpl.Tree == nil {
			continue
		}

		out, err = out.AddParseTree(name, tmpl.Tree)
		if err != nil {
			return panicOrError(err)
		}
	}

	return nil
}

// processExtensions processes all extend directives recorded on the root template.
// For each extension, it copies the source template and rewires references.
func (t *TemplateGroup) processExtensions(root *Template, out *htmpl.Template) error {
	return t.processExtensionsList(root.Extensions, out)
}

// processExtensionsList processes a list of extensions.
// For each extension, it copies the source template and rewires references.
func (t *TemplateGroup) processExtensionsList(extensions []Extension, out *htmpl.Template) error {
	if len(extensions) > 0 {
		// Log available templates for debugging
		var availableNames []string
		for _, tmpl := range out.Templates() {
			if tmpl.Tree != nil {
				availableNames = append(availableNames, tmpl.Name())
			}
		}
		slog.Debug("processExtensionsList: available templates", "count", len(availableNames), "templates", availableNames)
	}

	for _, ext := range extensions {
		slog.Debug("processExtensionsList: processing extension", "source", ext.SourceTemplate, "dest", ext.DestTemplate)
		// Find the source template
		sourceTmpl := out.Lookup(ext.SourceTemplate)
		if sourceTmpl == nil || sourceTmpl.Tree == nil {
			return fmt.Errorf("extend: source template not found: %s", ext.SourceTemplate)
		}

		// Copy the tree and apply rewrites
		copiedTree := CopyTreeWithRewrites(sourceTmpl.Tree, ext.Rewrites)
		copiedTree.Name = ext.DestTemplate

		// Add the new template
		var err error
		out, err = out.AddParseTree(ext.DestTemplate, copiedTree)
		if err != nil {
			return panicOrError(err)
		}
	}

	return nil
}

// RenderHtmlTemplate renders a template as HTML to the provided writer.
//
// It processes the template with its dependencies, executes it with the given data,
// and applies any additional template functions provided.
//
// If entry is specified, it executes that specific template within the processed template.
func (t *TemplateGroup) RenderHtmlTemplate(w io.Writer, root *Template, entry string, data any, funcs map[string]any) (err error) {
	out, err := t.PreProcessHtmlTemplate(root, funcs)
	if err != nil {
		return panicOrError(err)
	}
	tmpl := htmpl.Must(out, err)
	name := entry
	if name == "" {
		name = root.Name
	}
	if name == "" {
		err = tmpl.Execute(w, data)
	} else {
		err = tmpl.ExecuteTemplate(w, name, data)
	}
	if err != nil {
		slog.Error("error rendering template as html: ", "name", name, "error", err)
		return panicOrError(err)
	}
	return
}

// RenderTextTemplate renders a template as plain text to the provided writer.
//
// It processes the template with its dependencies, executes it with the given data,
// and applies any additional template functions provided.
//
// If entry is specified, it executes that specific template within the processed template.
func (t *TemplateGroup) RenderTextTemplate(w io.Writer, root *Template, entry string, data any, funcs map[string]any) (err error) {
	out, err := t.PreProcessTextTemplate(root, funcs)
	if err != nil {
		return panicOrError(err)
	}
	tmpl := ttmpl.Must(out, err)
	name := entry
	if name == "" {
		name = root.Name
	}
	if name == "" {
		err = tmpl.Execute(w, data)
	} else {
		err = tmpl.ExecuteTemplate(w, name, data)
	}
	if err != nil {
		slog.Error("error rendering template as text: ", "name", name, "error", err)
	}
	return
}
