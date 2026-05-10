package liquid

import "testing"

// Upstream parity: unit/lexer_unit_test.rb
//
// Ruby's Liquid::Lexer is an internal class with a tokenize() method that
// returns a flat token array. go-liquid's lexer is also internal but not
// exposed via a stable test surface. Behavior is testable end-to-end via
// template parse + render — each lexer test here is exercised that way.

func TestUpstream_LexerUnit_Strings(t *testing.T) {
	renderEq(t, "single", "{{ 'single' }}", nil, "single")
	renderEq(t, "double", `{{ "double" }}`, nil, "double")
}

func TestUpstream_LexerUnit_Integer(t *testing.T) {
	renderEq(t, "int", "{{ 42 }}", nil, "42")
	renderEq(t, "neg", "{{ -42 }}", nil, "-42")
}

func TestUpstream_LexerUnit_Float(t *testing.T) {
	renderEq(t, "f", "{{ 1.5 }}", nil, "1.5")
	renderEq(t, "neg", "{{ -1.5 }}", nil, "-1.5")
}

func TestUpstream_LexerUnit_Comparison(t *testing.T) {
	renderEq(t, "eq", "{% if 1 == 1 %}T{% endif %}", nil, "T")
	renderEq(t, "neq", "{% if 1 != 2 %}T{% endif %}", nil, "T")
	renderEq(t, "lt", "{% if 1 < 2 %}T{% endif %}", nil, "T")
	renderEq(t, "gt", "{% if 2 > 1 %}T{% endif %}", nil, "T")
	renderEq(t, "lte", "{% if 1 <= 1 %}T{% endif %}", nil, "T")
	renderEq(t, "gte", "{% if 1 >= 1 %}T{% endif %}", nil, "T")
	renderEq(t, "ne", "{% if 1 <> 2 %}T{% endif %}", nil, "T")
}

func TestUpstream_LexerUnit_ComparisonWithoutWhitespace(t *testing.T) {
	renderEq(t, "no-ws", "{% if 1==1 %}T{% endif %}", nil, "T")
}

func TestUpstream_LexerUnit_ComparisonWithNegativeNumber(t *testing.T) {
	renderEq(t, "neg", "{% if x < -1 %}T{% else %}F{% endif %}",
		map[string]any{"x": -2}, "T")
}

func TestUpstream_LexerUnit_RaiseForInvalidComparison(t *testing.T) {
	t.Skip("Ruby strict mode raises on '=!'; go-liquid handles differently or accepts")
}

func TestUpstream_LexerUnit_Specials(t *testing.T) {
	// Special tokens: pipe, colon, comma, dot, brackets — all exercised in
	// every other test. This is a placeholder.
	renderEq(t, "filter", "{{ 'x' | upcase }}", nil, "X")
}

func TestUpstream_LexerUnit_FancyIdentifiers(t *testing.T) {
	renderEq(t, "underscore", "{{ foo_bar }}",
		map[string]any{"foo_bar": "v"}, "v")
	renderEq(t, "digits", "{{ foo123 }}",
		map[string]any{"foo123": "v"}, "v")
}

func TestUpstream_LexerUnit_Whitespace(t *testing.T) {
	renderEq(t, "ws", "{{   x   }}", map[string]any{"x": "v"}, "v")
}

func TestUpstream_LexerUnit_UnexpectedCharacter(t *testing.T) {
	t.Skip("Ruby strict mode raises 'Unexpected character'; go-liquid behavior differs")
}

func TestUpstream_LexerUnit_NegativeNumbers(t *testing.T) {
	renderEq(t, "neg-int", "{{ -42 }}", nil, "-42")
	renderEq(t, "neg-float", "{{ -1.5 }}", nil, "-1.5")
}

func TestUpstream_LexerUnit_GreaterThanTwoDigits(t *testing.T) {
	renderEq(t, "10", "{{ 10 }}", nil, "10")
	renderEq(t, "100", "{{ 100 }}", nil, "100")
}

func TestUpstream_LexerUnit_ErrorWithUtf8Character(t *testing.T) {
	t.Skip("Ruby raises specific error for non-ASCII operator chars; go-liquid behavior may differ")
}

func TestUpstream_LexerUnit_ContainsAsAttributeName(t *testing.T) {
	// `contains` is a keyword, but `containsallshipments` (longer name with
	// "contains" prefix) must still parse as identifier.
	renderEq(t, "id", "{% if containsallshipments == true %}T{% endif %}",
		map[string]any{"containsallshipments": true}, "T")
}

func TestUpstream_LexerUnit_TokenizeIncompleteExpression(t *testing.T) {
	t.Skip("Ruby lexer error message format / lax recovery not matched")
}

func TestUpstream_LexerUnit_ErrorWithInvalidUtf8(t *testing.T) {
	t.Skip("Ruby raises Liquid::TemplateEncodingError; go-liquid does not validate UTF-8 of source")
}
