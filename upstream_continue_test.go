package liquid

// Ported from upstream Ruby Liquid test/integration/tags/continue_tag_test.rb.

import "testing"

func TestUpstreamContinueTag(t *testing.T) {
	t.Run("test_continue_with_no_block", func(t *testing.T) {
		renderEq(t, "continue-no-block", "{% continue %}", nil, "")
	})
}
