package liquid

// Ported from upstream Ruby Liquid test/integration/tags/increment_tag_test.rb.

import "testing"

func TestUpstreamIncrementTag(t *testing.T) {
	t.Run("test_inc", func(t *testing.T) {
		renderEq(t, "basic", "{%increment port %} {{ port }}", nil, "0 1")
		renderEq(t, "interleaved", "{{port}} {%increment port %} {%increment port%} {{port}}", nil, " 0 1 2")
		renderEq(t, "two-counters",
			"{%increment port %} {%increment starboard%} {%increment port %} {%increment port%} {%increment starboard %}",
			nil, "0 0 1 2 1")
	})

	t.Run("test_dec", func(t *testing.T) {
		renderEq(t, "dec-basic", "{%decrement port %} {{ port }}", map[string]any{"port": 10}, "-1 -1")
		renderEq(t, "dec-interleaved", "{{port}} {%decrement port %} {%decrement port%} {{port}}", nil, " -1 -2 -2")
		renderEq(t, "inc-dec-mixed",
			"{%increment starboard %} {%increment starboard%} {%increment starboard%} {%increment port %} {%increment starboard%} {%increment port %} {%decrement port%} {%decrement starboard %}",
			nil, "0 1 2 0 3 1 1 3")
	})

	t.Run("test_increment_rejects_invalid_variable_name", func(t *testing.T) {
		if _, err := Parse("{% increment foo bar %}"); err == nil {
			t.Fatal("expected parse error for `increment foo bar`")
		}
	})

	t.Run("test_increment_rejects_variable_starting_with_number", func(t *testing.T) {
		if _, err := Parse("{% increment 11aa %}"); err == nil {
			t.Fatal("expected parse error for `increment 11aa`")
		}
	})

	t.Run("test_increment_accepts_valid_variable_name", func(t *testing.T) {
		renderEq(t, "hyphen", "{% increment my-var %}", nil, "0")
	})

	t.Run("test_decrement_rejects_invalid_variable_name", func(t *testing.T) {
		if _, err := Parse("{% decrement foo bar %}"); err == nil {
			t.Fatal("expected parse error for `decrement foo bar`")
		}
	})

	t.Run("test_decrement_rejects_variable_starting_with_number", func(t *testing.T) {
		if _, err := Parse("{% decrement 11aa %}"); err == nil {
			t.Fatal("expected parse error for `decrement 11aa`")
		}
	})

	t.Run("test_decrement_accepts_valid_variable_name", func(t *testing.T) {
		renderEq(t, "hyphen", "{% decrement my-var %}", nil, "-1")
	})

	t.Run("test_increment_rejects_empty_variable_name", func(t *testing.T) {
		if _, err := Parse("{% increment %}"); err == nil {
			t.Fatal("expected parse error for empty `increment`")
		}
	})

	t.Run("test_decrement_rejects_empty_variable_name", func(t *testing.T) {
		if _, err := Parse("{% decrement %}"); err == nil {
			t.Fatal("expected parse error for empty `decrement`")
		}
	})
}
