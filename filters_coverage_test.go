package liquid

import (
	"strings"
	"testing"
)

// TestFiltersComprehensive covers filters that lack direct unit tests in
// liquid_test.go. Spec fixtures exercise many of these end-to-end, but
// these table tests pin down each filter's edge cases (nil input, empty
// input, unicode, type coercion) explicitly so a regression in a single
// filter doesn't have to be diagnosed from a noisy fixture diff.
func TestFiltersComprehensive(t *testing.T) {
	tests := []struct {
		name     string
		template string
		data     map[string]any
		expected string
	}{
		// Math
		{"abs negative int", "{{ n | abs }}", map[string]any{"n": -7}, "7"},
		{"abs negative float", "{{ n | abs }}", map[string]any{"n": -7.25}, "7.25"},
		{"abs nil", "{{ missing | abs }}", nil, "0"},
		{"at_least below", "{{ n | at_least: 5 }}", map[string]any{"n": 3}, "5"},
		{"at_least above", "{{ n | at_least: 5 }}", map[string]any{"n": 9}, "9"},
		{"at_most above", "{{ n | at_most: 5 }}", map[string]any{"n": 9}, "5"},
		{"at_most below", "{{ n | at_most: 5 }}", map[string]any{"n": 2}, "2"},
		{"ceil int", "{{ 3 | ceil }}", nil, "3"},
		{"ceil float up", "{{ 3.1 | ceil }}", nil, "4"},
		{"ceil negative", "{{ x | ceil }}", map[string]any{"x": -3.7}, "-3"},
		{"floor float", "{{ 3.9 | floor }}", nil, "3"},
		{"floor negative", "{{ x | floor }}", map[string]any{"x": -3.1}, "-4"},
		{"round default", "{{ 3.5 | round }}", nil, "4"},
		{"round precision", "{{ x | round: 2 }}", map[string]any{"x": 3.14159}, "3.14"},
		{"divided_by int", "{{ 10 | divided_by: 3 }}", nil, "3"},
		// Documented intentional divergence from Shopify: divisor decides
		// float-ness (so JSON-unmarshalled ints arriving as float64 don't
		// silently switch division mode). See filterDividedBy comment.
		// Float-mode kicks in only when the divisor has a non-zero fractional
		// part — `divided_by: 3.0` still does integer division, by design.
		{"divided_by fractional divisor", "{{ 10 | divided_by: 2.5 }}", nil, "4"},
		{"divided_by float dividend ignored", "{{ 10.0 | divided_by: 3 }}", nil, "3"},
		{"modulo", "{{ 10 | modulo: 3 }}", nil, "1"},
		{"times string-coerced", "{{ s | times: 3 }}", map[string]any{"s": "4"}, "12"},

		// String
		{"capitalize", "{{ s | capitalize }}", map[string]any{"s": "hello world"}, "Hello world"},
		{"capitalize empty", "{{ s | capitalize }}", map[string]any{"s": ""}, ""},
		{"lstrip", "{{ s | lstrip }}", map[string]any{"s": "  hi  "}, "hi  "},
		{"rstrip", "{{ s | rstrip }}", map[string]any{"s": "  hi  "}, "  hi"},
		{"prepend", `{{ s | prepend: ">> " }}`, map[string]any{"s": "x"}, ">> x"},
		{"remove", `{{ s | remove: "ab" }}`, map[string]any{"s": "abxabz"}, "xz"},
		{"remove_first", `{{ s | remove_first: "ab" }}`, map[string]any{"s": "abxabz"}, "xabz"},
		{"remove_last", `{{ s | remove_last: "ab" }}`, map[string]any{"s": "abxabz"}, "abxz"},
		{"replace", `{{ s | replace: "a", "X" }}`, map[string]any{"s": "aaba"}, "XXbX"},
		{"replace_first", `{{ s | replace_first: "a", "X" }}`, map[string]any{"s": "aaba"}, "Xaba"},
		{"replace_last", `{{ s | replace_last: "a", "X" }}`, map[string]any{"s": "aaba"}, "aabX"},
		{"truncate default", `{{ s | truncate: 5 }}`, map[string]any{"s": "Hello world"}, "He..."},
		{"truncate custom suffix", `{{ s | truncate: 8, "..!" }}`, map[string]any{"s": "Hello world"}, "Hello..!"},
		{"truncate shorter than length", `{{ s | truncate: 50 }}`, map[string]any{"s": "Hi"}, "Hi"},
		{"truncatewords", `{{ s | truncatewords: 2 }}`, map[string]any{"s": "one two three four"}, "one two..."},
		{"strip_html", `{{ s | strip_html }}`, map[string]any{"s": "a <b>B</b> c<script>x</script>"}, "a B c"},
		{"strip_newlines", `{{ s | strip_newlines }}`, map[string]any{"s": "a\nb\r\nc"}, "abc"},
		{"newline_to_br", `{{ s | newline_to_br }}`, map[string]any{"s": "a\nb"}, "a<br />\nb"},
		{"escape_once idempotent", `{{ s | escape_once | escape_once }}`, map[string]any{"s": "<b>&"}, "&lt;b&gt;&amp;"},
		{"squish", `{{ s | squish }}`, map[string]any{"s": "  a   b\n\tc  "}, "a b c"},
		{"split then join", `{{ s | split: "," | join: "|" }}`, map[string]any{"s": "a,b,c"}, "a|b|c"},
		{"slice 1 arg", `{{ s | slice: 1 }}`, map[string]any{"s": "hello"}, "e"},
		{"slice 2 args", `{{ s | slice: 1, 3 }}`, map[string]any{"s": "hello"}, "ell"},
		{"slice negative", `{{ s | slice: -2, 2 }}`, map[string]any{"s": "hello"}, "lo"},
		{"slice out of range", `{{ s | slice: 99, 5 }}`, map[string]any{"s": "hi"}, ""},
		{"slice unicode", `{{ s | slice: 1, 2 }}`, map[string]any{"s": "héllo"}, "él"},
		// truncate counts runes, not bytes — multi-byte UTF-8 inputs slice
		// cleanly to the requested character count.
		{"truncate by runes", `{{ s | truncate: 5 }}`, map[string]any{"s": "café au lait"}, "ca..."},
		{"truncate by runes - emoji", `{{ s | truncate: 4 }}`, map[string]any{"s": "hi👋👋👋"}, "h..."},
		{"size of utf8 string", `{{ s | size }}`, map[string]any{"s": "héllo"}, "5"},
		{"first of utf8 string", `{{ s | first }}`, map[string]any{"s": "héllo"}, "h"},

		// URL
		{"url_encode", `{{ s | url_encode }}`, map[string]any{"s": "a b&c"}, "a+b%26c"},
		{"url_decode", `{{ s | url_decode }}`, map[string]any{"s": "a+b%26c"}, "a b&c"},

		// Base64
		{"base64_encode", `{{ s | base64_encode }}`, map[string]any{"s": "hello"}, "aGVsbG8="},
		{"base64_decode round-trip", `{{ "aGVsbG8=" | base64_decode }}`, nil, "hello"},
		{"base64_url_safe_encode", `{{ s | base64_url_safe_encode }}`, map[string]any{"s": "??>"}, "Pz8-"},
		{"base64_url_safe_decode", `{{ "Pz8-" | base64_url_safe_decode }}`, nil, "??>"},

		// Array
		{"compact removes nil", `{{ a | compact | join: "," }}`, map[string]any{"a": []any{1, nil, 2, nil, 3}}, "1,2,3"},
		{"sort numbers", `{{ a | sort | join: "," }}`, map[string]any{"a": []any{3, 1, 2}}, "1,2,3"},
		{"sort_natural mixed case", `{{ a | sort_natural | join: "," }}`, map[string]any{"a": []any{"Banana", "apple", "Cherry"}}, "apple,Banana,Cherry"},
		// Mixed numeric+string sort: Ruby raises ArgumentError on this; go-liquid
		// falls back to lexicographic string comparison, which puts digits (0x30+)
		// before lowercase letters (0x60+).
		{"sort mixed types", `{{ a | sort | join: "," }}`, map[string]any{"a": []any{"banana", 1, "apple", 2}}, "1,2,apple,banana"},
		{"sum ints", `{{ a | sum }}`, map[string]any{"a": []any{1, 2, 3, 4}}, "10"},
		{"sum mixed", `{{ a | sum }}`, map[string]any{"a": []any{1, 2.5, "3"}}, "6.5"},
		{"sum empty", `{{ a | sum }}`, map[string]any{"a": []any{}}, "0"},
		{"uniq", `{{ a | uniq | join: "," }}`, map[string]any{"a": []any{1, 2, 1, 3, 2}}, "1,2,3"},
		{"first nil", `[{{ a | first }}]`, map[string]any{"a": nil}, "[]"},
		{"last empty array", `[{{ a | last }}]`, map[string]any{"a": []any{}}, "[]"},
		{"size of string", `{{ s | size }}`, map[string]any{"s": "hello"}, "5"},
		{"size of nil", `{{ x | size }}`, nil, "0"},
		{"join nil", `[{{ a | join: "," }}]`, map[string]any{"a": nil}, "[]"},

		// Type coercion through filters
		{"plus string to int", `{{ s | plus: 1 }}`, map[string]any{"s": "41"}, "42"},
		{"minus float to int", `{{ s | minus: 2 }}`, map[string]any{"s": "5.5"}, "3.5"},

		// Default with empty/blank/falsy
		{"default empty string", `{{ s | default: "FB" }}`, map[string]any{"s": ""}, "FB"},
		{"default empty array", `{{ a | default: "FB" }}`, map[string]any{"a": []any{}}, "FB"},
		{"default false (positional)", `{{ b | default: "FB" }}`, map[string]any{"b": false}, "FB"},
		{"default false allow_false", `{{ b | default: "FB", allow_false: true }}`, map[string]any{"b": false}, "false"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Render(tt.template, tt.data)
			if err != nil {
				t.Fatalf("render: %v", err)
			}
			if got != tt.expected {
				t.Errorf("template %q: got %q, want %q", tt.template, got, tt.expected)
			}
		})
	}
}

// TestFilterErrorPropagation: filters that signal errors should produce
// RenderError, not just the bare error string.
func TestFilterErrorPropagation(t *testing.T) {
	cases := []struct {
		name     string
		template string
		want     string // substring of error
	}{
		{"divide by zero", `{{ 10 | divided_by: 0 }}`, "zero"},
		{"modulo by zero", `{{ 10 | modulo: 0 }}`, "zero"},
		{"base64 decode bad input", `{{ "@@" | base64_decode }}`, "base64"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Render(tc.template, nil)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.want)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error %q does not contain %q", err.Error(), tc.want)
			}
		})
	}
}
