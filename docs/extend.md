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

## Complete Example: Product Catalog with Themed Cards

This example shows a reusable product card component that different pages can customize with their own styling.

### File Structure

```
templates/
├── shared/
│   └── card.html        # Base card component
├── pages/
│   ├── products.html    # Product listing (uses photos)
│   └── services.html    # Service listing (uses icons)
```

### Base Component (shared/card.html)

```html
{{/* Default placeholder - a generic box icon */}}
{{ define "cardIcon" }}
<svg class="icon" viewBox="0 0 24 24">
  <rect x="3" y="3" width="18" height="18" rx="2"/>
</svg>
{{ end }}

{{/* Card preview - shows icon by default */}}
{{ define "cardPreview" }}
<div class="card-preview">
  {{ template "cardIcon" . }}
</div>
{{ end }}

{{/* Individual card */}}
{{ define "card" }}
<div class="card">
  {{ template "cardPreview" . }}
  <h3>{{ .Title }}</h3>
  <p>{{ .Description }}</p>
</div>
{{ end }}

{{/* Card grid - renders multiple cards */}}
{{ define "cardGrid" }}
<div class="grid">
  {{ range .Items }}
    {{ template "card" . }}
  {{ end }}
</div>
{{ end }}
```

### Product Page (pages/products.html) - Uses Photos

```html
{{# namespace "Base" "shared/card.html" #}}

{{/* Product preview - shows actual product photo */}}
{{ define "productPreview" }}
<div class="card-preview">
  {{ if .ImageUrl }}
    <img src="{{ .ImageUrl }}" alt="{{ .Title }}">
  {{ else }}
    {{ template "productIcon" . }}
  {{ end }}
</div>
{{ end }}

{{/* Fallback icon for products without photos */}}
{{ define "productIcon" }}
<svg class="icon product-icon" viewBox="0 0 24 24">
  <path d="M20 7l-8-4-8 4m16 0v10l-8 4m8-14l-8 4m0 6v-6m0 6l-8-4V7"/>
</svg>
{{ end }}

{{/* Extend card to use product preview */}}
{{# extend "Base:card" "productCard"
           "Base:cardPreview" "productPreview" #}}

{{/* Extend grid to use product cards */}}
{{# extend "Base:cardGrid" "productGrid"
           "Base:card" "productCard" #}}

{{/* Page template */}}
{{ define "productPage" }}
<main>
  <h1>Our Products</h1>
  {{ template "productGrid" . }}
</main>
{{ end }}
```

### Service Page (pages/services.html) - Uses Custom Icons

```html
{{# namespace "Base" "shared/card.html" #}}

{{/* Service preview - uses service-specific icon */}}
{{ define "servicePreview" }}
<div class="card-preview service-style">
  {{ template "serviceIcon" . }}
</div>
{{ end }}

{{/* Custom icon for services */}}
{{ define "serviceIcon" }}
<svg class="icon service-icon" viewBox="0 0 24 24">
  <circle cx="12" cy="12" r="10"/>
  <path d="M12 6v6l4 2"/>
</svg>
{{ end }}

{{/* Extend card to use service preview */}}
{{# extend "Base:card" "serviceCard"
           "Base:cardPreview" "servicePreview" #}}

{{/* Extend grid to use service cards */}}
{{# extend "Base:cardGrid" "serviceGrid"
           "Base:card" "serviceCard" #}}

{{/* Page template */}}
{{ define "servicePage" }}
<main>
  <h1>Our Services</h1>
  {{ template "serviceGrid" . }}
</main>
{{ end }}
```

### How the Extensions Work

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  products.html extension chain:                                             │
│                                                                             │
│  Step 1: extend "Base:card" → "productCard"                                 │
│  ┌────────────────────────────┐       ┌────────────────────────────┐        │
│  │ Base:card                  │       │ productCard                │        │
│  │  calls Base:cardPreview    │  ──►  │  calls productPreview      │        │
│  └────────────────────────────┘       └────────────────────────────┘        │
│                                                                             │
│  Step 2: extend "Base:cardGrid" → "productGrid"                             │
│  ┌────────────────────────────┐       ┌────────────────────────────┐        │
│  │ Base:cardGrid              │       │ productGrid                │        │
│  │  calls Base:card           │  ──►  │  calls productCard         │        │
│  └────────────────────────────┘       └────────────────────────────┘        │
│                                                                             │
│  Result: productGrid → productCard → productPreview → (photo or icon)       │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│  services.html extension chain:                                             │
│                                                                             │
│  Step 1: extend "Base:card" → "serviceCard"                                 │
│  ┌────────────────────────────┐       ┌────────────────────────────┐        │
│  │ Base:card                  │       │ serviceCard                │        │
│  │  calls Base:cardPreview    │  ──►  │  calls servicePreview      │        │
│  └────────────────────────────┘       └────────────────────────────┘        │
│                                                                             │
│  Step 2: extend "Base:cardGrid" → "serviceGrid"                             │
│  ┌────────────────────────────┐       ┌────────────────────────────┐        │
│  │ Base:cardGrid              │       │ serviceGrid                │        │
│  │  calls Base:card           │  ──►  │  calls serviceCard         │        │
│  └────────────────────────────┘       └────────────────────────────┘        │
│                                                                             │
│  Result: serviceGrid → serviceCard → servicePreview → serviceIcon           │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Sample Output

With data `{Items: [{Title: "Widget", ImageUrl: "/img/widget.jpg"}, {Title: "Gadget"}]}`:

**Product page** renders:
```html
<div class="grid">
  <div class="card">
    <div class="card-preview">
      <img src="/img/widget.jpg" alt="Widget">  <!-- has image -->
    </div>
    <h3>Widget</h3>
  </div>
  <div class="card">
    <div class="card-preview">
      <svg class="icon product-icon">...</svg>  <!-- fallback icon -->
    </div>
    <h3>Gadget</h3>
  </div>
</div>
```

**Service page** renders:
```html
<div class="grid">
  <div class="card">
    <div class="card-preview service-style">
      <svg class="icon service-icon">...</svg>  <!-- always uses icon -->
    </div>
    <h3>Widget</h3>
  </div>
  ...
</div>
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
