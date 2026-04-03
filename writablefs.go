package templar

import "io/fs"

// WritableFS extends fs.FS with write operations, making template storage
// portable across local filesystems, S3, IndexedDB (WASM), in-memory (tests), etc.
//
// Implementations must satisfy fs.FS (read via Open) and additionally support
// file creation, modification, deletion, and renaming. All paths are relative
// to the filesystem root — no absolute paths.
type WritableFS interface {
	fs.FS

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
