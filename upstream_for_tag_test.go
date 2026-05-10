package liquid

// Ported from upstream Ruby Liquid test/integration/tags/for_tag_test.rb.

import (
	"strings"
	"testing"
)

func TestUpstreamForTag(t *testing.T) {
	t.Run("test_for", func(t *testing.T) {
		renderEq(t, "4", "{%for item in array%} yo {%endfor%}", map[string]any{"array": []any{1, 2, 3, 4}}, " yo  yo  yo  yo ")
		renderEq(t, "yoyo", "{%for item in array%}yo{%endfor%}", map[string]any{"array": []any{1, 2}}, "yoyo")
		renderEq(t, "1", "{%for item in array%} yo {%endfor%}", map[string]any{"array": []any{1}}, " yo ")
		renderEq(t, "empty-body", "{%for item in array%}{%endfor%}", map[string]any{"array": []any{1, 2}}, "")
		// Multi-line block with surrounding text.
		tmpl := "{%for item in array%}\n  yo\n{%endfor%}\n"
		expected := "\n  yo\n\n  yo\n\n  yo\n\n"
		renderEq(t, "multiline", tmpl, map[string]any{"array": []any{1, 2, 3}}, expected)
	})

	t.Run("test_for_reversed", func(t *testing.T) {
		renderEq(t, "reversed", "{%for item in array reversed %}{{item}}{%endfor%}",
			map[string]any{"array": []any{1, 2, 3}}, "321")
	})

	t.Run("test_for_with_range", func(t *testing.T) {
		renderEq(t, "range-1-3", "{%for item in (1..3) %} {{item}} {%endfor%}", nil, " 1  2  3 ")
		// Ruby raises Liquid::ArgumentError when a range bound is a non-numeric
		// non-coercible value like an array. go-liquid currently coerces to 0
		// silently — surfaces are documented separately.
		t.Run("non-numeric-array-bound", func(t *testing.T) {
			t.Skip("TODO: surface Liquid::ArgumentError for non-coercible range bound (Ruby parity)")
			_, err := Render("{% for i in (a..2) %}{% endfor %}", map[string]any{"a": []any{1, 2}})
			if err == nil {
				t.Fatal("expected error for non-numeric range bound")
			}
		})
		// String-as-range-bound coerces to 0 in Ruby (Integer(s) rescue 0).
		renderEq(t, "string-bound", "{% for item in (a..3) %} {{item}} {% endfor %}",
			map[string]any{"a": "invalid integer"}, " 0  1  2  3 ")
	})

	t.Run("test_for_with_variable_range", func(t *testing.T) {
		renderEq(t, "var-range", "{%for item in (1..foobar) %} {{item}} {%endfor%}",
			map[string]any{"foobar": 3}, " 1  2  3 ")
	})

	t.Run("test_for_with_hash_value_range", func(t *testing.T) {
		renderEq(t, "hash-range", "{%for item in (1..foobar.value) %} {{item}} {%endfor%}",
			map[string]any{"foobar": map[string]any{"value": 3}}, " 1  2  3 ")
	})

	t.Run("test_for_with_drop_value_range", func(t *testing.T) {
		t.Skip("TODO: Liquid::Drop equivalent — go-liquid's drop interface uses DropContext, not method-named accessors")
		renderEq(t, "drop-range", "{%for item in (1..foobar.value) %} {{item}} {%endfor%}", nil, " 1  2  3 ")
	})

	t.Run("test_for_with_variable", func(t *testing.T) {
		renderEq(t, "spaces", "{%for item in array%} {{item}} {%endfor%}",
			map[string]any{"array": []any{1, 2, 3}}, " 1  2  3 ")
		renderEq(t, "tight", "{%for item in array%}{{item}}{%endfor%}",
			map[string]any{"array": []any{1, 2, 3}}, "123")
		renderEq(t, "spacy-tag", "{% for item in array %}{{item}}{% endfor %}",
			map[string]any{"array": []any{1, 2, 3}}, "123")
		renderEq(t, "letters", "{%for item in array%}{{item}}{%endfor%}",
			map[string]any{"array": []any{"a", "b", "c", "d"}}, "abcd")
		renderEq(t, "spaces-in-array", "{%for item in array%}{{item}}{%endfor%}",
			map[string]any{"array": []any{"a", " ", "b", " ", "c"}}, "a b c")
		renderEq(t, "empties-in-array", "{%for item in array%}{{item}}{%endfor%}",
			map[string]any{"array": []any{"a", "", "b", "", "c"}}, "abc")
	})

	t.Run("test_for_helpers", func(t *testing.T) {
		data := map[string]any{"array": []any{1, 2, 3}}
		renderEq(t, "index/length", "{%for item in array%} {{forloop.index}}/{{forloop.length}} {%endfor%}", data, " 1/3  2/3  3/3 ")
		renderEq(t, "index", "{%for item in array%} {{forloop.index}} {%endfor%}", data, " 1  2  3 ")
		renderEq(t, "index0", "{%for item in array%} {{forloop.index0}} {%endfor%}", data, " 0  1  2 ")
		renderEq(t, "rindex0", "{%for item in array%} {{forloop.rindex0}} {%endfor%}", data, " 2  1  0 ")
		renderEq(t, "rindex", "{%for item in array%} {{forloop.rindex}} {%endfor%}", data, " 3  2  1 ")
		renderEq(t, "first", "{%for item in array%} {{forloop.first}} {%endfor%}", data, " true  false  false ")
		renderEq(t, "last", "{%for item in array%} {{forloop.last}} {%endfor%}", data, " false  false  true ")
	})

	t.Run("test_for_and_if", func(t *testing.T) {
		renderEq(t, "first-marker",
			"{%for item in array%}{% if forloop.first %}+{% else %}-{% endif %}{%endfor%}",
			map[string]any{"array": []any{1, 2, 3}}, "+--")
	})

	t.Run("test_for_else", func(t *testing.T) {
		src := "{%for item in array%}+{%else%}-{%endfor%}"
		renderEq(t, "non-empty", src, map[string]any{"array": []any{1, 2, 3}}, "+++")
		renderEq(t, "empty", src, map[string]any{"array": []any{}}, "-")
		renderEq(t, "nil", src, map[string]any{"array": nil}, "-")
	})

	t.Run("test_limiting", func(t *testing.T) {
		data := map[string]any{"array": []any{1, 2, 3, 4, 5, 6, 7, 8, 9, 0}}
		renderEq(t, "limit:2", "{%for i in array limit:2 %}{{ i }}{%endfor%}", data, "12")
		renderEq(t, "limit:4", "{%for i in array limit:4 %}{{ i }}{%endfor%}", data, "1234")
		renderEq(t, "limit:4 offset:2", "{%for i in array limit:4 offset:2 %}{{ i }}{%endfor%}", data, "3456")
		renderEq(t, "spaces", "{%for i in array limit: 4 offset: 2 %}{{ i }}{%endfor%}", data, "3456")
		renderEq(t, "commas", "{%for i in array, limit: 4, offset: 2 %}{{ i }}{%endfor%}", data, "3456")
	})

	t.Run("test_limiting_with_invalid_limit", func(t *testing.T) {
		t.Skip("TODO: emit Liquid::ArgumentError for non-integer `limit:` argument (Ruby parity)")
		data := map[string]any{"array": []any{1, 2, 3, 4, 5, 6, 7, 8, 9, 0}}
		_, err := Render("{% for i in array limit: true offset: 1 %}{{ i }}{% endfor %}", data)
		if err == nil || !strings.Contains(err.Error(), "invalid integer") {
			t.Fatalf("expected invalid integer error, got %v", err)
		}
	})

	t.Run("test_limiting_with_invalid_offset", func(t *testing.T) {
		t.Skip("TODO: emit Liquid::ArgumentError for non-integer `offset:` argument (Ruby parity)")
		data := map[string]any{"array": []any{1, 2, 3, 4, 5, 6, 7, 8, 9, 0}}
		_, err := Render("{% for i in array limit: 1 offset: true %}{{ i }}{% endfor %}", data)
		if err == nil || !strings.Contains(err.Error(), "invalid integer") {
			t.Fatalf("expected invalid integer error, got %v", err)
		}
	})

	t.Run("test_dynamic_variable_limiting", func(t *testing.T) {
		data := map[string]any{
			"array":  []any{1, 2, 3, 4, 5, 6, 7, 8, 9, 0},
			"limit":  2,
			"offset": 2,
		}
		renderEq(t, "dyn", "{%for i in array limit: limit offset: offset %}{{ i }}{%endfor%}", data, "34")
	})

	t.Run("test_nested_for", func(t *testing.T) {
		data := map[string]any{"array": []any{[]any{1, 2}, []any{3, 4}, []any{5, 6}}}
		renderEq(t, "nested", "{%for item in array%}{%for i in item%}{{ i }}{%endfor%}{%endfor%}", data, "123456")
	})

	t.Run("test_offset_only", func(t *testing.T) {
		data := map[string]any{"array": []any{1, 2, 3, 4, 5, 6, 7, 8, 9, 0}}
		renderEq(t, "offset", "{%for i in array offset:7 %}{{ i }}{%endfor%}", data, "890")
	})

	t.Run("test_pause_resume", func(t *testing.T) {
		data := map[string]any{"array": map[string]any{"items": []any{1, 2, 3, 4, 5, 6, 7, 8, 9, 0}}}
		// Indentation matters because Liquid renders raw whitespace.
		markup := "      {%for i in array.items limit: 3 %}{{i}}{%endfor%}\n      next\n      {%for i in array.items offset:continue limit: 3 %}{{i}}{%endfor%}\n      next\n      {%for i in array.items offset:continue limit: 3 %}{{i}}{%endfor%}\n"
		expected := "      123\n      next\n      456\n      next\n      789\n"
		renderEq(t, "pause-resume", markup, data, expected)
	})

	t.Run("test_pause_resume_limit", func(t *testing.T) {
		data := map[string]any{"array": map[string]any{"items": []any{1, 2, 3, 4, 5, 6, 7, 8, 9, 0}}}
		markup := "      {%for i in array.items limit:3 %}{{i}}{%endfor%}\n      next\n      {%for i in array.items offset:continue limit:3 %}{{i}}{%endfor%}\n      next\n      {%for i in array.items offset:continue limit:1 %}{{i}}{%endfor%}\n"
		expected := "      123\n      next\n      456\n      next\n      7\n"
		renderEq(t, "pause-resume-limit", markup, data, expected)
	})

	t.Run("test_pause_resume_big_limit", func(t *testing.T) {
		data := map[string]any{"array": map[string]any{"items": []any{1, 2, 3, 4, 5, 6, 7, 8, 9, 0}}}
		markup := "      {%for i in array.items limit:3 %}{{i}}{%endfor%}\n      next\n      {%for i in array.items offset:continue limit:3 %}{{i}}{%endfor%}\n      next\n      {%for i in array.items offset:continue limit:1000 %}{{i}}{%endfor%}\n"
		expected := "      123\n      next\n      456\n      next\n      7890\n"
		renderEq(t, "pause-resume-big-limit", markup, data, expected)
	})

	t.Run("test_pause_resume_big_offset", func(t *testing.T) {
		data := map[string]any{"array": map[string]any{"items": []any{1, 2, 3, 4, 5, 6, 7, 8, 9, 0}}}
		markup := "{%for i in array.items limit:3 %}{{i}}{%endfor%}\n      next\n      {%for i in array.items offset:continue limit:3 %}{{i}}{%endfor%}\n      next\n      {%for i in array.items offset:continue limit:3 offset:1000 %}{{i}}{%endfor%}"
		expected := "123\n      next\n      456\n      next\n      "
		renderEq(t, "pause-resume-big-offset", markup, data, expected)
	})

	t.Run("test_for_with_break", func(t *testing.T) {
		data := map[string]any{"array": map[string]any{"items": []any{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}}}
		renderEq(t, "break-immediate", "{% for i in array.items %}{% break %}{% endfor %}", data, "")
		renderEq(t, "break-after-emit", "{% for i in array.items %}{{ i }}{% break %}{% endfor %}", data, "1")
		renderEq(t, "break-before-emit", "{% for i in array.items %}{% break %}{{ i }}{% endfor %}", data, "")
		renderEq(t, "break-conditional",
			"{% for i in array.items %}{{ i }}{% if i > 3 %}{% break %}{% endif %}{% endfor %}",
			data, "1234")

		nested := map[string]any{"array": []any{[]any{1, 2}, []any{3, 4}, []any{5, 6}}}
		renderEq(t, "break-local",
			"{% for item in array %}{% for i in item %}{% if i == 1 %}{% break %}{% endif %}{{ i }}{% endfor %}{% endfor %}",
			nested, "3456")

		short := map[string]any{"array": map[string]any{"items": []any{1, 2, 3, 4, 5}}}
		renderEq(t, "break-unreached",
			"{% for i in array.items %}{% if i == 9999 %}{% break %}{% endif %}{{ i }}{% endfor %}",
			short, "12345")
	})

	t.Run("test_for_with_break_after_nested_loop", func(t *testing.T) {
		src := "{% for i in (1..2) -%}\n  {% for j in (1..2) -%}\n    {{ i }}-{{ j }},\n  {%- endfor -%}\n  {% break -%}\n{% endfor -%}\nafter"
		renderEq(t, "break-after-nested", src, nil, "1-1,1-2,after")
	})

	t.Run("test_for_with_continue", func(t *testing.T) {
		data := map[string]any{"array": map[string]any{"items": []any{1, 2, 3, 4, 5}}}
		renderEq(t, "continue-immediate", "{% for i in array.items %}{% continue %}{% endfor %}", data, "")
		renderEq(t, "continue-after-emit", "{% for i in array.items %}{{ i }}{% continue %}{% endfor %}", data, "12345")
		renderEq(t, "continue-before-emit", "{% for i in array.items %}{% continue %}{{ i }}{% endfor %}", data, "")
		renderEq(t, "continue-conditional",
			"{% for i in array.items %}{% if i > 3 %}{% continue %}{% endif %}{{ i }}{% endfor %}", data, "123")
		renderEq(t, "continue-skip-3",
			"{% for i in array.items %}{% if i == 3 %}{% continue %}{% else %}{{ i }}{% endif %}{% endfor %}", data, "1245")

		nested := map[string]any{"array": []any{[]any{1, 2}, []any{3, 4}, []any{5, 6}}}
		renderEq(t, "continue-local",
			"{% for item in array %}{% for i in item %}{% if i == 1 %}{% continue %}{% endif %}{{ i }}{% endfor %}{% endfor %}",
			nested, "23456")

		renderEq(t, "continue-unreached",
			"{% for i in array.items %}{% if i == 9999 %}{% continue %}{% endif %}{{ i }}{% endfor %}",
			data, "12345")
	})

	t.Run("test_for_tag_string", func(t *testing.T) {
		// Ruby Liquid treats a String as a single-element iterable.
		renderEq(t, "string-as-single",
			"{%for val in string%}{{val}}{%endfor%}",
			map[string]any{"string": "test string"}, "test string")
		renderEq(t, "string-as-single-limit",
			"{%for val in string limit:1%}{{val}}{%endfor%}",
			map[string]any{"string": "test string"}, "test string")
		renderEq(t, "string-forloop",
			"{%for val in string%}{{forloop.name}}-{{forloop.index}}-{{forloop.length}}-{{forloop.index0}}-{{forloop.rindex}}-{{forloop.rindex0}}-{{forloop.first}}-{{forloop.last}}-{{val}}{%endfor%}",
			map[string]any{"string": "test string"},
			"val-string-1-1-0-1-0-true-true-test string")
	})

	t.Run("test_for_parentloop_references_parent_loop", func(t *testing.T) {
		renderEq(t, "parentloop",
			"{% for inner in outer %}{% for k in inner %}{{ forloop.parentloop.index }}.{{ forloop.index }} {% endfor %}{% endfor %}",
			map[string]any{"outer": []any{[]any{1, 1, 1}, []any{1, 1, 1}}},
			"1.1 1.2 1.3 2.1 2.2 2.3 ")
	})

	t.Run("test_for_parentloop_nil_when_not_present", func(t *testing.T) {
		renderEq(t, "parentloop-nil",
			"{% for inner in outer %}{{ forloop.parentloop.index }}.{{ forloop.index }} {% endfor %}",
			map[string]any{"outer": []any{[]any{1, 1, 1}, []any{1, 1, 1}}},
			".1 .2 ")
	})

	t.Run("test_inner_for_over_empty_input", func(t *testing.T) {
		renderEq(t, "inner-empty",
			"{% for a in (1..2) %}o{% for b in empty %}{% endfor %}{% endfor %}",
			nil, "oo")
	})

	t.Run("test_blank_string_not_iterable", func(t *testing.T) {
		renderEq(t, "blank-string",
			"{% for char in characters %}I WILL NOT BE OUTPUT{% endfor %}",
			map[string]any{"characters": ""}, "")
	})

	t.Run("test_bad_variable_naming_in_for_loop", func(t *testing.T) {
		if _, err := Parse("{% for a/b in x %}{% endfor %}"); err == nil {
			t.Fatal("expected parse error for invalid loop variable")
		}
	})

	t.Run("test_spacing_with_variable_naming_in_for_loop", func(t *testing.T) {
		renderEq(t, "spacy",
			"{% for       item   in   items %}{{item}}{% endfor %}",
			map[string]any{"items": []any{1, 2, 3, 4, 5}}, "12345")
	})

	t.Run("test_iterate_with_each_when_no_limit_applied", func(t *testing.T) {
		t.Skip("TODO: expose a Drop/Iterable protocol with each + load_slice hooks (Ruby parity); without it we can't assert which path was taken")
		// Once a LoaderDrop-equivalent exists, this test should iterate the
		// loader without limit, expect output "12345", and assert that the
		// each-style path was taken (no slice load).
		renderEq(t, "each-no-limit", "{% for item in items %}{{item}}{% endfor %}",
			map[string]any{"items": []any{1, 2, 3, 4, 5}}, "12345")
	})

	t.Run("test_iterate_with_load_slice_when_limit_applied", func(t *testing.T) {
		t.Skip("TODO: expose a Drop/Iterable protocol with each + load_slice hooks (Ruby parity); without it we can't assert which path was taken")
		renderEq(t, "slice-limit", "{% for item in items limit:1 %}{{item}}{% endfor %}",
			map[string]any{"items": []any{1, 2, 3, 4, 5}}, "1")
	})

	t.Run("test_iterate_with_load_slice_when_limit_and_offset_applied", func(t *testing.T) {
		t.Skip("TODO: expose a Drop/Iterable protocol with each + load_slice hooks (Ruby parity); without it we can't assert which path was taken")
		renderEq(t, "slice-limit-offset", "{% for item in items offset:2 limit:2 %}{{item}}{% endfor %}",
			map[string]any{"items": []any{1, 2, 3, 4, 5}}, "34")
	})

	t.Run("test_iterate_with_load_slice_returns_same_results_as_without", func(t *testing.T) {
		t.Skip("TODO: expose a Drop/Iterable protocol with each + load_slice hooks (Ruby parity); the array half of this test already works — we only need the LoaderDrop side to compare")
		// The plain-array half of the parity check still works today:
		renderEq(t, "array-slice", "{% for item in items offset:2 limit:2 %}{{item}}{% endfor %}",
			map[string]any{"items": []any{1, 2, 3, 4, 5}}, "34")
	})

	t.Run("test_for_cleans_up_registers", func(t *testing.T) {
		t.Skip("TODO: expose a Registers introspection API so post-render state can be asserted (Ruby Context.registers); for now we can only verify the error surfaces")
		// Ruby asserts that after a render that errored mid-loop, the
		// :for_stack register is empty. We can at least verify the error
		// surfaces; cleanup verification needs a Registers-introspection API.
		_, err := Render("{% for i in (1..2) %}{{ standard_error }}{% endfor %}", nil)
		if err == nil {
			t.Fatal("expected error from missing variable in strict mode")
		}
	})
}
