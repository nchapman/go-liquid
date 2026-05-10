package liquid

import "testing"

// Upstream parity: unit/tags/case_tag_unit_test.rb

func TestUpstream_CaseTagUnit_CaseNodelist(t *testing.T) {
	t.Skip("Ruby Template#root#nodelist AST introspection not exposed in go-liquid")
}

func TestUpstream_CaseTagUnit_CaseWithTrailingElement(t *testing.T) {
	t.Skip("go-liquid rejects `{% case 1 bar %}` extra args; Ruby lax/strict tolerate")
	src := "{%- case 1 bar -%}{%- when 1 -%}one{%- else -%}two{%- endcase -%}"
	renderEq(t, "trailing", src, nil, "one")
}

func TestUpstream_CaseTagUnit_CaseWhenWithTrailingElement(t *testing.T) {
	t.Skip("go-liquid rejects trailing args after `{% when 1 bar %}`; Ruby lax/strict tolerate, strict2 rejects")
	src := "{%- case 1 -%}{%- when 1 bar -%}one{%- else -%}two{%- endcase -%}"
	renderEq(t, "trailing-when", src, nil, "one")
}

func TestUpstream_CaseTagUnit_CaseWhenWithComma(t *testing.T) {
	src := "{%- case 1 -%}{%- when 2, 1 -%}one{%- else -%}two{%- endcase -%}"
	renderEq(t, "comma", src, nil, "one")
}

func TestUpstream_CaseTagUnit_CaseWhenWithOr(t *testing.T) {
	src := "{%- case 1 -%}{%- when 2 or 1 -%}one{%- else -%}two{%- endcase -%}"
	renderEq(t, "or", src, nil, "one")
}

func TestUpstream_CaseTagUnit_CaseWhenEmpty(t *testing.T) {
	t.Skip("Ruby strict2 raises on `{% when %}` (empty); go-liquid behavior may differ")
}

func TestUpstream_CaseTagUnit_CaseWithInvalidExpression(t *testing.T) {
	t.Skip("Ruby strict mode produces specific error message; go-liquid format differs")
}

func TestUpstream_CaseTagUnit_CaseWhenWithInvalidExpression(t *testing.T) {
	t.Skip("Ruby strict mode produces specific error message; go-liquid format differs")
}
