# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and versions follow
[Semantic Versioning](https://semver.org/). Releases before v0.1.3 are on the
GitHub Releases page and are not back-filled.

## [v0.1.3] - 2026-10-03

### Fixed

- `MemFS.ReadDir` now lists subdirectories, so `fs.WalkDir` over a MemFS
  reaches nested files (issue 9, PR 11).
- `MemFS` is safe for concurrent use. A write during a read used to crash with
  `concurrent map read and map write` (PR 11).

### Changed

- `MemFS` wraps `goutils/memfs.FS` (goutils v0.1.14), shared with goapplib and
  agni. `MkdirAll` creates real empty directories, `Remove` on a directory with
  entries fails with `memfs.ErrNotEmpty`, a path can't be both a file and a
  directory, `SetFile` panics on an invalid or colliding path, and `GetFile`
  returns a copy (issue 10, PR 11).
- The minimum Go version is now 1.25.0, because goutils v0.1.14 requires it.
