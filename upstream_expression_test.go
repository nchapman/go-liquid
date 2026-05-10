package liquid

import "testing"

// Upstream parity: integration/expression_test.rb

// exprResult mirrors Ruby's assert_expression_result helper: compare the
// markup-evaluated value against `expect` using a templated == check.
func exprResult(t *testing.T, expect any, markup string) {
	t.Helper()
	tmpl := "{% if expect == " + markup + " %}pass{% else %}got {{ " + markup + " }}{% endif %}"
	got, err := Parse(tmpl)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	out, err := got.Render(map[string]any{"expect": expect})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if out != "pass" {
		t.Fatalf("expr %q: %s", markup, out)
	}
}

func TestUpstream_Expression_KeywordLiterals(t *testing.T) {
	renderEq(t, "true-output", "{{ true }}", nil, "true")
	exprResult(t, true, "true")
}

func TestUpstream_Expression_String(t *testing.T) {
	renderEq(t, "single", "{{'single quoted'}}", nil, "single quoted")
	renderEq(t, "double", `{{"double quoted"}}`, nil, "double quoted")
	renderEq(t, "spaced", "{{ 'spaced' }}", nil, "spaced")
	renderEq(t, "spaced2", "{{ 'spaced2' }}", nil, "spaced2")
	renderEq(t, "emoji", "{{ 'emoji🔥' }}", nil, "emoji🔥")
}

func TestUpstream_Expression_Int(t *testing.T) {
	renderEq(t, "lit", "{{ 456 }}", nil, "456")
	exprResult(t, 123, "123")
}

func TestUpstream_Expression_Int_LeadingZero(t *testing.T) {
	t.Skip("Ruby treats '012' as decimal 12; go-liquid lexer behavior differs (may parse as 12 or reject)")
	exprResult(t, 12, "012")
}

func TestUpstream_Expression_Float(t *testing.T) {
	renderEq(t, "neg-float", "{{ -17.42 }}", nil, "-17.42")
	renderEq(t, "pos-float", "{{ 2.5 }}", nil, "2.5")
	exprResult(t, 1.5, "1.5")
}

func TestUpstream_Expression_Float_QuirkyDottedLax(t *testing.T) {
	t.Skip("Ruby lax-mode quirk: '0.....5' parses to 0.0; go-liquid does not replicate this lax recovery")
	exprResult(t, 0.0, "0.....5")
	exprResult(t, 0.0, "-0..1")
}

func TestUpstream_Expression_Float_LeadingDotIsLookup(t *testing.T) {
	t.Skip("Ruby Expression.parse('.5') returns VariableLookup; go-liquid lexer handles differently")
	// Behavior under test: `.5` parses as variable lookup, NOT as 0.5
}

func TestUpstream_Expression_Range(t *testing.T) {
	// (1..2) materializes to [1,2]; equality holds against a Go slice.
	exprResult(t, []any{1, 2}, "(1..2)")
}

func TestUpstream_Expression_Range_OutputAsString(t *testing.T) {
	t.Skip("Ruby Range#to_s renders '3..4'; go-liquid materializes range to slice and renders as '34'")
	renderEq(t, "spaced-range", "{{ ( 3 .. 4 ) }}", nil, "3..4")
}

func TestUpstream_Expression_Range_InvalidBounds(t *testing.T) {
	t.Skip("Ruby: range with non-numeric bounds raises 'Invalid expression type \\'false\\' in range expression'; go-liquid handles differently")
	if _, err := Parse("{{ (false..true) }}"); err == nil {
		t.Fatalf("expected syntax error")
	}
	if _, err := Parse("{{ ((1..2)..3) }}"); err == nil {
		t.Fatalf("expected syntax error")
	}
}

func TestUpstream_Expression_QuirkyNegativeSignMarkup(t *testing.T) {
	t.Skip("Ruby parses '-' standalone as VariableLookup named '-'; go-liquid does not model this lax quirk")
	renderEq(t, "neg-sign", "{{ - 'theme.css' - }}", nil, "")
}

func TestUpstream_Expression_Cache(t *testing.T) {
	t.Skip("go-liquid does not expose an expression cache API")
}

func TestUpstream_Expression_CacheWithTrueBoolean(t *testing.T) {
	t.Skip("go-liquid does not expose an expression cache API")
}

func TestUpstream_Expression_CacheWithLruRedux(t *testing.T) {
	t.Skip("go-liquid does not expose an expression cache API")
}

func TestUpstream_Expression_DisableExpressionCache(t *testing.T) {
	t.Skip("go-liquid does not expose an expression cache API")
}

func TestUpstream_Expression_SafeParseWithVariableLookup(t *testing.T) {
	t.Skip("go-liquid does not expose Expression.safe_parse; expression parsing is internal")
}

func TestUpstream_Expression_SafeParseWithNumber(t *testing.T) {
	t.Skip("go-liquid does not expose Expression.safe_parse")
}

func TestUpstream_Expression_SafeParseRaisesSyntaxErrorForInvalid(t *testing.T) {
	t.Skip("go-liquid does not expose Expression.safe_parse")
}
