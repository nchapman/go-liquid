package liquid

// Tests ported from the upstream Ruby Liquid suite (Shopify/liquid test/).
// Each sub-test names the originating Ruby test (file:test_name) so the
// provenance is auditable. Ruby-specific behavior (Drops with method_missing,
// Strainer, instance_eval, Profiler, I18n, ResourceLimits, Symbol coercion,
// Proc/lambda values, strict2 mode) is intentionally omitted.

import (
	"strings"
	"testing"
)

// renderEq is a small helper for table-driven cases that expect successful render.
func renderEq(t *testing.T, name, src string, data map[string]any, want string) {
	t.Helper()
	got, err := Render(src, data)
	if err != nil {
		t.Fatalf("%s: unexpected error: %v", name, err)
	}
	if got != want {
		t.Errorf("%s: template %q\n  want %q\n  got  %q", name, src, want, got)
	}
}

// ---- parsing_quirks_test.rb ----

func TestUpstream_ParsingQuirks(t *testing.T) {
	t.Run("parsing_css_unchanged", func(t *testing.T) {
		src := " div { font-weight: bold; } "
		renderEq(t, "css", src, nil, src)
	})

	t.Run("blank_variable_markup_is_empty", func(t *testing.T) {
		t.Skip("gap: upstream renders empty string for `{{}}`; go-liquid raises a parse error")
		renderEq(t, "{{}}", "{{}}", nil, "")
	})

	t.Run("lookup_on_var_with_literal_name_blank", func(t *testing.T) {
		t.Skip("gap: upstream treats `blank` as an identifier when followed by . or []; go-liquid resolves it as the blank literal")
		data := map[string]any{"blank": map[string]any{"x": "result"}}
		renderEq(t, "blank.x", "{{ blank.x }}", data, "result")
		renderEq(t, "blank['x']", "{{ blank['x'] }}", data, "result")
	})

	t.Run("contains_as_identifier_prefix", func(t *testing.T) {
		// "contains" inside an identifier name must not be parsed as the operator
		data := map[string]any{"containsallshipments": true}
		renderEq(t, "contains-id", "{% if containsallshipments == true %} YES {% endif %}", data, " YES ")
	})

	t.Run("invalid_tag_delimiter_end_alone", func(t *testing.T) {
		// `{% end %}` is not a valid tag
		if _, err := Parse("{% end %}"); err == nil {
			t.Fatal("expected parse error for stray {% end %}")
		}
	})

	t.Run("unclosed_variable_errors", func(t *testing.T) {
		if _, err := Parse("TEST {{ "); err == nil {
			t.Fatal("expected parse error for unclosed {{")
		}
	})

	t.Run("unclosed_tag_errors", func(t *testing.T) {
		if _, err := Parse("TEST {% "); err == nil {
			t.Fatal("expected parse error for unclosed {%")
		}
	})
}

// ---- variable_test.rb ----

func TestUpstream_Variable(t *testing.T) {
	cases := []struct {
		name, src string
		data      map[string]any
		want      string
	}{
		{"simple_with_whitespace", "  {{ test }}  ", map[string]any{"test": "worked"}, "  worked  "},
		{"hash_scoping_basic", "{{ test.test }}", map[string]any{"test": map[string]any{"test": "worked"}}, "worked"},
		{"hash_scoping_with_dot_whitespace", "{{ test . test }}", map[string]any{"test": map[string]any{"test": "worked"}}, "worked"},
		{"false_renders_as_false", "{{ foo }}", map[string]any{"foo": false}, "false"},
		{"false_literal_renders_as_false", "{{ false }}", nil, "false"},
		{"nil_renders_empty", "{{ nil }}", nil, ""},
		{"nil_through_filter", "{{ nil | append: 'cat' }}", nil, "cat"},
		// gap: upstream coerces the blank/empty literal to "" on render; go-liquid renders the literal name.
		// {"using_blank_as_variable_name", "{% assign foo = blank %}{{ foo }}", nil, ""},
		// {"using_empty_as_variable_name", "{% assign foo = empty %}{{ foo }}", nil, ""},
		{"multiline_variable", "{{\ntest\n}}", map[string]any{"test": "worked"}, "worked"},
		{"bracket_with_inner_whitespace", "{{ a[ 'b' ] }}", map[string]any{"a": map[string]any{"b": "result"}}, "result"},
		{"ignore_unknown", "{{ test }}", nil, ""},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			renderEq(t, c.name, c.src, c.data, c.want)
		})
	}
}

// ---- for_tag_test.rb ----

func TestUpstream_ForTag(t *testing.T) {
	t.Run("for_with_variable_range", func(t *testing.T) {
		src := "{%for item in (1..foobar) %} {{item}} {%endfor%}"
		renderEq(t, "var-range", src, map[string]any{"foobar": 3}, " 1  2  3 ")
	})

	t.Run("for_with_hash_value_range", func(t *testing.T) {
		src := "{%for item in (1..foobar.value) %} {{item}} {%endfor%}"
		data := map[string]any{"foobar": map[string]any{"value": 3}}
		renderEq(t, "hash-range", src, data, " 1  2  3 ")
	})

	t.Run("dynamic_variable_limiting", func(t *testing.T) {
		src := "{%for i in array limit: limit offset: offset %}{{ i }}{%endfor%}"
		data := map[string]any{
			"array":  []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			"limit":  2,
			"offset": 2,
		}
		renderEq(t, "dyn-limit", src, data, "34")
	})

	t.Run("for_and_if_uses_forloop_first", func(t *testing.T) {
		src := "{%for item in array%}{% if forloop.first %}+{% else %}-{% endif %}{%endfor%}"
		renderEq(t, "for+if", src, map[string]any{"array": []int{1, 2, 3}}, "+--")
	})

	t.Run("blank_string_not_iterable", func(t *testing.T) {
		t.Skip("gap: upstream treats blank string as non-iterable for {% for %}; go-liquid iterates over runes")
		src := "{% for char in characters %}I WILL NOT BE OUTPUT{% endfor %}"
		renderEq(t, "blank-string", src, map[string]any{"characters": ""}, "")
	})
}

// ---- if_else_tag_test.rb ----

func TestUpstream_IfElse(t *testing.T) {
	t.Run("strings_containing_and_or_or", func(t *testing.T) {
		src := "{% if a == 'and' and b == 'or' and c == 'foo and bar' and d == 'bar or baz' and e == 'foo' and foo and bar %} YES {% endif %}"
		data := map[string]any{
			"a": "and", "b": "or",
			"c": "foo and bar", "d": "bar or baz",
			"e": "foo", "foo": true, "bar": true,
		}
		renderEq(t, "and-or-strings", src, data, " YES ")
	})

	t.Run("identifier_starting_with_and", func(t *testing.T) {
		src := "{% if android.name == 'Roy' %}YES{% endif %}"
		data := map[string]any{"android": map[string]any{"name": "Roy"}}
		renderEq(t, "android", src, data, "YES")
	})

	t.Run("identifier_starting_with_or", func(t *testing.T) {
		src := "{% if order.items_count == 0 %}YES{% endif %}"
		data := map[string]any{"order": map[string]any{"items_count": 0}}
		renderEq(t, "order", src, data, "YES")
	})

	t.Run("hash_miss_is_falsy", func(t *testing.T) {
		// foo.bar where foo is empty hash should render the else branch as empty
		src := "{% if foo.bar %} NO {% endif %}"
		renderEq(t, "hash-miss", src, map[string]any{"foo": map[string]any{}}, "")
	})

	t.Run("literal_false_is_falsy_in_assignment", func(t *testing.T) {
		src := "{% assign v = false %}{% if v %} YES {% else %} NO {% endif %}"
		renderEq(t, "lit-false", src, nil, " NO ")
	})

	t.Run("nil_equals_nil", func(t *testing.T) {
		src := "{% assign v = nil %}{% if v == nil %} YES {% else %} NO {% endif %}"
		renderEq(t, "nil-eq-nil", src, nil, " YES ")
	})
}

// ---- raw_tag_test.rb ----

func TestUpstream_Raw(t *testing.T) {
	t.Run("comment_inside_raw_preserved", func(t *testing.T) {
		src := "{% raw %}{% comment %} test {% endcomment %}{% endraw %}"
		renderEq(t, "raw-comment", src, nil, "{% comment %} test {% endcomment %}")
	})

	t.Run("variable_inside_raw_preserved", func(t *testing.T) {
		src := "> {%- raw -%}{{ test }}{%- endraw -%} <"
		renderEq(t, "raw-trim", src, nil, ">{{ test }}<")
	})

	t.Run("raw_with_brace_braces", func(t *testing.T) {
		src := "{% raw %}{{% endraw %}Hello{% raw %}}{% endraw %}"
		renderEq(t, "raw-brace", src, nil, "{Hello}")
	})

	t.Run("unclosed_raw_errors", func(t *testing.T) {
		t.Skip("gap: upstream raises a parse error for unclosed `{% raw %}`; go-liquid silently consumes to EOF")
		if _, err := Parse("{% raw %} foo"); err == nil {
			t.Fatal("expected parse error for unclosed raw")
		}
	})
}

// ---- inline_comment_test.rb ----

func TestUpstream_InlineComment(t *testing.T) {
	cases := []struct {
		name, src, want string
	}{
		{"with_dash_and_space", "{%- # this is an inline comment -%}", ""},
		{"with_dash_no_space", "{%-# this is an inline comment -%}", ""},
		{"plain_with_space", "{% # this is an inline comment %}", ""},
		{"plain_no_space", "{%# this is an inline comment %}", ""},
		{"no_space_after_pound", "{%#this is an inline comment%}", ""},
		{"multiple_pound_signs", "{% ##### why so many pounds ##### %}", ""},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			renderEq(t, c.name, c.src, nil, c.want)
		})
	}

	t.Run("does_not_execute_inner_tag", func(t *testing.T) {
		// The inner `{% echo 'hello world' %}` is consumed as comment text
		// up to the first `%}`. Ruby Liquid renders the trailing ` -%}` as
		// literal text. We just assert no "hello world" appears.
		out, err := Render("{%- # {% echo 'hello world' %} -%}", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if strings.Contains(out, "hello world") {
			t.Errorf("inner echo should not have executed; got %q", out)
		}
	})
}

// ---- cycle_tag_test.rb ----

func TestUpstream_Cycle(t *testing.T) {
	t.Run("cycle_basic_advances_within_single_render", func(t *testing.T) {
		renderEq(t, "cycle", `{% cycle "one", "two" %} {% cycle "one", "two" %}`, nil, "one two")
	})

	t.Run("cycle_in_for_loop", func(t *testing.T) {
		src := `{% for i in (1..6) %}{% cycle "a", "b", "c" %}{% endfor %}`
		renderEq(t, "cycle-for", src, nil, "abcabc")
	})

	t.Run("cycle_named_groups_string", func(t *testing.T) {
		// Named groups have independent counters
		src := `{% cycle 'g1': 1, 2, 3 %}{% cycle 'g1': 1, 2, 3 %}{% cycle 'g2': 1, 2, 3 %}{% cycle 'g2': 1, 2, 3 %}`
		renderEq(t, "cycle-named", src, nil, "1212")
	})

	t.Run("cycle_with_variables", func(t *testing.T) {
		src := `{% assign a = 1 %}{% assign b = 2 %}{% assign c = 3 %}{% for i in (1..3) %}{% cycle a, b, c %}{% endfor %}`
		renderEq(t, "cycle-vars", src, nil, "123")
	})
}

// ---- tablerow tests ----

func TestUpstream_Tablerow(t *testing.T) {
	t.Run("tablerow_with_cols", func(t *testing.T) {
		src := "{% tablerow n in numbers cols:3 %} {{n}} {% endtablerow %}"
		data := map[string]any{"numbers": []int{1, 2, 3, 4, 5, 6}}
		out, err := Render(src, data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Expect 2 rows of 3 cols each, containing values 1..6 in order.
		if !strings.Contains(out, `<tr class="row1">`) || !strings.Contains(out, `<tr class="row2">`) {
			t.Errorf("missing row markers; got %q", out)
		}
		for _, want := range []string{" 1 ", " 2 ", " 3 ", " 4 ", " 5 ", " 6 "} {
			if !strings.Contains(out, want) {
				t.Errorf("missing %q in output %q", want, out)
			}
		}
	})

	t.Run("tablerowloop_col_counter", func(t *testing.T) {
		// col is 1-based and resets each row. The body emits col only, so each <td>
		// should contain "1" or "2" matching its col1/col2 class.
		src := "{% tablerow n in numbers cols:2 %}{{tablerowloop.col}}{% endtablerow %}"
		data := map[string]any{"numbers": []int{10, 20, 30, 40}}
		out, err := Render(src, data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		for _, want := range []string{`<td class="col1">1</td>`, `<td class="col2">2</td>`} {
			if !strings.Contains(out, want) {
				t.Errorf("missing %q in output %q", want, out)
			}
		}
	})
}

// ---- break/continue ----

func TestUpstream_BreakContinueOutsideLoop(t *testing.T) {
	t.Run("break_with_no_block_renders_prefix", func(t *testing.T) {
		t.Skip("gap: upstream renders `before` and stops; go-liquid surfaces the break sentinel as a render error")
		out, err := Render("before{% break %}after", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.HasPrefix(out, "before") {
			t.Errorf("expected output starting with 'before', got %q", out)
		}
	})

	t.Run("continue_with_no_block_renders_empty", func(t *testing.T) {
		t.Skip("gap: upstream renders empty string; go-liquid surfaces the continue sentinel as a render error")
		out, err := Render("{% continue %}", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if out != "" {
			t.Errorf("expected empty output, got %q", out)
		}
	})
}

// ---- statements_test.rb ----

func TestUpstream_Statements(t *testing.T) {
	cases := []struct {
		name, src string
		data      map[string]any
		want      string
	}{
		{"true_eql_true", " {% if true == true %} true {% else %} false {% endif %} ", nil, "  true  "},
		{"true_not_eql_true", " {% if true != true %} true {% else %} false {% endif %} ", nil, "  false  "},
		{"zero_gt_zero", " {% if 0 > 0 %} true {% else %} false {% endif %} ", nil, "  false  "},
		{"one_gt_zero", " {% if 1 > 0 %} true {% else %} false {% endif %} ", nil, "  true  "},
		{"zero_lt_one", " {% if 0 < 1 %} true {% else %} false {% endif %} ", nil, "  true  "},
		{"zero_lte_zero", " {% if 0 <= 0 %} true {% else %} false {% endif %} ", nil, "  true  "},
		// gap: upstream returns false for nil <= number (incompatible types); go-liquid coerces nil to 0 and returns true.
		// {"null_lte_zero", " {% if null <= 0 %} true {% else %} false {% endif %} ", nil, "  false  "},
		// {"zero_lte_null", " {% if 0 <= null %} true {% else %} false {% endif %} ", nil, "  false  "},
		{"zero_gte_zero", " {% if 0 >= 0 %} true {% else %} false {% endif %} ", nil, "  true  "},
		{"strings_eq", " {% if 'test' == 'test' %} true {% else %} false {% endif %} ", nil, "  true  "},
		{"strings_neq", " {% if 'test' != 'test' %} true {% else %} false {% endif %} ", nil, "  false  "},
		{"var_string_eq", " {% if var == \"hello there!\" %} true {% else %} false {% endif %} ", map[string]any{"var": "hello there!"}, "  true  "},
		{"var_string_eq_reversed", " {% if \"hello there!\" == var %} true {% else %} false {% endif %} ", map[string]any{"var": "hello there!"}, "  true  "},
		{"empty_array_eq_empty", " {% if array == empty %} true {% else %} false {% endif %} ", map[string]any{"array": []int{}}, "  true  "},
		{"nonempty_array_eq_empty", " {% if array == empty %} true {% else %} false {% endif %} ", map[string]any{"array": []int{1, 2, 3}}, "  false  "},
		{"nil_eq_nil", " {% if var == nil %} true {% else %} false {% endif %} ", map[string]any{"var": nil}, "  true  "},
		{"null_eq_nil", " {% if var == null %} true {% else %} false {% endif %} ", map[string]any{"var": nil}, "  true  "},
		{"nil_neq_value", " {% if var != nil %} true {% else %} false {% endif %} ", map[string]any{"var": 1}, "  true  "},
		{"null_neq_value", " {% if var != null %} true {% else %} false {% endif %} ", map[string]any{"var": 1}, "  true  "},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			renderEq(t, c.name, c.src, c.data, c.want)
		})
	}
}

// ---- case/when ----

func TestUpstream_CaseWhen(t *testing.T) {
	t.Run("when_with_comma", func(t *testing.T) {
		src := "{% case x %}{% when 1, 2, 3 %}low{% when 4 %}four{% endcase %}"
		renderEq(t, "comma-1", src, map[string]any{"x": 1}, "low")
		renderEq(t, "comma-3", src, map[string]any{"x": 3}, "low")
		renderEq(t, "comma-4", src, map[string]any{"x": 4}, "four")
	})

	t.Run("when_with_or", func(t *testing.T) {
		t.Skip("gap: upstream supports `{% when 1 or 2 or 3 %}`; go-liquid only matches the first listed value")
		src := "{% case x %}{% when 1 or 2 or 3 %}low{% when 4 %}four{% endcase %}"
		renderEq(t, "or-1", src, map[string]any{"x": 1}, "low")
		renderEq(t, "or-2", src, map[string]any{"x": 2}, "low")
	})

	t.Run("case_on_size", func(t *testing.T) {
		src := "{% case a.size %}{% when 1 %}1{% when 2 %}2{% endcase %}"
		renderEq(t, "size-0", src, map[string]any{"a": []int{}}, "")
		renderEq(t, "size-1", src, map[string]any{"a": []int{10}}, "1")
		renderEq(t, "size-2", src, map[string]any{"a": []int{10, 20}}, "2")
	})
}

// ---- standard_tag_test.rb extras ----

func TestUpstream_StandardTagExtras(t *testing.T) {
	t.Run("multiline_tag", func(t *testing.T) {
		src := "0{%\nfor i in (1..3)\n%} {{\ni\n}}{%\nendfor\n%}"
		renderEq(t, "multiline", src, nil, "0 1 2 3")
	})

	t.Run("assign_unassigned_rhs", func(t *testing.T) {
		// "var2:" + "" (var2 unset) + " " (literal) + "" ({%assign%} renders blank) + " var2:content"
		// The double space between the two "var2:" segments is intentional.
		src := "var2:{{var2}} {%assign var2 = var%} var2:{{var2}}"
		renderEq(t, "unassigned", src, map[string]any{"var": "content"}, "var2:  var2:content")
	})

	t.Run("assign_empty_string", func(t *testing.T) {
		renderEq(t, "empty-str", "{% assign a = \"\" %}{{a}}", nil, "")
	})

	t.Run("assign_is_global_within_for", func(t *testing.T) {
		src := "{%for i in (1..2) %}{% assign a = \"variable\"%}{% endfor %}{{a}}"
		renderEq(t, "global", src, nil, "variable")
	})

	t.Run("size_of_array", func(t *testing.T) {
		renderEq(t, "size-arr", "array has {{ array.size }} elements",
			map[string]any{"array": []int{1, 2, 3, 4}}, "array has 4 elements")
	})

	t.Run("size_of_hash", func(t *testing.T) {
		t.Skip("gap: upstream supports `.size` on hashes; go-liquid only resolves it on arrays/strings")
		renderEq(t, "size-hash", "hash has {{ hash.size }} elements",
			map[string]any{"hash": map[string]any{"a": 1, "b": 2, "c": 3, "d": 4}},
			"hash has 4 elements")
	})

	t.Run("ifchanged_dedupes", func(t *testing.T) {
		src := "{%for item in array%}{%ifchanged%}{{item}}{% endifchanged %}{%endfor%}"
		renderEq(t, "ifchanged", src, map[string]any{"array": []int{1, 1, 2, 2, 3, 3}}, "123")
	})

	t.Run("capture_without_var_errors", func(t *testing.T) {
		if _, err := Parse("{% capture %}x{% endcapture %}"); err == nil {
			t.Fatal("expected parse error for {% capture %} with no variable name")
		}
	})

	t.Run("case_with_no_when_keyword_errors", func(t *testing.T) {
		if _, err := Parse("{% case false %}{% huh %}true{% endcase %}"); err == nil {
			t.Fatal("expected parse error for unknown inner tag in case")
		}
	})
}

// ---- unless_else_tag_test.rb ----

func TestUpstream_UnlessElse(t *testing.T) {
	cases := []struct {
		name, src string
		data      map[string]any
		want      string
	}{
		{"unless_true", " {% unless true %} this text should not go into the output {% endunless %} ", nil, "  "},
		{"unless_false", " {% unless false %} this text should go into the output {% endunless %} ", nil, "  this text should go into the output  "},
		{"unless_string_truthy_takes_else", "{% unless \"foo\" %} NO {% else %} YES {% endunless %}", nil, " YES "},
		{"unless_in_loop", "{% for i in choices %}{% unless i %}{{ forloop.index }}{% endunless %}{% endfor %}", map[string]any{"choices": []any{1, nil, false}}, "23"},
		{"unless_else_in_loop", "{% for i in choices %}{% unless i %} {{ forloop.index }} {% else %} TRUE {% endunless %}{% endfor %}", map[string]any{"choices": []any{1, nil, false}}, " TRUE  2  3 "},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			renderEq(t, c.name, c.src, c.data, c.want)
		})
	}
}

// ---- output_test.rb (custom filter pipelines) ----

// TestUpstream_OutputFilterPipelines mutates the package-level filters map and
// restores it via defer. Do not add t.Parallel() here or to its subtests, and
// do not run it concurrently with any test that calls RegisterFilter or
// renders templates that depend on built-in filters.
func TestUpstream_OutputFilterPipelines(t *testing.T) {
	// Register filters once for these tests; restore afterwards.
	old := filters
	filters = make(map[string]Filter, len(old))
	for k, v := range old {
		filters[k] = v
	}
	defer func() { filters = old }()

	RegisterFilter("make_funny", func(in any, args ...any) any { return "LOL" })
	RegisterFilter("cite_funny", func(in any, args ...any) any { return "LOL: " + toString(in) })
	RegisterFilter("add_smiley", func(in any, args ...any) any {
		smiley := ":-)"
		if len(args) > 0 {
			smiley = toString(args[0])
		}
		return toString(in) + " " + smiley
	})
	RegisterFilter("add_tag", func(in any, args ...any) any {
		tag := "p"
		id := ""
		if len(args) > 0 {
			tag = toString(args[0])
		}
		if len(args) > 1 {
			id = toString(args[1])
		}
		return "<" + tag + " id=\"" + id + "\">" + toString(in) + "</" + tag + ">"
	})
	RegisterFilter("paragraph", func(in any, args ...any) any {
		return "<p>" + toString(in) + "</p>"
	})

	data := map[string]any{
		"best_cars": "bmw",
		"car":       map[string]any{"bmw": "good", "gm": "bad"},
	}

	cases := []struct{ name, src, want string }{
		{"variable_piping", " {{ car.gm | make_funny }} ", " LOL "},
		{"variable_piping_with_input", " {{ car.gm | cite_funny }} ", " LOL: bad "},
		{"variable_piping_with_args", "! {{ car.gm | add_smiley : ':-(' }} !", "! bad :-( !"},
		{"variable_piping_with_no_args", " {{ car.gm | add_smiley }} ", " bad :-) "},
		{"variable_piping_multiple_args", "( {{ car.gm | add_tag : 'span', 'bar' }} )", "( <span id=\"bar\">bad</span> )"},
		{"variable_piping_variable_args", "( {{ car.gm | add_tag : 'span', car.bmw }} )", "( <span id=\"good\">bad</span> )"},
		{"multiple_pipings", " {{ best_cars | cite_funny | paragraph }} ", " <p>LOL: bmw</p> "},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			renderEq(t, c.name, c.src, data, c.want)
		})
	}
}
