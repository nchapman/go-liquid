package liquid

// Ported from upstream Ruby Liquid test/integration/tags/standard_tag_test.rb.

import "testing"

func TestUpstreamStandardTag(t *testing.T) {
	t.Run("test_no_transform", func(t *testing.T) {
		texts := []string{
			"this text should come out of the template without change...",
			"blah",
			"<blah>",
			"|,.:",
			"",
			"this shouldnt see any transformation either but has multiple lines\n              as you can clearly see here ...",
		}
		for _, txt := range texts {
			renderEq(t, "passthrough", txt, nil, txt)
		}
	})

	t.Run("test_has_a_block_which_does_nothing", func(t *testing.T) {
		cases := []struct{ src, want string }{
			{"the comment block should be removed {%comment%} be gone.. {%endcomment%} .. right?", "the comment block should be removed  .. right?"},
			{"{%comment%}{%endcomment%}", ""},
			{"{%comment%}{% endcomment %}", ""},
			{"{% comment %}{%endcomment%}", ""},
			{"{% comment %}{% endcomment %}", ""},
			{"{%comment%}comment{%endcomment%}", ""},
			{"{% comment %}comment{% endcomment %}", ""},
			{"{% comment %} 1 {% comment %} 2 {% endcomment %} 3 {% endcomment %}", ""},
			{"{%comment%}{%blabla%}{%endcomment%}", ""},
			{"{% comment %}{% blabla %}{% endcomment %}", ""},
			{"{%comment%}{% endif %}{%endcomment%}", ""},
			{"{% comment %}{% endwhatever %}{% endcomment %}", ""},
			{"{% comment %}{% raw %} {{%%%%}}  }} { {% endcomment %} {% comment {% endraw %} {% endcomment %}", ""},
			{`{% comment %}{% " %}{% endcomment %}`, ""},
			{"{% comment %}{%%}{% endcomment %}", ""},
			{"foo{%comment%}comment{%endcomment%}bar", "foobar"},
			{"foo{% comment %}comment{% endcomment %}bar", "foobar"},
			{"foo{%comment%} comment {%endcomment%}bar", "foobar"},
			{"foo{% comment %} comment {% endcomment %}bar", "foobar"},
			{"foo {%comment%} {%endcomment%} bar", "foo  bar"},
			{"foo {%comment%}comment{%endcomment%} bar", "foo  bar"},
			{"foo {%comment%} comment {%endcomment%} bar", "foo  bar"},
			{"foo{%comment%}\n                                     {%endcomment%}bar", "foobar"},
		}
		for _, c := range cases {
			renderEq(t, "comment", c.src, nil, c.want)
		}
	})

	t.Run("test_hyphenated_assign", func(t *testing.T) {
		data := map[string]any{"a-b": "1"}
		renderEq(t, "hyphen-assign", "a-b:{{a-b}} {%assign a-b = 2 %}a-b:{{a-b}}", data, "a-b:1 a-b:2")
	})

	t.Run("test_assign_with_colon_and_spaces", func(t *testing.T) {
		data := map[string]any{"var": map[string]any{"a:b c": map[string]any{"paged": "1"}}}
		renderEq(t, "colon-spaces", `{%assign var2 = var["a:b c"].paged %}var2: {{var2}}`, data, "var2: 1")
	})

	t.Run("test_capture", func(t *testing.T) {
		data := map[string]any{"var": "content"}
		renderEq(t, "capture",
			"{{ var2 }}{% capture var2 %}{{ var }} foo {% endcapture %}{{ var2 }}{{ var2 }}",
			data, "content foo content foo ")
	})

	t.Run("test_capture_detects_bad_syntax", func(t *testing.T) {
		_, err := Parse("{{ var2 }}{% capture %}{{ var }} foo {% endcapture %}{{ var2 }}{{ var2 }}")
		if err == nil {
			t.Fatal("expected parse error for {% capture %} without variable")
		}
	})

	t.Run("test_case", func(t *testing.T) {
		caseSrc := "{% case condition %}{% when 1 %} its 1 {% when 2 %} its 2 {% endcase %}"
		renderEq(t, "2", caseSrc, map[string]any{"condition": 2}, " its 2 ")
		renderEq(t, "1", caseSrc, map[string]any{"condition": 1}, " its 1 ")
		renderEq(t, "3", caseSrc, map[string]any{"condition": 3}, "")
		renderEq(t, "str-hit", `{% case condition %}{% when "string here" %} hit {% endcase %}`,
			map[string]any{"condition": "string here"}, " hit ")
		renderEq(t, "str-miss", `{% case condition %}{% when "string here" %} hit {% endcase %}`,
			map[string]any{"condition": "bad string here"}, "")
	})

	t.Run("test_case_with_else", func(t *testing.T) {
		src := "{% case condition %}{% when 5 %} hit {% else %} else {% endcase %}"
		renderEq(t, "hit", src, map[string]any{"condition": 5}, " hit ")
		renderEq(t, "else", src, map[string]any{"condition": 6}, " else ")
		renderEq(t, "leading-ws",
			"{% case condition %} {% when 5 %} hit {% else %} else {% endcase %}",
			map[string]any{"condition": 6}, " else ")
	})

	t.Run("test_case_on_size", func(t *testing.T) {
		src := "{% case a.size %}{% when 1 %}1{% when 2 %}2{% endcase %}"
		cases := []struct {
			arr  []any
			want string
		}{
			{[]any{}, ""},
			{[]any{1}, "1"},
			{[]any{1, 1}, "2"},
			{[]any{1, 1, 1}, ""},
			{[]any{1, 1, 1, 1}, ""},
			{[]any{1, 1, 1, 1, 1}, ""},
		}
		for _, c := range cases {
			renderEq(t, "case-size", src, map[string]any{"a": c.arr}, c.want)
		}
	})

	t.Run("test_case_on_size_with_else", func(t *testing.T) {
		src := "{% case a.size %}{% when 1 %}1{% when 2 %}2{% else %}else{% endcase %}"
		cases := []struct {
			arr  []any
			want string
		}{
			{[]any{}, "else"},
			{[]any{1}, "1"},
			{[]any{1, 1}, "2"},
			{[]any{1, 1, 1}, "else"},
			{[]any{1, 1, 1, 1}, "else"},
			{[]any{1, 1, 1, 1, 1}, "else"},
		}
		for _, c := range cases {
			renderEq(t, "case-size-else", src, map[string]any{"a": c.arr}, c.want)
		}
	})

	t.Run("test_case_on_length_with_else", func(t *testing.T) {
		renderEq(t, "empty?",
			"{% case a.empty? %}{% when true %}true{% when false %}false{% else %}else{% endcase %}",
			nil, "else")
	})

	t.Run("test_case_on_literal_true_false", func(t *testing.T) {
		// Sub-case from test_case_on_length_with_else that doesn't depend on .empty?
		src := "{% case false %}{% when true %}true{% when false %}false{% else %}else{% endcase %}"
		renderEq(t, "false", src, nil, "false")
		src = "{% case true %}{% when true %}true{% when false %}false{% else %}else{% endcase %}"
		renderEq(t, "true", src, nil, "true")
		src = "{% case NULL %}{% when true %}true{% when false %}false{% else %}else{% endcase %}"
		renderEq(t, "NULL", src, nil, "else")
	})

	t.Run("test_assign_from_case", func(t *testing.T) {
		code := "{% case collection.handle %}{% when 'menswear-jackets' %}{% assign ptitle = 'menswear' %}{% when 'menswear-t-shirts' %}{% assign ptitle = 'menswear' %}{% else %}{% assign ptitle = 'womenswear' %}{% endcase %}{{ ptitle }}"
		for _, h := range []string{"menswear-jackets", "menswear-t-shirts"} {
			renderEq(t, "men-"+h, code, map[string]any{"collection": map[string]any{"handle": h}}, "menswear")
		}
		for _, h := range []string{"x", "y", "z"} {
			renderEq(t, "women-"+h, code, map[string]any{"collection": map[string]any{"handle": h}}, "womenswear")
		}
	})

	t.Run("test_case_when_or", func(t *testing.T) {
		code := `{% case condition %}{% when 1 or 2 or 3 %} its 1 or 2 or 3 {% when 4 %} its 4 {% endcase %}`
		renderEq(t, "1", code, map[string]any{"condition": 1}, " its 1 or 2 or 3 ")
		renderEq(t, "2", code, map[string]any{"condition": 2}, " its 1 or 2 or 3 ")
		renderEq(t, "3", code, map[string]any{"condition": 3}, " its 1 or 2 or 3 ")
		renderEq(t, "4", code, map[string]any{"condition": 4}, " its 4 ")
		renderEq(t, "5", code, map[string]any{"condition": 5}, "")

		code = `{% case condition %}{% when 1 or "string" or null %} its 1 or 2 or 3 {% when 4 %} its 4 {% endcase %}`
		renderEq(t, "mixed-1", code, map[string]any{"condition": 1}, " its 1 or 2 or 3 ")
		renderEq(t, "mixed-string", code, map[string]any{"condition": "string"}, " its 1 or 2 or 3 ")
		renderEq(t, "mixed-nil", code, map[string]any{"condition": nil}, " its 1 or 2 or 3 ")
		renderEq(t, "mixed-other", code, map[string]any{"condition": "something else"}, "")
	})

	t.Run("test_case_when_comma", func(t *testing.T) {
		code := `{% case condition %}{% when 1, 2, 3 %} its 1 or 2 or 3 {% when 4 %} its 4 {% endcase %}`
		renderEq(t, "1", code, map[string]any{"condition": 1}, " its 1 or 2 or 3 ")
		renderEq(t, "2", code, map[string]any{"condition": 2}, " its 1 or 2 or 3 ")
		renderEq(t, "3", code, map[string]any{"condition": 3}, " its 1 or 2 or 3 ")
		renderEq(t, "4", code, map[string]any{"condition": 4}, " its 4 ")
		renderEq(t, "5", code, map[string]any{"condition": 5}, "")

		code = `{% case condition %}{% when 1, "string", null %} its 1 or 2 or 3 {% when 4 %} its 4 {% endcase %}`
		renderEq(t, "comma-1", code, map[string]any{"condition": 1}, " its 1 or 2 or 3 ")
		renderEq(t, "comma-string", code, map[string]any{"condition": "string"}, " its 1 or 2 or 3 ")
		renderEq(t, "comma-nil", code, map[string]any{"condition": nil}, " its 1 or 2 or 3 ")
		renderEq(t, "comma-other", code, map[string]any{"condition": "something else"}, "")
	})

	t.Run("test_case_when_comma_and_blank_body", func(t *testing.T) {
		code := `{% case condition %}{% when 1, 2 %} {% assign r = "result" %} {% endcase %}{{ r }}`
		renderEq(t, "blank-body-runs-side-effect", code, map[string]any{"condition": 2}, "result")
	})

	t.Run("test_assign", func(t *testing.T) {
		renderEq(t, "assign", `{% assign a = "variable"%}{{a}}`, nil, "variable")
	})

	t.Run("test_assign_unassigned", func(t *testing.T) {
		renderEq(t, "unassigned",
			"var2:{{var2}} {%assign var2 = var%} var2:{{var2}}",
			map[string]any{"var": "content"}, "var2:  var2:content")
	})

	t.Run("test_assign_an_empty_string", func(t *testing.T) {
		renderEq(t, "empty-str", `{% assign a = ""%}{{a}}`, nil, "")
	})

	t.Run("test_assign_is_global", func(t *testing.T) {
		renderEq(t, "global",
			`{%for i in (1..2) %}{% assign a = "variable"%}{% endfor %}{{a}}`,
			nil, "variable")
	})

	t.Run("test_case_detects_bad_syntax", func(t *testing.T) {
		if _, err := Parse("{% case false %}{% when %}true{% endcase %}"); err == nil {
			t.Fatal("expected parse error for empty when")
		}
		if _, err := Parse("{% case false %}{% huh %}true{% endcase %}"); err == nil {
			t.Fatal("expected parse error for unknown inner tag")
		}
	})

	t.Run("test_cycle", func(t *testing.T) {
		renderEq(t, "single", `{%cycle "one", "two"%}`, nil, "one")
		renderEq(t, "double", `{%cycle "one", "two"%} {%cycle "one", "two"%}`, nil, "one two")
		renderEq(t, "blank-and-two", `{%cycle "", "two"%} {%cycle "", "two"%}`, nil, " two")
		renderEq(t, "triple", `{%cycle "one", "two"%} {%cycle "one", "two"%} {%cycle "one", "two"%}`, nil, "one two one")
		renderEq(t, "css",
			`{%cycle "text-align: left", "text-align: right" %} {%cycle "text-align: left", "text-align: right"%}`,
			nil, "text-align: left text-align: right")
	})

	t.Run("test_multiple_cycles", func(t *testing.T) {
		renderEq(t, "two-and-three",
			`{%cycle 1,2%} {%cycle 1,2%} {%cycle 1,2%} {%cycle 1,2,3%} {%cycle 1,2,3%} {%cycle 1,2,3%} {%cycle 1,2,3%}`,
			nil, "1 2 1 1 2 3 1")
	})

	t.Run("test_multiple_named_cycles", func(t *testing.T) {
		renderEq(t, "named",
			`{%cycle 1: "one", "two" %} {%cycle 2: "one", "two" %} {%cycle 1: "one", "two" %} {%cycle 2: "one", "two" %} {%cycle 1: "one", "two" %} {%cycle 2: "one", "two" %}`,
			nil, "one one two two one one")
	})

	t.Run("test_multiple_named_cycles_with_names_from_context", func(t *testing.T) {
		renderEq(t, "named-from-ctx",
			`{%cycle var1: "one", "two" %} {%cycle var2: "one", "two" %} {%cycle var1: "one", "two" %} {%cycle var2: "one", "two" %} {%cycle var1: "one", "two" %} {%cycle var2: "one", "two" %}`,
			map[string]any{"var1": 1, "var2": 2}, "one one two two one one")
	})

	t.Run("test_size_of_array", func(t *testing.T) {
		renderEq(t, "size-arr", "array has {{ array.size }} elements",
			map[string]any{"array": []any{1, 2, 3, 4}}, "array has 4 elements")
	})

	t.Run("test_size_of_hash", func(t *testing.T) {
		renderEq(t, "size-hash", "hash has {{ hash.size }} elements",
			map[string]any{"hash": map[string]any{"a": 1, "b": 2, "c": 3, "d": 4}},
			"hash has 4 elements")
	})

	t.Run("test_illegal_symbols", func(t *testing.T) {
		renderEq(t, "true-empty", "{% if true == empty %}?{% endif %}", nil, "")
		renderEq(t, "true-null", "{% if true == null %}?{% endif %}", nil, "")
		renderEq(t, "empty-true", "{% if empty == true %}?{% endif %}", nil, "")
		renderEq(t, "null-true", "{% if null == true %}?{% endif %}", nil, "")
	})

	t.Run("test_ifchanged", func(t *testing.T) {
		src := "{%for item in array%}{%ifchanged%}{{item}}{% endifchanged %}{%endfor%}"
		renderEq(t, "changes", src, map[string]any{"array": []any{1, 1, 2, 2, 3, 3}}, "123")
		renderEq(t, "no-changes", src, map[string]any{"array": []any{1, 1, 1, 1}}, "1")
	})

	t.Run("test_multiline_tag", func(t *testing.T) {
		renderEq(t, "multiline",
			"0{%\nfor i in (1..3)\n%} {{\ni\n}}{%\nendfor\n%}",
			nil, "0 1 2 3")
	})
}
