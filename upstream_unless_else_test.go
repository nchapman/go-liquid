package liquid

// Ported from upstream Ruby Liquid test/integration/tags/unless_else_tag_test.rb.

import "testing"

func TestUpstreamUnlessElseTag(t *testing.T) {
	t.Run("test_unless", func(t *testing.T) {
		renderEq(t, "true-skip", " {% unless true %} this text should not go into the output {% endunless %} ", nil, "  ")
		renderEq(t, "false-render", " {% unless false %} this text should go into the output {% endunless %} ", nil, "  this text should go into the output  ")
		renderEq(t, "mixed", "{% unless true %} you suck {% endunless %} {% unless false %} you rock {% endunless %}?", nil, "  you rock ?")
	})

	t.Run("test_unless_else", func(t *testing.T) {
		renderEq(t, "true-else", "{% unless true %} NO {% else %} YES {% endunless %}", nil, " YES ")
		renderEq(t, "false-then", "{% unless false %} YES {% else %} NO {% endunless %}", nil, " YES ")
		renderEq(t, "string-truthy", `{% unless "foo" %} NO {% else %} YES {% endunless %}`, nil, " YES ")
	})

	t.Run("test_unless_in_loop", func(t *testing.T) {
		renderEq(t, "loop", "{% for i in choices %}{% unless i %}{{ forloop.index }}{% endunless %}{% endfor %}",
			map[string]any{"choices": []any{1, nil, false}}, "23")
	})

	t.Run("test_unless_else_in_loop", func(t *testing.T) {
		renderEq(t, "loop-else", "{% for i in choices %}{% unless i %} {{ forloop.index }} {% else %} TRUE {% endunless %}{% endfor %}",
			map[string]any{"choices": []any{1, nil, false}}, " TRUE  2  3 ")
	})
}
