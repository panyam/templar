package templar

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

// --- WritableFS interface compliance tests ---

// TestLocalFSWriteReadRoundtrip verifies that LocalFS can write a file
// and read it back, proving the WritableFS contract for local disk.
func TestLocalFSWriteReadRoundtrip(t *testing.T) {
	dir := t.TempDir()
	lfs := NewLocalFS(dir)

	if err := lfs.WriteFile("hello.txt", []byte("world"), 0644); err != nil {
		t.Fatal(err)
	}

	data, err := fs.ReadFile(lfs, "hello.txt")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "world" {
		t.Errorf("got %q, want world", data)
	}
}

// TestLocalFSMkdirAll verifies that LocalFS.MkdirAll creates nested directories.
func TestLocalFSMkdirAll(t *testing.T) {
	dir := t.TempDir()
	lfs := NewLocalFS(dir)

	if err := lfs.MkdirAll("a/b/c", 0755); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(filepath.Join(dir, "a", "b", "c"))
	if err != nil {
		t.Fatal(err)
	}
	if !info.IsDir() {
		t.Error("expected directory")
	}
}

// TestLocalFSRemove verifies that LocalFS.Remove deletes a file.
func TestLocalFSRemove(t *testing.T) {
	dir := t.TempDir()
	lfs := NewLocalFS(dir)

	lfs.WriteFile("tmp.txt", []byte("data"), 0644)
	if err := lfs.Remove("tmp.txt"); err != nil {
		t.Fatal(err)
	}
	if _, err := lfs.Stat("tmp.txt"); err == nil {
		t.Error("file should be gone after Remove")
	}
}

// TestLocalFSRename verifies that LocalFS.Rename moves a file.
func TestLocalFSRename(t *testing.T) {
	dir := t.TempDir()
	lfs := NewLocalFS(dir)

	lfs.WriteFile("old.txt", []byte("content"), 0644)
	if err := lfs.Rename("old.txt", "new.txt"); err != nil {
		t.Fatal(err)
	}
	if _, err := lfs.Stat("old.txt"); err == nil {
		t.Error("old file should not exist after Rename")
	}
	data, err := fs.ReadFile(lfs, "new.txt")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "content" {
		t.Errorf("got %q, want content", data)
	}
}

// TestLocalFSReadDir verifies directory listing through LocalFS.
func TestLocalFSReadDir(t *testing.T) {
	dir := t.TempDir()
	lfs := NewLocalFS(dir)

	lfs.WriteFile("a.txt", []byte("a"), 0644)
	lfs.WriteFile("b.txt", []byte("b"), 0644)

	entries, err := fs.ReadDir(lfs, ".")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}
}

// --- MemFS tests ---

// TestMemFSWriteReadRoundtrip verifies that MemFS can write and read back.
func TestMemFSWriteReadRoundtrip(t *testing.T) {
	m := NewMemFS()
	if err := m.WriteFile("test.txt", []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	data, err := m.ReadFile("test.txt")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello" {
		t.Errorf("got %q, want hello", data)
	}
}

// TestMemFSOpenRead verifies that MemFS.Open returns a readable fs.File.
func TestMemFSOpenRead(t *testing.T) {
	m := NewMemFS()
	m.SetFile("doc.txt", []byte("content"))

	f, err := m.Open("doc.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	buf := make([]byte, 20)
	n, _ := f.Read(buf)
	if string(buf[:n]) != "content" {
		t.Errorf("Read got %q", buf[:n])
	}
}

// TestMemFSRemove verifies that MemFS.Remove deletes a file.
func TestMemFSRemove(t *testing.T) {
	m := NewMemFS()
	m.SetFile("del.txt", []byte("x"))

	if err := m.Remove("del.txt"); err != nil {
		t.Fatal(err)
	}
	if m.HasFile("del.txt") {
		t.Error("file should be gone")
	}
}

// TestMemFSRemoveNotFound verifies that removing a nonexistent file returns an error.
func TestMemFSRemoveNotFound(t *testing.T) {
	m := NewMemFS()
	err := m.Remove("nope.txt")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

// TestMemFSRename verifies that MemFS.Rename moves a file.
func TestMemFSRename(t *testing.T) {
	m := NewMemFS()
	m.SetFile("src.txt", []byte("data"))

	if err := m.Rename("src.txt", "dst.txt"); err != nil {
		t.Fatal(err)
	}
	if m.HasFile("src.txt") {
		t.Error("src should not exist")
	}
	if !m.HasFile("dst.txt") {
		t.Error("dst should exist")
	}
	if string(m.GetFile("dst.txt")) != "data" {
		t.Error("dst content mismatch")
	}
}

// TestMemFSReadDir verifies directory listing within MemFS.
func TestMemFSReadDir(t *testing.T) {
	m := NewMemFS()
	m.SetFile("slides/a.html", []byte("a"))
	m.SetFile("slides/b.html", []byte("b"))
	m.SetFile("slides/sub/c.html", []byte("c")) // nested, should NOT appear

	entries, err := m.ReadDir("slides")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2 (a.html, b.html)", len(entries))
	}
	if entries[0].Name() != "a.html" {
		t.Errorf("entries[0] = %q, want a.html", entries[0].Name())
	}
}

// TestMemFSStat verifies that MemFS.Stat returns file info.
func TestMemFSStat(t *testing.T) {
	m := NewMemFS()
	m.SetFile("file.txt", []byte("12345"))

	info, err := m.Stat("file.txt")
	if err != nil {
		t.Fatal(err)
	}
	if info.Name() != "file.txt" {
		t.Errorf("Name = %q", info.Name())
	}
	if info.Size() != 5 {
		t.Errorf("Size = %d, want 5", info.Size())
	}
}

// TestMemFSStatNotFound verifies that Stat on missing file returns error.
func TestMemFSStatNotFound(t *testing.T) {
	m := NewMemFS()
	_, err := m.Stat("nope.txt")
	if err == nil {
		t.Error("expected error")
	}
}

// TestMemFSWriteIsolation verifies that writes don't alias the input slice.
func TestMemFSWriteIsolation(t *testing.T) {
	m := NewMemFS()
	data := []byte("original")
	m.WriteFile("f.txt", data, 0644)

	// Mutate the input after writing
	data[0] = 'X'

	got, _ := m.ReadFile("f.txt")
	if string(got) != "original" {
		t.Errorf("got %q — write aliased input slice", got)
	}
}

// TestMemFSFileCount verifies the FileCount convenience method.
func TestMemFSFileCount(t *testing.T) {
	m := NewMemFS()
	if m.FileCount() != 0 {
		t.Errorf("empty FS count = %d", m.FileCount())
	}
	m.SetFile("a", []byte("a"))
	m.SetFile("b", []byte("b"))
	if m.FileCount() != 2 {
		t.Errorf("count = %d, want 2", m.FileCount())
	}
}

// --- FileSystemLoader with fs.FS tests ---

// TestFileSystemLoaderWithMemFS verifies that FileSystemLoader can load
// templates from an in-memory fs.FS via the FileSystems field.
func TestFileSystemLoaderWithMemFS(t *testing.T) {
	m := NewMemFS()
	m.SetFile("templates/greeting.html", []byte("<h1>Hello</h1>"))

	loader := &FileSystemLoader{
		Folders:    []FSFolder{{FS: m, Path: "templates"}},
		Extensions: []string{"html"},
	}

	tmpl, err := loader.Load("greeting", "")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if len(tmpl) != 1 {
		t.Fatalf("got %d templates, want 1", len(tmpl))
	}
	if string(tmpl[0].RawSource) != "<h1>Hello</h1>" {
		t.Errorf("content = %q", tmpl[0].RawSource)
	}
}

// TestFileSystemLoaderMixedFSAndOS verifies that some folders can use fs.FS
// while others fall back to OS (nil in FileSystems).
func TestFileSystemLoaderMixedFSAndOS(t *testing.T) {
	// OS folder
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "local.html"), []byte("from disk"), 0644)

	// MemFS folder
	m := NewMemFS()
	m.SetFile("virtual/remote.html", []byte("from memory"))

	loader := &FileSystemLoader{
		Folders:    []FSFolder{
			{Path: dir},         // nil FS = LocalFS auto-resolve
			{FS: m, Path: "virtual"},
		},
		Extensions: []string{"html"},
	}

	// Load from OS folder
	tmpl, err := loader.Load("local", "")
	if err != nil {
		t.Fatalf("Load from OS failed: %v", err)
	}
	if string(tmpl[0].RawSource) != "from disk" {
		t.Errorf("OS content = %q", tmpl[0].RawSource)
	}

	// Load from MemFS folder
	tmpl, err = loader.Load("remote", "")
	if err != nil {
		t.Fatalf("Load from MemFS failed: %v", err)
	}
	if string(tmpl[0].RawSource) != "from memory" {
		t.Errorf("MemFS content = %q", tmpl[0].RawSource)
	}
}

// TestFileSystemLoaderNotFound verifies that missing templates return TemplateNotFound.
func TestFileSystemLoaderNotFound(t *testing.T) {
	m := NewMemFS()
	loader := &FileSystemLoader{
		Folders:    []FSFolder{{FS: m, Path: "templates"}},
		Extensions: []string{"html"},
	}

	_, err := loader.Load("nonexistent", "")
	if err != TemplateNotFound {
		t.Errorf("err = %v, want TemplateNotFound", err)
	}
}

// TestFileSystemLoaderBackwardCompat verifies that existing code using only
// Folders (no FileSystems) continues to work exactly as before.
func TestFileSystemLoaderBackwardCompat(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "page.html"), []byte("old way"), 0644)

	loader := NewFileSystemLoader(LocalFolder(dir))
	tmpl, err := loader.Load("page", "")
	if err != nil {
		t.Fatalf("backward compat Load failed: %v", err)
	}
	if string(tmpl[0].RawSource) != "old way" {
		t.Errorf("content = %q", tmpl[0].RawSource)
	}
}
