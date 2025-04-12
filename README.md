# Templar: Go Template Loader

[![Go Reference](https://pkg.go.dev/badge/github.com/panyam/templar.svg)](https://pkg.go.dev/github.com/panyam/templar)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Templar is a powerful extension to Go's standard templating libraries that adds dependency management, simplifies template composition, and solves common pain points in template organization.

## Why Templar?

Templar is designed to integrate smoothly with Go's standard templating libraries while solving common issues:

1. **Minimal Learning Curve**: If you know Go templates, you already know 99% of Templar.
2. **Zero New Runtime Syntax**: The include directives are processed before rendering (variable based inclusion in the
   works).
3. **Flexible and Extensible**: Create custom loaders for any template source (file loader for now, more in the works).
4. **Production Ready**: Handles complex dependencies, prevents cycles, and provides clear error messages (and aiming to
   get better at this).

## Background

Go's built-in templating libraries (`text/template` and `html/template`) are powerful but have limitations when working with complex template structures:

1. **No Native Dependency Management**: When templates reference other templates, you must manually ensure they're loaded in the correct order.

2. **Global Template Namespace**: All template definitions share a global namespace, making it challenging to use different versions of the same template in different contexts.

3. **Brittle Template Resolution**: In large applications, templates often load differently in development vs. production environments.

4. **Verbose Template Loading**: Loading templates with their dependencies typically requires repetitive boilerplate code:

```go
// Standard approach - verbose and error-prone
func renderIndexPage(w http.ResponseWriter, r *http.Request) {
  t := template.ParseFiles("Base1.tmpl", "a2.tmpl", "IndexPage.tmpl")
  t.Execute(w, data)
}

func renderProductListPage(w http.ResponseWriter, r *http.Request) {
  t := template.ParseFiles("AnotherBase.tmpl", "a2.tmpl", "ProductListPage.tmpl")
  t.Execute(w, data)
}
```

## Proposal

Templar solves these problems by providing:

1. **Dependency Declaration**: Templates can declare their own dependencies using a simple include syntax:

```
{{# include "base.tmpl" #}}
{{# include "components/header.tmpl" #}}

<div class="content">
  {{ template "content" . }}
</div>
```

2. **Automatic Template Loading**: Templar automatically loads and processes all dependencies:

```go
// With Templar - clean and maintainable
func renderIndexPage(w http.ResponseWriter, r *http.Request) {
  tmpl := loadTemplate("IndexPage.tmpl")  // Dependencies automatically handled
  tmpl.Execute(w, data)
}
```

3. **Flexible Template Resolution**: Multiple loaders can be configured to find templates in different locations.

4. **Template Reuse**: The same template name can be reused in different contexts without conflict.

## Getting Started

### [Basic Example](https://github.com/panyam/templar/blob/main/examples/main.go)

```go
package main

import (
    "os"
    "github.com/panyam/templar"
)

func main() {
  // Create a template group
  group := templar.NewTemplateGroup()
  
  // Create a filesystem loader that searches multiple directories
  group.Loader = templar.NewFileSystemLoader(
      "templates/",
      "templates/shared/",
  )
  
  // Load a root template (dependencies handled automatically)
  rootTemplate := group.MustLoad("pages/homepage.tmpl", "")

  // Prepare data for the template
  data := map[string]any{
    "Title": "Home Page",
    "User": User{
      ID:   1,
      Name: "John Doe",
    },
    "Updates": []Update{
      {Title: "New Feature Released", Date: "2023-06-15"},
      {Title: "System Maintenance", Date: "2023-06-10"},
      {Title: "Welcome to our New Site", Date: "2023-06-01"},
    },
    "Featured": FeaturedContent{
      Title:       "Summer Sale",
      Description: "Get 20% off on all products until July 31st!",
      URL:         "/summer-sale",
    },
  }

  // Render the template to stdout (for this example)
  if err = group.RenderHtmlTemplate(os.Stdout, rootTemplate[0], "", data, nil); err != nil {
    fmt.Printf("Error rendering template: %v\n", err)
  }
}
```

## Key Features

### 1. Template Dependencies

In your templates, use the `{{# include "path/to/template" #}}` directive to include dependencies:

```html
{{# include "layouts/base.tmpl" #}}
{{# include "components/navbar.tmpl" #}}

{{ define "content" }}
  <h1>Welcome to our site</h1>
  <p>This is the homepage content.</p>
{{ end }}
```

### 2. Multiple Template Loaders

Templar allows you to configure multiple template loaders with fallback behavior:

```go
// Create a list of loaders to search in order
loaderList := &templar.LoaderList{}

// Add loaders in priority order
loaderList.AddLoader(templar.NewFileSystemLoader("app/templates/"))
loaderList.AddLoader(templar.NewFileSystemLoader("shared/templates/"))

// Set a default loader as final fallback
loaderList.DefaultLoader = templar.NewFileSystemLoader("default/templates/")
```

### 3. Template Groups

Template groups manage collections of templates and their dependencies:

```go
group := templar.NewTemplateGroup()
group.Loader = loaderList
group.AddFuncs(map[string]any{
    "formatDate": func(t time.Time) string {
        return t.Format("2006-01-02")
    },
})
```

## Advanced Usage

### Conditional Template Loading

You can implement conditional template loading based on application state:

```go
folder := "desktop"
if isMobile {
  folder = "mobile"
}
tmpl, err := loader.Load(fmt.Sprintf("%s/homepage.tmpl", folder))
```

### Dynamic Templates

Generate templates dynamically and use them immediately:

```go
dynamicTemplate := &templar.Template{
    Name:      "dynamic-template",
    RawSource: []byte(`Hello, {{.Name}}!`),
}

group.RenderTextTemplate(w, dynamicTemplate, "", map[string]any{"Name": "World"}, nil)
```

## Comparison with Other Solutions

| Feature                          | Standard Go Templates | Templar |
|----------------------------------|-----------------------|---------|
| Dependency Management            | ❌                    | ✅      |
| Self-describing Templates (*)    | ❌                    | ✅      |
| Standard Go Template Syntax      | ✅                    | ✅      |
| Supports Cycles Prevention (**)  | ❌                    | ✅      |
| HTML Escaping                    | ✅                    | ✅      |
| Template Grouping (***)          | ⚠️ Partial             | ✅      |

`*`: Self-describing here refers to a template specifying all the dependencies it needs so a template author can be
clear about what is required and include them instead of hoping they exist somehow.
`**`: Cycles are caught by the preprocessor and is clearer.
`***`: Grouping in standard templates is done in code by the template user instead of the author.

### Other alternatives

* [Pongo2](https://github.com/flosch/pongo2) is amazing for its reverence for Django syntax.
* [Templ](https://github.com/a-h/templ) is amazing as a typed template library and being able to perform compile time validations of templates. 

My primary goal here was to have as much alignment with Go's template stdlib.   Beyond this library for managing
dependencies, the goal itself was to have strict adherence to Go's templating syntax.   Using the same Go template
syntax also allows extra features during preprocessing of templates.  (eg using same set of variables for both
pre-processing as well as for final rendering).

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the LICENSE file for details.
