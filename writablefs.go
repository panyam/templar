package templar

import "io/fs"

// WritableFS is a complete read-write filesystem interface for template storage.
// It extends fs.FS with efficient read operations (ReadFile, ReadDir) and
// write operations (WriteFile, MkdirAll, Remove, Rename).
//
// Portable across local filesystems, S3, IndexedDB (WASM), in-memory (tests), etc.
// All paths are relative to the filesystem root — no absolute paths.
type WritableFS interface {
	fs.FS

	// ReadFile reads the named file and returns its contents.
	// Equivalent to fs.ReadFileFS but required (not optional) here.
	ReadFile(name string) ([]byte, error)

	// ReadDir reads the named directory and returns a list of directory entries
	// sorted by filename.
	ReadDir(name string) ([]fs.DirEntry, error)

	// WriteFile writes data to the named file, creating it if necessary.
	// If the file exists, it is truncated before writing.
	WriteFile(name string, data []byte, perm fs.FileMode) error

	// MkdirAll creates a directory path and all parents that don't exist.
	MkdirAll(path string, perm fs.FileMode) error

	// Remove deletes the named file or empty directory.
	Remove(name string) error

	// Rename renames (moves) a file within the filesystem.
	Rename(oldname, newname string) error
}
