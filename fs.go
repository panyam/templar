package templar

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// FileSystemLoader loads templates from the file system based on
// a set of directories and file extensions.
type FileSystemLoader struct {
	// Folders is a list of directories to search for templates.
	Folders []string

	// Extensions is a list of file extensions to consider as templates.
	Extensions []string
}

// NewFileSystemLoader creates a new file system loader that will search
// in the provided folders for template files.
// By default, it recognizes files with .tmpl, .tmplus, and .html extensions.
func NewFileSystemLoader(folders ...string) *FileSystemLoader {
	return &FileSystemLoader{
		Folders: folders,
		Extensions: []string{
			"tmpl", "tmplus", "html",
		},
	}
}

// Load attempts to find and load a template with the given name.
// If the name includes an extension, only files with that extension are considered.
// Otherwise, files with any of the loader's recognized extensions are searched.
// If cwd is provided, it's used for resolving relative paths.
// Returns the loaded templates or TemplateNotFound if no matching templates were found.
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
				slog.Debug("folder does not exist: ", "folder", folder)
				continue
			}
			if !folderinfo.IsDir() {
				slog.Debug("folder is not a directory: ", "folder", folder)
				continue
			}
		} else {
			slog.Debug("Invalid folder: ", "folder", folder)
			continue
		}
		for _, ext := range extensions {
			// check if folder/name.ext exists
			withext := fmt.Sprintf("%s.%s", withoutext, ext)
			fname, err := filepath.Abs(filepath.Join(folder, withext))
			if err != nil {
				slog.Info("fs template not found", "folder", folder, "ext", withext, "fname", fname, "error", err)
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
	slog.Warn("Template not found", "name", name, "cwd", cwd)
	return nil, TemplateNotFound
}

// LoaderList is a composite loader that tries multiple loaders in sequence
// and returns the first successful match.
type LoaderList struct {
	// DefaultLoader is used as a fallback if no other loaders succeed.
	DefaultLoader TemplateLoader

	// loaders is the ordered list of template loaders to try.
	loaders []TemplateLoader
}

// AddLoader adds a new loader to the list of loaders to try.
// Returns the updated LoaderList for method chaining.
func (t *LoaderList) AddLoader(loader TemplateLoader) *LoaderList {
	t.loaders = append(t.loaders, loader)
	return t
}

// Load attempts to load a template with the given name by trying each loader in sequence.
// It returns the first successful match, or falls back to the DefaultLoader if all others fail.
// If cwd is provided, it's used for resolving relative paths.
// Returns TemplateNotFound if no loader can find the template.
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
