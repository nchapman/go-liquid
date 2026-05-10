package liquid

// Ported from upstream Ruby Liquid test/integration/tags/break_tag_test.rb.

import "testing"

func TestUpstreamBreakTag(t *testing.T) {
	t.Run("test_break_with_no_block", func(t *testing.T) {
		renderEq(t, "break-no-block", "before{% break %}after", map[string]any{"i": 1}, "before")
	})
}
