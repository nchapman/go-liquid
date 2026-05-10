package liquid

// Ported from upstream Ruby Liquid test/integration/document_test.rb.

import (
	"strings"
	"testing"
)

func TestUpstreamDocument(t *testing.T) {
	t.Run("test_unexpected_outer_tag", func(t *testing.T) {
		_, err := Parse("{% else %}")
		if err == nil {
			t.Fatal("expected syntax error for outer {% else %}")
		}
	})

	t.Run("test_unknown_tag", func(t *testing.T) {
		_, err := Parse("{% foo %}")
		if err == nil {
			t.Fatal("expected syntax error for {% foo %}")
		}
		if !strings.Contains(strings.ToLower(err.Error()), "foo") {
			t.Errorf("error should mention the tag name; got %q", err.Error())
		}
	})
}
