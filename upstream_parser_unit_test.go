package liquid

import "testing"

// Upstream parity: unit/parser_unit_test.rb
//
// Ruby's Liquid::Parser is internal-only. Tests target consume/jump/look
// primitives, which go-liquid doesn't expose. End-to-end parse coverage
// lives in the integration suites.

func TestUpstream_ParserUnit_Consume(t *testing.T)  { t.Skip("Liquid::Parser internals not exposed") }
func TestUpstream_ParserUnit_Jump(t *testing.T)     { t.Skip("Liquid::Parser internals not exposed") }
func TestUpstream_ParserUnit_ConsumeQ(t *testing.T) { t.Skip("Liquid::Parser internals not exposed") }
func TestUpstream_ParserUnit_IdQ(t *testing.T)      { t.Skip("Liquid::Parser internals not exposed") }
func TestUpstream_ParserUnit_Look(t *testing.T)     { t.Skip("Liquid::Parser internals not exposed") }

func TestUpstream_ParserUnit_Expressions(t *testing.T) {
	// End-to-end equivalents of the Ruby parser test_expressions cases.
	renderEq(t, "string", "{{ 'hello' }}", nil, "hello")
	renderEq(t, "int", "{{ 42 }}", nil, "42")
	renderEq(t, "var", "{{ foo }}", map[string]any{"foo": "bar"}, "bar")
	renderEq(t, "filter", "{{ 'x' | upcase }}", nil, "X")
}

func TestUpstream_ParserUnit_Ranges(t *testing.T) {
	renderEq(t, "for-range", "{% for i in (1..3) %}{{ i }}{% endfor %}", nil, "123")
}

func TestUpstream_ParserUnit_Arguments(t *testing.T) {
	renderEq(t, "split", "{{ 'a,b,c' | split: ',' | join: '|' }}", nil, "a|b|c")
}

func TestUpstream_ParserUnit_InvalidExpression(t *testing.T) {
	if _, err := Parse("{{ }}"); err != nil {
		t.Fatalf("'{{ }}' should parse to empty output, got %v", err)
	}
	if _, err := Parse("{% if %}{% endif %}"); err == nil {
		t.Fatalf("expected error for empty if condition")
	}
}
