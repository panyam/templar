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

Without namespaces, template names are global. If two files define a template with the same name, the second overwrites the first:

```
buttons.html:     {{ define "button" }} Primary style {{ end }}
alt-buttons.html: {{ define "button" }} Alt style {{ end }}

<!-- Problem: which "button" do you get? -->
{{ template "button" . }}
```

With namespaces, both can coexist:

```html
{{# namespace "Main" "buttons.html" #}}
{{# namespace "Alt" "alt-buttons.html" #}}

{{ template "Main:button" . }}  <!-- Primary style -->
{{ template "Alt:button" . }}   <!-- Alt style -->
```

## Namespace Resolution Rules

When templates are imported into a namespace, their internal references are also prefixed:

### Before namespacing (original file)
```html
<!-- components/card.html -->
{{ define "card" }}
<div class="card">
    {{ template "cardHeader" . }}
    {{ template "cardBody" . }}
</div>
{{ end }}

{{ define "cardHeader" }}<h3>{{ .Title }}</h3>{{ end }}
{{ define "cardBody" }}<p>{{ .Content }}</p>{{ end }}
```

### After namespacing
```html
{{# namespace "UI" "components/card.html" #}}

<!-- Templates become: -->
<!-- UI:card        - calls UI:cardHeader and UI:cardBody -->
<!-- UI:cardHeader  -->
<!-- UI:cardBody    -->

{{ template "UI:card" . }}
```

## Reference Syntax

Within namespaced templates, there are three ways to reference other templates:

| Syntax | Meaning | Example |
|--------|---------|---------|
| `name` | Same namespace | `{{ template "helper" }}` → `NS:helper` |
| `Other:name` | Explicit namespace | `{{ template "UI:button" }}` → `UI:button` |
| `::name` | Global (no namespace) | `{{ template "::global" }}` → `global` |

### Example

```html
{{# namespace "Page" "page.html" #}}
{{# namespace "UI" "components.html" #}}

<!-- In page.html: -->
{{ define "layout" }}
<div>
    {{ template "header" . }}      <!-- becomes Page:header -->
    {{ template "UI:navbar" . }}   <!-- stays UI:navbar -->
    {{ template "::baseMeta" . }}  <!-- stays baseMeta (global) -->
</div>
{{ end }}
```

## Tree-Shaking with Namespaces

You can selectively import only specific templates (and their dependencies):

```html
{{# namespace "UI" "widgets.html" "button" "icon" #}}
```

This imports only:
- `UI:button`
- `UI:icon`
- Any templates that `button` and `icon` depend on

Templates not referenced are excluded, reducing the final template size.

### Without tree-shaking
```html
{{# namespace "UI" "widgets.html" #}}
<!-- Imports ALL templates: button, icon, card, modal, tooltip, ... -->
```

### With tree-shaking
```html
{{# namespace "UI" "widgets.html" "button" #}}
<!-- Imports only: button (and any templates button calls) -->
```

## Combining Namespace and Extend

Namespaces work together with the `extend` directive. First namespace to import, then extend to customize:

```html
{{# namespace "Base" "layouts/base.html" #}}

{{# extend "Base:layout" "MyLayout"
           "Base:header" "myHeader"
           "Base:footer" "myFooter" #}}

{{ define "myHeader" }}
<header>My Custom Header</header>
{{ end }}

{{ define "myFooter" }}
<footer>My Custom Footer</footer>
{{ end }}

{{ template "MyLayout" . }}
```

## Practical Example: Component Library

```html
<!-- app/pages/dashboard.html -->

{{# namespace "UI" "shared/components.html" #}}
{{# namespace "Charts" "shared/charts.html" #}}
{{# namespace "Layout" "shared/layouts.html" #}}

{{ define "DashboardPage" }}
{{ template "Layout:twoColumn" . }}
{{ end }}

{{ define "Layout:sidebar" }}
<nav>
    {{ template "UI:navMenu" .MenuItems }}
</nav>
{{ end }}

{{ define "Layout:main" }}
<div class="dashboard">
    {{ template "Charts:lineChart" .SalesData }}
    {{ template "Charts:barChart" .InventoryData }}
    {{ template "UI:dataTable" .RecentOrders }}
</div>
{{ end }}
```

## Key Points

1. **Prefixes are added automatically** - All templates from the imported file get the namespace prefix
2. **Internal references are updated** - Template calls within the file are also prefixed
3. **Cross-namespace calls use explicit prefix** - Use `Other:name` to call templates from different namespaces
4. **Global references use `::`** - Use `::name` to reference templates without any namespace
5. **Tree-shaking is optional** - List specific template names after the path to import only those

## The Diamond Problem

When multiple libraries include the same shared template with different namespaces, each gets its own isolated copy:

```
        Page
       /    \
     LibA   LibB
       \    /
       Shared
```

```html
<!-- shared.html -->
{{ define "widget" }}[WIDGET]{{ end }}

<!-- libA.html -->
{{# namespace "A" "shared.html" #}}
{{ define "libA" }}LibA uses {{ template "A:widget" . }}{{ end }}

<!-- libB.html -->
{{# namespace "B" "shared.html" #}}
{{ define "libB" }}LibB uses {{ template "B:widget" . }}{{ end }}

<!-- page.html -->
{{# include "libA.html" #}}
{{# include "libB.html" #}}
{{ define "page" }}
{{ template "libA" . }} AND {{ template "libB" . }}
{{ end }}
```

Result: Both `A:widget` and `B:widget` exist independently. Each library gets its own namespaced version of the shared template.

## Gotchas and Common Mistakes

### 1. Empty namespace is invalid

```html
{{/* WRONG - empty namespace */}}
{{# namespace "" "components.html" #}}

{{/* CORRECT - provide a namespace name */}}
{{# namespace "UI" "components.html" #}}
```

### 2. Missing namespace prefix when calling

After namespacing, you must use the prefix to call templates:

```html
{{# namespace "UI" "components.html" #}}

{{/* WRONG - "button" doesn't exist, only "UI:button" */}}
{{ template "button" . }}

{{/* CORRECT - use the namespace prefix */}}
{{ template "UI:button" . }}
```

### 3. Internal references within namespaced templates

Templates inside a namespaced file that call each other are automatically prefixed:

```html
<!-- components.html -->
{{ define "card" }}
<div>{{ template "cardBody" . }}</div>  <!-- becomes UI:cardBody -->
{{ end }}
{{ define "cardBody" }}<p>Content</p>{{ end }}
```

When loaded with `{{# namespace "UI" "components.html" #}}`:
- `card` becomes `UI:card`
- `cardBody` becomes `UI:cardBody`
- The call to `cardBody` inside `card` becomes `UI:cardBody`

### 4. Referencing global templates from namespaced code

Use `::` prefix to escape the namespace and reference global templates:

```html
<!-- helpers.html (loaded globally) -->
{{ define "formatDate" }}2024-01-01{{ end }}

<!-- components.html -->
{{ define "card" }}
<div>Date: {{ template "::formatDate" . }}</div>
{{ end }}
```

```html
{{# include "helpers.html" #}}
{{# namespace "UI" "components.html" #}}
<!-- UI:card correctly calls global formatDate, not UI:formatDate -->
```

### 5. Tree-shaking includes dependencies

When you tree-shake, transitive dependencies are automatically included:

```html
<!-- components.html -->
{{ define "form" }}{{ template "input" . }}{{ template "button" . }}{{ end }}
{{ define "input" }}<input/>{{ end }}
{{ define "button" }}<button/>{{ end }}
{{ define "unused" }}not needed{{ end }}

{{/* Only request "form", but input and button come along */}}
{{# namespace "UI" "components.html" "form" #}}
<!-- Available: UI:form, UI:input, UI:button -->
<!-- NOT available: UI:unused -->
```

### 6. Namespace before extend

When combining with `extend`, always namespace first:

```html
{{/* CORRECT ORDER */}}
{{# namespace "Base" "layouts.html" #}}
{{# extend "Base:layout" "MyLayout" ... #}}

{{/* WRONG - can't extend what doesn't exist yet */}}
{{# extend "Base:layout" "MyLayout" ... #}}
{{# namespace "Base" "layouts.html" #}}
```

## Debugging Tips

1. **List available templates**: After loading, check what templates exist with `templar debug --defines`
2. **Check references**: Use `templar debug --refs` to see cross-template dependencies
3. **Trace loading**: Use `templar debug --trace` to see how paths are resolved
