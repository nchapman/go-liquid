package liquid

import "testing"

// Upstream parity: unit/variable_unit_test.rb
//
// Ruby's Liquid::Variable parses a single `{{ markup }}` and exposes
// `name`, `filters`, and `raw`. go-liquid does not expose this — variable
// parsing is internal. Each test is exercised through Parse+Render.

func TestUpstream_VariableUnit_Variable(t *testing.T) {
	renderEq(t, "ident", "{{hello}}", map[string]any{"hello": "world"}, "world")
}

func TestUpstream_VariableUnit_Filters(t *testing.T) {
	renderEq(t, "1", "{{ hello | textileze }}", map[string]any{"hello": "x"}, "x")
	renderEq(t, "2", "{{ hello | textileze | paragraph }}", map[string]any{"hello": "x"}, "x")
}

func TestUpstream_VariableUnit_Filters_StrftimeOnNonDate(t *testing.T) {
	t.Skip("Ruby strftime on non-date string returns ''; go-liquid passes through unknown 'strftime' as no-op (input unchanged)")
	renderEq(t, "3", "{{ hello | strftime: '%Y'}}", map[string]any{"hello": "now"}, "")
}

func TestUpstream_VariableUnit_FilterWithDateParameter(t *testing.T) {
	renderEq(t, "date", "{{ '2006-06-06' | date: \"%m/%d\" }}", nil, "06/06")
}

func TestUpstream_VariableUnit_FiltersWithoutWhitespace(t *testing.T) {
	renderEq(t, "no-ws", "{{hello|textileze|paragraph}}", map[string]any{"hello": "x"}, "x")
}

func TestUpstream_VariableUnit_Symbol(t *testing.T) {
	t.Skip("Ruby Symbol type not modeled in Go")
}

func TestUpstream_VariableUnit_StringToFilter(t *testing.T) {
	renderEq(t, "s", "{{ 'hello' | textileze }}", nil, "hello")
}

func TestUpstream_VariableUnit_StringSingleQuoted(t *testing.T) {
	renderEq(t, "s", "{{ 'hello' }}", nil, "hello")
}

func TestUpstream_VariableUnit_StringDoubleQuoted(t *testing.T) {
	renderEq(t, "d", `{{ "hello" }}`, nil, "hello")
}

func TestUpstream_VariableUnit_Integer(t *testing.T) {
	renderEq(t, "i", "{{ 42 }}", nil, "42")
}

func TestUpstream_VariableUnit_Float(t *testing.T) {
	renderEq(t, "f", "{{ 1.5 }}", nil, "1.5")
}

func TestUpstream_VariableUnit_Dashes(t *testing.T) {
	renderEq(t, "dash", "{{ foo-bar }}", map[string]any{"foo-bar": "v"}, "v")
}

func TestUpstream_VariableUnit_StringWithSpecialChars(t *testing.T) {
	// Ruby Liquid doesn't process escape sequences in strings; you use the
	// opposite quote type when you need the other inside. Original Ruby:
	//   ' hello! $!@.;"ddasd" '  (single-quoted, contains literal double quotes)
	renderEq(t, "specials", `{{ 'hello! $!@.;"ddasd" ' }}`, nil, `hello! $!@.;"ddasd" `)
}

func TestUpstream_VariableUnit_StringDot(t *testing.T) {
	renderEq(t, "dot-in-string", `{{ "foo.bar" }}`, nil, "foo.bar")
}

func TestUpstream_VariableUnit_FilterWithKeywordArguments(t *testing.T) {
	renderEq(t, "kw", "{{ hello | default: 'x', allow_false: true }}",
		map[string]any{"hello": ""}, "x")
}

func TestUpstream_VariableUnit_LaxFilterArgumentParsing(t *testing.T) {
	t.Skip("Ruby lax mode tolerates malformed filter args; go-liquid is stricter")
}
func TestUpstream_VariableUnit_StrictFilterArgumentParsing(t *testing.T) {
	t.Skip("Ruby :strict mode behavior not modeled")
}
func TestUpstream_VariableUnit_Strict2FilterArgumentParsing(t *testing.T) {
	t.Skip("Ruby :strict2 mode behavior not modeled")
}

func TestUpstream_VariableUnit_OutputRawSourceOfVariable(t *testing.T) {
	t.Skip("Ruby Variable#raw attribute not exposed")
}

func TestUpstream_VariableUnit_VariableLookupInterface(t *testing.T) {
	t.Skip("Ruby VariableLookup#name/#lookups attributes not exposed")
}
