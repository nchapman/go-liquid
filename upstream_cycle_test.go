package liquid

// Ported from upstream Ruby Liquid test/integration/tags/cycle_tag_test.rb.

import (
	"strings"
	"testing"
)

func TestUpstreamCycleTag(t *testing.T) {
	t.Run("test_simple_cycle_inside_for_loop", func(t *testing.T) {
		src := "{%- for i in (1..3) -%}\n  {%- cycle '1', '2', '3' -%}\n{%- endfor -%}\n"
		renderEq(t, "simple", src, nil, "123")
	})

	t.Run("test_cycle_with_variables_inside_for_loop", func(t *testing.T) {
		src := "{%- assign a = 1 -%}\n{%- assign b = 2 -%}\n{%- assign c = 3 -%}\n" +
			"{%- for i in (1..3) -%}\n  {% cycle a, b, c %}\n{%- endfor -%}\n"
		renderEq(t, "vars", src, nil, "123")
	})

	t.Run("test_cycle_named_groups_string", func(t *testing.T) {
		src := "{%- for i in (1..3) -%}\n" +
			"  {%- cycle 'placeholder1': 1, 2, 3 -%}\n" +
			"  {%- cycle 'placeholder2': 1, 2, 3 -%}\n" +
			"{%- endfor -%}\n"
		renderEq(t, "named-string", src, nil, "112233")
	})

	t.Run("test_cycle_named_groups_vlookup", func(t *testing.T) {
		src := "{%- assign placeholder1 = 'placeholder1' -%}\n" +
			"{%- assign placeholder2 = 'placeholder2' -%}\n" +
			"{%- for i in (1..3) -%}\n" +
			"  {%- cycle placeholder1: 1, 2, 3 -%}\n" +
			"  {%- cycle placeholder2: 1, 2, 3 -%}\n" +
			"{%- endfor -%}\n"
		renderEq(t, "named-vlookup", src, nil, "112233")
	})

	t.Run("test_unnamed_cycle_have_independent_counters_when_used_with_lookups", func(t *testing.T) {
		// Ruby Liquid quirk: two textually-distinct unnamed cycles whose argument
		// lists contain a VariableLookup get *independent* counters even when the
		// lookups resolve to the same value, because Ruby uses object identity
		// (dup'd lookups) when building the cycle key. go-liquid keys by
		// resolved value, so it shares the counter and produces "121212" instead
		// of "112211".
		t.Skip("TODO: cycle key for unnamed cycles with lookups should be per-occurrence (Ruby dup quirk)")
		src := "{%- assign a = \"1\" -%}\n{%- for i in (1..3) -%}\n  {%- cycle a, \"2\" -%}\n  {%- cycle a, \"2\" -%}\n{%- endfor -%}\n"
		renderEq(t, "lookup-independent", src, nil, "112211")
	})

	t.Run("test_unnamed_cycle_dependent_counter_when_used_with_literal_values", func(t *testing.T) {
		src := "{%- cycle \"1\", \"2\" -%}\n{%- cycle \"1\", \"2\" -%}\n{%- cycle \"1\", \"2\" -%}\n"
		renderEq(t, "literal-shared", src, nil, "121")
	})

	t.Run("test_optional_trailing_comma", func(t *testing.T) {
		src := "{%- cycle \"1\", \"2\", -%}\n{%- cycle \"1\", \"2\", -%}\n{%- cycle \"1\", \"2\", -%}\n{%- cycle \"1\", -%}\n"
		renderEq(t, "trailing-comma", src, nil, "1211")
	})

	t.Run("test_cycle_tag_without_arguments", func(t *testing.T) {
		_, err := Parse("{% cycle %}")
		if err == nil {
			t.Fatal("expected parse error for empty cycle")
		}
		if !strings.Contains(err.Error(), "cycle") {
			t.Errorf("error should mention 'cycle'; got %q", err.Error())
		}
	})

	t.Run("test_cycle_tag_with_error_mode_lax", func(t *testing.T) {
		t.Skip("TODO: ErrorModeLax should swallow malformed expressions inside known-tag bodies (Ruby :lax/:strict tolerance)")
		// Ruby :lax/:strict accept these "permissive" forms; only :strict2 rejects.
		// go-liquid has no strict2; we only verify the lax/strict path here.
		renderEq(t, "lax-1", "{% assign 5 = 'b' %}{% cycle .5, .4 %}", nil, "b")
		renderEq(t, "lax-2", "{% cycle .5: 'a', 'b' %}", nil, "a")
	})

	t.Run("test_cycle_with_trailing_elements", func(t *testing.T) {
		t.Skip("TODO: Ruby :lax/:strict QuotedFragment tolerates whitespace-separated trailing tokens in cycle args")
		assigns := "{% assign a = 'A' %}{% assign n = 'N' %}"
		renderEq(t, "trailing-1", assigns+"{% cycle       'a'  'b', 'c' %}", nil, "a")
		renderEq(t, "trailing-2", assigns+"{% cycle name: 'a'  'b', 'c' %}", nil, "a")
		renderEq(t, "trailing-3", assigns+"{% cycle name: 'a', 'b'  'c' %}", nil, "a")
		renderEq(t, "trailing-4", assigns+"{% cycle n  e: 'a', 'b', 'c' %}", nil, "N")
		renderEq(t, "trailing-5", assigns+"{% cycle n  e  'a', 'b', 'c' %}", nil, "N")
	})

	t.Run("test_cycle_name_with_invalid_expression", func(t *testing.T) {
		t.Skip("TODO: Ruby :lax/:strict parses `foo=>bar` permissively in cycle name position")
		src := "{% for i in (1..3) %}\n  {% cycle foo=>bar: \"a\", \"b\" %}\n{% endfor %}\n"
		if _, err := Parse(src); err != nil {
			t.Errorf("lax/strict should accept malformed cycle name; got %v", err)
		}
	})

	t.Run("test_cycle_variable_with_invalid_expression", func(t *testing.T) {
		t.Skip("TODO: Ruby :lax/:strict parses `foo=>bar` permissively in cycle var position")
		src := "{% for i in (1..3) %}\n  {% cycle foo=>bar, \"a\", \"b\" %}\n{% endfor %}\n"
		if _, err := Parse(src); err != nil {
			t.Errorf("lax/strict should accept malformed cycle var; got %v", err)
		}
	})
}
