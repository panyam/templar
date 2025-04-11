package gotl

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

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
