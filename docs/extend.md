# Template Extension with `extend`

The `extend` directive creates new templates by copying existing ones while rewiring their internal template calls. This enables template inheritance patterns where child templates can override specific blocks.

## Basic Syntax

```html
{{# extend "SourceTemplate" "NewTemplate" "OldRef" "NewRef" ... #}}
```

- **SourceTemplate**: The template to copy from
- **NewTemplate**: The name for the new template being created
- **OldRef/NewRef pairs**: Template references to rewrite (multiple pairs allowed)

## How It Works

When you use `extend`, templar:
1. **Copies** the source template's parse tree
2. **Scans** for `{{ template "X" }}` calls within that tree
3. **Rewrites** any matching template names according to the rewrite pairs
4. **Registers** the result as a new template with the destination name

## Visual Example: Simple Extension

Consider a base layout template:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  base.html (source file - plain names, no namespace prefix)                 │
│                                                                             │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │ {{ define "layout" }}                                                 │  │
│  │ <html>                                                                │  │
│  │   <head>{{ template "title" . }}</head>         ───────┐              │  │
│  │   <body>                                               │              │  │
│  │     {{ template "header" . }}       ───────────────────┼──┐           │  │
│  │     {{ template "content" . }}      ───────────────────┼──┼──┐        │  │
│  │     {{ template "footer" . }}       ───────────────────┼──┼──┼──┐     │  │
│  │   </body>                                              │  │  │  │     │  │
│  │ </html>                                                │  │  │  │     │  │
│  │ {{ end }}                                              │  │  │  │     │  │
│  └────────────────────────────────────────────────────────│──│──│──│─────┘  │
│                                                           │  │  │  │        │
│  ┌────────────────────────────────────────────────────────│──│──│──│─────┐  │
│  │ {{ define "title" }}    ◄──────────────────────────────┘  │  │  │     │  │
│  │   <title>Default Title</title>                            │  │  │     │  │
│  │ {{ end }}                                                 │  │  │     │  │
│  ├───────────────────────────────────────────────────────────│──│──│─────┤  │
│  │ {{ define "header" }}   ◄─────────────────────────────────┘  │  │     │  │
│  │   <header>Default Header</header>                            │  │     │  │
│  │ {{ end }}                                                    │  │     │  │
│  ├──────────────────────────────────────────────────────────────│──│─────┤  │
│  │ {{ define "content" }}  ◄────────────────────────────────────┘  │     │  │
│  │   <main>Default Content</main>                                  │     │  │
│  │ {{ end }}                                                       │     │  │
│  ├─────────────────────────────────────────────────────────────────│─────┤  │
│  │ {{ define "footer" }}   ◄───────────────────────────────────────┘     │  │
│  │   <footer>Default Footer</footer>                                     │  │
│  │ {{ end }}                                                             │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────────┘
```

### After namespacing:

```
{{# namespace "Base" "base.html" #}}

┌─────────────────────────────────────────────────────────────────────────────┐
│  Templates now available (all prefixed with "Base:"):                       │
│                                                                             │
│  Base:layout  ──► calls Base:title, Base:header, Base:content, Base:footer  │
│  Base:title   ──► <title>Default Title</title>                              │
│  Base:header  ──► <header>Default Header</header>                           │
│  Base:content ──► <main>Default Content</main>                              │
│  Base:footer  ──► <footer>Default Footer</footer>                           │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Using extend to customize:

```html
{{# namespace "Base" "base.html" #}}
{{# extend "Base:layout" "MyLayout"
           "Base:title" "myTitle"
           "Base:content" "myContent" #}}

{{ define "myTitle" }}<title>My Custom Page</title>{{ end }}
{{ define "myContent" }}<main>Hello World!</main>{{ end }}
```

### Result after extension:

```
┌────────────────────────────────────────────────────────────────────────────────┐
│  BEFORE extend                              AFTER extend                       │
│                                                                                │
│  ┌─────────────────────────────────┐       ┌────────────────────────────────┐  │
│  │ Base:layout                     │       │ MyLayout                       │  │
│  │ ┌─────────────────────────────┐ │       │ ┌────────────────────────────┐ │  │
│  │ │ {{ template "Base:title" }} │ │  ───► │ │ {{ template "myTitle" }}   │ │  │
│  │ │ {{ template "Base:header" }}│ │       │ │ {{ template "Base:header"}}│ │  │
│  │ │ {{ template "Base:content"}}│ │  ───► │ │ {{ template "myContent"}}  │ │  │
│  │ │ {{ template "Base:footer" }}│ │       │ │ {{ template "Base:footer"}}│ │  │
│  │ └─────────────────────────────┘ │       │ └────────────────────────────┘ │  │
│  └─────────────────────────────────┘       └────────────────────────────────┘  │
│                                                                                │
│  Rewrite Map:                                                                  │
│  ┌─────────────────────────────────────────────────────────────────────┐       │
│  │  "Base:title"   ──────────────────────────────────►  "myTitle"      │       │
│  │  "Base:content" ──────────────────────────────────►  "myContent"    │       │
│  │  "Base:header"  ─── (not in map) ─────────────────►  unchanged      │       │
│  │  "Base:footer"  ─── (not in map) ─────────────────►  unchanged      │       │
│  └─────────────────────────────────────────────────────────────────────┘       │
└────────────────────────────────────────────────────────────────────────────────┘
```

## Critical Concept: Rewrites Only Apply to the Extended Template

**The most important thing to understand**: `extend` only rewrites template calls **within the source template itself**. It does NOT recursively rewrite calls in templates that the source template calls.

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  What extend DOES rewrite:                                                  │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │  Source Template (being copied)                                     │    │
│  │  ┌───────────────────────────────────────────────────────────────┐  │    │
│  │  │  {{ template "X" }}  ◄──────────── THIS gets rewritten        │  │    │
│  │  │  {{ template "Y" }}  ◄──────────── THIS gets rewritten        │  │    │
│  │  └───────────────────────────────────────────────────────────────┘  │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                                                                             │
│  What extend does NOT rewrite:                                              │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │  Template "X" (called by source, but separate template)             │    │
│  │  ┌───────────────────────────────────────────────────────────────┐  │    │
│  │  │  {{ template "Z" }}  ◄──────────── NOT rewritten!             │  │    │
│  │  └───────────────────────────────────────────────────────────────┘  │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────────────┘
```

## The Nested Template Problem

### The Goal

You have a shared component library (e.g., `EntityListing.html`) that provides a reusable listing UI with grid cards. Each card shows a preview image using a default placeholder icon. You want to customize this for your app - for example, showing actual preview images for worlds, or using a custom globe icon instead of the default placeholder.

The intuitive approach would be: "I'll just extend `EntityListing` and override `GridCardPreview`." But this doesn't work as expected due to how `extend` processes templates.

### The Setup

Consider a component library with nested templates:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  EntityListing.html - A reusable listing component                          │
│                                                                             │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │ {{ define "EntityListing" }}                                          │  │
│  │   {{ template "PageHeader" . }}                                       │  │
│  │   {{ if .Items }}                                                     │  │
│  │       {{ template "Grid" . }}  ─────────────────────────┐             │  │
│  │       {{ template "Table" . }}                          │             │  │
│  │   {{ else }}                                            │             │  │
│  │       {{ template "EmptyState" . }}                     │             │  │
│  │   {{ end }}                                             │             │  │
│  │ {{ end }}                                               │             │  │
│  └─────────────────────────────────────────────────────────│─────────────┘  │
│                                                            │                │
│                                                            ▼                │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │ {{ define "Grid" }}                    ◄── SEPARATE template          │  │
│  │   <div class="grid">                                                  │  │
│  │     {{ range .Items }}                                                │  │
│  │       <div class="card">                                              │  │
│  │         {{ template "GridCardPreview" . }}  ◄── call inside Grid      │  │
│  │         {{ template "GridCardMeta" . }}                               │  │
│  │         <h3>{{ .Name }}</h3>                                          │  │
│  │       </div>                                                          │  │
│  │     {{ end }}                                                         │  │
│  │   </div>                                                              │  │
│  │ {{ end }}                                                             │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
│                                                                             │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │ {{ define "GridCardPreview" }}                                        │  │
│  │   <div class="preview">                                               │  │
│  │     {{ template "GridCardPlaceholder" . }}                            │  │
│  │   </div>                                                              │  │
│  │ {{ end }}                                                             │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
│                                                                             │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │ {{ define "GridCardPlaceholder" }}                                    │  │
│  │   <svg class="placeholder-icon">...</svg>   ◄── default blue icon     │  │
│  │ {{ end }}                                                             │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────────┘
```

### The Wrong Approach (doesn't work):

```html
{{# namespace "EL" "EntityListing.html" #}}

{{/* Try to override GridCardPreview */}}
{{# extend "EL:EntityListing" "MyListing"
           "EL:GridCardPreview" "MyPreview" #}}

{{ define "MyPreview" }}
  <img src="{{ .ImageUrl }}" />
{{ end }}
```

### Why it fails:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  extend "EL:EntityListing" → "MyListing"                                    │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │ MyListing (copied from EL:EntityListing)                            │    │
│  │                                                                     │    │
│  │   {{ template "EL:Grid" . }}    ◄── This call is in EntityListing   │    │
│  │                │                    so it COULD be rewritten...     │    │
│  │                │                    but we didn't include it in     │    │
│  │                │                    the rewrite map!                │    │
│  │                │                                                    │    │
│  │                ▼                                                    │    │
│  │   ┌─────────────────────────────────────────────────────────────┐   │    │
│  │   │ EL:Grid (NOT copied, original template)                     │   │    │
│  │   │                                                             │   │    │
│  │   │   {{ template "EL:GridCardPreview" . }}  ◄── This call is   │   │    │
│  │   │                │                            inside Grid,    │   │    │
│  │   │                │                            NOT in the      │   │    │
│  │   │                │                            copied template │   │    │
│  │   │                │                            so it's NEVER   │   │    │
│  │   │                │                            rewritten!      │   │    │
│  │   │                ▼                                            │   │    │
│  │   │   Still calls EL:GridCardPreview (default blue icon)        │   │    │
│  │   └─────────────────────────────────────────────────────────────┘   │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                                                                             │
│  The rewrite map only affected EntityListing's direct template calls,       │
│  but GridCardPreview is called from Grid, not from EntityListing!           │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Summary of the Problem

The `extend` directive only rewrites template calls that appear directly in the template being extended. It doesn't "look inside" the templates that get called. So even though we wanted to override `GridCardPreview`, that template is called from `Grid`, not from `EntityListing`. Our rewrite never gets applied because `extend` only processed `EntityListing`'s own template calls.

## Solution: Chained Extensions

To override templates at any level, you need to extend each level in the chain:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  STEP 1: Extend Grid to use custom preview                                  │
│                                                                             │
│  {{# extend "EL:Grid" "MyGrid"                                              │
│             "EL:GridCardPreview" "MyPreview" #}}                            │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │ EL:Grid                              MyGrid                         │    │
│  │ ┌───────────────────────┐           ┌───────────────────────┐       │    │
│  │ │ {{ template           │   ───►    │ {{ template           │       │    │
│  │ │   "EL:GridCardPreview"│           │   "MyPreview" . }}    │       │    │
│  │ │    . }}               │           │                       │       │    │
│  │ └───────────────────────┘           └───────────────────────┘       │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                                                                             │
│  STEP 2: Extend EntityListing to use custom Grid                            │
│                                                                             │
│  {{# extend "EL:EntityListing" "MyListing"                                  │
│             "EL:Grid" "MyGrid" #}}                                          │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │ EL:EntityListing                     MyListing                      │    │
│  │ ┌───────────────────────┐           ┌───────────────────────┐       │    │
│  │ │ {{ template           │   ───►    │ {{ template           │       │    │
│  │ │   "EL:Grid" . }}      │           │   "MyGrid" . }}       │       │    │
│  │ └───────────────────────┘           └───────────────────────┘       │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                                                                             │
│  STEP 3: Define custom templates                                            │
│                                                                             │
│  {{ define "MyPreview" }}                                                   │
│    <img src="{{ .ImageUrl }}" />                                            │
│  {{ end }}                                                                  │
└─────────────────────────────────────────────────────────────────────────────┘
```

### The complete call chain after chained extensions:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  BEFORE (original chain):                                                   │
│                                                                             │
│  EL:EntityListing ──► EL:Grid ──► EL:GridCardPreview ──► EL:Placeholder     │
│        │                 │               │                    │             │
│        │                 │               │                    └─ blue icon  │
│        │                 │               └─ default preview                 │
│        │                 └─ default grid                                    │
│        └─ main listing                                                      │
│                                                                             │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  AFTER (extended chain):                                                    │
│                                                                             │
│  MyListing ──────────► MyGrid ──────► MyPreview ──────────► MyPlaceholder   │
│        │                  │               │                       │         │
│        │                  │               │                       └─ custom │
│        │                  │               │                          icon   │
│        │                  │               └─ shows actual image             │
│        │                  └─ rewired grid                                   │
│        └─ rewired listing                                                   │
│                                                                             │
│  Each arrow represents a {{ template "..." }} call that was rewritten       │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Complete Real-World Example

### Base Component (goapplib/components/EntityListing.html)

```html
{{ define "GridCardPlaceholder" }}
<svg class="w-16 h-16 text-blue-200"><!-- default icon --></svg>
{{ end }}

{{ define "GridCardPreview" }}
<div class="preview">{{ template "GridCardPlaceholder" $ }}</div>
{{ end }}

{{ define "Grid" }}
<div class="grid">
    {{ range .Items }}
    <div class="card">
        {{ template "GridCardPreview" . }}
        <h3>{{ .Name }}</h3>
    </div>
    {{ end }}
</div>
{{ end }}

{{ define "EntityListing" }}
<div>
    {{ template "PageHeader" . }}
    {{ if .Items }}{{ template "Grid" . }}{{ end }}
</div>
{{ end }}
```

### App-Specific Template (WorldListingPage.html)

```html
{{# namespace "EL" "goapplib/components/EntityListing.html" #}}

{{/* Define world-specific templates */}}
{{ define "WorldPlaceholder" }}
<svg class="w-16 h-16 text-green-200"><!-- globe icon --></svg>
{{ end }}

{{ define "WorldPreview" }}
{{ if .PreviewUrl }}
    <img src="{{ .PreviewUrl }}" class="w-full h-full object-cover">
{{ else }}
    <div class="preview">{{ template "WorldPlaceholder" $ }}</div>
{{ end }}
{{ end }}

{{/* Chain of extensions - from inside out */}}
{{# extend "EL:Grid" "WorldGrid"
           "EL:GridCardPreview" "WorldPreview" #}}

{{# extend "EL:EntityListing" "WorldListing"
           "EL:Grid" "WorldGrid" #}}

{{/* Use the fully customized listing */}}
{{ define "BodySection" }}
<main>{{ template "WorldListing" .ListingData }}</main>
{{ end }}
```

### Visual representation of the extension:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  Extension Chain for WorldListingPage                                       │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │  Step 1: extend "EL:Grid" → "WorldGrid"                             │    │
│  │                                                                     │    │
│  │  EL:Grid                                WorldGrid                   │    │
│  │  ┌──────────────────────────┐          ┌──────────────────────────┐ │    │
│  │  │ calls EL:GridCardPreview │   ──►    │ calls WorldPreview       │ │    │
│  │  │ calls EL:GridCardMeta    │          │ calls EL:GridCardMeta    │ │    │
│  │  └──────────────────────────┘          └──────────────────────────┘ │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │  Step 2: extend "EL:EntityListing" → "WorldListing"                 │    │
│  │                                                                     │    │
│  │  EL:EntityListing                       WorldListing                │    │
│  │  ┌──────────────────────────┐          ┌──────────────────────────┐ │    │
│  │  │ calls EL:Grid            │   ──►    │ calls WorldGrid          │ │    │
│  │  │ calls EL:Table           │          │ calls EL:Table           │ │    │
│  │  │ calls EL:EmptyState      │          │ calls EL:EmptyState      │ │    │
│  │  └──────────────────────────┘          └──────────────────────────┘ │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │  Final Result:                                                      │    │
│  │                                                                     │    │
│  │  WorldListing                                                       │    │
│  │       │                                                             │    │
│  │       ├──► WorldGrid                                                │    │
│  │       │        │                                                    │    │
│  │       │        └──► WorldPreview                                    │    │
│  │       │                  │                                          │    │
│  │       │                  └──► (shows image or WorldPlaceholder)     │    │
│  │       │                                                             │    │
│  │       ├──► EL:Table (unchanged - we didn't extend it)               │    │
│  │       │                                                             │    │
│  │       └──► EL:EmptyState (unchanged)                                │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Common Patterns

### Pattern 1: Override a leaf template

When you only need to change a template that doesn't call other templates:

```
┌─────────────────────────────────────────────────────────────────┐
│  EntityListing ──► EmptyStateIcon (leaf - no further calls)    │
│                                                                 │
│  Just one extend needed:                                        │
│  {{# extend "EL:EntityListing" "MyListing"                      │
│             "EL:EmptyStateIcon" "MyEmptyIcon" #}}               │
└─────────────────────────────────────────────────────────────────┘
```

### Pattern 2: Override at multiple levels

When you need to change templates that are called from other templates:

```
┌─────────────────────────────────────────────────────────────────┐
│  EntityListing ──► Grid ──► GridCardPreview                     │
│                                                                 │
│  Two extends needed (inside-out order):                         │
│  {{# extend "EL:Grid" "MyGrid"                                  │
│             "EL:GridCardPreview" "MyPreview" #}}                │
│  {{# extend "EL:EntityListing" "MyListing"                      │
│             "EL:Grid" "MyGrid" #}}                              │
└─────────────────────────────────────────────────────────────────┘
```

### Pattern 3: Multiple independent overrides

When different branches need different customizations:

```
┌─────────────────────────────────────────────────────────────────┐
│  EntityListing ──► Grid ──► GridCardPreview                     │
│               └──► Table ──► TableRowPreview                    │
│                                                                 │
│  Three extends needed:                                          │
│  {{# extend "EL:Grid" "MyGrid"                                  │
│             "EL:GridCardPreview" "MyPreview" #}}                │
│  {{# extend "EL:Table" "MyTable"                                │
│             "EL:TableRowPreview" "MyRowPreview" #}}             │
│  {{# extend "EL:EntityListing" "MyListing"                      │
│             "EL:Grid" "MyGrid"                                  │
│             "EL:Table" "MyTable" #}}                            │
└─────────────────────────────────────────────────────────────────┘
```

### Pattern 4: Partial override (keeping some defaults)

Only override what you need - non-specified calls remain unchanged:

```
┌─────────────────────────────────────────────────────────────────┐
│  Base:layout ──► Base:header                                    │
│             └──► Base:content                                   │
│             └──► Base:footer                                    │
│                                                                 │
│  Only override content:                                         │
│  {{# extend "Base:layout" "MyLayout"                            │
│             "Base:content" "myContent" #}}                      │
│                                                                 │
│  Result:                                                        │
│  MyLayout ──► Base:header  (unchanged - uses default)           │
│          └──► myContent    (customized)                         │
│          └──► Base:footer  (unchanged - uses default)           │
└─────────────────────────────────────────────────────────────────┘
```

## Gotchas and Common Mistakes

### 1. Source template must exist before extend

```
┌─────────────────────────────────────────────────────────────────┐
│  WRONG ORDER:                                                   │
│  {{# extend "Base:layout" "MyLayout" ... #}}   ◄── Base:layout  │
│  {{# namespace "Base" "base.html" #}}              doesn't      │
│                                                    exist yet!   │
│                                                                 │
│  CORRECT ORDER:                                                 │
│  {{# namespace "Base" "base.html" #}}          ◄── load first   │
│  {{# extend "Base:layout" "MyLayout" ... #}}   ◄── then extend  │
└─────────────────────────────────────────────────────────────────┘
```

### 2. Rewrite pairs must be even

```html
{{/* WRONG - odd number of arguments after dest */}}
{{# extend "Base:layout" "MyLayout" "Base:title" #}}

{{/* CORRECT - pairs of old/new */}}
{{# extend "Base:layout" "MyLayout" "Base:title" "myTitle" #}}
```

### 3. Template names must match exactly

```
┌───────────────────────────────────────────────────────────────────┐
│  If base calls {{ template "Base:header" . }}:                    │
│                                                                   │
│  WRONG - name doesn't match:                                      │
│  {{# extend "Base:layout" "MyLayout" "header" "myHeader" #}}      │
│                                     ^^^^^^^^                      │
│                                     should be "Base:header"       │
│                                                                   │
│  CORRECT - exact match:                                           │
│  {{# extend "Base:layout" "MyLayout" "Base:header" "myHeader" #}} │
└───────────────────────────────────────────────────────────────────┘
```

### 4. Order matters for chained extensions

```
┌─────────────────────────────────────────────────────────────────┐
│  WRONG ORDER:                                                   │
│  {{# extend "EL:EntityListing" "MyListing"                      │
│             "EL:Grid" "MyGrid" #}}         ◄── MyGrid doesn't   │
│  {{# extend "EL:Grid" "MyGrid" ... #}}         exist yet!       │
│                                                                 │
│  CORRECT ORDER (inside-out):                                    │
│  {{# extend "EL:Grid" "MyGrid" ... #}}     ◄── create MyGrid    │
│  {{# extend "EL:EntityListing" "MyListing"                      │
│             "EL:Grid" "MyGrid" #}}         ◄── then use it      │
└─────────────────────────────────────────────────────────────────┘
```

### 5. Don't forget all call sites

```
┌───────────────────────────────────────────────────────────────────┐
│  If EntityListing has BOTH Grid and Table views:                  │
│                                                                   │
│  EntityListing ──► Grid ──► GridCardPreview                       │
│               └──► Table ──► TableRowPreview ──► uses same icon   │
│                                                                   │
│  INCOMPLETE - only fixes Grid view:                               │
│  {{# extend "EL:Grid" "MyGrid" ... #}}                            │
│  {{# extend "EL:EntityListing" "MyListing" "EL:Grid" "MyGrid" #}} │
│                                                                   │
│  COMPLETE - fixes both views:                                     │
│  {{# extend "EL:Grid" "MyGrid" ... #}}                            │
│  {{# extend "EL:Table" "MyTable" ... #}}                          │
│  {{# extend "EL:EntityListing" "MyListing"                        │
│             "EL:Grid" "MyGrid"                                    │
│             "EL:Table" "MyTable" #}}                              │
└───────────────────────────────────────────────────────────────────┘
```

## Debugging Tips

1. **Check template names**: Use `templar debug --defines` to see all defined templates
2. **Verify the call chain**: Use `templar debug --refs` to see what templates call what
3. **Look at preprocessed output**: Use `templar debug --flatten` to see the final template after all includes and extends are processed
