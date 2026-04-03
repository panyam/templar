package templar

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// FileSystemLoader loads templates from the file system based on
// a set of directories and file extensions.
//
// Each folder can optionally be backed by an fs.FS via the FileSystems slice.
// When FileSystems[i] is set, Folders[i] is a relative path within that FS.
// When FileSystems[i] is nil (or FileSystems is shorter than Folders),
// Folders[i] is an OS filesystem path (original behavior).
type FileSystemLoader struct {
	// Folders is a list of directories to search for templates.
	Folders []string

	// FileSystems optionally backs each Folder with an fs.FS.
	// FileSystems[i] serves Folders[i]. If nil or missing, OS filesystem is used.
	FileSystems []fs.FS

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

	// Build search list: (folder, fs.FS or nil)
	type searchEntry struct {
		folder string
		fsys   fs.FS // nil = use OS
	}
	var entries []searchEntry
	for i, folder := range g.Folders {
		var fsys fs.FS
		if i < len(g.FileSystems) {
			fsys = g.FileSystems[i]
		}
		entries = append(entries, searchEntry{folder, fsys})
	}

	if cwd != "" {
		cwdEntry := searchEntry{folder: cwd} // cwd always uses OS (backward compat)
		if isRelative {
			entries = []searchEntry{cwdEntry}
		} else {
			entries = append(entries, cwdEntry)
		}
	}

	for _, entry := range entries {
		if !g.folderExists(entry.folder, entry.fsys) {
			continue
		}
		for _, ext := range extensions {
			withext := fmt.Sprintf("%s.%s", withoutext, ext)
			contents, fullPath, err := g.readTemplate(entry.folder, withext, entry.fsys)
			if err != nil {
				continue
			}
			return []*Template{{RawSource: contents, Path: fullPath}}, nil
		}
	}
	slog.Warn("Template not found", "name", name, "cwd", cwd)
	return nil, TemplateNotFound
}

// folderExists checks if a folder exists, using fs.FS or OS as appropriate.
func (g *FileSystemLoader) folderExists(folder string, fsys fs.FS) bool {
	if fsys != nil {
		info, err := fs.Stat(fsys, folder)
		return err == nil && info.IsDir()
	}
	// OS path
	abs, err := filepath.Abs(folder)
	if err != nil {
		slog.Debug("Invalid folder", "folder", folder)
		return false
	}
	info, err := os.Stat(abs)
	if os.IsNotExist(err) {
		slog.Debug("folder does not exist", "folder", abs)
		return false
	}
	if !info.IsDir() {
		slog.Debug("folder is not a directory", "folder", abs)
		return false
	}
	return true
}

// readTemplate reads a template file from within a folder, using fs.FS or OS.
// Returns the file contents and the resolved path.
func (g *FileSystemLoader) readTemplate(folder, name string, fsys fs.FS) ([]byte, string, error) {
	if fsys != nil {
		fpath := folder + "/" + name
		data, err := fs.ReadFile(fsys, fpath)
		if err != nil {
			return nil, "", err
		}
		return data, fpath, nil
	}
	// OS path
	fname, err := filepath.Abs(filepath.Join(folder, name))
	if err != nil {
		return nil, "", err
	}
	info, err := os.Stat(fname)
	if err != nil || info.IsDir() {
		return nil, "", fmt.Errorf("not found: %s", fname)
	}
	data, err := os.ReadFile(filepath.Clean(fname))
	return data, fname, err
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
	if t.DefaultLoader != nil {
		return t.DefaultLoader.Load(name, cwd)
	}
	return nil, TemplateNotFound
}
