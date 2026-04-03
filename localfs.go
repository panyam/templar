package templar

import (
	"io/fs"
	"os"
	"path/filepath"
)

// LocalFS implements WritableFS backed by the local operating system filesystem.
// All paths are relative to Root.
type LocalFS struct {
	// Root is the absolute base directory. All operations are relative to it.
	Root string
}

// NewLocalFS creates a WritableFS backed by the local filesystem at the given root.
// The root should be an absolute path.
func NewLocalFS(root string) *LocalFS {
	return &LocalFS{Root: root}
}

// Open implements fs.FS.
func (f *LocalFS) Open(name string) (fs.File, error) {
	return os.Open(f.abs(name))
}

// ReadDir implements fs.ReadDirFS.
func (f *LocalFS) ReadDir(name string) ([]fs.DirEntry, error) {
	return os.ReadDir(f.abs(name))
}

// ReadFile implements fs.ReadFileFS.
func (f *LocalFS) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(f.abs(name))
}

// Stat implements fs.StatFS.
func (f *LocalFS) Stat(name string) (fs.FileInfo, error) {
	return os.Stat(f.abs(name))
}

// WriteFile implements WritableFS.
func (f *LocalFS) WriteFile(name string, data []byte, perm fs.FileMode) error {
	return os.WriteFile(f.abs(name), data, perm)
}

// MkdirAll implements WritableFS.
func (f *LocalFS) MkdirAll(path string, perm fs.FileMode) error {
	return os.MkdirAll(f.abs(path), perm)
}

// Remove implements WritableFS.
func (f *LocalFS) Remove(name string) error {
	return os.Remove(f.abs(name))
}

// Rename implements WritableFS.
func (f *LocalFS) Rename(oldname, newname string) error {
	return os.Rename(f.abs(oldname), f.abs(newname))
}

// AbsPath returns the absolute path for a relative name within the FS.
func (f *LocalFS) AbsPath(name string) string {
	return f.abs(name)
}

func (f *LocalFS) abs(name string) string {
	return filepath.Join(f.Root, name)
}
