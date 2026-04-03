# templar

## Version
0.0.35

## Provides
- template-loading: Go template loader with dependency management
- template-includes: {{# include }} directive for template composition
- template-namespacing: Namespace support to avoid template name collisions
- template-inheritance: {{# extend }} directive for template extension
- tree-shaking: Selective template loading
- multi-loader: Multiple template loaders with fallback behavior
- template-groups: Managing template collections
- external-sources: Fetch templates from URLs/GitHub
- template-cli: CLI tool for template serving and debugging
- dependency-visualization: GraphViz visualization of template dependency graph
- llm-workflow-guidance: Generated files include AI/LLM-readable markers to prevent incorrect copying of templar_modules/
- writable-fs: WritableFS interface extending fs.FS with Write/Mkdir/Remove/Rename for portable template storage
- local-fs: LocalFS implementation of WritableFS backed by OS filesystem
- mem-fs: MemFS implementation of WritableFS for in-memory testing and WASM
- fs-backed-loader: FileSystemLoader.FileSystems field for loading templates from any fs.FS

## Module
github.com/panyam/templar

## Location
newstack/templar/main

## Stack Dependencies
- goutils (github.com/panyam/goutils)

## Integration

### Go Module
```go
// go.mod
require github.com/panyam/templar 0.0.29

// Local development
replace github.com/panyam/templar => ~/newstack/templar/main
```

### Key Imports
```go
import "github.com/panyam/templar/loader"
```

## Status
Stable

## Conventions
- Directive-based includes (pre-rendering)
- Namespace colon syntax
- @source/ prefix for external templates
- templar.yaml is source of truth; templar_modules/ and templar.lock are generated
- `templar get` auto-writes README.md inside templar_modules/ with LLM-readable markers
- Lock file header includes explicit "do not copy" instructions for AI agents
