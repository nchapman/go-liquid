package liquid

import (
	"io"
	"unicode"
)

// blankWriter returns io.Discard when the given branch is entirely blank
// (Ruby's BlockBody#blank? semantics), so side effects inside the branch
// still run while the rendered whitespace is dropped. Otherwise returns w.
func blankWriter(w io.Writer, body []Node) io.Writer {
	if nodesBlank(body) {
		return io.Discard
	}
	return w
}

// nodesBlank reports whether a list of AST nodes can produce only
// whitespace output. Ruby Liquid uses this to elide entirely-blank
// block bodies (for/if/case/unless) so that wrapper whitespace does
// not bleed into the final rendering. Mirrors BlockBody#blank? — side
// effects (assigns, captures) still run but their output is discarded.
func nodesBlank(nodes []Node) bool {
	for _, n := range nodes {
		if !nodeBlank(n) {
			return false
		}
	}
	return true
}

func nodeBlank(n Node) bool {
	switch n := n.(type) {
	case *TextNode:
		for _, r := range n.Text {
			if !unicode.IsSpace(r) {
				return false
			}
		}
		return true
	case *AssignTag, *CommentTag, *CaptureTag, *DocTag:
		return true
	case *IfTag:
		if !nodesBlank(n.ThenBranch) {
			return false
		}
		for _, b := range n.ElsifBranches {
			if !nodesBlank(b.Body) {
				return false
			}
		}
		if n.ElseBranch != nil && !nodesBlank(n.ElseBranch) {
			return false
		}
		return true
	case *UnlessTag:
		if !nodesBlank(n.Body) {
			return false
		}
		if n.ElseBranch != nil && !nodesBlank(n.ElseBranch) {
			return false
		}
		return true
	case *CaseTag:
		for _, w := range n.Whens {
			if !nodesBlank(w.Body) {
				return false
			}
		}
		if n.Else != nil && !nodesBlank(n.Else) {
			return false
		}
		return true
	case *ForTag:
		if !nodesBlank(n.Body) {
			return false
		}
		if n.ElseBody != nil && !nodesBlank(n.ElseBody) {
			return false
		}
		return true
	case *LiquidTag:
		return nodesBlank(n.Body)
	case *IfchangedTag:
		return nodesBlank(n.Body)
	}
	return false
}
