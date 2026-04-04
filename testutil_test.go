package templar

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"testing"
)

// TarGzOpts configures test tarball generation.
type TarGzOpts struct {
	TopDir     string // "" = flat, "repo-abc123" = nested under top-level dir (GitHub style)
	IncludePAX bool   // prepend a PAX global header (tests PAX header skipping)
}

// makeTestTarGz creates an in-memory gzipped tarball for testing.
// Supports flat or nested layout, optional PAX headers — one function for all variations.
func makeTestTarGz(t *testing.T, opts TarGzOpts, files map[string]string) *bytes.Buffer {
	t.Helper()

	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)

	// PAX global header (tests that extractors skip these correctly)
	if opts.IncludePAX {
		tw.WriteHeader(&tar.Header{
			Name:     "pax_global_header",
			Typeflag: tar.TypeXGlobalHeader,
		})
	}

	// Top-level directory (GitHub tarball style)
	if opts.TopDir != "" {
		tw.WriteHeader(&tar.Header{Name: opts.TopDir + "/", Typeflag: tar.TypeDir, Mode: 0755})
	}

	for name, content := range files {
		fullName := name
		if opts.TopDir != "" {
			fullName = opts.TopDir + "/" + name
		}

		isDir := len(content) == 0 && name[len(name)-1] == '/'
		if isDir {
			tw.WriteHeader(&tar.Header{Name: fullName, Mode: 0755, Typeflag: tar.TypeDir})
		} else {
			tw.WriteHeader(&tar.Header{
				Name:     fullName,
				Size:     int64(len(content)),
				Mode:     0644,
				Typeflag: tar.TypeReg,
			})
			tw.Write([]byte(content))
		}
	}

	tw.Close()
	gzw.Close()
	return &buf
}
