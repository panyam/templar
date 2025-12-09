# Template Vendoring with External Sources

Templar supports loading templates from external sources like GitHub repositories. This enables sharing template libraries across projects while maintaining explicit dependency management and reproducible builds.

## Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  Your Project                                                               │
│                                                                             │
│  templates/                                                                 │
│  ├── pages/                                                                 │
│  │   └── dashboard.html    ──► {{# namespace "EL" "@goapplib/..." #}}       │
│  └── components/                                                            │
│      └── custom.html                                                        │
│                                                                             │
│  templar.yaml              ──► sources:                                     │
│                                  goapplib:                                  │
│                                    url: github.com/panyam/goapplib          │
│                                    path: templates                          │
│                                    ref: v1.2.0                              │
│                                                                             │
│  templar_modules/          ──► Vendored dependencies (after templar get)    │
│  └── github.com/                                                            │
│      └── panyam/                                                            │
│          └── goapplib/                                                      │
│              └── templates/                                                 │
│                  └── EntityListing.html                                     │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Quick Start

### 1. Configure Sources

Create a `templar.yaml` in your project root:

```yaml
# Define external template sources
sources:
  goapplib:
    url: github.com/panyam/goapplib
    path: templates              # Subdirectory within repo
    ref: v1.2.0                  # Tag, branch, or commit hash

  shared:
    url: github.com/myorg/shared-templates
    ref: main

# Where vendored templates are stored
vendor_dir: ./templar_modules

# Template search paths (in order)
search_paths:
  - ./templates                  # Local templates first
  - ./templar_modules            # Then vendored dependencies
```

### 2. Fetch Dependencies

```bash
templar get
```

This downloads all configured sources to `templar_modules/`:

```
templar_modules/
├── github.com/
│   ├── panyam/
│   │   └── goapplib/
│   │       └── templates/
│   │           ├── EntityListing.html
│   │           └── components/
│   │               └── Grid.html
│   └── myorg/
│       └── shared-templates/
│           └── ...
└── templar.lock                 # Lock file with exact versions
```

### 3. Use in Templates

Reference external templates with the `@sourcename` prefix:

```html
{{# namespace "EL" "@goapplib/components/EntityListing.html" #}}
{{# include "@shared/layouts/base.html" #}}

{{ define "MyPage" }}
    {{ template "EL:EntityListing" .Items }}
{{ end }}
```

## Template Reference Syntax

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  Reference Syntax                                                           │
│                                                                             │
│  ┌────────────────────────┬─────────────────────────────────────────────┐   │
│  │ Syntax                 │ Resolution                                  │   │
│  ├────────────────────────┼─────────────────────────────────────────────┤   │
│  │ "@goapplib/foo.html"   │ Vendored: templar_modules/.../foo.html      │   │
│  │ "./components/bar.html"│ Relative to current template                │   │
│  │ "layouts/base.html"    │ Searched in search_paths order              │   │
│  └────────────────────────┴─────────────────────────────────────────────┘   │
│                                                                             │
│  The @ prefix maps to a configured source in templar.yaml                   │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Resolution Example

With this configuration:
```yaml
sources:
  goapplib:
    url: github.com/panyam/goapplib
    path: templates
    ref: v1.2.0
```

This template reference:
```html
{{# namespace "EL" "@goapplib/components/EntityListing.html" #}}
```

Resolves to:
```
./templar_modules/github.com/panyam/goapplib/templates/components/EntityListing.html
```

## CLI Commands

### `templar get` - Fetch Dependencies

```bash
# Fetch all configured sources
templar get

# Update to latest versions matching refs
templar get --update

# Fetch only a specific source
templar get @goapplib

# Verify local files match lock file
templar get --verify

# Show what would be fetched (dry run)
templar get --dry-run
```

### `templar sources` - List Sources

```bash
# Show configured sources and their status
templar sources

# Output:
# SOURCE      URL                              REF      STATUS
# goapplib    github.com/panyam/goapplib       v1.2.0   ✓ vendored (abc123)
# shared      github.com/myorg/shared-templates main    ✗ not fetched
```

## Configuration Reference

### templar.yaml

```yaml
# External template sources
sources:
  # Source name (used with @ prefix in templates)
  goapplib:
    # Repository URL (GitHub shorthand supported)
    url: github.com/panyam/goapplib

    # Subdirectory within repo containing templates (optional)
    path: templates

    # Git ref: tag, branch, or commit hash
    ref: v1.2.0

  # Another source example
  company-templates:
    url: github.com/mycompany/templates
    ref: main

# Directory for vendored templates (default: ./templar_modules)
vendor_dir: ./templar_modules

# Template search paths (in order of priority)
search_paths:
  - ./templates              # Check local first
  - ./templar_modules        # Then check vendored

# Optional: Require lock file for reproducible builds
require_lock: true
```

### templar.lock

Auto-generated lock file with exact versions:

```yaml
# AUTO-GENERATED - Do not edit manually
# Run 'templar get' to regenerate

version: 1
sources:
  goapplib:
    url: github.com/panyam/goapplib
    ref: v1.2.0
    resolved_commit: abc123def456789...
    fetched_at: 2024-12-08T10:30:00Z

  shared:
    url: github.com/myorg/shared-templates
    ref: main
    resolved_commit: def456abc789012...
    fetched_at: 2024-12-08T10:30:05Z
```

## Deployment Strategies

### Strategy 1: Vendor and Check In (Recommended)

Check vendored templates into version control for reproducible builds:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  Development                                                                │
│                                                                             │
│  1. templar get                    # Fetch dependencies                     │
│  2. git add templar_modules/       # Check in vendored files                │
│  3. git add templar.lock           # Check in lock file                     │
│  4. git commit                                                              │
│                                                                             │
├─────────────────────────────────────────────────────────────────────────────┤
│  Production                                                                 │
│                                                                             │
│  1. git clone / pull               # Get code + vendored templates          │
│  2. go build                       # Build app                              │
│  3. ./app                          # Run - no network needed                │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

**Pros:**
- Reproducible builds without network access
- See template changes in diffs during code review
- No external service dependencies during build

**Cons:**
- Larger repository size
- Potential merge conflicts when updating

### Strategy 2: Lock File Only

Check in only the lock file, fetch during build:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  Development                                                                │
│                                                                             │
│  1. templar get                    # Fetch dependencies                     │
│  2. git add templar.lock           # Only check in lock file                │
│  3. echo "templar_modules/" >> .gitignore                                   │
│  4. git commit                                                              │
│                                                                             │
├─────────────────────────────────────────────────────────────────────────────┤
│  CI / Production Build                                                      │
│                                                                             │
│  1. git clone / pull                                                        │
│  2. templar get                    # Fetch using lock file                  │
│  3. go build                                                                │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

**Pros:**
- Smaller repository
- No merge conflicts on vendored files

**Cons:**
- Requires network during build
- Depends on external service availability

### Strategy 3: CI-Generated Vendor

Generate vendored files in CI, cache or artifact them:

```yaml
# .github/workflows/build.yml
jobs:
  build:
    steps:
      - uses: actions/checkout@v4

      - name: Cache templar modules
        uses: actions/cache@v4
        with:
          path: templar_modules
          key: templar-${{ hashFiles('templar.lock') }}

      - name: Fetch templates
        run: templar get --verify || templar get

      - name: Build
        run: go build ./...
```

## Comparison with Go Modules

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  Concept              │ Go Modules          │ Templar Vendoring             │
├───────────────────────┼─────────────────────┼───────────────────────────────┤
│  Config file          │ go.mod              │ templar.yaml                  │
│  Lock file            │ go.sum              │ templar.lock                  │
│  Fetch command        │ go mod download     │ templar get                   │
│  Update command       │ go get -u           │ templar get --update          │
│  Vendor command       │ go mod vendor       │ (automatic with get)          │
│  Vendor directory     │ vendor/             │ templar_modules/              │
│  Reference syntax     │ import "pkg/..."    │ @source/path/...              │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Complete Example

### Project Structure

```
myapp/
├── templar.yaml
├── templar.lock
├── templates/
│   ├── pages/
│   │   ├── WorldListingPage.html
│   │   └── GameListingPage.html
│   └── BasePage.html
└── templar_modules/
    └── github.com/
        └── panyam/
            └── goapplib/
                └── templates/
                    └── components/
                        └── EntityListing.html
```

### templar.yaml

```yaml
sources:
  goapplib:
    url: github.com/panyam/goapplib
    path: templates
    ref: v1.0.0

vendor_dir: ./templar_modules

search_paths:
  - ./templates
  - ./templar_modules
```

### templates/pages/WorldListingPage.html

```html
{{# include "BasePage.html" #}}
{{# namespace "EL" "@goapplib/components/EntityListing.html" #}}

{{/* Custom world grid card */}}
{{ define "WorldGridCardPlaceholder" }}
<svg class="w-16 h-16 text-green-200">
    <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2z"/>
</svg>
{{ end }}

{{/* Extend Grid to use world-specific templates */}}
{{# extend "EL:Grid" "WorldGrid"
           "EL:GridCardPlaceholder" "WorldGridCardPlaceholder" #}}

{{# extend "EL:EntityListing" "WorldEntityListing"
           "EL:Grid" "WorldGrid" #}}

{{ define "BodySection" }}
<main class="max-w-7xl mx-auto px-4 py-8">
    {{ template "WorldEntityListing" .ListingData }}
</main>
{{ end }}

{{ define "WorldListingPage" }}
{{ template "BasePage" . }}
{{ end }}
```

### Go Code

```go
package main

import (
    "github.com/panyam/templar"
)

func main() {
    group := templar.NewTemplateGroup()

    // Configure loader with search paths from templar.yaml
    // (or programmatically)
    group.Loader = templar.NewLoaderList(
        templar.NewFileSystemLoader("templates/"),
        templar.NewFileSystemLoader("templar_modules/"),
    )

    // Load template - @goapplib references resolve automatically
    tmpl := group.MustLoad("pages/WorldListingPage.html", "")

    // Render
    group.RenderHtmlTemplate(w, tmpl[0], "WorldListingPage", data, nil)
}
```

## Gotchas and Tips

### 1. Always run `templar get` after cloning

If vendored files aren't checked in, fetch them before building:

```bash
git clone myrepo
cd myrepo
templar get          # Fetch dependencies
go build ./...
```

### 2. Lock file should always be checked in

Even if you check in `templar_modules/`, the lock file ensures:
- Exact commit hashes are recorded
- `templar get --verify` can validate local files
- Reproducible fetches if re-vendoring is needed

### 3. Use specific refs for stability

```yaml
# Avoid (unstable):
sources:
  lib:
    url: github.com/example/lib
    ref: main                    # May change unexpectedly

# Prefer (stable):
sources:
  lib:
    url: github.com/example/lib
    ref: v1.2.3                  # Specific version
    # or
    ref: abc123def               # Specific commit
```

### 4. The @ prefix is required for external sources

```html
{{/* WRONG - looks in search_paths */}}
{{# namespace "EL" "goapplib/components/EntityListing.html" #}}

{{/* CORRECT - looks up source "goapplib" */}}
{{# namespace "EL" "@goapplib/components/EntityListing.html" #}}
```

### 5. Source names are case-sensitive

```yaml
sources:
  GoAppLib:              # This name...
    url: ...
```

```html
{{/* Must match exactly */}}
{{# namespace "EL" "@GoAppLib/..." #}}    {{/* Correct */}}
{{# namespace "EL" "@goapplib/..." #}}    {{/* Wrong - case mismatch */}}
```

## Debugging

### Check source resolution

```bash
# See how a template path resolves
templar debug --trace templates/pages/WorldListingPage.html

# Output shows resolution chain:
# @goapplib/components/EntityListing.html
#   → source: goapplib
#   → url: github.com/panyam/goapplib
#   → path: templates
#   → resolved: templar_modules/github.com/panyam/goapplib/templates/components/EntityListing.html
```

### Verify vendored files

```bash
# Check if local files match lock file
templar get --verify

# Output:
# ✓ goapplib: matches lock (abc123def)
# ✗ shared: modified locally (expected def456, found ghi789)
```

### List what would be fetched

```bash
templar get --dry-run

# Output:
# Would fetch:
#   goapplib: github.com/panyam/goapplib@v1.2.0 → templar_modules/...
#   shared: github.com/myorg/shared@main → templar_modules/...
```
