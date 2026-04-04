package templar

import (
	"testing"
)


// TestExtractTarGzFS verifies that extractTarGzFS writes files to a MemFS
// from an in-memory tarball — zero disk I/O.
func TestExtractTarGzFS(t *testing.T) {
	tarball := makeTestTarGz(t, TarGzOpts{TopDir: "repo-abc123"}, map[string]string{
		"hello.txt":       "world",
		"sub/nested.html": "<h1>Nested</h1>",
	})

	mfs := NewMemFS()
	count, err := extractTarGzFS(mfs, tarball, "vendor/src", "", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Errorf("extracted %d files, want 2", count)
	}

	// Verify files written to FS
	data, err := mfs.ReadFile("vendor/src/hello.txt")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "world" {
		t.Errorf("hello.txt = %q, want world", data)
	}

	data, err = mfs.ReadFile("vendor/src/sub/nested.html")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "<h1>Nested</h1>" {
		t.Errorf("nested.html = %q", data)
	}
}

// TestExtractTarGzFSWithSubPath verifies subPath filtering.
func TestExtractTarGzFSWithSubPath(t *testing.T) {
	tarball := makeTestTarGz(t, TarGzOpts{TopDir: "repo-abc123"}, map[string]string{
		"templates/a.html": "A",
		"templates/b.html": "B",
		"other/c.txt":      "C",
	})

	mfs := NewMemFS()
	count, err := extractTarGzFS(mfs, tarball, "out", "templates", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Errorf("extracted %d files, want 2 (only templates/)", count)
	}

	if !mfs.HasFile("out/a.html") {
		t.Error("missing a.html")
	}
	if mfs.HasFile("out/c.txt") {
		t.Error("c.txt should not be extracted (outside subPath)")
	}
}

// TestExtractTarGzFSWithPatterns verifies include/exclude filtering.
func TestExtractTarGzFSWithPatterns(t *testing.T) {
	tarball := makeTestTarGz(t, TarGzOpts{TopDir: "repo-abc123"}, map[string]string{
		"a.html": "A",
		"b.css":  "B",
		"c.html": "C",
	})

	mfs := NewMemFS()
	count, err := extractTarGzFS(mfs, tarball, "out", "", []string{"*.html"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Errorf("extracted %d files, want 2 (only *.html)", count)
	}

	if mfs.HasFile("out/b.css") {
		t.Error("b.css should be excluded by include pattern")
	}
}

// TestExtractTarGzFSWithPAX verifies that PAX global headers are skipped.
func TestExtractTarGzFSWithPAX(t *testing.T) {
	tarball := makeTestTarGz(t, TarGzOpts{TopDir: "repo-abc", IncludePAX: true}, map[string]string{
		"hello.txt": "world",
	})
	mfs := NewMemFS()
	count, err := extractTarGzFS(mfs, tarball, "out", "", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("extracted %d, want 1", count)
	}
	if mfs.HasFile("out/pax_global_header") {
		t.Error("PAX header should not be extracted")
	}
}

// TestExtractTarGzFSWithExclude verifies exclude pattern filtering.
func TestExtractTarGzFSWithExclude(t *testing.T) {
	tarball := makeTestTarGz(t, TarGzOpts{TopDir: "repo-abc"}, map[string]string{
		"keep.html": "keep", "remove.css": "rm", "also.html": "also",
	})
	mfs := NewMemFS()
	count, _ := extractTarGzFS(mfs, tarball, "out", "", nil, []string{"*.css"})
	if count != 2 {
		t.Errorf("extracted %d, want 2", count)
	}
	if mfs.HasFile("out/remove.css") {
		t.Error("*.css should be excluded")
	}
}

// TestExtractTarGzFSIncludeAndExclude verifies both filters together.
func TestExtractTarGzFSIncludeAndExclude(t *testing.T) {
	tarball := makeTestTarGz(t, TarGzOpts{TopDir: "repo-abc"}, map[string]string{
		"a.html": "A", "b.html": "B", "c.css": "C", "test.html": "test",
	})
	mfs := NewMemFS()
	count, _ := extractTarGzFS(mfs, tarball, "out", "", []string{"*.html"}, []string{"test*"})
	if count != 2 {
		t.Errorf("extracted %d, want 2 (a.html, b.html)", count)
	}
	if mfs.HasFile("out/c.css") {
		t.Error("c.css excluded by include")
	}
	if mfs.HasFile("out/test.html") {
		t.Error("test.html excluded by exclude")
	}
}

// TestWriteLockFileFS verifies lock file writing to MemFS.
func TestWriteLockFileFS(t *testing.T) {
	mfs := NewMemFS()
	lock := &VendorLock{
		Version: 1,
		Sources: map[string]LockedSource{
			"test": {URL: "github.com/test/repo", ResolvedCommit: "abc123"},
		},
	}
	info := ToolInfo{Name: "test", ConfigNames: []string{"test.yaml"}, VendorDir: "modules", FetchCmd: "test fetch"}

	err := WriteLockFileFS(mfs, "test.lock", lock, info)
	if err != nil {
		t.Fatal(err)
	}

	if !mfs.HasFile("test.lock") {
		t.Fatal("lock file not written")
	}

	// Verify it can be loaded back
	loaded, err := LoadLockFileFS(mfs, "test.lock")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Version != 1 {
		t.Errorf("version = %d, want 1", loaded.Version)
	}
	if loaded.Sources["test"].ResolvedCommit != "abc123" {
		t.Errorf("commit = %q", loaded.Sources["test"].ResolvedCommit)
	}
}

// TestWriteVendorReadmeFS verifies readme writing to MemFS.
func TestWriteVendorReadmeFS(t *testing.T) {
	mfs := NewMemFS()
	info := ToolInfo{Name: "slyds", ConfigNames: []string{".slyds.yaml"}, VendorDir: ".slyds-modules", FetchCmd: "slyds update", ProjectURL: "https://github.com/panyam/slyds"}

	err := WriteVendorReadmeFS(mfs, ".slyds-modules", info)
	if err != nil {
		t.Fatal(err)
	}

	data, err := mfs.ReadFile(".slyds-modules/README.md")
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Error("readme is empty")
	}
}
