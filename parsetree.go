package templar

import (
	"strings"
	"text/template/parse"
)

// TransformName applies namespace resolution rules to a template reference name.
//
// Resolution rules:
//   - If name starts with "::" → strip "::", return as global (no namespace)
//   - If name contains ":" → return unchanged (explicit cross-namespace reference)
//   - Otherwise → prepend namespace (e.g., "icon" → "NS:icon")
func TransformName(name, namespace string) string {
	// Global reference: strip :: prefix
	if strings.HasPrefix(name, "::") {
		return strings.TrimPrefix(name, "::")
	}

	// Already has namespace (explicit cross-namespace): leave unchanged
	if strings.Contains(name, ":") {
		return name
	}

	// Local reference: add namespace
	return namespace + ":" + name
}

// WalkParseTree walks a parse tree and calls the visitor function for each TemplateNode.
// The visitor can modify the node's Name field to apply namespace transformations.
func WalkParseTree(node parse.Node, visitor func(*parse.TemplateNode)) {
	if node == nil {
		return
	}

	switch n := node.(type) {
	case *parse.ListNode:
		if n != nil {
			for _, child := range n.Nodes {
				WalkParseTree(child, visitor)
			}
		}

	case *parse.TemplateNode:
		// This is what we're looking for - a {{ template "name" }} call
		visitor(n)

	case *parse.IfNode:
		WalkParseTree(n.List, visitor)
		WalkParseTree(n.ElseList, visitor)

	case *parse.RangeNode:
		WalkParseTree(n.List, visitor)
		WalkParseTree(n.ElseList, visitor)

	case *parse.WithNode:
		WalkParseTree(n.List, visitor)
		WalkParseTree(n.ElseList, visitor)

	case *parse.ActionNode:
		// ActionNodes contain pipelines, not nested template calls
		// Template calls are always TemplateNodes at the top level

	case *parse.TextNode, *parse.CommentNode, *parse.PipeNode:
		// Leaf nodes, nothing to recurse into
	}
}

// ApplyNamespaceToTree applies a namespace transformation to all template
// references within a parse tree. It modifies the tree in place.
//
// This transforms:
//   - {{ template "foo" }} → {{ template "NS:foo" }}
//   - {{ template "Other:bar" }} → {{ template "Other:bar" }} (unchanged)
//   - {{ template "::global" }} → {{ template "global" }}
func ApplyNamespaceToTree(tree *parse.Tree, namespace string) {
	if tree == nil || tree.Root == nil {
		return
	}

	WalkParseTree(tree.Root, func(node *parse.TemplateNode) {
		node.Name = TransformName(node.Name, namespace)
	})
}

// CopyTreeWithNamespace creates a deep copy of a parse tree and applies
// namespace transformation to both the tree name and all template references.
func CopyTreeWithNamespace(tree *parse.Tree, namespace string) *parse.Tree {
	if tree == nil {
		return nil
	}

	// Deep copy the tree
	copied := tree.Copy()

	// Apply namespace to the tree's own name
	copied.Name = TransformName(tree.Name, namespace)

	// Apply namespace to all template references within the tree
	ApplyNamespaceToTree(copied, namespace)

	return copied
}

// CollectTemplateNames walks a parse tree and collects all template names
// that are referenced via {{ template "name" }} calls.
func CollectTemplateNames(tree *parse.Tree) []string {
	if tree == nil || tree.Root == nil {
		return nil
	}

	var names []string
	WalkParseTree(tree.Root, func(node *parse.TemplateNode) {
		names = append(names, node.Name)
	})
	return names
}

// CreateDelegationTree creates a parse tree that simply delegates to another template.
// The resulting tree, when executed, will call {{ template "delegateTo" . }}
//
// This is used for template extension: when a child template overrides a block,
// we replace the base template's block with a delegation to the child's definition.
func CreateDelegationTree(treeName string, delegateTo string) *parse.Tree {
	// Create a minimal tree that delegates to another template
	// Equivalent to: {{ template "delegateTo" . }}
	tree := &parse.Tree{
		Name: treeName,
		Root: &parse.ListNode{
			NodeType: parse.NodeList,
			Nodes: []parse.Node{
				&parse.TemplateNode{
					NodeType: parse.NodeTemplate,
					Name:     delegateTo,
					Pipe: &parse.PipeNode{
						NodeType: parse.NodePipe,
						Cmds: []*parse.CommandNode{
							{
								NodeType: parse.NodeCommand,
								Args: []parse.Node{
									&parse.DotNode{
										NodeType: parse.NodeDot,
									},
								},
							},
						},
					},
				},
			},
		},
	}
	return tree
}

// IsLocalReference returns true if the name is a local reference (not namespaced, not global).
// Local references are plain names like "header" that should be namespaced.
// Non-local references include:
//   - "NS:header" (already namespaced)
//   - "::header" (explicitly global)
func IsLocalReference(name string) bool {
	return !strings.HasPrefix(name, "::") && !strings.Contains(name, ":")
}

// CollectLocalReferences collects all local (non-namespaced, non-global) template
// references from a parse tree. These are the references that would be transformed
// when applying a namespace.
func CollectLocalReferences(tree *parse.Tree) []string {
	if tree == nil || tree.Root == nil {
		return nil
	}

	seen := make(map[string]bool)
	WalkParseTree(tree.Root, func(node *parse.TemplateNode) {
		if IsLocalReference(node.Name) {
			seen[node.Name] = true
		}
	})

	var names []string
	for name := range seen {
		names = append(names, name)
	}
	return names
}

// ComputeReachableTemplates computes the transitive closure of templates reachable
// from the given entry points. This is used for tree-shaking: only namespace
// templates that are actually used.
//
// Parameters:
//   - templates: map of template name to parse tree (all available templates)
//   - entryPoints: starting template names to trace from
//
// Returns: set of template names that are reachable (including entry points)
func ComputeReachableTemplates(templates map[string]*parse.Tree, entryPoints []string) map[string]bool {
	reachable := make(map[string]bool)
	queue := make([]string, 0, len(entryPoints))

	// Start with entry points
	for _, name := range entryPoints {
		if _, exists := templates[name]; exists {
			if !reachable[name] {
				reachable[name] = true
				queue = append(queue, name)
			}
		}
	}

	// BFS to find all reachable templates
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		tree := templates[current]
		if tree == nil {
			continue
		}

		// Find all local references from this template
		refs := CollectLocalReferences(tree)
		for _, ref := range refs {
			if _, exists := templates[ref]; exists {
				if !reachable[ref] {
					reachable[ref] = true
					queue = append(queue, ref)
				}
			}
		}
	}

	return reachable
}

// CopyTreeWithRewrites creates a deep copy of a parse tree and rewrites
// template references according to the provided mapping.
//
// Parameters:
//   - tree: the source parse tree to copy
//   - rewrites: map of old name -> new name for template references
//
// Returns: a new tree with references rewritten
func CopyTreeWithRewrites(tree *parse.Tree, rewrites map[string]string) *parse.Tree {
	if tree == nil {
		return nil
	}

	copied := tree.Copy()

	WalkParseTree(copied.Root, func(node *parse.TemplateNode) {
		if newName, ok := rewrites[node.Name]; ok {
			node.Name = newName
		}
	})

	return copied
}
