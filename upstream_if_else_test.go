package liquid

// Ported from upstream Ruby Liquid test/integration/tags/if_else_tag_test.rb.

import "testing"

func TestUpstreamIfElseTag(t *testing.T) {
	t.Run("test_if", func(t *testing.T) {
		renderEq(t, "false", " {% if false %} this text should not go into the output {% endif %} ", nil, "  ")
		renderEq(t, "true", " {% if true %} this text should go into the output {% endif %} ", nil, "  this text should go into the output  ")
		renderEq(t, "two-ifs", "{% if false %} you suck {% endif %} {% if true %} you rock {% endif %}?", nil, "  you rock ?")
	})

	t.Run("test_literal_comparisons", func(t *testing.T) {
		renderEq(t, "assign-false", "{% assign v = false %}{% if v %} YES {% else %} NO {% endif %}", nil, " NO ")
		renderEq(t, "assign-nil", "{% assign v = nil %}{% if v == nil %} YES {% else %} NO {% endif %}", nil, " YES ")
	})

	t.Run("test_if_else", func(t *testing.T) {
		renderEq(t, "false", "{% if false %} NO {% else %} YES {% endif %}", nil, " YES ")
		renderEq(t, "true", "{% if true %} YES {% else %} NO {% endif %}", nil, " YES ")
		renderEq(t, "string", `{% if "foo" %} YES {% else %} NO {% endif %}`, nil, " YES ")
	})

	t.Run("test_if_boolean", func(t *testing.T) {
		renderEq(t, "var-true", "{% if var %} YES {% endif %}", map[string]any{"var": true}, " YES ")
	})

	t.Run("test_if_or", func(t *testing.T) {
		renderEq(t, "tt", "{% if a or b %} YES {% endif %}", map[string]any{"a": true, "b": true}, " YES ")
		renderEq(t, "tf", "{% if a or b %} YES {% endif %}", map[string]any{"a": true, "b": false}, " YES ")
		renderEq(t, "ft", "{% if a or b %} YES {% endif %}", map[string]any{"a": false, "b": true}, " YES ")
		renderEq(t, "ff", "{% if a or b %} YES {% endif %}", map[string]any{"a": false, "b": false}, "")
		renderEq(t, "fft", "{% if a or b or c %} YES {% endif %}", map[string]any{"a": false, "b": false, "c": true}, " YES ")
		renderEq(t, "fff", "{% if a or b or c %} YES {% endif %}", map[string]any{"a": false, "b": false, "c": false}, "")
	})

	t.Run("test_if_or_with_operators", func(t *testing.T) {
		renderEq(t, "1", "{% if a == true or b == true %} YES {% endif %}", map[string]any{"a": true, "b": true}, " YES ")
		renderEq(t, "2", "{% if a == true or b == false %} YES {% endif %}", map[string]any{"a": true, "b": true}, " YES ")
		renderEq(t, "3", "{% if a == false or b == false %} YES {% endif %}", map[string]any{"a": true, "b": true}, "")
	})

	t.Run("test_comparison_of_strings_containing_and_or_or", func(t *testing.T) {
		src := "{% if a == 'and' and b == 'or' and c == 'foo and bar' and d == 'bar or baz' and e == 'foo' and foo and bar %} YES {% endif %}"
		data := map[string]any{
			"a": "and", "b": "or", "c": "foo and bar", "d": "bar or baz",
			"e": "foo", "foo": true, "bar": true,
		}
		renderEq(t, "and-or-strings", src, data, " YES ")
	})

	t.Run("test_comparison_of_expressions_starting_with_and_or_or", func(t *testing.T) {
		data := map[string]any{
			"order":   map[string]any{"items_count": 0},
			"android": map[string]any{"name": "Roy"},
		}
		renderEq(t, "android", "{% if android.name == 'Roy' %}YES{% endif %}", data, "YES")
		renderEq(t, "order", "{% if order.items_count == 0 %}YES{% endif %}", data, "YES")
	})

	t.Run("test_if_and", func(t *testing.T) {
		renderEq(t, "tt", "{% if true and true %} YES {% endif %}", nil, " YES ")
		renderEq(t, "ft", "{% if false and true %} YES {% endif %}", nil, "")
		renderEq(t, "tf", "{% if true and false %} YES {% endif %}", nil, "")
	})

	t.Run("test_hash_miss_generates_false", func(t *testing.T) {
		renderEq(t, "miss", "{% if foo.bar %} NO {% endif %}", map[string]any{"foo": map[string]any{}}, "")
	})

	t.Run("test_if_from_variable", func(t *testing.T) {
		falsy := []struct {
			name string
			data map[string]any
		}{
			{"var-false", map[string]any{"var": false}},
			{"var-nil", map[string]any{"var": nil}},
			{"foo-bar-false", map[string]any{"foo": map[string]any{"bar": false}}},
			{"foo-empty", map[string]any{"foo": map[string]any{}}},
			{"foo-nil", map[string]any{"foo": nil}},
			{"foo-true", map[string]any{"foo": true}},
		}
		for _, f := range falsy {
			renderEq(t, f.name, "{% if var %} NO {% endif %}", f.data, "")
			renderEq(t, "miss-"+f.name, "{% if foo.bar %} NO {% endif %}", f.data, "")
		}

		truthy := []struct {
			name string
			data map[string]any
		}{
			{"var-text", map[string]any{"var": "text"}},
			{"var-true", map[string]any{"var": true}},
			{"var-1", map[string]any{"var": 1}},
			{"var-empty-hash", map[string]any{"var": map[string]any{}}},
			{"var-empty-arr", map[string]any{"var": []any{}}},
		}
		for _, tc := range truthy {
			renderEq(t, tc.name, "{% if var %} YES {% endif %}", tc.data, " YES ")
		}
		renderEq(t, "literal-foo", `{% if "foo" %} YES {% endif %}`, nil, " YES ")
		renderEq(t, "foo-bar-true", "{% if foo.bar %} YES {% endif %}", map[string]any{"foo": map[string]any{"bar": true}}, " YES ")
		renderEq(t, "foo-bar-text", "{% if foo.bar %} YES {% endif %}", map[string]any{"foo": map[string]any{"bar": "text"}}, " YES ")
		renderEq(t, "foo-bar-1", "{% if foo.bar %} YES {% endif %}", map[string]any{"foo": map[string]any{"bar": 1}}, " YES ")
		renderEq(t, "foo-bar-empty-hash", "{% if foo.bar %} YES {% endif %}", map[string]any{"foo": map[string]any{"bar": map[string]any{}}}, " YES ")
		renderEq(t, "foo-bar-empty-arr", "{% if foo.bar %} YES {% endif %}", map[string]any{"foo": map[string]any{"bar": []any{}}}, " YES ")

		renderEq(t, "else-var-false", "{% if var %} NO {% else %} YES {% endif %}", map[string]any{"var": false}, " YES ")
		renderEq(t, "else-var-nil", "{% if var %} NO {% else %} YES {% endif %}", map[string]any{"var": nil}, " YES ")
		renderEq(t, "else-var-true", "{% if var %} YES {% else %} NO {% endif %}", map[string]any{"var": true}, " YES ")
		renderEq(t, "else-literal-foo", `{% if "foo" %} YES {% else %} NO {% endif %}`, map[string]any{"var": "text"}, " YES ")
		renderEq(t, "else-foo-bar-false", "{% if foo.bar %} NO {% else %} YES {% endif %}", map[string]any{"foo": map[string]any{"bar": false}}, " YES ")
		renderEq(t, "else-foo-bar-true", "{% if foo.bar %} YES {% else %} NO {% endif %}", map[string]any{"foo": map[string]any{"bar": true}}, " YES ")
		renderEq(t, "else-foo-bar-text", "{% if foo.bar %} YES {% else %} NO {% endif %}", map[string]any{"foo": map[string]any{"bar": "text"}}, " YES ")
		renderEq(t, "else-foo-notbar", "{% if foo.bar %} NO {% else %} YES {% endif %}", map[string]any{"foo": map[string]any{"notbar": true}}, " YES ")
		renderEq(t, "else-foo-empty", "{% if foo.bar %} NO {% else %} YES {% endif %}", map[string]any{"foo": map[string]any{}}, " YES ")
		renderEq(t, "else-notfoo", "{% if foo.bar %} NO {% else %} YES {% endif %}", map[string]any{"notfoo": map[string]any{"bar": true}}, " YES ")
	})

	t.Run("test_nested_if", func(t *testing.T) {
		renderEq(t, "ff", "{% if false %}{% if false %} NO {% endif %}{% endif %}", nil, "")
		renderEq(t, "ft", "{% if false %}{% if true %} NO {% endif %}{% endif %}", nil, "")
		renderEq(t, "tf", "{% if true %}{% if false %} NO {% endif %}{% endif %}", nil, "")
		renderEq(t, "tt", "{% if true %}{% if true %} YES {% endif %}{% endif %}", nil, " YES ")
		renderEq(t, "tt-else", "{% if true %}{% if true %} YES {% else %} NO {% endif %}{% else %} NO {% endif %}", nil, " YES ")
		renderEq(t, "tf-else", "{% if true %}{% if false %} NO {% else %} YES {% endif %}{% else %} NO {% endif %}", nil, " YES ")
		renderEq(t, "ft-else", "{% if false %}{% if true %} NO {% else %} NONO {% endif %}{% else %} YES {% endif %}", nil, " YES ")
	})

	t.Run("test_comparisons_on_null", func(t *testing.T) {
		for _, op := range []string{"<", "<=", ">=", ">"} {
			renderEq(t, "null-"+op+"-10", "{% if null "+op+" 10 %} NO {% endif %}", nil, "")
			renderEq(t, "10-"+op+"-null", "{% if 10 "+op+" null %} NO {% endif %}", nil, "")
		}
	})

	t.Run("test_else_if", func(t *testing.T) {
		renderEq(t, "first", "{% if 0 == 0 %}0{% elsif 1 == 1%}1{% else %}2{% endif %}", nil, "0")
		renderEq(t, "elsif", "{% if 0 != 0 %}0{% elsif 1 == 1%}1{% else %}2{% endif %}", nil, "1")
		renderEq(t, "else", "{% if 0 != 0 %}0{% elsif 1 != 1%}1{% else %}2{% endif %}", nil, "2")
		renderEq(t, "elsif-no-else", "{% if false %}if{% elsif true %}elsif{% endif %}", nil, "elsif")
	})

	t.Run("test_syntax_error_no_variable", func(t *testing.T) {
		if _, err := Parse("{% if jerry == 1 %}"); err == nil {
			t.Fatal("expected parse error for unclosed if")
		}
	})

	t.Run("test_syntax_error_no_expression", func(t *testing.T) {
		if _, err := Parse("{% if %}"); err == nil {
			t.Fatal("expected parse error for empty if")
		}
	})

	t.Run("test_if_with_custom_condition", func(t *testing.T) {
		t.Skip("TODO: Liquid::Condition.operators is a Ruby-specific extension API; go-liquid has no equivalent for plugging in user-defined operators")
		// Ruby monkey-patches Condition.operators['contains'] = :[] (Ruby's
		// String#[] returns a substring or nil) to redefine `contains`.
		renderEq(t, "yes", `{% if 'bob' contains 'o' %}yes{% endif %}`, nil, "yes")
		renderEq(t, "no", `{% if 'bob' contains 'f' %}yes{% else %}no{% endif %}`, nil, "no")
	})

	t.Run("test_operators_are_ignored_unless_isolated", func(t *testing.T) {
		// Operators inside identifier-like strings must not match. The
		// embedded ' and ' / ' or ' substrings must not break parsing.
		renderEq(t, "hyphen-name",
			`{% if 'gnomeslab-and-or-liquid' contains 'gnomeslab-and-or-liquid' %}yes{% endif %}`,
			nil, "yes")
	})

	t.Run("test_operators_are_whitelisted", func(t *testing.T) {
		if _, err := Parse(`{% if 1 or throw or or 1 %}yes{% endif %}`); err == nil {
			t.Fatal("expected syntax error for `or or`")
		}
	})

	t.Run("test_multiple_conditions", func(t *testing.T) {
		// `a or b and c` — Ruby Liquid evaluates strictly right-to-left,
		// so this is `a or (b and c)`.
		tpl := "{% if a or b and c %}true{% else %}false{% endif %}"
		cases := []struct {
			a, b, c bool
			want    string
		}{
			{true, true, true, "true"},
			{true, true, false, "true"},
			{true, false, true, "true"},
			{true, false, false, "true"},
			{false, true, true, "true"},
			{false, true, false, "false"},
			{false, false, true, "false"},
			{false, false, false, "false"},
		}
		for _, c := range cases {
			data := map[string]any{"a": c.a, "b": c.b, "c": c.c}
			renderEq(t, "abc", tpl, data, c.want)
		}
	})
}
