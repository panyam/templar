# Templar Integration Guide

Quick reference for using Templar's vendoring and SourceLoader features. See docs/vendoring.md for full documentation.

## Overview

Templar supports loading templates from external sources (like GitHub repositories) using the `@source/` prefix syntax. This enables:

- Sharing template libraries across projects
- Explicit dependency management
- Reproducible builds with lock files

## Quick Setup

### 1. Create templar.yaml

In your templates directory:

```yaml
sources:
  goapplib:
    url: github.com/panyam/goapplib
    path: templates
    ref: main

vendor_dir: templar_modules
search_paths:
  - .
  - ./templar_modules
```

### 2. Fetch Dependencies

```bash
cd templates
templar get
```

This creates:
```
templates/
├── templar.yaml
├── templar.lock           # Lock file (auto-generated)
├── templar_modules/
│   └── goapplib/          # Vendored templates
│       ├── BasePage.html
│       └── components/
└── MyPage.html            # Your templates
```

### 3. Reference in Templates

Use `@sourcename/` prefix to reference vendored templates:

```html
{{# namespace "Base" "@goapplib/BasePage.html" #}}
                     ^^^^^^^^^^
                     Source name from templar.yaml

{{ define "MyPage" }}
{{ template "Base:BasePage" . }}
{{ end }}
```

### 4. Configure Go Loader

```go
import (
    "path/filepath"
    tmplr "github.com/panyam/templar"
)

templates := tmplr.NewTemplateGroup()

// CRITICAL: Use absolute path
configPath, _ := filepath.Abs(filepath.Join(TEMPLATES_FOLDER, "templar.yaml"))
sourceLoader, err := tmplr.NewSourceLoaderFromConfig(configPath)
if err != nil {
    // Fallback to basic loader
    templates.Loader = tmplr.NewFileSystemLoader(TEMPLATES_FOLDER)
} else {
    templates.Loader = sourceLoader
}
```

---

## Critical Patterns

### 1. Use Absolute Path for Config

```go
// CORRECT
configPath, _ := filepath.Abs(filepath.Join(TEMPLATES_FOLDER, "templar.yaml"))
sourceLoader, _ := tmplr.NewSourceLoaderFromConfig(configPath)

// WRONG - Relative paths may fail
sourceLoader, _ := tmplr.NewSourceLoaderFromConfig("./templates/templar.yaml")
```

**Why?** The SourceLoader resolves `vendor_dir` and `search_paths` relative to the config file location. With relative paths, the working directory affects resolution.

### 2. @ Prefix is Required for External Sources

```html
<!-- CORRECT -->
{{# namespace "EL" "@goapplib/components/EntityListing.html" #}}

<!-- WRONG - Looks in search_paths, not vendored sources -->
{{# namespace "EL" "goapplib/components/EntityListing.html" #}}
```

### 3. Source Names Are Case-Sensitive

```yaml
sources:
  GoAppLib:    # This name...
    url: github.com/panyam/goapplib
```

```html
<!-- Must match exactly -->
{{# namespace "EL" "@GoAppLib/..." #}}    <!-- Correct -->
{{# namespace "EL" "@goapplib/..." #}}    <!-- Wrong - case mismatch -->
```

### 4. Run `templar get` After Cloning

If `templar_modules/` isn't committed:

```bash
git clone myrepo
cd myrepo/templates
templar get              # Fetch dependencies
```

---

## SourceLoader API

### NewSourceLoaderFromConfig

Creates a SourceLoader from a templar.yaml file:

```go
func NewSourceLoaderFromConfig(configPath string) (*SourceLoader, error)
```

### NewSourceLoaderFromDir

Searches for templar.yaml starting from a directory:

```go
func NewSourceLoaderFromDir(dir string) (*SourceLoader, error)
```

### VendorConfig Structure

```go
type VendorConfig struct {
    Sources     map[string]SourceConfig  // Named sources
    VendorDir   string                   // Where vendored files live
    SearchPaths []string                 // Template search order
    RequireLock bool                     // Require lock file
}

type SourceConfig struct {
    URL     string   // Repository URL
    Path    string   // Subdirectory within repo
    Version string   // Semantic version (takes precedence)
    Ref     string   // Git ref (branch/commit)
    Include []string // Glob patterns to include
    Exclude []string // Glob patterns to exclude
}
```

---

## Template Resolution

The SourceLoader resolves templates in this order:

1. **@source/path** - Look up source, resolve to vendored location
2. **./relative/path** - Relative to current template
3. **path** - Search in `search_paths` order

### Resolution Example

```yaml
# templar.yaml
sources:
  goapplib:
    url: github.com/panyam/goapplib
    path: templates
    ref: main

vendor_dir: templar_modules
search_paths:
  - .
  - ./templar_modules
```

```html
{{# namespace "Base" "@goapplib/BasePage.html" #}}
```

Resolves to:
```
./templar_modules/goapplib/BasePage.html
```

Note: The flat structure uses just the source name, not the full GitHub path.

---

## Common Errors

### Error: `source 'xyz' not defined in templar.yaml`
**Cause**: Using `@xyz/` but no source named `xyz` in config
**Fix**: Add source to templar.yaml

### Error: `template not found` (after @source reference)
**Cause**: templar_modules not fetched, or wrong path
**Fix**: Run `templar get`, verify file exists in templar_modules

### Error: `failed to read config file`
**Cause**: templar.yaml not found or relative path issue
**Fix**: Use `filepath.Abs()` for configPath

### Error: `invalid source pattern '@xyz': expected @sourcename/path`
**Cause**: Using `@xyz` without a path
**Fix**: Use full path like `@xyz/file.html`

---

## Deployment Strategies

### Check In templar_modules (Recommended)

```bash
git add templar_modules/
git add templar.lock
```

**Pros**: Reproducible builds, no network needed at deploy time

### Lock File Only

```bash
echo "templar_modules/" >> .gitignore
git add templar.lock
```

In CI:
```bash
templar get   # Fetches using lock file
```

**Pros**: Smaller repo, cleaner diffs

---

## Reference Examples

- **excaliframe/site/templates/** - Simple site with goapplib dependency
- **lilbattle/web/templates/** - Full app with multiple template sources
