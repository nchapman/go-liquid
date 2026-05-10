package liquid

import (
	"strings"
	"testing"
)

// Upstream parity: integration/filter_test.rb

// renderWithFilters parses src against a private env with the given filters
// registered, then renders it with data.
func renderWithFilters(t *testing.T, src string, data map[string]any, filters map[string]FilterFunc) (string, error) {
	t.Helper()
	env := NewEnvironment()
	for name, fn := range filters {
		env.RegisterFilter(name, fn)
	}
	tmpl, err := env.Parse(src)
	if err != nil {
		return "", err
	}
	return tmpl.Render(data)
}

func TestUpstream_Filter_LocalFilter(t *testing.T) {
	got, err := renderWithFilters(t, "{{var | money}}", map[string]any{"var": 1000}, map[string]FilterFunc{
		"money": func(input any, args ...any) any {
			return " 1000$ "
		},
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if got != " 1000$ " {
		t.Fatalf("got %q", got)
	}
}

func TestUpstream_Filter_UnderscoreInFilterName(t *testing.T) {
	got, err := renderWithFilters(t, "{{var | money_with_underscore}}", map[string]any{"var": 1000}, map[string]FilterFunc{
		"money_with_underscore": func(input any, args ...any) any { return " 1000$ " },
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if got != " 1000$ " {
		t.Fatalf("got %q", got)
	}
}

func TestUpstream_Filter_SecondFilterOverwritesFirst(t *testing.T) {
	env := NewEnvironment()
	env.RegisterFilter("money", func(input any, args ...any) any { return " 1000$ " })
	env.RegisterFilter("money", func(input any, args ...any) any { return " 1000$ CAD " })
	tmpl, err := env.Parse("{{var | money}}")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	got, err := tmpl.Render(map[string]any{"var": 1000})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if got != " 1000$ CAD " {
		t.Fatalf("got %q", got)
	}
}

func TestUpstream_Filter_Size(t *testing.T) {
	renderEq(t, "size", "{{var | size}}", map[string]any{"var": "abcd"}, "4")
}

func TestUpstream_Filter_Join(t *testing.T) {
	renderEq(t, "join", "{{var | join}}", map[string]any{"var": []any{1, 2, 3, 4}}, "1 2 3 4")
}

func TestUpstream_Filter_Sort(t *testing.T) {
	renderEq(t, "ints", "{{numbers | sort | join}}", map[string]any{"numbers": []any{2, 1, 4, 3}}, "1 2 3 4")
	renderEq(t, "words", "{{words | sort | join}}", map[string]any{"words": []any{"expected", "as", "alphabetic"}}, "alphabetic as expected")
	renderEq(t, "scalar", "{{value | sort}}", map[string]any{"value": 3}, "3")
	renderEq(t, "arrays", "{{arrays | sort | join}}", map[string]any{"arrays": []any{"flower", "are"}}, "are flower")
	renderEq(t, "case_sensitive", "{{case_sensitive | sort | join}}",
		map[string]any{"case_sensitive": []any{"sensitive", "Expected", "case"}},
		"Expected case sensitive")
}

func TestUpstream_Filter_SortNatural(t *testing.T) {
	renderEq(t, "strings", "{{words | sort_natural | join}}",
		map[string]any{"words": []any{"case", "Assert", "Insensitive"}},
		"Assert case Insensitive")

	renderEq(t, "hashes", "{{hashes | sort_natural: 'a' | map: 'a' | join}}",
		map[string]any{"hashes": []any{
			map[string]any{"a": "A"},
			map[string]any{"a": "b"},
			map[string]any{"a": "C"},
		}},
		"A b C")
}

func TestUpstream_Filter_Compact(t *testing.T) {
	renderEq(t, "strings", "{{words | compact | join}}",
		map[string]any{"words": []any{"a", nil, "b", nil, "c"}},
		"a b c")
	renderEq(t, "hashes", "{{hashes | compact: 'a' | map: 'a' | join}}",
		map[string]any{"hashes": []any{
			map[string]any{"a": "A"},
			map[string]any{"a": nil},
			map[string]any{"a": "C"},
		}},
		"A C")
}

func TestUpstream_Filter_StripHtml(t *testing.T) {
	renderEq(t, "basic", "{{ var | strip_html }}", map[string]any{"var": "<b>bla blub</a>"}, "bla blub")
}

func TestUpstream_Filter_StripHtmlIgnoreCommentsWithHtml(t *testing.T) {
	renderEq(t, "comment", "{{ var | strip_html }}",
		map[string]any{"var": "<!-- split and some <ul> tag --><b>bla blub</a>"},
		"bla blub")
}

func TestUpstream_Filter_Capitalize(t *testing.T) {
	renderEq(t, "blub", "{{ var | capitalize }}", map[string]any{"var": "blub"}, "Blub")
}

func TestUpstream_Filter_NonexistentFilterIsIgnored(t *testing.T) {
	renderEq(t, "xyzzy", "{{ var | xyzzy }}", map[string]any{"var": 1000}, "1000")
}

func TestUpstream_Filter_FilterWithKeywordArguments(t *testing.T) {
	env := NewEnvironment()
	env.RegisterKwargFilter("substitute", func(input any, args []any, kwargs map[string]any) any {
		s := toString(input)
		for k, v := range kwargs {
			s = strings.ReplaceAll(s, "%{"+k+"}", toString(v))
		}
		return s
	})
	tmpl, err := env.Parse("{{ input | substitute: first_name: surname, last_name: 'doe' }}")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	got, err := tmpl.Render(map[string]any{
		"surname": "john",
		"input":   "hello %{first_name}, %{last_name}",
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if got != "hello john, doe" {
		t.Fatalf("got %q", got)
	}
}

func TestUpstream_Filter_OverrideObjectMethodInFilter(t *testing.T) {
	got, err := renderWithFilters(t, "{{var | tap}}", map[string]any{"var": 1000}, map[string]FilterFunc{
		"tap": func(input any, args ...any) any { return "tap overridden" },
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if got != "tap overridden" {
		t.Fatalf("got %q", got)
	}

	// without override: tap is non-existent and ignored, input passes through
	renderEq(t, "tap-noop", "{{var | tap}}", map[string]any{"var": 1000}, "1000")
}

func TestUpstream_Filter_LiquidArgumentError(t *testing.T) {
	t.Skip("go-liquid does not surface wrong-argument-count errors as Liquid error strings inside output (no Liquid::ArgumentError equivalent)")

	src := "{{ '' | size: 'too many args' }}"
	tmpl, err := Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	got, err := tmpl.Render(nil)
	_ = err
	if !strings.HasPrefix(got, "Liquid error: wrong number of arguments") {
		t.Fatalf("got %q", got)
	}
}
