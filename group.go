package gotl

import (
	htmpl "html/template"
	"io"
	"log"
	"path/filepath"
	ttmpl "text/template"
)

// A group of templates
type TemplateGroup struct {
	templates map[string]*Template
	// Hnderlying html and text template that map to given names (NOT PATHS)
	Funcs         map[string]any
	Loader        TemplateLoader
	htmlTemplates map[string]*htmpl.Template
	textTemplates map[string]*ttmpl.Template
	dependencies  map[string]map[string]bool
}

func NewTemplateGroup() *TemplateGroup {
	return &TemplateGroup{
		Funcs:         make(map[string]any),
		htmlTemplates: make(map[string]*htmpl.Template),
		textTemplates: make(map[string]*ttmpl.Template),
		templates:     make(map[string]*Template),
		dependencies:  make(map[string]map[string]bool),
	}
}

func (t *TemplateGroup) AddFuncs(funcs map[string]any) *TemplateGroup {
	for k, v := range funcs {
		t.Funcs[k] = v
	}
	return t
}

func (t *TemplateGroup) NewHtmlTemplate(name string, funcs map[string]any) (out *htmpl.Template) {
	out = htmpl.New(name).Funcs(t.Funcs)
	if funcs != nil {
		out = out.Funcs(funcs)
	}
	return out
}

func (t *TemplateGroup) NewTextTemplate(name string, funcs map[string]any) (out *ttmpl.Template) {
	out = ttmpl.New(name).Funcs(t.Funcs)
	if funcs != nil {
		out = out.Funcs(funcs)
	}
	return out
}

func (t *TemplateGroup) PreProcessTextTemplate(root *Template, funcs ttmpl.FuncMap) (out *ttmpl.Template, err error) {
	name := root.Name
	if name == "" {
		name = root.Path
	}
	if name != "" {
		out = t.textTemplates[name]
	}
	if out == nil {
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

func (t *TemplateGroup) PreProcessHtmlTemplate(root *Template, funcs htmpl.FuncMap) (out *htmpl.Template, err error) {
	name := root.Name
	if name == "" {
		name = root.Path
	}
	if name != "" {
		out = t.htmlTemplates[name]
	}
	if out == nil {
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

// Preprocesses and Renders a template either as html or as text
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

/*
func (t *TemplateGroup) RenderHtmlTemplate(w io.Writer, name string, data any) error {
	t2, err := t.PreProcessHtmlTemplate(name, nil)
	if err != nil {
		log.Println("Error loading: ", name, err)
		panic(err)
		return panicOrError(err)
	}
	tmpl := htmpl.Must(t2, err)
	err = tmpl.ExecuteTemplate(w, name, data)
	if err != nil {
		log.Println("error rendering template: ", name, err)
	}
	return panicOrError(err)
}

func (t *TemplateGroup) RenderTextTemplate(w io.Writer, name string, data any) error {
	tmpl := ttmpl.Must(t.PreProcessTextTemplate(name, nil))
	err := tmpl.ExecuteTemplate(w, name, data)
	if err != nil {
		log.Println("error rendering template: ", name, err)
	}
	return panicOrError(err)
}
*/
