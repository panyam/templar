package templar

import (
	"io"
	"io/fs"
	"path"
	"sort"
	"strings"
	"time"
)

// MemFS implements WritableFS as a fully in-memory filesystem.
// Useful for testing, WASM/IndexedDB backends, and any context where
// files don't live on a local filesystem.
//
// Not safe for concurrent use without external synchronization.
type MemFS struct {
	files map[string][]byte
}

// NewMemFS creates an empty in-memory filesystem.
func NewMemFS() *MemFS {
	return &MemFS{files: make(map[string][]byte)}
}

// SetFile adds or overwrites a file. Convenience for test setup.
func (m *MemFS) SetFile(name string, data []byte) {
	m.files[name] = data
}

// GetFile returns the raw bytes of a file, or nil if it doesn't exist.
// Convenience for test assertions.
func (m *MemFS) GetFile(name string) []byte {
	return m.files[name]
}

// HasFile returns true if the named file exists.
func (m *MemFS) HasFile(name string) bool {
	_, ok := m.files[name]
	return ok
}

// Open implements fs.FS.
func (m *MemFS) Open(name string) (fs.File, error) {
	data, ok := m.files[name]
	if !ok {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}
	return &memFile{name: name, data: data}, nil
}

// ReadFile implements fs.ReadFileFS.
func (m *MemFS) ReadFile(name string) ([]byte, error) {
	data, ok := m.files[name]
	if !ok {
		return nil, &fs.PathError{Op: "read", Path: name, Err: fs.ErrNotExist}
	}
	cp := make([]byte, len(data))
	copy(cp, data)
	return cp, nil
}

// Stat implements fs.StatFS.
// Returns file info for files, and a synthetic directory entry for any prefix
// that has files under it (e.g., "slides" exists if "slides/01.html" exists).
func (m *MemFS) Stat(name string) (fs.FileInfo, error) {
	// Check for exact file match
	if data, ok := m.files[name]; ok {
		return &memFileInfo{name: path.Base(name), size: int64(len(data))}, nil
	}
	// Check if it's a directory (any file has this as a prefix)
	prefix := name + "/"
	if name == "." || name == "" {
		return &memFileInfo{name: ".", size: 0, dir: true}, nil
	}
	for k := range m.files {
		if strings.HasPrefix(k, prefix) {
			return &memFileInfo{name: path.Base(name), size: 0, dir: true}, nil
		}
	}
	return nil, &fs.PathError{Op: "stat", Path: name, Err: fs.ErrNotExist}
}

// ReadDir implements fs.ReadDirFS.
func (m *MemFS) ReadDir(name string) ([]fs.DirEntry, error) {
	prefix := name + "/"
	if name == "." || name == "" {
		prefix = ""
	}
	seen := map[string]bool{}
	var entries []fs.DirEntry
	for k := range m.files {
		if !strings.HasPrefix(k, prefix) {
			continue
		}
		rest := strings.TrimPrefix(k, prefix)
		if rest == "" {
			continue
		}
		// Only immediate children (no further slashes)
		if i := strings.Index(rest, "/"); i >= 0 {
			continue
		}
		if seen[rest] {
			continue
		}
		seen[rest] = true
		entries = append(entries, &memDirEntry{name: rest, size: int64(len(m.files[k]))})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	return entries, nil
}

// WriteFile implements WritableFS.
func (m *MemFS) WriteFile(name string, data []byte, _ fs.FileMode) error {
	cp := make([]byte, len(data))
	copy(cp, data)
	m.files[name] = cp
	return nil
}

// MkdirAll implements WritableFS (no-op for in-memory FS).
func (m *MemFS) MkdirAll(_ string, _ fs.FileMode) error { return nil }

// Remove implements WritableFS.
func (m *MemFS) Remove(name string) error {
	if _, ok := m.files[name]; !ok {
		return &fs.PathError{Op: "remove", Path: name, Err: fs.ErrNotExist}
	}
	delete(m.files, name)
	return nil
}

// Rename implements WritableFS.
func (m *MemFS) Rename(oldname, newname string) error {
	data, ok := m.files[oldname]
	if !ok {
		return &fs.PathError{Op: "rename", Path: oldname, Err: fs.ErrNotExist}
	}
	m.files[newname] = data
	delete(m.files, oldname)
	return nil
}

// FileCount returns the total number of files in the FS. Convenience for tests.
func (m *MemFS) FileCount() int {
	return len(m.files)
}

// --- fs.File implementation ---

type memFile struct {
	name   string
	data   []byte
	offset int
}

func (f *memFile) Stat() (fs.FileInfo, error) {
	return &memFileInfo{name: path.Base(f.name), size: int64(len(f.data))}, nil
}

func (f *memFile) Read(b []byte) (int, error) {
	if f.offset >= len(f.data) {
		return 0, io.EOF
	}
	n := copy(b, f.data[f.offset:])
	f.offset += n
	return n, nil
}

func (f *memFile) Close() error { return nil }

type memFileInfo struct {
	name string
	size int64
	dir  bool
}

func (fi *memFileInfo) Name() string        { return fi.name }
func (fi *memFileInfo) Size() int64         { return fi.size }
func (fi *memFileInfo) Mode() fs.FileMode   { if fi.dir { return fs.ModeDir | 0755 }; return 0444 }
func (fi *memFileInfo) ModTime() time.Time  { return time.Time{} }
func (fi *memFileInfo) IsDir() bool         { return fi.dir }
func (fi *memFileInfo) Sys() any            { return nil }

type memDirEntry struct {
	name string
	size int64
}

func (de *memDirEntry) Name() string               { return de.name }
func (de *memDirEntry) IsDir() bool                { return false }
func (de *memDirEntry) Type() fs.FileMode          { return 0 }
func (de *memDirEntry) Info() (fs.FileInfo, error)  { return &memFileInfo{name: de.name, size: de.size}, nil }
