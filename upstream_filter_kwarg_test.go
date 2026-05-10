package liquid

import (
	"sort"
	"testing"
)

// Upstream parity: integration/filter_kwarg_test.rb

func TestUpstream_FilterKwarg_CanParseDataKwargs(t *testing.T) {
	env := NewEnvironment()
	env.RegisterKwargFilter("html_tag", func(input any, args []any, kwargs map[string]any) any {
		// Sort keys so output is stable (Go map iteration is unordered).
		keys := make([]string, 0, len(kwargs))
		for k := range kwargs {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		out := ""
		for i, k := range keys {
			if i > 0 {
				out += " "
			}
			out += k + "='" + toString(kwargs[k]) + "'"
		}
		return out
	})

	tmpl, err := env.Parse("{{ 'img' | html_tag: data-src: 'src', data-widths: '100, 200' }}")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	got, err := tmpl.Render(nil)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if got != "data-src='src' data-widths='100, 200'" {
		t.Fatalf("got %q", got)
	}
}
