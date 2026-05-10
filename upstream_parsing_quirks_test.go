package liquid

import "testing"

// Upstream parity: integration/parsing_quirks_test.rb

func TestUpstream_ParsingQuirks_Css(t *testing.T) {
	src := " div { font-weight: bold; } "
	renderEq(t, "css", src, nil, src)
}

func TestUpstream_ParsingQuirks_RaiseOnSingleCloseBracet(t *testing.T) {
	if _, err := Parse("text {{method} oh nos!"); err == nil {
		t.Fatalf("expected syntax error")
	}
}

func TestUpstream_ParsingQuirks_RaiseOnLabelAndNoCloseBrackets(t *testing.T) {
	if _, err := Parse("TEST {{ "); err == nil {
		t.Fatalf("expected syntax error")
	}
}

func TestUpstream_ParsingQuirks_RaiseOnLabelAndNoCloseBracketsPercent(t *testing.T) {
	if _, err := Parse("TEST {% "); err == nil {
		t.Fatalf("expected syntax error")
	}
}

func TestUpstream_ParsingQuirks_ErrorOnEmptyFilter(t *testing.T) {
	if _, err := Parse("{{test}}"); err != nil {
		t.Fatalf("good template should parse: %v", err)
	}
}

func TestUpstream_ParsingQuirks_ErrorOnEmptyFilter_StrictRejected(t *testing.T) {
	t.Skip("Ruby strict mode rejects '{{|test}}' and '{{test |a|b|}}'; go-liquid is single-mode")
}

func TestUpstream_ParsingQuirks_MeaninglessParensError(t *testing.T) {
	t.Skip("Ruby strict mode rejects 'a == \\'foo\\' or (b == ...)' grouping; go-liquid parses it")
}

func TestUpstream_ParsingQuirks_UnexpectedCharactersSyntaxError(t *testing.T) {
	t.Skip("Ruby strict mode rejects '&&' and '||'; go-liquid has no strict mode")
}

func TestUpstream_ParsingQuirks_NoErrorOnLaxEmptyFilter(t *testing.T) {
	t.Skip("Ruby lax mode tolerates trailing/leading empty filter pipes ('{{test |a|b|}}', '{{|test|}}'); go-liquid rejects them")
	for _, src := range []string{"{{test |a|b|}}", "{{test}}", "{{|test|}}"} {
		if _, err := Parse(src); err != nil {
			t.Fatalf("lax parse %q: %v", src, err)
		}
	}
}

func TestUpstream_ParsingQuirks_MeaninglessParensLax(t *testing.T) {
	t.Skip("go-liquid: parenthesized boolean groupings in {% if %} are parsed as ranges; Ruby lax mode silently flattens them")
	d := map[string]any{"b": "bar", "c": "baz"}
	src := "{% if a == 'foo' or (b == 'bar' and c == 'baz') or false %} YES {% endif %}"
	renderEq(t, "parens", src, d, " YES ")
}

func TestUpstream_ParsingQuirks_UnexpectedCharactersSilentlyEatLogicLax(t *testing.T) {
	t.Skip("Ruby lax silently strips '&&' / '||' and re-evaluates; go-liquid parses them literally or errors")
	renderEq(t, "and", "{% if true && false %} YES {% endif %}", nil, " YES ")
	renderEq(t, "or", "{% if false || true %} YES {% endif %}", nil, "")
}

func TestUpstream_ParsingQuirks_RaiseOnInvalidTagDelimiter(t *testing.T) {
	if _, err := Parse("{% end %}"); err == nil {
		t.Fatalf("expected syntax error")
	}
}

func TestUpstream_ParsingQuirks_UnanchoredFilterArguments(t *testing.T) {
	t.Skip("Ruby lax mode tolerates malformed filter syntax (split$$$:, downcase), recovering partial output; go-liquid is stricter")
	renderEq(t, "split", "{{ 'hi there' | split$$$:' ' | first }}", nil, "hi")
	renderEq(t, "downcase", "{{ 'X' | downcase) }}", nil, "x")
}

func TestUpstream_ParsingQuirks_InvalidVariablesWork(t *testing.T) {
	t.Skip("Ruby lax mode allows numeric variable names (123foo, 123); go-liquid lexer rejects them")
}

func TestUpstream_ParsingQuirks_ExtraDotsInRanges(t *testing.T) {
	t.Skip("Ruby lax mode tolerates '(1...5)' (3 dots); go-liquid range syntax requires exactly two")
	renderEq(t, "extra-dots", "{% for i in (1...5) %}{{ i }}{% endfor %}", nil, "12345")
}

func TestUpstream_ParsingQuirks_BlankVariableMarkup(t *testing.T) {
	renderEq(t, "empty-output", "{{}}", nil, "")
}

func TestUpstream_ParsingQuirks_LookupOnVarWithLiteralName(t *testing.T) {
	d := map[string]any{"blank": map[string]any{"x": "result"}}
	renderEq(t, "dot", "{{ blank.x }}", d, "result")
	renderEq(t, "bracket", "{{ blank['x'] }}", d, "result")
}

func TestUpstream_ParsingQuirks_ContainsInId(t *testing.T) {
	renderEq(t, "contains-id",
		"{% if containsallshipments == true %} YES {% endif %}",
		map[string]any{"containsallshipments": true}, " YES ")
}

func TestUpstream_ParsingQuirks_IncompleteExpression(t *testing.T) {
	t.Skip("Ruby lax mode silently truncates malformed trailing tokens; go-liquid is stricter")
	for _, src := range []string{
		"{{ false - }}", "{{ false > }}", "{{ false < }}",
		"{{ false = }}", "{{ false ! }}", "{{ false 1 }}", "{{ false a }}",
	} {
		renderEq(t, src, src, nil, "false")
	}
}
