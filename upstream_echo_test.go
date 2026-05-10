package liquid

// Ported from upstream Ruby Liquid test/integration/tags/echo_test.rb.

import "testing"

func TestUpstreamEchoTag(t *testing.T) {
	t.Run("test_echo_outputs_its_input", func(t *testing.T) {
		renderEq(t, "echo", "{%- echo variable-name | upcase -%}\n", map[string]any{"variable-name": "bar"}, "BAR")
	})
}
