package templar

import (
	"fmt"
	"io/fs"
	"log/slog"
	"path"
	"strings"
)

// FSFolder pairs a filesystem with a folder path within it.
type FSFolder struct {
	FS   fs.FS  // filesystem to search in
	Path string // folder path within the FS
}

// FileSystemLoader loads templates from one or more filesystem folders.
// Each folder is backed by an fs.FS — use NewLocalFS for local disk, NewMemFS for tests.
type FileSystemLoader struct {
	// Folders is the list of FS+path pairs to search for templates.
	Folders []FSFolder

	// Extensions is a list of file extensions to consider as templates.
	Extensions []string
}

// NewFileSystemLoader creates a loader that searches the given FS+path pairs.
// Default extensions: .tmpl, .tmplus, .html.
func NewFileSystemLoader(folders ...FSFolder) *FileSystemLoader {
	return &FileSystemLoader{
		Folders: folders,
		Extensions: []string{
			"tmpl", "tmplus", "html",
		},
	}
}

// LocalFolder is a convenience for creating an FSFolder from a local directory path.
func LocalFolder(dir string) FSFolder {
	return FSFolder{FS: NewLocalFS(dir), Path: "."}
}

// Load attempts to find and load a template with the given name.
func (g *FileSystemLoader) Load(name string, cwd string) (template []*Template, err error) {
	ext := path.Ext(name)
	extensions := g.Extensions
	withoutext := name
	if ext != "" {
		extensions = []string{ext[1:]}
		withoutext = name[:len(name)-len(ext)]
	}
	isRelative := strings.HasPrefix(name, "./") || strings.HasPrefix(name, "../")

	entries := g.Folders
	if cwd != "" {
		// cwd is always an FS path — find which folder's FS it belongs to, or assume first
		cwdEntry := FSFolder{Path: cwd}
		if len(g.Folders) > 0 {
			cwdEntry.FS = g.Folders[0].FS
		}
		if isRelative {
			entries = []FSFolder{cwdEntry}
		} else {
			entries = append(append([]FSFolder{}, entries...), cwdEntry)
		}
	}

	for _, entry := range entries {
		if !g.folderExists(entry) {
			continue
		}
		for _, ext := range extensions {
			withext := fmt.Sprintf("%s.%s", withoutext, ext)
			contents, fullPath, err := g.readTemplate(entry, withext)
			if err != nil {
				continue
			}
			return []*Template{{RawSource: contents, Path: fullPath}}, nil
		}
	}
	slog.Warn("Template not found", "name", name, "cwd", cwd)
	return nil, TemplateNotFound
}

// resolve ensures FSFolder has an FS set — defaults to LocalFS if nil.
func (entry *FSFolder) resolve() {
	if entry.FS == nil {
		entry.FS = NewLocalFS(entry.Path)
		entry.Path = "."
	}
}

// folderExists checks if a folder exists in its FS.
func (g *FileSystemLoader) folderExists(entry FSFolder) bool {
	entry.resolve()
	info, err := fs.Stat(entry.FS, entry.Path)
	if err != nil {
		// "." always exists conceptually
		if entry.Path == "." || entry.Path == "" {
			return true
		}
		slog.Debug("folder does not exist", "folder", entry.Path)
		return false
	}
	return info.IsDir()
}

// readTemplate reads a template file from an FSFolder.
func (g *FileSystemLoader) readTemplate(entry FSFolder, name string) ([]byte, string, error) {
	entry.resolve()
	fpath := name
	if entry.Path != "" && entry.Path != "." {
		fpath = entry.Path + "/" + name
	}
	data, err := fs.ReadFile(entry.FS, fpath)
	if err != nil {
		return nil, "", err
	}
	return data, fpath, nil
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
func (t *LoaderList) AddLoader(loader TemplateLoader) *LoaderList {
	t.loaders = append(t.loaders, loader)
	return t
}

// Load attempts to load a template with the given name by trying each loader in sequence.
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

// LocalFolders converts a list of directory paths to FSFolder entries.
// Convenience for migrating code that passes string paths.
func LocalFolders(dirs ...string) []FSFolder {
	var folders []FSFolder
	for _, d := range dirs {
		folders = append(folders, LocalFolder(d))
	}
	return folders
}
