package liquid

// Ported from upstream Ruby Liquid test/integration/output_test.rb.

import (
	"testing"
)

func TestUpstreamOutput(t *testing.T) {
	assigns := map[string]any{
		"car": map[string]any{"bmw": "good", "gm": "bad"},
	}

	t.Run("test_variable", func(t *testing.T) {
		renderEq(t, "var", " {{best_cars}} ", map[string]any{"best_cars": "bmw"}, " bmw ")
	})

	t.Run("test_variable_traversing_with_two_brackets", func(t *testing.T) {
		data := map[string]any{
			"site":    map[string]any{"data": map[string]any{"menu": map[string]any{"foo": map[string]any{"bar": "it works!"}}}},
			"include": map[string]any{"menu": "foo", "locale": "bar"},
		}
		renderEq(t, "two-brk", "{{ site.data.menu[include.menu][include.locale] }}", data, "it works!")
	})

	t.Run("test_variable_traversing", func(t *testing.T) {
		renderEq(t, "trav", " {{car.bmw}} {{car.gm}} {{car.bmw}} ", assigns, " good bad good ")
	})
}

// TestUpstreamOutputCustomFilters covers the FunnyFilter battery from
// output_test.rb. Filter registration mutates the default environment;
// this test must not run in parallel with other tests that register
// filters or rely on un-overridden built-ins.
//
//nolint:tparallel
func TestUpstreamOutputCustomFilters(t *testing.T) {
	d := Default()
	old := d.filters
	d.filters = make(map[string]Filter, len(old))
	for k, v := range old {
		d.filters[k] = v
	}
	defer func() { d.filters = old }()

	RegisterFilter("make_funny", func(_ any, _ ...any) any { return "LOL" })
	RegisterFilter("cite_funny", func(in any, _ ...any) any { return "LOL: " + toString(in) })
	RegisterFilter("add_smiley", func(in any, args ...any) any {
		smiley := ":-)"
		if len(args) > 0 {
			smiley = toString(args[0])
		}
		return toString(in) + " " + smiley
	})
	RegisterFilter("add_tag", func(in any, args ...any) any {
		tag, id := "p", "foo"
		if len(args) > 0 {
			tag = toString(args[0])
		}
		if len(args) > 1 {
			id = toString(args[1])
		}
		return "<" + tag + ` id="` + id + `">` + toString(in) + "</" + tag + ">"
	})
	RegisterFilter("paragraph", func(in any, _ ...any) any { return "<p>" + toString(in) + "</p>" })
	RegisterFilter("link_to", func(name any, args ...any) any {
		url := ""
		if len(args) > 0 {
			url = toString(args[0])
		}
		return `<a href="` + url + `">` + toString(name) + `</a>`
	})

	assigns := map[string]any{
		"car": map[string]any{"bmw": "good", "gm": "bad"},
	}

	cases := []struct{ name, src, want string }{
		{"piping", " {{ car.gm | make_funny }} ", " LOL "},
		{"piping_with_input", " {{ car.gm | cite_funny }} ", " LOL: bad "},
		{"piping_with_args", " {{ car.gm | add_smiley : ':-(' }} ", " bad :-( "},
		{"piping_no_args", " {{ car.gm | add_smiley }} ", " bad :-) "},
		{"multi_piping_with_args", " {{ car.gm | add_smiley : ':-(' | add_smiley : ':-('}} ", " bad :-( :-( "},
		{"piping_with_multiple_args", " {{ car.gm | add_tag : 'span', 'bar'}} ", ` <span id="bar">bad</span> `},
		{"piping_with_variable_args", " {{ car.gm | add_tag : 'span', car.bmw}} ", ` <span id="good">bad</span> `},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			renderEq(t, c.name, c.src, assigns, c.want)
		})
	}

	t.Run("multiple_pipings", func(t *testing.T) {
		renderEq(t, "multi", " {{ best_cars | cite_funny | paragraph }} ", map[string]any{"best_cars": "bmw"}, " <p>LOL: bmw</p> ")
	})

	t.Run("link_to", func(t *testing.T) {
		renderEq(t, "link", " {{ 'Typo' | link_to: 'http://typo.leetsoft.com' }} ", assigns, ` <a href="http://typo.leetsoft.com">Typo</a> `)
	})
}
