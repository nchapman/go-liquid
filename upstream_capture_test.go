package liquid

// Ported from upstream Ruby Liquid test/integration/capture_test.rb.

import (
	"testing"
)

func TestUpstreamCapture(t *testing.T) {
	t.Run("test_captures_block_content_in_variable", func(t *testing.T) {
		renderEq(t, "basic", "{% capture var %}test string{% endcapture %}{{var}}", nil, "test string")
	})

	t.Run("test_capture_with_hyphen_in_variable_name", func(t *testing.T) {
		src := "{% capture this-thing %}Print this-thing{% endcapture -%}\n{{ this-thing -}}\n"
		renderEq(t, "hyphen", src, nil, "Print this-thing")
	})

	t.Run("test_capture_to_variable_from_outer_scope_if_existing", func(t *testing.T) {
		src := "" +
			"{% assign var = '' -%}\n" +
			"{% if true -%}\n" +
			"  {% capture var %}first-block-string{% endcapture -%}\n" +
			"{% endif -%}\n" +
			"{% if true -%}\n" +
			"  {% capture var %}test-string{% endcapture -%}\n" +
			"{% endif -%}\n" +
			"{{var-}}\n"
		renderEq(t, "outer-scope", src, nil, "test-string")
	})

	t.Run("test_assigning_from_capture", func(t *testing.T) {
		src := "" +
			"{% assign first = '' -%}\n" +
			"{% assign second = '' -%}\n" +
			"{% for number in (1..3) -%}\n" +
			"  {% capture first %}{{number}}{% endcapture -%}\n" +
			"  {% assign second = first -%}\n" +
			"{% endfor -%}\n" +
			"{{ first }}-{{ second -}}\n"
		renderEq(t, "for-capture", src, nil, "3-3")
	})

	t.Run("test_increment_assign_score_by_bytes_not_characters", func(t *testing.T) {
		tmpl := MustParse("{% capture foo %}すごい{% endcapture %}")
		limits := &ResourceLimits{}
		if _, err := tmpl.Render(nil, WithLimits(limits)); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := limits.AssignScore(); got != 9 {
			t.Errorf("assign_score for 'すごい' = %d, want 9 (bytes)", got)
		}
	})

	t.Run("test_capture_with_valid_identifier", func(t *testing.T) {
		renderEq(t, "valid-id", "{% capture my_var %}hello{% endcapture %}{{ my_var }}", nil, "hello")
	})

	t.Run("test_capture_with_hyphen", func(t *testing.T) {
		renderEq(t, "hyphen-id", "{% capture my-var %}hello{% endcapture %}{{ my-var }}", nil, "hello")
	})

	t.Run("test_capture_rejects_parentheses_in_variable_name", func(t *testing.T) {
		if _, err := Parse("{% capture (x[y %}hello{% endcapture %}"); err == nil {
			t.Fatal("expected parse error for parens in capture variable name")
		}
	})

	t.Run("test_capture_rejects_dot_in_variable_name", func(t *testing.T) {
		if _, err := Parse("{% capture a.b %}hello{% endcapture %}"); err == nil {
			t.Fatal("expected parse error for dot in capture variable name")
		}
	})

	t.Run("test_capture_rejects_numeric_variable_name", func(t *testing.T) {
		if _, err := Parse("{% capture 1abc %}hello{% endcapture %}"); err == nil {
			t.Fatal("expected parse error for numeric-prefixed capture variable name")
		}
	})
}
