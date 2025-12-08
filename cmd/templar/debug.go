package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/panyam/templar"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var debugCmd = &cobra.Command{
	Use:   "debug <template-file>",
	Short: "Analyze template dependencies and debug issues",
	Long: `Analyze template files and their dependencies.

Features:
  - Detect dependency cycles
  - Show template definitions and references
  - Output GraphViz DOT format for visualization
  - Flatten/preprocess templates
  - Trace path resolution

Config file options (debug section):
  debug:
    path: "templates,../shared"
    verbose: false
    cycles: true
    defines: false
    refs: false

Examples:
  templar debug -p templates,../shared WorldListingPage.html
  templar debug -v --cycles WorldListingPage.html
  templar debug --dot WorldListingPage.html > deps.dot
  templar debug --flatten WorldListingPage.html
  templar debug --trace WorldListingPage.html`,
	Args: cobra.ExactArgs(1),
	Run:  runDebug,
}

func init() {
	debugCmd.Flags().StringP("path", "p", ".", "Comma-separated search paths for templates")
	debugCmd.Flags().BoolP("verbose", "v", false, "Verbose output")
	debugCmd.Flags().Bool("defines", false, "Show template definitions")
	debugCmd.Flags().Bool("refs", false, "Show template references")
	debugCmd.Flags().Bool("cycles", true, "Detect dependency cycles")
	debugCmd.Flags().Bool("dot", false, "Output GraphViz DOT format")
	debugCmd.Flags().Bool("flatten", false, "Output flattened/preprocessed template")
	debugCmd.Flags().Bool("trace", false, "Trace path resolution for includes")

	// Bind flags to viper
	viper.BindPFlag("debug.path", debugCmd.Flags().Lookup("path"))
	viper.BindPFlag("debug.verbose", debugCmd.Flags().Lookup("verbose"))
	viper.BindPFlag("debug.defines", debugCmd.Flags().Lookup("defines"))
	viper.BindPFlag("debug.refs", debugCmd.Flags().Lookup("refs"))
	viper.BindPFlag("debug.cycles", debugCmd.Flags().Lookup("cycles"))
	viper.BindPFlag("debug.dot", debugCmd.Flags().Lookup("dot"))
	viper.BindPFlag("debug.flatten", debugCmd.Flags().Lookup("flatten"))
	viper.BindPFlag("debug.trace", debugCmd.Flags().Lookup("trace"))

	// Set defaults
	viper.SetDefault("debug.path", ".")
	viper.SetDefault("debug.cycles", true)
}

// Directive represents a parsed templar directive
type Directive struct {
	Type      string   // "include", "namespace", "extend"
	File      string   // for include/namespace: the file path
	Namespace string   // for namespace: the namespace name
	Args      []string // additional arguments
	Line      int      // line number in source
}

// TemplateInfo holds parsed information about a template file
type TemplateInfo struct {
	Path         string
	Directives   []Directive
	Defines      []string // template names defined in this file
	TemplateRefs []string // templates referenced via {{ template "X" }}
	Error        error
}

// DependencyGraph tracks template dependencies
type DependencyGraph struct {
	templates    map[string]*TemplateInfo
	searchPaths  []string
	extensions   map[string][]string // namespace prefixes to expand
	traceResolve bool                // show path resolution
}

var (
	// Regex patterns for parsing
	includePattern     = regexp.MustCompile(`\{\{#\s*include\s+"([^"]+)"(?:\s+"([^"]+)")*\s*#\}\}`)
	namespacePattern   = regexp.MustCompile(`\{\{#\s*namespace\s+"([^"]+)"\s+"([^"]+)"(?:\s+"([^"]+)")*\s*#\}\}`)
	extendPattern      = regexp.MustCompile(`\{\{#\s*extend\s+"([^"]+)"\s+"([^"]+)"(?:\s+"([^"]+)"\s+"([^"]+)")*\s*#\}\}`)
	definePattern      = regexp.MustCompile(`\{\{\s*define\s+"([^"]+)"`)
	templateRefPattern = regexp.MustCompile(`\{\{\s*(?:template|block)\s+"([^"]+)"`)
	// Pattern to strip comments (both HTML and Go template comments)
	htmlCommentPattern = regexp.MustCompile(`<!--[\s\S]*?-->`)
	goCommentPattern   = regexp.MustCompile(`\{\{/\*[\s\S]*?\*/\}\}`)
	// Pattern to strip commented directive examples in documentation
	commentedDirectivePattern = regexp.MustCompile(`\{\{#/\*[\s\S]*?\*/\s*#\}\}`)
)

func runDebug(cmd *cobra.Command, args []string) {
	templateFile := args[0]

	// Get config values from viper
	searchPath := viper.GetString("debug.path")
	verbose := viper.GetBool("debug.verbose")
	showDefines := viper.GetBool("debug.defines")
	showRefs := viper.GetBool("debug.refs")
	detectCycles := viper.GetBool("debug.cycles")
	outputDot := viper.GetBool("debug.dot")
	flatten := viper.GetBool("debug.flatten")
	traceResolve := viper.GetBool("debug.trace")

	paths := strings.Split(searchPath, ",")

	// Handle flatten mode separately using the actual templar library
	if flatten {
		flattenTemplate(templateFile, paths, traceResolve)
		return
	}

	graph := &DependencyGraph{
		templates:    make(map[string]*TemplateInfo),
		searchPaths:  paths,
		extensions:   make(map[string][]string),
		traceResolve: traceResolve,
	}

	// Parse the root template and all dependencies
	fmt.Printf("Analyzing: %s\n", templateFile)
	fmt.Printf("Search paths: %v\n\n", paths)

	rootInfo, err := graph.analyzeTemplate(templateFile, "")
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		os.Exit(1)
	}

	if outputDot {
		graph.outputDOT(templateFile)
		return
	}

	// Print dependency tree
	fmt.Println("=== Dependency Tree ===")
	graph.printTree(templateFile, "", make(map[string]bool), verbose)

	// Show defines
	if showDefines {
		fmt.Println("\n=== Template Definitions ===")
		for path, info := range graph.templates {
			if len(info.Defines) > 0 {
				fmt.Printf("%s:\n", filepath.Base(path))
				for _, def := range info.Defines {
					fmt.Printf("  - %s\n", def)
				}
			}
		}
	}

	// Show references
	if showRefs {
		fmt.Println("\n=== Template References ===")
		for path, info := range graph.templates {
			if len(info.TemplateRefs) > 0 {
				fmt.Printf("%s:\n", filepath.Base(path))
				for _, ref := range info.TemplateRefs {
					fmt.Printf("  → %s\n", ref)
				}
			}
		}
	}

	// Detect cycles
	if detectCycles {
		fmt.Println("\n=== Cycle Detection ===")
		cycles := graph.detectCycles(templateFile)
		if len(cycles) == 0 {
			fmt.Println("No cycles detected in include/namespace graph.")
		} else {
			fmt.Printf("Found %d cycle(s):\n", len(cycles))
			for i, cycle := range cycles {
				// Shorten paths for readability
				shortCycle := make([]string, len(cycle))
				for j, p := range cycle {
					shortCycle[j] = filepath.Base(p)
				}
				fmt.Printf("  Cycle %d: %s\n", i+1, strings.Join(shortCycle, " → "))
			}
		}

		// Check for extension issues
		fmt.Println("\n=== Extension Analysis ===")
		issues := graph.analyzeExtensions(rootInfo)
		if len(issues) == 0 {
			fmt.Println("No extension issues detected.")
		} else {
			fmt.Println("Potential issues:")
			for _, issue := range issues {
				fmt.Printf("  ! %s\n", issue)
			}
		}
	}

	// Summary
	fmt.Println("\n=== Summary ===")
	fmt.Printf("Total templates analyzed: %d\n", len(graph.templates))

	var totalDefines, totalRefs int
	for _, info := range graph.templates {
		totalDefines += len(info.Defines)
		totalRefs += len(info.TemplateRefs)
	}
	fmt.Printf("Total definitions: %d\n", totalDefines)
	fmt.Printf("Total references: %d\n", totalRefs)
}

// flattenTemplate uses the actual templar library to flatten a template
func flattenTemplate(templateFile string, searchPaths []string, trace bool) {
	// Create loader
	loader := templar.NewFileSystemLoader(searchPaths...)

	// Create a custom tracing loader if trace is enabled
	var actualLoader templar.TemplateLoader = loader
	if trace {
		actualLoader = &TracingLoader{
			inner:       loader,
			searchPaths: searchPaths,
		}
	}

	// Load the template
	templates, err := actualLoader.Load(templateFile, "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR loading template: %v\n", err)
		os.Exit(1)
	}

	if len(templates) == 0 {
		fmt.Fprintf(os.Stderr, "ERROR: no templates found for %s\n", templateFile)
		os.Exit(1)
	}

	root := templates[0]

	// Walk and preprocess
	if trace {
		fmt.Fprintln(os.Stderr, "\n=== Path Resolution Trace ===")
	}

	// Collect all extensions from all processed templates
	var allExtensions []templar.Extension

	walker := &templar.Walker{
		Loader: actualLoader,
		FoundInclude: func(included string) bool {
			return false // process all includes
		},
		ProcessedTemplate: func(t *templar.Template) error {
			// Collect extensions from each template
			allExtensions = append(allExtensions, t.Extensions...)
			return nil
		},
	}

	err = walker.Walk(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR preprocessing template: %v\n", err)
		os.Exit(1)
	}

	// Output the flattened template
	if trace {
		fmt.Fprintln(os.Stderr, "\n=== Flattened Template ===")
	}
	fmt.Println(root.ParsedSource)

	// Show extensions that were collected
	if len(allExtensions) > 0 {
		fmt.Fprintln(os.Stderr, "\n=== Extensions (will create templates at render time) ===")
		for _, ext := range allExtensions {
			fmt.Fprintf(os.Stderr, "  - %s -> %s\n", ext.SourceTemplate, ext.DestTemplate)
			for old, newTmpl := range ext.Rewrites {
				fmt.Fprintf(os.Stderr, "      rewire: %s -> %s\n", old, newTmpl)
			}
		}
	}
}

// TracingLoader wraps a loader to trace path resolution
type TracingLoader struct {
	inner       templar.TemplateLoader
	searchPaths []string
	depth       int
}

func (t *TracingLoader) Load(pattern string, cwd string) ([]*templar.Template, error) {
	indent := strings.Repeat("  ", t.depth)
	fmt.Fprintf(os.Stderr, "%s-> Loading \"%s\"", indent, pattern)
	if cwd != "" {
		fmt.Fprintf(os.Stderr, " (from: %s)", filepath.Base(cwd))
	}
	fmt.Fprintln(os.Stderr)

	t.depth++
	defer func() { t.depth-- }()

	templates, err := t.inner.Load(pattern, cwd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s  X Not found: %v\n", indent, err)
		return nil, err
	}

	for _, tmpl := range templates {
		if tmpl.Path != "" {
			fmt.Fprintf(os.Stderr, "%s  OK Resolved to: %s\n", indent, tmpl.Path)
		}
	}

	return templates, nil
}

func (g *DependencyGraph) analyzeTemplate(name string, fromDir string) (*TemplateInfo, error) {
	// Resolve the full path
	fullPath, err := g.resolvePath(name, fromDir)
	if err != nil {
		return nil, err
	}

	// Check if already analyzed
	if info, ok := g.templates[fullPath]; ok {
		return info, nil
	}

	// Read and parse the file
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %w", fullPath, err)
	}

	info := &TemplateInfo{
		Path: fullPath,
	}
	g.templates[fullPath] = info

	// Strip comments before parsing to avoid false positives
	cleanContent := stripComments(string(content))

	// Parse directives
	info.Directives = g.parseDirectives(cleanContent)
	info.Defines = g.parseDefines(cleanContent)
	info.TemplateRefs = g.parseTemplateRefs(cleanContent)

	// Recursively analyze dependencies
	dir := filepath.Dir(fullPath)
	for _, directive := range info.Directives {
		switch directive.Type {
		case "include", "namespace":
			if g.traceResolve {
				fmt.Printf("  -> Loading \"%s\" from %s\n", directive.File, filepath.Base(fullPath))
			}
			resolvedPath, err := g.resolvePath(directive.File, dir)
			if err != nil {
				fmt.Printf("  Warning: could not resolve %s: %v\n", directive.File, err)
				continue
			}
			if g.traceResolve {
				fmt.Printf("    Resolved to: %s\n", resolvedPath)
			}
			_, err = g.analyzeTemplate(directive.File, dir)
			if err != nil {
				fmt.Printf("  Warning: could not analyze %s: %v\n", directive.File, err)
			}
			if directive.Type == "namespace" && directive.Namespace != "" {
				g.extensions[directive.Namespace] = append(g.extensions[directive.Namespace], directive.File)
			}
		}
	}

	return info, nil
}

// stripComments removes HTML and Go template comments to avoid false positives
func stripComments(content string) string {
	// Remove commented directive examples like {{#/* ... */#}}
	content = commentedDirectivePattern.ReplaceAllString(content, "")
	// Remove HTML comments
	content = htmlCommentPattern.ReplaceAllString(content, "")
	// Remove Go template comments
	content = goCommentPattern.ReplaceAllString(content, "")
	return content
}

func (g *DependencyGraph) resolvePath(name string, fromDir string) (string, error) {
	// Try relative to fromDir first
	if fromDir != "" {
		candidate := filepath.Join(fromDir, name)
		if _, err := os.Stat(candidate); err == nil {
			return filepath.Abs(candidate)
		}
	}

	// Try search paths
	for _, searchPath := range g.searchPaths {
		candidate := filepath.Join(searchPath, name)
		if _, err := os.Stat(candidate); err == nil {
			return filepath.Abs(candidate)
		}
	}

	// Try as absolute path
	if _, err := os.Stat(name); err == nil {
		return filepath.Abs(name)
	}

	return "", fmt.Errorf("template not found: %s (searched in %s and %v)", name, fromDir, g.searchPaths)
}

func (g *DependencyGraph) parseDirectives(content string) []Directive {
	var directives []Directive
	lines := strings.Split(content, "\n")

	for lineNum, line := range lines {
		// Skip if line looks like it's in a comment block
		if strings.Contains(line, "USAGE") || strings.Contains(line, "Example") {
			continue
		}

		// Parse include directives
		if matches := includePattern.FindAllStringSubmatch(line, -1); matches != nil {
			for _, match := range matches {
				d := Directive{
					Type: "include",
					File: match[1],
					Line: lineNum + 1,
				}
				if len(match) > 2 && match[2] != "" {
					d.Args = append(d.Args, match[2])
				}
				directives = append(directives, d)
			}
		}

		// Parse namespace directives
		if matches := namespacePattern.FindAllStringSubmatch(line, -1); matches != nil {
			for _, match := range matches {
				d := Directive{
					Type:      "namespace",
					Namespace: match[1],
					File:      match[2],
					Line:      lineNum + 1,
				}
				if len(match) > 3 && match[3] != "" {
					d.Args = append(d.Args, match[3])
				}
				directives = append(directives, d)
			}
		}

		// Parse extend directives
		if strings.Contains(line, "extend") && strings.Contains(line, "{{#") {
			// More flexible parsing for extend
			re := regexp.MustCompile(`\{\{#\s*extend\s+(.+?)\s*#\}\}`)
			if match := re.FindStringSubmatch(line); match != nil {
				args := parseQuotedStrings(match[1])
				if len(args) >= 2 {
					d := Directive{
						Type: "extend",
						Args: args,
						Line: lineNum + 1,
					}
					directives = append(directives, d)
				}
			}
		}
	}

	return directives
}

func parseQuotedStrings(s string) []string {
	re := regexp.MustCompile(`"([^"]+)"`)
	matches := re.FindAllStringSubmatch(s, -1)
	var result []string
	for _, m := range matches {
		result = append(result, m[1])
	}
	return result
}

func (g *DependencyGraph) parseDefines(content string) []string {
	var defines []string
	seen := make(map[string]bool)
	matches := definePattern.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		name := match[1]
		if !seen[name] {
			defines = append(defines, name)
			seen[name] = true
		}
	}
	sort.Strings(defines)
	return defines
}

func (g *DependencyGraph) parseTemplateRefs(content string) []string {
	var refs []string
	seen := make(map[string]bool)
	matches := templateRefPattern.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		name := match[1]
		if !seen[name] {
			refs = append(refs, name)
			seen[name] = true
		}
	}
	sort.Strings(refs)
	return refs
}

func (g *DependencyGraph) printTree(path string, indent string, visited map[string]bool, verbose bool) {
	info, ok := g.templates[path]
	if !ok {
		fmt.Printf("%s%s (not analyzed)\n", indent, path)
		return
	}

	// Show short path
	shortPath := filepath.Base(path)
	if visited[path] {
		fmt.Printf("%s%s (already shown)\n", indent, shortPath)
		return
	}
	visited[path] = true

	fmt.Printf("%s%s\n", indent, shortPath)

	for _, d := range info.Directives {
		switch d.Type {
		case "include":
			depPath, _ := g.resolvePath(d.File, filepath.Dir(path))
			if verbose {
				fmt.Printf("%s  +- include \"%s\" (line %d)\n", indent, d.File, d.Line)
			} else {
				fmt.Printf("%s  +- include \"%s\"\n", indent, d.File)
			}
			if depPath != "" {
				g.printTree(depPath, indent+"  |  ", visited, verbose)
			}

		case "namespace":
			depPath, _ := g.resolvePath(d.File, filepath.Dir(path))
			if verbose {
				fmt.Printf("%s  +- namespace \"%s\" \"%s\" (line %d)\n", indent, d.Namespace, d.File, d.Line)
			} else {
				fmt.Printf("%s  +- namespace \"%s\" \"%s\"\n", indent, d.Namespace, d.File)
			}
			if depPath != "" {
				g.printTree(depPath, indent+"  |  ", visited, verbose)
			}

		case "extend":
			if len(d.Args) >= 2 {
				if verbose {
					fmt.Printf("%s  +- extend \"%s\" -> \"%s\" (line %d)\n", indent, d.Args[0], d.Args[1], d.Line)
				} else {
					fmt.Printf("%s  +- extend \"%s\" -> \"%s\"\n", indent, d.Args[0], d.Args[1])
				}
				if len(d.Args) > 2 {
					for i := 2; i < len(d.Args)-1; i += 2 {
						fmt.Printf("%s  |    \\- rewire \"%s\" -> \"%s\"\n", indent, d.Args[i], d.Args[i+1])
					}
				}
			}
		}
	}
}

func (g *DependencyGraph) detectCycles(startPath string) [][]string {
	var cycles [][]string
	visited := make(map[string]bool)
	inStack := make(map[string]bool)
	path := []string{}

	var dfs func(current string)
	dfs = func(current string) {
		if inStack[current] {
			// Found a cycle - find where in path
			for i, p := range path {
				if p == current {
					cycle := append([]string{}, path[i:]...)
					cycle = append(cycle, current)
					cycles = append(cycles, cycle)
					return
				}
			}
			return
		}

		if visited[current] {
			return
		}

		visited[current] = true
		inStack[current] = true
		path = append(path, current)
		defer func() {
			path = path[:len(path)-1]
			inStack[current] = false
		}()

		info, ok := g.templates[current]
		if !ok {
			return
		}

		for _, d := range info.Directives {
			if d.Type == "include" || d.Type == "namespace" {
				depPath, err := g.resolvePath(d.File, filepath.Dir(current))
				if err == nil {
					dfs(depPath)
				}
			}
		}
	}

	fullPath, _ := g.resolvePath(startPath, "")
	dfs(fullPath)
	return cycles
}

func (g *DependencyGraph) analyzeExtensions(rootInfo *TemplateInfo) []string {
	var issues []string

	// Collect all namespaces and what templates they provide
	namespaceDefines := make(map[string][]string) // namespace -> defines

	for path, info := range g.templates {
		// Check which namespace this file belongs to
		for ns, files := range g.extensions {
			for _, f := range files {
				resolved, _ := g.resolvePath(f, "")
				if resolved == path {
					for _, def := range info.Defines {
						namespaceDefines[ns] = append(namespaceDefines[ns], ns+":"+def)
					}
				}
			}
		}
	}

	// Check all extend directives
	for path, info := range g.templates {
		for _, d := range info.Directives {
			if d.Type == "extend" && len(d.Args) >= 2 {
				source := d.Args[0]
				dest := d.Args[1]

				// Check if source exists
				if strings.Contains(source, ":") {
					parts := strings.SplitN(source, ":", 2)
					ns, name := parts[0], parts[1]
					found := false
					for _, def := range namespaceDefines[ns] {
						if def == source || strings.HasSuffix(def, ":"+name) {
							found = true
							break
						}
					}
					if !found {
						issues = append(issues, fmt.Sprintf(
							"%s: extend references \"%s\" but namespace \"%s\" may not define \"%s\"",
							filepath.Base(path), source, ns, name))
					}
				}

				// Check rewrites
				for i := 2; i < len(d.Args)-1; i += 2 {
					oldRef := d.Args[i]
					if strings.Contains(oldRef, ":") && !strings.HasPrefix(oldRef, "::") {
						parts := strings.SplitN(oldRef, ":", 2)
						ns := parts[0]
						if _, ok := g.extensions[ns]; !ok {
							issues = append(issues, fmt.Sprintf(
								"%s: extend rewrites \"%s\" but namespace \"%s\" is not defined",
								filepath.Base(path), oldRef, ns))
						}
					}
				}

				// Check for potential infinite recursion
				if dest == source {
					issues = append(issues, fmt.Sprintf(
						"%s: extend creates \"%s\" from itself (infinite recursion)",
						filepath.Base(path), dest))
				}

				// Check for same name without namespace
				if !strings.Contains(dest, ":") {
					for _, def := range info.Defines {
						if def == dest {
							// This is fine - local override
							continue
						}
					}
				}
			}
		}
	}

	return issues
}

func (g *DependencyGraph) outputDOT(rootPath string) {
	fmt.Println("digraph TemplateDependencies {")
	fmt.Println("  rankdir=TB;")
	fmt.Println("  node [shape=box];")

	// Nodes
	for path := range g.templates {
		name := filepath.Base(path)
		fmt.Printf("  \"%s\" [label=\"%s\"];\n", path, name)
	}

	// Edges
	for path, info := range g.templates {
		for _, d := range info.Directives {
			switch d.Type {
			case "include":
				depPath, _ := g.resolvePath(d.File, filepath.Dir(path))
				if depPath != "" {
					fmt.Printf("  \"%s\" -> \"%s\" [label=\"include\"];\n", path, depPath)
				}
			case "namespace":
				depPath, _ := g.resolvePath(d.File, filepath.Dir(path))
				if depPath != "" {
					fmt.Printf("  \"%s\" -> \"%s\" [label=\"namespace:%s\", style=dashed];\n", path, depPath, d.Namespace)
				}
			case "extend":
				if len(d.Args) >= 2 {
					fmt.Printf("  \"%s\" -> \"%s\" [label=\"extend:%s->%s\", style=dotted, color=blue];\n",
						path, path, d.Args[0], d.Args[1])
				}
			}
		}
	}

	fmt.Println("}")
}

// Ensure TracingLoader implements TemplateLoader
var _ templar.TemplateLoader = (*TracingLoader)(nil)
