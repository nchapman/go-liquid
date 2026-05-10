package liquid

import "testing"

// Upstream parity: unit/parse_context_unit_test.rb
//
// Ruby's Liquid::ParseContext exposes new_parser / safe_parse_expression
// for plugin authors. go-liquid does not expose the parser layer.

func TestUpstream_ParseContextUnit_SafeParseExpressionWithVariableLookup(t *testing.T) {
	t.Skip("Ruby ParseContext#new_parser + Expression.safe_parse not exposed in go-liquid")
}
func TestUpstream_ParseContextUnit_SafeParseExpressionRaisesSyntaxErrorForInvalidExpression(t *testing.T) {
	t.Skip("Ruby ParseContext#new_parser + Expression.safe_parse not exposed")
}
func TestUpstream_ParseContextUnit_ParseExpressionWithVariableLookup(t *testing.T) {
	t.Skip("Ruby ParseContext#parse_expression not exposed")
}
func TestUpstream_ParseContextUnit_ParseExpressionWithSafeTrue(t *testing.T) {
	t.Skip("Ruby ParseContext#parse_expression(safe:) not exposed")
}
func TestUpstream_ParseContextUnit_ParseExpressionWithEmptyString(t *testing.T) {
	t.Skip("Ruby ParseContext#parse_expression not exposed")
}
func TestUpstream_ParseContextUnit_ParseExpressionWithEmptyStringAndSafeTrue(t *testing.T) {
	t.Skip("Ruby ParseContext#parse_expression(safe:) not exposed")
}
func TestUpstream_ParseContextUnit_SafeParseExpressionAdvancesParserPointer(t *testing.T) {
	t.Skip("Ruby Parser pointer / Expression.safe_parse not exposed")
}
func TestUpstream_ParseContextUnit_ParseExpressionWithWhitespaceInStrict2Mode(t *testing.T) {
	t.Skip("Ruby :strict2 error mode + ParseContext not exposed")
}
