package liquid

// Ported from upstream Ruby Liquid test/integration/tags/inline_comment_test.rb.

import (
	"strings"
	"testing"
)

func TestUpstreamInlineComment(t *testing.T) {
	t.Run("test_inline_comment_returns_nothing", func(t *testing.T) {
		renderEq(t, "trim-space", "{%- # this is an inline comment -%}", nil, "")
		renderEq(t, "trim-no-space", "{%-# this is an inline comment -%}", nil, "")
		renderEq(t, "plain-space", "{% # this is an inline comment %}", nil, "")
		renderEq(t, "plain-no-space", "{%# this is an inline comment %}", nil, "")
	})

	t.Run("test_inline_comment_does_not_require_a_space_after_the_pound_sign", func(t *testing.T) {
		renderEq(t, "no-space", "{%#this is an inline comment%}", nil, "")
	})

	t.Run("test_liquid_inline_comment_returns_nothing", func(t *testing.T) {
		src := "{%- liquid\n" +
			"  # This is how you'd write a block comment in a liquid tag.\n" +
			"  # It looks a lot like what you'd have in ruby.\n\n" +
			"  # You can use it as inline documentation in your\n" +
			"  # liquid blocks to explain why you're doing something.\n" +
			"  echo \"Hey there, \"\n\n" +
			"  # It won't affect the output.\n" +
			"  echo \"how are you doing today?\"\n" +
			"-%}\n"
		renderEq(t, "liquid-comment", src, nil, "Hey there, how are you doing today?")
	})

	t.Run("test_inline_comment_can_be_written_on_multiple_lines", func(t *testing.T) {
		src := "{%-\n" +
			"  # That kind of block comment is also allowed.\n" +
			"  # It would only be a stylistic difference.\n\n" +
			"  # Much like JavaScript's /* */ comments and their\n" +
			"  # leading * on new lines.\n" +
			"-%}\n"
		renderEq(t, "multi-line", src, nil, "")
	})

	t.Run("test_inline_comment_multiple_pound_signs", func(t *testing.T) {
		src := "{%- liquid\n" +
			"  ######################################\n" +
			"  # We support comments like this too. #\n" +
			"  ######################################\n" +
			"-%}\n"
		renderEq(t, "multi-pound", src, nil, "")
	})

	t.Run("test_inline_comments_require_the_pound_sign_on_every_new_line", func(t *testing.T) {
		src := "{%-\n  # some comment\n  echo 'hello world'\n-%}\n"
		_, err := Parse(src)
		if err == nil {
			t.Fatal("expected parse error when comment lines miss the '#' prefix")
		}
		if !strings.Contains(err.Error(), "#") {
			t.Errorf("error should mention '#'; got %q", err.Error())
		}
	})

	t.Run("test_inline_comment_does_not_support_nested_tags", func(t *testing.T) {
		// Ruby parses the inline comment up to the first %} then renders ` -%}` as text.
		out, err := Render("{%- # {% echo 'hello world' %} -%}", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if out != " -%}" {
			t.Errorf("want %q, got %q", " -%}", out)
		}
	})
}
