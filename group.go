package templar

import (
	htmpl "html/template"
	"io"
	"log"
	"maps"
	"path/filepath"
	ttmpl "text/template"
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
		err = root.WalkTemplate(t.Loader, func(t *Template) error {
			if t.Path == "" {
				out, err = out.Parse(t.ParsedSource)
				return panicOrError(err)
			} else {
				x, err := out.Parse(t.ParsedSource)
				if err != nil {
					return panicOrError(err)
				}
				base := filepath.Base(t.Path)
				out, err = out.AddParseTree(base, x.Tree)
				return panicOrError(err)
			}
		})
		if err == nil && name != "" {
			t.htmlTemplates[name] = out
		}
	}
	return out, err
}

// RenderHtmlTemplate renders a template as HTML to the provided writer.
// It processes the template with its dependencies, executes it with the given data,
// and applies any additional template functions provided.
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
		log.Println("error rendering template as html: ", name, err)
		return panicOrError(err)
	}
	return
}

// RenderTextTemplate renders a template as plain text to the provided writer.
// It processes the template with its dependencies, executes it with the given data,
// and applies any additional template functions provided.
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
		log.Println("error rendering template as text: ", name, err)
	}
	return
}
