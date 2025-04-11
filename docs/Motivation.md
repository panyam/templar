
Go templates are very powerful and quite robust.   One of the irksome patterns is their lack of dependency management.

For example even though it is easier to just load multiple files with ParseFS or ParseGlob, sometimes order matters.  Consider the following templates:

a1.tmpl:

```
{{ define "A" }}
Prints A1
{{ end }}
```

a2.tmpl:

```
{{ define "A" }}
Prints A2
{{ end }}
```

IndexPage.tmpl:

```
// Needs A1
{{ template "A" }}
```

ProductListPage.tmpl:

```
// Needs A2
{{ template "A" }}
```

In this typical scenario, IndexPage "needs" A1 where as ProductListPage needs "A2".  There is no clear way to indicate this unless we fall back to code, eg:

```
func renderIndexPage(w http.ResponseWriter, r *http.Request) {
  t := template.ParseFiles("a1.tmpl", "IndexPage.tmpl")
  t.Execute(w, ...)
}

func renderProductListPage(w http.ResponseWriter, r *http.Request) {
  t := template.ParseFiles("a1.tmpl", "ProductListPage.tmpl")
  t.Execute(w, ...)
}
```

Instead it would be easier if we could something like:

```
func renderIndexPage(w http.ResponseWriter, r *http.Request) {
  t := template.ParseFiles("IndexPage.tmpl")    // <---- IndexPage is all that is called as it is the "root"
  t.Execute(w, ...)
}
```

with the template being:

```IndexPage.tmpl
{{ include "a1" }}      // <--- includes A from a1

{{ template "A" }}
```

Goal of this small library is to enable dependencies and imports in a template.  Note that inheritance mechanisms do not need to be changed if we have this kind of "mixin" ability.  Before the proposal is shared, note:

1. Go templates only allows one definition for a given name and it is global.  
2. Also these values are only resolved at render time and not at parse time.   So even if two definitions are parsed the second will overwrite the first.
3. Finally definitions cannot be changed or parsed after being executed.

Typical use case is that when a view/template is rendered, it knows what dependencies it needs (or it can be parametrized but it is still known).   So why not have something like:

```
indexPage = loadTemplate("IndexPage.html")    // of type text.Template or html.Template
indexPage.Execute(writer, data)               // Like before
```

[###](###) Proposal

Introduce include tags that will manage dependencies when loading templates.   So this is a template template of sorts.   First we load the templates and take care of dependencies from the include tags, and *then* the resultant template is parsed and returned.

Since in the final template we want to remove all include tags, we can simply have different delimiters in the first phase:

```
{{# include "a1" #}}   // a1 is parsed - note "a1" can be a literal or a variable - Parse trees from all of a1's DefinedTemplates are added here
{{# include b1 #}}     // b1 is parsed - but provided as a variable to allow caller to override}
{{# include "/afolder/*.templ" #}}     // b1 is parsed - but provided as a variable to allow caller to override}
{{# from "b1" include C D as D1 "E" as E1 "F" #}}   // parse b1 but add only some parse trees (and ensure they can be renamed to avoid duplicates)
                                                    // C will be "C", D will be "D1" and so on, Problem with partial imports is that imported
                                                    // templates *may* depend on unimported templates. It is not clear whether this will implcitly bring them in too.

... Rest of the template ...

{{ template "C" }}
```

One of the goal is this should only act as a pre-processor for go templates.   The "final" template should be a valid go (text or html) template and not need any syntax changes.  So the above will look like:

```
.... load all templates in "a1" ...
.... load all templates in b1 ...
.... load C, D, E and F (with renames) from "b1"

... rest of the template ...

{{ template "C" }}
```

### Considerations

1. Cycle detection

Cycles are possible but must be detected and rejected.  For example "a1" may contain an import of "b1" which may import "a1".  This should be caught and rejected.  Similarly it could happen dynamically when include has "variables".

2. Efficiency

Loading templates may be slow with all the transitive loads and cycle checking etc - but this happens today when we load N template files anyway.  Just like today these should be cached.   Typically templates do not change often so binary reloads are fine.

3. Memory consumption

This can be an issue.  If two different root pages - load/include the same base template then the two templates would have duplicate parse trees for the base template.   This is an issue today when template order and lists are curated per page.   In the future we can see how to fix this via template DAGs (if references instead of cloning is available with go templates).

4. Invalidation and Reloads

While refreshing to changes are not a high prioirity, sometimes (for dev mode) it may be desirable to update templates based on changes.   This is to be investigated more.  Even if a graph of template dependency is maintained it can fail in unrecoverable ways forcing a reload anyway.
