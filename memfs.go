package templar

import (
	"fmt"
	"io/fs"

	"github.com/panyam/goutils/memfs"
)

// MemFS implements WritableFS as a fully in-memory filesystem.
// Useful for testing, WASM/IndexedDB backends, and any context where
// files don't live on a local filesystem.
//
// It wraps goutils/memfs.FS, so it is safe for concurrent use, a file already
// open keeps its contents when the file is rewritten, and directories are implied
// by file paths ("slides/01.html" makes "slides" a directory). A path is a file
// or a directory, never both. MkdirAll creates real empty directories, and
// Remove of a directory that still has entries fails with memfs.ErrNotEmpty.
type MemFS struct {
	memfs.FS
}

// NewMemFS creates an empty in-memory filesystem.
func NewMemFS() *MemFS {
	return &MemFS{}
}

// SetFile adds or overwrites a file. Convenience for test setup.
// It keeps data rather than copying it, so don't modify data afterwards.
// It panics if name is not a valid fs path or collides with a directory, since a
// test that sets up an impossible tree should fail at the line that did it.
func (m *MemFS) SetFile(name string, data []byte) {
	if err := m.Put(name, data); err != nil {
		panic(fmt.Sprintf("MemFS.SetFile: %v", err))
	}
}

// GetFile returns a copy of a file's bytes, or nil if it doesn't exist or is a
// directory. Convenience for test assertions.
func (m *MemFS) GetFile(name string) []byte {
	data, err := m.ReadFile(name)
	if err != nil {
		return nil
	}
	return data
}

// HasFile returns true if the named file exists. A directory is not a file.
func (m *MemFS) HasFile(name string) bool {
	info, err := m.Stat(name)
	return err == nil && !info.IsDir()
}

// FileCount returns the total number of files in the FS, not counting
// directories. Convenience for tests.
func (m *MemFS) FileCount() int {
	n := 0
	_ = fs.WalkDir(m, ".", func(_ string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			n++
		}
		return nil
	})
	return n
}
