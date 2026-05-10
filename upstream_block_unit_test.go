package liquid

import (
	"strings"
	"testing"
)

// Upstream parity: unit/block_unit_test.rb

func TestUpstream_BlockUnit_Blankspace(t *testing.T) {
	// Pure-whitespace template renders as itself; no tags to parse.
	src := "    "
	renderEq(t, "blank", src, nil, src)
}

func TestUpstream_BlockUnit_VariableBeginning(t *testing.T) {
	renderEq(t, "begin", "{{funk}}  ", map[string]any{"funk": "X"}, "X  ")
}

func TestUpstream_BlockUnit_VariableEnd(t *testing.T) {
	renderEq(t, "end", "  {{funk}}", map[string]any{"funk": "X"}, "  X")
}

func TestUpstream_BlockUnit_VariableMiddle(t *testing.T) {
	renderEq(t, "mid", "  {{funk}}  ", map[string]any{"funk": "X"}, "  X  ")
}

func TestUpstream_BlockUnit_VariableWithMultibyteCharacter(t *testing.T) {
	renderEq(t, "mb", "“{{funk}}”", map[string]any{"funk": "X"}, "“X”")
}

func TestUpstream_BlockUnit_VariableManyEmbeddedFragments(t *testing.T) {
	renderEq(t, "many", "{{a}} {{b}} {{c}} {{d}}",
		map[string]any{"a": "1", "b": "2", "c": "3", "d": "4"}, "1 2 3 4")
}

func TestUpstream_BlockUnit_CommentTagWithBlock(t *testing.T) {
	renderEq(t, "comment", "{% comment %}{{ foo }}{% endcomment %}", nil, "")
}

func TestUpstream_BlockUnit_DocTagWithBlock(t *testing.T) {
	// {% doc %} captures and discards the body. Verify a tag inside is not
	// parsed (would otherwise error).
	tmpl, err := Parse("{% doc %}{% if invalid %}{% enddoc %}")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	got, err := tmpl.Render(nil)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if strings.Contains(got, "invalid") {
		t.Fatalf("doc body should not be rendered: %q", got)
	}
}
