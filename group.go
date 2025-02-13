package gotl

import (
	"html/template"
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

func (t *TemplateGroup) RenderHtmlTemplate(w io.Writer, name string, data any) error {
	tmpl := template.Must(t.GetHtmlTemplate(name, nil))
	err := tmpl.ExecuteTemplate(w, name, data)
	if err != nil {
		log.Println("error rendering template: ", name, err)
	}
	return err
}

func (t *TemplateGroup) GetHtmlTemplate(name string, funcs htmpl.FuncMap) (out *htmpl.Template, err error) {
	root, err := t.Loader.Load(name, "")
	if err != nil {
		log.Println("Error getting template: ", name, err)
		return nil, err
	}

	out = t.htmlTemplates[name]
	if out == nil {
		// try and load it
		out = htmpl.New(name).Funcs(t.Funcs)
		if funcs != nil {
			out = out.Funcs(funcs)
		}
		err = root[0].WalkTemplate(t.Loader, func(t *Template) error {
			if t.Path == "" {
				out, err = out.Parse(t.ParsedSource)
				return err
			} else {
				x, err := out.Parse(t.ParsedSource)
				if err != nil {
					return err
				}
				base := filepath.Base(t.Path)
				out, err = out.AddParseTree(base, x.Tree)
				return err
			}
		})
		if err == nil {
			t.htmlTemplates[name] = out
		}
	}
	return out, err
}

func (g *TemplateGroup) GetTextTemplate(template string) (*ttmpl.Template, error) {
	// TODO
	return nil, nil
}
