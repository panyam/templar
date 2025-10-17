package templar

import (
	"embed"
	"fmt"
	"io"
	"log"
	"log/slog"
	"path/filepath"
)

// EmbedFSLoader loads templates from the file system based on
// a set of directories and file extensions.
type EmbedFSLoader struct {
	// Embeds is a list of directories to search for templates.
	Embeds []embed.FS

	// Extensions is a list of file extensions to consider as templates.
	Extensions []string
}

// NewEmbedFSLoader creates a new file system loader that will search
// in the provided folders for template files.
// By default, it recognizes files with .tmpl, .tmplus, and .html extensions.
func NewEmbedFSLoader(fss ...embed.FS) *EmbedFSLoader {
	return &EmbedFSLoader{
		Embeds: fss,
		Extensions: []string{
			"tmpl", "tmplus", "html",
		},
	}
}

// Load attempts to find and load a template with the given name.
// If the name includes an extension, only files with that extension are considered.
// Otherwise, files with any of the loader's recognized extensions are searched.
// The cwd parameter is ignored as we can only provided templates from embedded FS
// Returns the loaded templates or TemplateNotFound if no matching templates were found.
func (g *EmbedFSLoader) Load(name string, _ string) (template []*Template, err error) {
	ext := filepath.Ext(name)
	extensions := g.Extensions
	withoutext := name
	if ext != "" {
		extensions = []string{ext[1:]}
		withoutext = name[:len(name)-len(ext)]
	}
	// log.Printf("Loading in CWD: %s, Name: %s, WithoutExt: %s, Ext: %s, Embeds: %v", cwd, name, withoutext, ext, folders)
	for _, embedfs := range g.Embeds {
		for _, ext := range extensions {
			// check if folder/name.ext exists
			withext := fmt.Sprintf("%s.%s", withoutext, ext)
			f, err := embedfs.Open(withext)
			if err != nil {
				log.Println("Found error: ", withext, err)
			} else {
				// Found it so laod it
				contents, err := io.ReadAll(f)
				return []*Template{{RawSource: contents, Path: withext}}, err
			}
		}
	}
	slog.Warn("Template not found", "name", name)
	return nil, TemplateNotFound
}
