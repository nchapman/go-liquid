package liquid

import "testing"

// Upstream parity: unit/tokenizer_unit_test.rb
//
// Ruby's Liquid::Tokenizer splits the source string into a flat array of
// raw-text and tag/variable tokens. go-liquid's tokenizer is internal.
// The user-observable result is testable via Parse+Render.

func TestUpstream_TokenizerUnit_TokenizeStrings(t *testing.T) {
	// Token boundaries: every text segment between {% %} / {{ }} is one
	// token, preserved as-is on render.
	renderEq(t, "1", "  ", nil, "  ")
	renderEq(t, "2", "hello world", nil, "hello world")
}

func TestUpstream_TokenizerUnit_TokenizeVariables(t *testing.T) {
	renderEq(t, "v", "{{funk}}", map[string]any{"funk": "X"}, "X")
	renderEq(t, "ws", " {{funk}} ", map[string]any{"funk": "X"}, " X ")
}

func TestUpstream_TokenizerUnit_TokenizeBlocks(t *testing.T) {
	renderEq(t, "b", "{% comment %}{% endcomment %}", nil, "")
	renderEq(t, "ws", " {% comment %}{% endcomment %} ", nil, "  ")
}

func TestUpstream_TokenizerUnit_CalculateLineNumbersPerTokenWithProfiling(t *testing.T) {
	t.Skip("Ruby Tokenizer line_numbers/profiling token attributes not exposed")
}

func TestUpstream_TokenizerUnit_TokenizeWithNilSourceReturnsEmptyArray(t *testing.T) {
	t.Skip("Ruby Tokenizer.new(nil) coerces; go-liquid Parse takes a Go string only")
}

func TestUpstream_TokenizerUnit_IncompleteCurlyBraces(t *testing.T) {
	// A single '{' with no matching is plain text.
	renderEq(t, "single", "{ not a tag", nil, "{ not a tag")
}

func TestUpstream_TokenizerUnit_UnmatchingStartAndEnd(t *testing.T) {
	if _, err := Parse("{{ foo "); err == nil {
		t.Fatalf("expected parse error for unmatched {{")
	}
}
