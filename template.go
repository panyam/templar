package gotl

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	ttmpl "text/template"
)

var TemplateNotFound = errors.New("template not found")
var TemplateUnsupported = errors.New("template type not supported")

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

// Loads templates based contents of a file system
type FileSystemLoader struct {
	Folders    []string
	Extensions []string
}

// Creates a new file system loader
func NewFileSystemLoader(folders ...string) *FileSystemLoader {
	return &FileSystemLoader{
		Folders: folders,
		Extensions: []string{
			"tmpl", "tmplus", "html",
		},
	}
}

func (g *FileSystemLoader) Load(name string, cwd string) (template []*Template, err error) {
	ext := filepath.Ext(name)
	extensions := g.Extensions
	withoutext := name
	if ext != "" {
		extensions = []string{ext[1:]}
		withoutext = name[:len(name)-len(ext)]
	}
	isRelative := strings.HasPrefix(name, "./") || strings.HasPrefix(name, "../")
	folders := g.Folders
	if cwd != "" {
		if isRelative {
			// TODO - should we even look at other folders if it is a relative path?
			// give cwd a higher priority
			// folders = append([]string{cwd}, g.Folders...)
			folders = []string{cwd}
		} else {
			// make it lower than other folders
			folders = append(folders, cwd)
		}
	}
	// log.Printf("Loading in CWD: %s, Name: %s, WithoutExt: %s, Ext: %s, Folders: %v", cwd, name, withoutext, ext, folders)
	for _, folder := range folders {
		folder, err := filepath.Abs(folder)
		if err == nil {
			folderinfo, err := os.Stat(folder)
			if os.IsNotExist(err) {
				log.Println("folder does not exist: ", folder)
				continue
			}
			if !folderinfo.IsDir() {
				log.Println("folder is not a directory: ", folder)
				continue
			}
		} else {
			log.Println("Invalid folder: ", folder)
			continue
		}
		for _, ext := range extensions {
			// check if folder/name.ext exists
			withext := fmt.Sprintf("%s.%s", withoutext, ext)
			fname, err := filepath.Abs(filepath.Join(folder, withext))
			if err != nil {
				log.Println("Not found: ", folder, withext, fname, err)
				continue
			}
			info, err := os.Stat(fname)
			if err == nil && !info.IsDir() {
				// Found it so laod it
				contents, err := os.ReadFile(fname)
				return []*Template{{RawSource: contents, Path: fname}}, err
			}
		}
	}
	return nil, TemplateNotFound
}

// A loader that tries all loaders and returns the first match
type LoaderList struct {
	DefaultLoader TemplateLoader
	loaders       []TemplateLoader
}

// Adds a new loader to the list
func (t *LoaderList) AddLoader(loader TemplateLoader) *LoaderList {
	t.loaders = append(t.loaders, loader)
	return t
}

// Gets the loader for a particular template by item by its name.
func (t *LoaderList) Load(name string, cwd string) (matched []*Template, err error) {
	// TODO - should we cache this?  Or is a CacheLoader just another type?
	for _, loader := range t.loaders {
		matched, err = loader.Load(name, cwd)
		if err == nil && matched != nil && len(matched) > 0 {
			return matched, err
		} else if err == TemplateNotFound {
			continue
		} else {
			break
		}
	}
	// try loading with the default laoder too
	if t.DefaultLoader != nil {
		return t.DefaultLoader.Load(name, cwd)
	}
	return nil, TemplateNotFound
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

	// Now go through all included templates first
	// log.Println("Starting at root: ", root.Path)
	// log.Println("Includes: ", includes)

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
