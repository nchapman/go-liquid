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

		// h is Ruby's alias for escape.
		{"h aliases escape", `{{ s | h }}`, "&lt;b&gt;hi&lt;/b&gt;", map[string]any{"s": "<b>hi</b>"}},

		// compact: "prop" keeps items whose prop is non-nil.
		{"compact with property", `{{ items | compact: "name" | size }}`, "2", map[string]any{
			"items": []any{
				map[string]any{"name": "a", "n": 1},
				map[string]any{"n": 2},
				map[string]any{"name": "c", "n": 3},
			},
		}},

		// strftime directives previously unmapped (%j year-day, %w weekday,
		// %s unix epoch, %x/%X locale shorthands, %U/%W week numbers).
		// Reference date: 2024-03-15 = Friday, day 75 of the year.
		{"strftime %j", `{{ d | date: "%j" }}`, "075", map[string]any{"d": "2024-03-15T12:00:00Z"}},
		{"strftime %w", `{{ d | date: "%w" }}`, "5", map[string]any{"d": "2024-03-15T12:00:00Z"}},
		{"strftime %s", `{{ d | date: "%s" }}`, "1710504000", map[string]any{"d": "2024-03-15T12:00:00Z"}},
		{"strftime %x", `{{ d | date: "%x" }}`, "03/15/24", map[string]any{"d": "2024-03-15T12:00:00Z"}},
		{"strftime %X", `{{ d | date: "%X" }}`, "12:00:00", map[string]any{"d": "2024-03-15T12:00:00Z"}},
		{"strftime %F shorthand", `{{ d | date: "%F" }}`, "2024-03-15", map[string]any{"d": "2024-03-15T12:00:00Z"}},

		// Ruby strftime flag modifiers: `-` strips zero-padding, `_` pads
		// with spaces, `0` forces zero-padding. The `%-d` form is what most
		// Jekyll/Shopify templates use to render "Jan 2, 2006".
		{"strftime %-d no pad", `{{ d | date: "%-d" }}`, "5", map[string]any{"d": "2024-03-05T00:00:00Z"}},
		{"strftime %-m no pad", `{{ d | date: "%-m" }}`, "3", map[string]any{"d": "2024-03-05T00:00:00Z"}},
		{"strftime %_d space pad", `{{ d | date: "%_d" }}`, " 5", map[string]any{"d": "2024-03-05T00:00:00Z"}},
		{"strftime %0e zero pad overrides default space", `{{ d | date: "%0e" }}`, "05", map[string]any{"d": "2024-03-05T00:00:00Z"}},
		{"strftime %-Y no pad year", `{{ d | date: "%-Y" }}`, "2024", map[string]any{"d": "2024-03-05T00:00:00Z"}},
		{"strftime composite Jan 2, 2006", `{{ d | date: "%b %-d, %Y" }}`, "Mar 5, 2024", map[string]any{"d": "2024-03-05T00:00:00Z"}},

		// Ruby's Utils.to_date accepts integer/numeric-string Unix
		// timestamps and the literal strings "now"/"today".
		{"date int unix timestamp", `{{ d | date: "%Y-%m-%d" }}`, "2024-03-15", map[string]any{"d": int64(1710504000)}},
		{"date numeric string timestamp", `{{ d | date: "%Y-%m-%d" }}`, "2024-03-15", map[string]any{"d": "1710504000"}},

		// Scientific notation in number literals (Ruby Liquid accepts via
		// the underlying Float coercion). Previously these would lex as
		// identifiers and silently render as variable names.
		{"sci notation int exponent", `{{ 1e3 }}`, "1000", nil},
		{"sci notation negative exponent", `{{ 1.5e-2 }}`, "0.015", nil},
		{"sci notation explicit positive", `{{ 2E+2 }}`, "200", nil},

		// Ruby coerces hash keys for integer subscript lookup, so
		// `obj[1]` finds the entry stored under the string key "1".
		{"hash int-key coercion", `{{ h[1] }}`, "v", map[string]any{"h": map[string]any{"1": "v"}}},

		// Ruby's `truncate` and `escape` pass nil through unchanged
		// rather than coercing to "".
		{"truncate nil passthrough", `[{{ x | truncate: 5 }}]`, "[]", nil},
		{"escape nil passthrough", `[{{ x | escape }}]`, "[]", nil},

		// Backslash escapes are NOT interpreted in string literals
		// (Ruby's lexer takes content verbatim). A template author who
		// wants a real newline must put one in the source.
		{"string literal keeps backslash escapes", `{{ "a\nb" }}`, `a\nb`, nil},

		// truncatewords clamps to 1 word minimum (Ruby clamps `words <= 0`).
		{"truncatewords clamps zero", `{{ s | truncatewords: 0 }}`, "one...", map[string]any{"s": "one two three"}},
		{"truncatewords clamps negative", `{{ s | truncatewords: -5 }}`, "one...", map[string]any{"s": "one two three"}},
		{"truncatewords nil passthrough", `[{{ x | truncatewords: 2 }}]`, "[]", nil},

		// Sort: nil values sort last (Ruby's nil_safe_compare).
		{"sort nil-last", `{{ a | sort | join: "," }}`, "1,2,3,", map[string]any{"a": []any{2, nil, 1, 3}}},
		{"sort_natural nil-last", `{{ a | sort_natural | join: "," }}`, "alpha,beta,", map[string]any{"a": []any{"beta", nil, "alpha"}}},

		// for ... offset: continue (Shopify pagination idiom): a second
		// for-tag with offset:continue resumes from where the previous
		// for-tag stopped within the same Render.
		{
			"offset:continue resumes after limited loop",
			`{% for i in items limit: 3 %}{{ i }}{% endfor %}|{% for i in items offset: continue %}{{ i }}{% endfor %}`,
			"123|45678",
			map[string]any{"items": []any{1, 2, 3, 4, 5, 6, 7, 8}},
		},
		{
			"offset:continue with explicit limit on resume",
			`{% for i in items limit: 2 %}{{ i }}{% endfor %}-{% for i in items offset: continue limit: 3 %}{{ i }}{% endfor %}`,
			"12-345",
			map[string]any{"items": []any{1, 2, 3, 4, 5, 6, 7, 8}},
		},
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
