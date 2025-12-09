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

### Important: Rewrites Only Apply to the Extended Template

The key thing to understand is that `extend` only rewrites template calls **within the source template itself**. It does not recursively rewrite calls in templates that the source template calls.

## Visual Example: Simple Extension

Consider a base layout template:

```
┌─────────────────────────────────────────┐
│  Base:layout                            │
│  ┌───────────────────────────────────┐  │
│  │ <html>                            │  │
│  │   {{ template "Base:header" . }}  │──┼──► calls Base:header
│  │   {{ template "Base:content" . }} │──┼──► calls Base:content
│  │   {{ template "Base:footer" . }}  │──┼──► calls Base:footer
│  │ </html>                           │  │
│  └───────────────────────────────────┘  │
└─────────────────────────────────────────┘
```

Using extend to create a custom layout:

```html
{{# extend "Base:layout" "MyLayout"
           "Base:header" "myHeader"
           "Base:content" "myContent" #}}

{{ define "myHeader" }}<h1>My Site</h1>{{ end }}
{{ define "myContent" }}<p>Welcome!</p>{{ end }}
```

Result:

```
┌─────────────────────────────────────────┐
│  MyLayout (copied from Base:layout)     │
│  ┌───────────────────────────────────┐  │
│  │ <html>                            │  │
│  │   {{ template "myHeader" . }}     │──┼──► rewired to myHeader
│  │   {{ template "myContent" . }}    │──┼──► rewired to myContent
│  │   {{ template "Base:footer" . }}  │──┼──► unchanged (not in rewrite list)
│  │ </html>                           │  │
│  └───────────────────────────────────┘  │
└─────────────────────────────────────────┘
```

## The Nested Template Problem

Consider a more complex scenario with nested templates:

```
┌──────────────────────────────────────────────────────────────┐
│  EntityListing                                               │
│  ┌────────────────────────────────────────────────────────┐  │
│  │ {{ template "PageHeader" . }}                          │  │
│  │ {{ template "Grid" . }}        ─────────────┐          │  │
│  │ {{ template "Table" . }}                    │          │  │
│  └─────────────────────────────────────────────│──────────┘  │
└────────────────────────────────────────────────│─────────────┘
                                                 │
                                                 ▼
┌──────────────────────────────────────────────────────────────┐
│  Grid (separate template)                                    │
│  ┌────────────────────────────────────────────────────────┐  │
│  │ {{ range .Items }}                                     │  │
│  │   {{ template "GridCardPreview" . }}  ◄── NOT rewired! │  │
│  │   {{ template "GridCardMeta" . }}                      │  │
│  │ {{ end }}                                              │  │
│  └────────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────┘
```

If you try to extend just EntityListing:

```html
{{# extend "EL:EntityListing" "MyListing"
           "EL:GridCardPreview" "MyPreview" #}}
```

**This won't work as expected!** The `GridCardPreview` call is inside `Grid`, not inside `EntityListing`. When `extend` copies `EntityListing`, it only sees and rewrites calls in `EntityListing` itself - it doesn't look inside `Grid`.

```
EntityListing calls Grid ──► Grid calls GridCardPreview
      │                              │
      │ extend copies this           │ but NOT this!
      ▼                              ▼
MyListing calls Grid ──────► Grid still calls GridCardPreview (unchanged)
```

## Solution: Chained Extensions

To properly override nested template calls, extend each level of the template hierarchy:

```html
{{/* Step 1: Extend Grid with our custom preview */}}
{{# extend "EL:Grid" "MyGrid"
           "EL:GridCardPreview" "MyPreview"
           "EL:GridCardMeta" "MyMeta" #}}

{{/* Step 2: Extend Table with our custom row preview */}}
{{# extend "EL:Table" "MyTable"
           "EL:TableRowPreview" "MyRowPreview" #}}

{{/* Step 3: Extend EntityListing to use our custom Grid and Table */}}
{{# extend "EL:EntityListing" "MyListing"
           "EL:Grid" "MyGrid"
           "EL:Table" "MyTable"
           "EL:EmptyStateIcon" "MyEmptyIcon" #}}
```

This creates a chain of extended templates:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  BEFORE: Original template chain                                            │
│                                                                             │
│  EntityListing ──► Grid ──► GridCardPreview ──► GridCardPlaceholder         │
│                      │                                                       │
│                      └──► GridCardMeta                                       │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│  AFTER: Extended template chain                                             │
│                                                                             │
│  MyListing ──► MyGrid ──► MyPreview ──► MyPlaceholder                       │
│       │           │                                                          │
│       │           └──► MyMeta                                                │
│       │                                                                      │
│       └──► MyTable ──► MyRowPreview                                          │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Complete Real-World Example

Here's a complete example showing how to extend a shared EntityListing component with app-specific customizations:

### Base Template (goapplib/components/EntityListing.html)

```html
{{ define "GridCardPlaceholder" }}
<svg class="w-16 h-16 text-blue-200">...</svg>
{{ end }}

{{ define "GridCardPreview" }}
<div class="flex items-center justify-center h-full">
    {{ template "GridCardPlaceholder" $ }}
</div>
{{ end }}

{{ define "Grid" }}
<div class="grid grid-cols-4 gap-6">
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
    {{ if .Items }}
        {{ template "Grid" . }}
    {{ else }}
        {{ template "EmptyState" . }}
    {{ end }}
</div>
{{ end }}
```

### App-Specific Template (WorldListingPage.html)

```html
{{# namespace "EL" "goapplib/components/EntityListing.html" #}}

{{/* Define world-specific templates */}}
{{ define "WorldPlaceholder" }}
<svg class="w-16 h-16 text-green-200">
    <!-- globe icon -->
</svg>
{{ end }}

{{ define "WorldPreview" }}
{{ if and .PreviewUrls (index .PreviewUrls 0) }}
    <img src="{{ index .PreviewUrls 0 }}" class="w-full h-full object-cover">
{{ else }}
    <div class="flex items-center justify-center h-full">
        {{ template "WorldPlaceholder" $ }}
    </div>
{{ end }}
{{ end }}

{{/* Chain of extensions */}}
{{# extend "EL:Grid" "WorldGrid"
           "EL:GridCardPreview" "WorldPreview" #}}

{{# extend "EL:EntityListing" "WorldListing"
           "EL:Grid" "WorldGrid" #}}

{{/* Use the extended template */}}
{{ define "BodySection" }}
<main>
    {{ template "WorldListing" .ListingData }}
</main>
{{ end }}
```

## Visualization of the Extension Process

```
STEP 1: Load with namespace
┌──────────────────────────────────────────────────┐
│ Available templates after namespace:             │
│   EL:GridCardPlaceholder                         │
│   EL:GridCardPreview  ─► calls GridCardPlaceholder
│   EL:Grid             ─► calls GridCardPreview   │
│   EL:EntityListing    ─► calls Grid              │
└──────────────────────────────────────────────────┘

STEP 2: Define custom templates
┌──────────────────────────────────────────────────┐
│ Added templates:                                 │
│   WorldPlaceholder  (custom globe icon)          │
│   WorldPreview      ─► calls WorldPlaceholder    │
└──────────────────────────────────────────────────┘

STEP 3: extend "EL:Grid" → "WorldGrid"
┌──────────────────────────────────────────────────┐
│ Copy EL:Grid, rewrite EL:GridCardPreview         │
│                                                  │
│   WorldGrid ─► calls WorldPreview (rewired!)     │
└──────────────────────────────────────────────────┘

STEP 4: extend "EL:EntityListing" → "WorldListing"
┌──────────────────────────────────────────────────┐
│ Copy EL:EntityListing, rewrite EL:Grid           │
│                                                  │
│   WorldListing ─► calls WorldGrid (rewired!)     │
└──────────────────────────────────────────────────┘

FINAL CALL CHAIN:
WorldListing → WorldGrid → WorldPreview → WorldPlaceholder
     │              │            │              │
     │              │            │              └── custom globe icon
     │              │            └── custom image handling
     │              └── rewired Grid template
     └── rewired EntityListing template
```

## Key Takeaways

1. **`extend` only rewrites the immediate template** - not templates it calls
2. **For nested overrides, extend each level** of the template hierarchy
3. **Work from inside out** - extend inner templates first, then outer ones
4. **The rewrite map is literal** - template names must match exactly
5. **Non-rewritten calls remain unchanged** - only specified pairs are rewritten

## Common Patterns

### Pattern 1: Override a leaf template
When you only need to change a template that doesn't call other templates:
```html
{{# extend "EL:EntityListing" "MyListing"
           "EL:EmptyStateIcon" "MyEmptyIcon" #}}
```

### Pattern 2: Override at multiple levels
When you need to change templates that are called from other templates:
```html
{{# extend "EL:Grid" "MyGrid" "EL:GridCardPreview" "MyPreview" #}}
{{# extend "EL:EntityListing" "MyListing" "EL:Grid" "MyGrid" #}}
```

### Pattern 3: Multiple independent overrides
When different parts need different customizations:
```html
{{# extend "EL:Grid" "MyGrid" "EL:GridCardPreview" "MyPreview" #}}
{{# extend "EL:Table" "MyTable" "EL:TableRowPreview" "MyRowPreview" #}}
{{# extend "EL:EntityListing" "MyListing"
           "EL:Grid" "MyGrid"
           "EL:Table" "MyTable" #}}
```

### Pattern 4: Partial override (keeping some defaults)
Only override what you need - non-specified blocks use the original templates:
```html
{{# namespace "Base" "base.html" #}}
{{# extend "Base:layout" "MyLayout" "Base:content" "myContent" #}}

{{ define "myContent" }}Custom Content Only{{ end }}

{{/* header and footer still use Base:header and Base:footer */}}
{{ template "MyLayout" . }}
```

## Gotchas and Common Mistakes

### 1. Source template must exist before extend

The `extend` directive looks up the source template at preprocessing time. The source must be loaded via `include` or `namespace` before the `extend` directive:

```html
{{/* WRONG - Base:layout doesn't exist yet */}}
{{# extend "Base:layout" "MyLayout" ... #}}
{{# namespace "Base" "base.html" #}}

{{/* CORRECT - namespace first, then extend */}}
{{# namespace "Base" "base.html" #}}
{{# extend "Base:layout" "MyLayout" ... #}}
```

### 2. Rewrite pairs must be even

The extend directive requires pairs of old/new template names after the destination:

```html
{{/* WRONG - odd number of arguments after dest */}}
{{# extend "Base:layout" "MyLayout" "Base:title" #}}

{{/* CORRECT - pairs of old/new */}}
{{# extend "Base:layout" "MyLayout" "Base:title" "myTitle" #}}
```

### 3. Empty destination name is invalid

```html
{{/* WRONG - empty destination */}}
{{# extend "Base:layout" "" "Base:title" "myTitle" #}}
```

### 4. Template names are exact matches

Rewrites only happen when the template name matches exactly:

```html
{{/* If base calls {{ template "Base:header" . }} */}}
{{# extend "Base:layout" "MyLayout" "header" "myHeader" #}}
{{/* This WON'T work - "header" != "Base:header" */}}

{{/* CORRECT - use full namespaced name */}}
{{# extend "Base:layout" "MyLayout" "Base:header" "myHeader" #}}
```

### 5. Order matters for chained extensions

When creating a chain of extended templates, process them in dependency order (inside-out):

```html
{{/* WRONG ORDER - MyListing references MyGrid before it exists */}}
{{# extend "EL:EntityListing" "MyListing" "EL:Grid" "MyGrid" #}}
{{# extend "EL:Grid" "MyGrid" ... #}}

{{/* CORRECT ORDER - define MyGrid first, then use it */}}
{{# extend "EL:Grid" "MyGrid" ... #}}
{{# extend "EL:EntityListing" "MyListing" "EL:Grid" "MyGrid" #}}
```

### 6. Don't forget about all call sites

If a template is called from multiple places, you need to rewrite all of them:

```html
{{/* If EntityListing has both Grid and Table calling GridCardPreview... */}}

{{/* This only fixes Grid, Table still calls the original */}}
{{# extend "EL:Grid" "MyGrid" "EL:GridCardPreview" "MyPreview" #}}
{{# extend "EL:EntityListing" "MyListing" "EL:Grid" "MyGrid" #}}

{{/* Need to also extend Table */}}
{{# extend "EL:Grid" "MyGrid" "EL:GridCardPreview" "MyPreview" #}}
{{# extend "EL:Table" "MyTable" "EL:TableRowPreview" "MyPreview" #}}
{{# extend "EL:EntityListing" "MyListing"
           "EL:Grid" "MyGrid"
           "EL:Table" "MyTable" #}}
```

## Debugging Tips

1. **Check template names**: Use `templar debug --defines` to see all defined templates
2. **Verify the call chain**: Use `templar debug --refs` to see what templates call what
3. **Look at preprocessed output**: Use `templar debug --flatten` to see the final template after all includes and extends are processed
