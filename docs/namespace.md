# Template Namespacing with `namespace`

The `namespace` directive imports templates with a prefix, preventing name collisions when combining templates from different sources.

## Basic Syntax

```html
{{# namespace "Prefix" "path/to/template.html" #}}
```

- **Prefix**: The namespace prefix to add to all template names
- **Path**: The template file to import

After importing, all templates from that file are available with the prefix:

```html
{{# namespace "UI" "components/buttons.html" #}}

{{ template "UI:primaryButton" . }}
{{ template "UI:secondaryButton" . }}
```

## Why Namespaces?

Without namespaces, template names are global. If two files define a template with the same name, the second one causes go template loader to throw a duplicate-definition error.

### The Problem: Name Collision

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  buttons.html                        alt-buttons.html                       │
│  ┌─────────────────────────┐         ┌─────────────────────────┐            │
│  │ {{ define "button" }}   │         │ {{ define "button" }}   │            │
│  │   Primary style         │         │   Alt style             │            │
│  │ {{ end }}               │         │ {{ end }}               │            │
│  └─────────────────────────┘         └─────────────────────────┘            │
│              │                                   │                          │
│              │ include                           │ include                  │
│              ▼                                   ▼                          │
│  ┌───────────────────────────────────────────────────────────────┐          │
│  │                     Global Template Space                     │          │
│  │  ┌─────────────────────────────────────────────────────────┐  │          │
│  │  │  "button" = ???  (which one wins?) (dup error thrown)   │  │          │
│  │  └─────────────────────────────────────────────────────────┘  │          │
│  └───────────────────────────────────────────────────────────────┘          │
└─────────────────────────────────────────────────────────────────────────────┘
```

### The Solution: Namespaced Imports

```
┌────────────────────────────────────────────────────────────────────────────────┐
│  buttons.html                        alt-buttons.html                          │
│  ┌─────────────────────────┐         ┌─────────────────────────┐               │
│  │ {{ define "button" }}   │         │ {{ define "button" }}   │               │
│  │   Primary style         │         │   Alt style             │               │
│  │ {{ end }}               │         │ {{ end }}               │               │
│  └─────────────────────────┘         └─────────────────────────┘               │
│              │                                   │                             │
│              │ namespace "Main"                  │ namespace "Alt"             │
│              ▼                                   ▼                             │
│  ┌───────────────────────────────────────────────────────────────┐             │
│  │                     Global Template Space                     │             │
│  │  ┌───────────────────────┐    ┌───────────────────────┐       │             │
│  │  │  "Main:button"        │    │  "Alt:button"         │       │             │
│  │  │   Primary style       │    │   Alt style           │       │             │
│  │  └───────────────────────┘    └───────────────────────┘       │             │
│  └───────────────────────────────────────────────────────────────┘             │
│                                                                                │
│  Both coexist! Use {{ template "Main:button" }} or {{ template "Alt:button" }} │
└────────────────────────────────────────────────────────────────────────────────┘
```

## How Namespacing Transforms Templates

When you namespace a file, all template definitions AND their internal references get prefixed:

### Before: Original File

```
┌─────────────────────────────────────────────────────────────────┐
│  components/card.html                                           │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │ {{ define "card" }}                                       │  │
│  │ <div class="card">                                        │  │
│  │     {{ template "cardHeader" . }}  ──────┐                │  │
│  │     {{ template "cardBody" . }}    ──────┼──┐             │  │
│  │ </div>                                   │  │             │  │
│  │ {{ end }}                                │  │             │  │
│  └──────────────────────────────────────────│──│─────────────┘  │
│                                             │  │                │
│  ┌──────────────────────────────────────────│──│─────────────┐  │
│  │ {{ define "cardHeader" }}  ◄─────────────┘  │             │  │
│  │ <h3>{{ .Title }}</h3>                       │             │  │
│  │ {{ end }}                                   │             │  │
│  └─────────────────────────────────────────────│─────────────┘  │
│                                                │                │
│  ┌─────────────────────────────────────────────│─────────────┐  │
│  │ {{ define "cardBody" }}  ◄──────────────────┘             │  │
│  │ <p>{{ .Content }}</p>                                     │  │
│  │ {{ end }}                                                 │  │
│  └───────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

### After: Namespaced with "UI"

```
{{# namespace "UI" "components/card.html" #}}

┌─────────────────────────────────────────────────────────────────┐
│  Result in Template Space                                       │
│                                                                 │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │ {{ define "UI:card" }}         ◄── name prefixed          │  │
│  │ <div class="card">                                        │  │
│  │     {{ template "UI:cardHeader" . }}  ◄── call prefixed   │  │
│  │     {{ template "UI:cardBody" . }}    ◄── call prefixed   │  │
│  │ </div>                                                    │  │
│  │ {{ end }}                                                 │  │
│  └───────────────────────────────────────────────────────────┘  │
│                                                                 │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │ {{ define "UI:cardHeader" }}   ◄── name prefixed          │  │
│  │ <h3>{{ .Title }}</h3>                                     │  │
│  │ {{ end }}                                                 │  │
│  └───────────────────────────────────────────────────────────┘  │
│                                                                 │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │ {{ define "UI:cardBody" }}     ◄── name prefixed          │  │
│  │ <p>{{ .Content }}</p>                                     │  │
│  │ {{ end }}                                                 │  │
│  └───────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘

Usage: {{ template "UI:card" .CardData }}
```

## Reference Syntax

Within namespaced templates, there are three ways to reference other templates:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  Reference Syntax                                                           │
│                                                                             │
│  ┌─────────────────┬─────────────────────────┬───────────────────────────┐  │
│  │ Syntax          │ Meaning                 │ Example                   │  │
│  ├─────────────────┼─────────────────────────┼───────────────────────────┤  │
│  │ "name"          │ Same namespace          │ "helper" → "NS:helper"    │  │
│  │ "Other:name"    │ Explicit namespace      │ "UI:button" → "UI:button" │  │
│  │ "::name"        │ Global (no namespace)   │ "::global" → "global"     │  │
│  └─────────────────┴─────────────────────────┴───────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Visual Example

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  page.html (loaded with namespace "Page")                                   │
│                                                                             │
│  {{ define "layout" }}                                                      │
│  <div>                                                                      │
│      {{ template "header" . }}       ─────► becomes "Page:header"           │
│      {{ template "UI:navbar" . }}    ─────► stays "UI:navbar" (explicit)    │
│      {{ template "::baseMeta" . }}   ─────► becomes "baseMeta" (global)     │
│  </div>                                                                     │
│  {{ end }}                                                                  │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │ Template Resolution:                                                │    │
│  │                                                                     │    │
│  │   "header"      + namespace "Page"  =  "Page:header"                │    │
│  │   "UI:navbar"   (has colon)         =  "UI:navbar"    (unchanged)   │    │
│  │   "::baseMeta"  (double colon)      =  "baseMeta"     (global)      │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Tree-Shaking with Namespaces

You can selectively import only specific templates (and their dependencies):

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  widgets.html                                                               │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │ button ──────► icon                                                 │    │
│  │ card ────────► cardHeader ──► icon                                  │    │
│  │          └───► cardBody                                             │    │
│  │ modal ───────► button                                               │    │
│  │ tooltip                                                             │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                                                                             │
│  {{# namespace "UI" "widgets.html" "button" #}}                             │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │ INCLUDED (button + dependencies):          EXCLUDED:                │    │
│  │ ┌──────────────────────────────┐          ┌──────────────────────┐  │    │
│  │ │ ✓ UI:button                  │          │ ✗ card               │  │    │
│  │ │ ✓ UI:icon (dependency)       │          │ ✗ cardHeader         │  │    │
│  │ └──────────────────────────────┘          │ ✗ cardBody           │  │    │
│  │                                           │ ✗ modal              │  │    │
│  │                                           │ ✗ tooltip            │  │    │
│  │                                           └──────────────────────┘  │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Without vs With Tree-Shaking

```
WITHOUT tree-shaking:
{{# namespace "UI" "widgets.html" #}}
┌───────────────────────────────────┐
│ UI:button, UI:icon, UI:card,      │
│ UI:cardHeader, UI:cardBody,       │
│ UI:modal, UI:tooltip              │
│ (ALL templates imported)          │
└───────────────────────────────────┘

WITH tree-shaking:
{{# namespace "UI" "widgets.html" "button" #}}
┌───────────────────────────────────┐
│ UI:button, UI:icon                │
│ (only button + its dependencies)  │
└───────────────────────────────────┘
```

## The Diamond Problem

When multiple libraries include the same shared template with different namespaces, each gets its own isolated copy:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              page.html                                      │
│                      {{# include "libA.html" #}}                            │
│                      {{# include "libB.html" #}}                            │
│                           /            \                                    │
│                          /              \                                   │
│                         ▼                ▼                                  │
│  ┌─────────────────────────────┐  ┌─────────────────────────────┐           │
│  │         libA.html           │  │         libB.html           │           │
│  │ {{# namespace "A"           │  │ {{# namespace "B"           │           │
│  │    "shared.html" #}}        │  │    "shared.html" #}}        │           │
│  │                             │  │                             │           │
│  │ {{ define "libA" }}         │  │ {{ define "libB" }}         │           │
│  │   Uses {{ template          │  │   Uses {{ template          │           │
│  │         "A:widget" }}       │  │         "B:widget" }}       │           │
│  │ {{ end }}                   │  │ {{ end }}                   │           │
│  └──────────────┬──────────────┘  └──────────────┬──────────────┘           │
│                 │                                │                          │
│                 │ namespace "A"                  │ namespace "B"            │
│                 ▼                                ▼                          │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │                          shared.html                                │    │
│  │                   {{ define "widget" }}                             │    │
│  │                       [WIDGET]                                      │    │
│  │                   {{ end }}                                         │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │ RESULT: Both coexist independently!                                 │    │
│  │                                                                     │    │
│  │   "A:widget" ──► [WIDGET]  (for libA)                               │    │
│  │   "B:widget" ──► [WIDGET]  (for libB)                               │    │
│  │   "libA"     ──► Uses A:widget                                      │    │
│  │   "libB"     ──► Uses B:widget                                      │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────────────┘

Output: "LibA uses [WIDGET] AND LibB uses [WIDGET]"
```

## Combining Namespace and Extend

Namespaces work together with the `extend` directive. First namespace to import, then extend to customize:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  STEP 1: Namespace loads the base templates                                 │
│                                                                             │
│  {{# namespace "Base" "layouts/base.html" #}}                               │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │ Base:layout ──► Base:header                                         │    │
│  │            └──► Base:content                                        │    │
│  │            └──► Base:footer                                         │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                                                                             │
│  STEP 2: Extend creates customized version                                  │
│                                                                             │
│  {{# extend "Base:layout" "MyLayout"                                        │
│             "Base:header" "myHeader"                                        │
│             "Base:footer" "myFooter" #}}                                    │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │ MyLayout ────► myHeader        (rewired from Base:header)           │    │
│  │          └──► Base:content     (unchanged - not in rewrite list)    │    │
│  │          └──► myFooter         (rewired from Base:footer)           │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                                                                             │
│  STEP 3: Define your custom templates                                       │
│                                                                             │
│  {{ define "myHeader" }}<header>Custom Header</header>{{ end }}             │
│  {{ define "myFooter" }}<footer>Custom Footer</footer>{{ end }}             │
│                                                                             │
│  {{ template "MyLayout" . }}                                                │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Practical Example: Component Library

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  app/pages/dashboard.html                                                   │
│                                                                             │
│  {{# namespace "UI" "shared/components.html" #}}                            │
│  {{# namespace "Charts" "shared/charts.html" #}}                            │
│  {{# namespace "Layout" "shared/layouts.html" #}}                           │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │                        Template Space                               │    │
│  │                                                                     │    │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐               │    │
│  │  │ UI:          │  │ Charts:      │  │ Layout:      │               │    │
│  │  │  navMenu     │  │  lineChart   │  │  twoColumn   │               │    │
│  │  │  dataTable   │  │  barChart    │  │  sidebar     │               │    │
│  │  │  button      │  │  pieChart    │  │  main        │               │    │
│  │  └──────────────┘  └──────────────┘  └──────────────┘               │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                                                                             │
│  {{ define "DashboardPage" }}                                               │
│      {{ template "Layout:twoColumn" . }}                                    │
│  {{ end }}                                                                  │
│                                                                             │
│  {{ define "Layout:sidebar" }}                                              │
│      <nav>{{ template "UI:navMenu" .MenuItems }}</nav>                      │
│  {{ end }}                                                                  │
│                                                                             │
│  {{ define "Layout:main" }}                                                 │
│      <div class="dashboard">                                                │
│          {{ template "Charts:lineChart" .SalesData }}                       │
│          {{ template "Charts:barChart" .InventoryData }}                    │
│          {{ template "UI:dataTable" .RecentOrders }}                        │
│      </div>                                                                 │
│  {{ end }}                                                                  │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Key Points

1. **Prefixes are added automatically** - All templates from the imported file get the namespace prefix
2. **Internal references are updated** - Template calls within the file are also prefixed
3. **Cross-namespace calls use explicit prefix** - Use `Other:name` to call templates from different namespaces
4. **Global references use `::`** - Use `::name` to reference templates without any namespace
5. **Tree-shaking is optional** - List specific template names after the path to import only those

## Gotchas and Common Mistakes

### 1. Empty namespace is invalid

```html
{{/* WRONG - empty namespace */}}
{{# namespace "" "components.html" #}}

{{/* CORRECT - provide a namespace name */}}
{{# namespace "UI" "components.html" #}}
```

### 2. Missing namespace prefix when calling

```
┌─────────────────────────────────────────────────────────────────┐
│  {{# namespace "UI" "components.html" #}}                       │
│                                                                 │
│  {{ template "button" . }}      ✗ WRONG - "button" not found   │
│  {{ template "UI:button" . }}   ✓ CORRECT - use prefix         │
└─────────────────────────────────────────────────────────────────┘
```

### 3. Referencing global templates from namespaced code

```
┌─────────────────────────────────────────────────────────────────┐
│  helpers.html (included globally):                              │
│  {{ define "formatDate" }}2024-01-01{{ end }}                   │
│                                                                 │
│  components.html (namespaced as "UI"):                          │
│  {{ define "card" }}                                            │
│      Date: {{ template "::formatDate" . }}   ◄── use :: prefix  │
│  {{ end }}                                                      │
│                                                                 │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │ Without ::  →  looks for "UI:formatDate"  →  NOT FOUND    │  │
│  │ With ::     →  looks for "formatDate"     →  FOUND        │  │
│  └───────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

**Important**: The `::` prefix is consumed during namespace application. After namespacing, `{{ template "::formatDate" }}` becomes `{{ template "formatDate" }}` in the processed template. This is a one-shot escape - if the resulting template were re-namespaced (which is not a typical use case), the now-plain `formatDate` would get the new namespace prefix. Templates are generally not designed to be re-namespaced.

### 4. Order matters: namespace before extend

```
┌─────────────────────────────────────────────────────────────────┐
│  WRONG ORDER:                                                   │
│  {{# extend "Base:layout" "MyLayout" ... #}}   ◄── Base:layout  │
│  {{# namespace "Base" "layouts.html" #}}           doesn't      │
│                                                    exist yet!   │
│                                                                 │
│  CORRECT ORDER:                                                 │
│  {{# namespace "Base" "layouts.html" #}}       ◄── load first   │
│  {{# extend "Base:layout" "MyLayout" ... #}}   ◄── then extend  │
└─────────────────────────────────────────────────────────────────┘
```

## Debugging Tips

1. **List available templates**: Use `templar debug --defines` to see all defined templates
2. **Check references**: Use `templar debug --refs` to see cross-template dependencies
3. **Trace loading**: Use `templar debug --trace` to see how paths are resolved
