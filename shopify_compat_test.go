package liquid

import (
	"testing"
)

// TestShopifyCompat collects regression tests pinning down Shopify-Liquid
// behaviors discovered during a cross-check against the canonical Ruby
// reference at /Users/nchapman/Code/liquid/. Each subtest names the bug
// it locks down so a future regression is obvious in CI output.
func TestShopifyCompat(t *testing.T) {
	cases := []struct {
		name, tmpl, want string
		data             map[string]any
	}{
		// Ruby IDENTIFIER allows [a-zA-Z_][\w-]*\??.
		{"hyphen in variable name", `{{ my-var }}`, "hi", map[string]any{"my-var": "hi"}},
		{"hyphen in property", `{{ obj.product-name }}`, "Widget", map[string]any{"obj": map[string]any{"product-name": "Widget"}}},
		{"hyphen does not split identifier", `{{ a-b | minus: 1 }}`, "2", map[string]any{"a-b": 3}},

		// UTF-8 corrections.
		{"truncate counts runes", `{{ s | truncate: 5 }}`, "ca...", map[string]any{"s": "café au lait"}},
		{"size of utf8 is char count", `{{ s | size }}`, "5", map[string]any{"s": "héllo"}},
		{"first of utf8 is rune", `{{ s | first }}`, "h", map[string]any{"s": "héllo"}},
		{"last of utf8 is rune", `{{ s | last }}`, "o", map[string]any{"s": "héllo"}},
		{"first of empty string returns empty string", `[{{ s | first }}]`, "[]", map[string]any{"s": ""}},
		{"last of empty string returns empty string", `[{{ s | last }}]`, "[]", map[string]any{"s": ""}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Render(tc.tmpl, tc.data)
			if err != nil {
				t.Fatalf("render: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %q want %q", got, tc.want)
			}
		})
	}
}
