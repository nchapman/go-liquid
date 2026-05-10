package liquid

import "testing"

// Upstream parity: integration/filter_kwarg_test.rb

func TestUpstream_FilterKwarg_CanParseDataKwargs(t *testing.T) {
	t.Skip("go-liquid lexer does not allow '-' in identifiers; kebab-case kwarg keys (e.g. data-src:) require lexer support")

	env := NewEnvironment()
	env.RegisterKwargFilter("html_tag", func(input any, args []any, kwargs map[string]any) any {
		// emit in stable insertion order — but go map iteration is unstable,
		// so this test depends on ordered kwargs. The body is retained for
		// when both kebab-case kwarg keys and ordered kwargs are supported.
		out := ""
		first := true
		for k, v := range kwargs {
			if !first {
				out += " "
			}
			first = false
			out += k + "='" + toString(v) + "'"
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
