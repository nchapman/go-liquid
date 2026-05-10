package liquid

import "testing"

// Upstream parity: unit/condition_unit_test.rb
//
// Ruby's Liquid::Condition is an internal AST class with its own
// evaluate method. go-liquid has no equivalent public type, so the
// internal-API tests are skipped; the user-observable blank/empty
// semantics ARE testable through `{% if x == blank %}` and similar.

func TestUpstream_ConditionUnit_BasicCondition(t *testing.T) {
	renderEq(t, "eq", "{% if 1 == 1 %}true{% else %}false{% endif %}", nil, "true")
	renderEq(t, "neq", "{% if 1 == 2 %}true{% else %}false{% endif %}", nil, "false")
}

func TestUpstream_ConditionUnit_DefaultOperatorsEvaluteTrue(t *testing.T) {
	cases := [][3]string{
		{"==", "1", "1"},
		{"!=", "1", "2"},
		{"<>", "1", "2"},
		{"<", "0", "1"},
		{">", "1", "0"},
		{"<=", "1", "1"},
		{">=", "1", "1"},
	}
	for _, c := range cases {
		src := "{% if " + c[1] + " " + c[0] + " " + c[2] + " %}true{% endif %}"
		renderEq(t, c[0], src, nil, "true")
	}
	// contains
	renderEq(t, "contains-str", "{% if 'bob' contains 'o' %}true{% endif %}", nil, "true")
}

func TestUpstream_ConditionUnit_DefaultOperatorsEvaluteFalse(t *testing.T) {
	renderEq(t, "neq-same", "{% if 1 != 1 %}T{% else %}F{% endif %}", nil, "F")
	renderEq(t, "lt-rev", "{% if 1 < 0 %}T{% else %}F{% endif %}", nil, "F")
}

func TestUpstream_ConditionUnit_ContainsWorksOnStrings(t *testing.T) {
	renderEq(t, "yes", "{% if 'bob' contains 'o' %}T{% endif %}", nil, "T")
	renderEq(t, "no", "{% if 'bob' contains 'x' %}T{% else %}F{% endif %}", nil, "F")
}

func TestUpstream_ConditionUnit_ContainsBinaryEncodingCompatibilityWithUtf8(t *testing.T) {
	t.Skip("Ruby Encoding::BINARY vs UTF-8 not modeled in Go (strings are byte slices)")
}

func TestUpstream_ConditionUnit_InvalidComparationOperator(t *testing.T) {
	t.Skip("Ruby Condition.new('1','=!','2').evaluate raises Liquid::ArgumentError; go-liquid surfaces at parse time")
}

func TestUpstream_ConditionUnit_ComparationOfIntAndStr(t *testing.T) {
	t.Skip("Ruby raises Liquid::ArgumentError on int<>string compare; go-liquid coerces and may return true/false")
}

func TestUpstream_ConditionUnit_HashCompareBackwardsCompatibility(t *testing.T) {
	t.Skip("Ruby hash-vs-string compare quirks not modeled")
}

func TestUpstream_ConditionUnit_ContainsWorksOnArrays(t *testing.T) {
	renderEq(t, "yes", "{% if a contains 2 %}T{% endif %}",
		map[string]any{"a": []any{1, 2, 3}}, "T")
	renderEq(t, "no", "{% if a contains 9 %}T{% else %}F{% endif %}",
		map[string]any{"a": []any{1, 2, 3}}, "F")
}

func TestUpstream_ConditionUnit_ContainsReturnsFalseForNilOperands(t *testing.T) {
	renderEq(t, "nil-l", "{% if missing contains 'x' %}T{% else %}F{% endif %}", nil, "F")
}

func TestUpstream_ConditionUnit_ContainsReturnsFalseForNilOperands_NilRight(t *testing.T) {
	renderEq(t, "nil-r", "{% if 'foo' contains missing %}T{% else %}F{% endif %}", nil, "F")
}

func TestUpstream_ConditionUnit_ContainsReturnFalseOnWrongDataType(t *testing.T) {
	renderEq(t, "wrong", "{% if 1 contains 2 %}T{% else %}F{% endif %}", nil, "F")
}

func TestUpstream_ConditionUnit_ContainsWithStringLeftOperandCoercesRightOperandToString(t *testing.T) {
	renderEq(t, "coerce", "{% if 'bob123' contains 123 %}T{% else %}F{% endif %}", nil, "T")
}

func TestUpstream_ConditionUnit_OrCondition(t *testing.T) {
	renderEq(t, "or-tf", "{% if false or true %}T{% else %}F{% endif %}", nil, "T")
	renderEq(t, "or-ff", "{% if false or false %}T{% else %}F{% endif %}", nil, "F")
}

func TestUpstream_ConditionUnit_AndCondition(t *testing.T) {
	renderEq(t, "and-tt", "{% if true and true %}T{% else %}F{% endif %}", nil, "T")
	renderEq(t, "and-tf", "{% if true and false %}T{% else %}F{% endif %}", nil, "F")
}

func TestUpstream_ConditionUnit_ShouldAllowCustomProcOperator(t *testing.T) {
	t.Skip("Ruby Condition.operators[] custom proc registration not exposed in go-liquid")
}

func TestUpstream_ConditionUnit_LeftOrRightMayContainOperators(t *testing.T) {
	// Identifiers containing operator-like substrings should still work:
	renderEq(t, "ok", "{% if foo_bar == 'foo' %}T{% endif %}",
		map[string]any{"foo_bar": "foo"}, "T")
}

func TestUpstream_ConditionUnit_DefaultContextIsDeprecated(t *testing.T) {
	t.Skip("Ruby Condition deprecation warning; not applicable")
}

func TestUpstream_ConditionUnit_ParseExpressionInStrictMode(t *testing.T) {
	t.Skip("Ruby :strict error mode not modeled in go-liquid")
}

func TestUpstream_ConditionUnit_ParseExpressionInStrict2ModeRaisesInternalError(t *testing.T) {
	t.Skip("Ruby :strict2 error mode not modeled")
}

func TestUpstream_ConditionUnit_ParseExpressionWithSafeTrueInStrict2Mode(t *testing.T) {
	t.Skip("Ruby :strict2 + safe-parse not modeled")
}

// ----- Blank / Empty literal tests -----

func TestUpstream_ConditionUnit_BlankWithWhitespaceString(t *testing.T) {
	renderEq(t, "ws", "{% if x == blank %}T{% else %}F{% endif %}",
		map[string]any{"x": "   "}, "T")
}

func TestUpstream_ConditionUnit_BlankWithEmptyString(t *testing.T) {
	renderEq(t, "empty", "{% if x == blank %}T{% else %}F{% endif %}",
		map[string]any{"x": ""}, "T")
}

func TestUpstream_ConditionUnit_BlankWithEmptyArray(t *testing.T) {
	renderEq(t, "arr", "{% if x == blank %}T{% else %}F{% endif %}",
		map[string]any{"x": []any{}}, "T")
}

func TestUpstream_ConditionUnit_BlankWithEmptyHash(t *testing.T) {
	renderEq(t, "hash", "{% if x == blank %}T{% else %}F{% endif %}",
		map[string]any{"x": map[string]any{}}, "T")
}

func TestUpstream_ConditionUnit_BlankWithNil(t *testing.T) {
	renderEq(t, "nil", "{% if x == blank %}T{% else %}F{% endif %}",
		map[string]any{"x": nil}, "T")
}

func TestUpstream_ConditionUnit_BlankWithFalse(t *testing.T) {
	renderEq(t, "false", "{% if x == blank %}T{% else %}F{% endif %}",
		map[string]any{"x": false}, "T")
}

func TestUpstream_ConditionUnit_NotBlankWithTrue(t *testing.T) {
	renderEq(t, "true", "{% if x == blank %}T{% else %}F{% endif %}",
		map[string]any{"x": true}, "F")
}

func TestUpstream_ConditionUnit_NotBlankWithNumber(t *testing.T) {
	renderEq(t, "num", "{% if x == blank %}T{% else %}F{% endif %}",
		map[string]any{"x": 42}, "F")
}

func TestUpstream_ConditionUnit_NotBlankWithStringContent(t *testing.T) {
	renderEq(t, "str", "{% if x == blank %}T{% else %}F{% endif %}",
		map[string]any{"x": "hello"}, "F")
}

func TestUpstream_ConditionUnit_NotBlankWithNonEmptyArray(t *testing.T) {
	renderEq(t, "arr", "{% if x == blank %}T{% else %}F{% endif %}",
		map[string]any{"x": []any{1, 2, 3}}, "F")
}

func TestUpstream_ConditionUnit_NotBlankWithNonEmptyHash(t *testing.T) {
	renderEq(t, "hash", "{% if x == blank %}T{% else %}F{% endif %}",
		map[string]any{"x": map[string]any{"a": 1}}, "F")
}

func TestUpstream_ConditionUnit_EmptyWithEmptyString(t *testing.T) {
	renderEq(t, "empty", "{% if x == empty %}T{% else %}F{% endif %}",
		map[string]any{"x": ""}, "T")
}

func TestUpstream_ConditionUnit_EmptyWithWhitespaceStringNotEmpty(t *testing.T) {
	renderEq(t, "ws", "{% if x == empty %}T{% else %}F{% endif %}",
		map[string]any{"x": "   "}, "F")
}

func TestUpstream_ConditionUnit_EmptyWithEmptyArray(t *testing.T) {
	renderEq(t, "arr", "{% if x == empty %}T{% else %}F{% endif %}",
		map[string]any{"x": []any{}}, "T")
}

func TestUpstream_ConditionUnit_EmptyWithEmptyHash(t *testing.T) {
	renderEq(t, "hash", "{% if x == empty %}T{% else %}F{% endif %}",
		map[string]any{"x": map[string]any{}}, "T")
}

func TestUpstream_ConditionUnit_NilIsNotEmpty(t *testing.T) {
	renderEq(t, "nil", "{% if x == empty %}T{% else %}F{% endif %}",
		map[string]any{"x": nil}, "F")
}
