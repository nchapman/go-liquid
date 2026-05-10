package liquid

// Ported from upstream Ruby Liquid test/integration/tags/raw_tag_test.rb.

import (
	"strings"
	"testing"
)

func TestUpstreamRawTag(t *testing.T) {
	t.Run("test_tag_in_raw", func(t *testing.T) {
		renderEq(t, "tag-in-raw",
			"{% raw %}{% comment %} test {% endcomment %}{% endraw %}",
			nil, "{% comment %} test {% endcomment %}")
	})

	t.Run("test_output_in_raw", func(t *testing.T) {
		renderEq(t, "trim", "> {%- raw -%}{{ test }}{%- endraw -%} <", nil, ">{{ test }}<")
		renderEq(t, "trim-mixed-1", "> {%- raw -%} inner {%- endraw %} <", nil, "> inner  <")
		renderEq(t, "trim-mixed-2", "> {%- raw -%} inner {%- endraw -%} <", nil, "> inner <")
		renderEq(t, "brace-brace", "{% raw %}{{% endraw %}Hello{% raw %}}{% endraw %}", nil, "{Hello}")
	})

	t.Run("test_open_tag_in_raw", func(t *testing.T) {
		cases := [][2]string{
			{"{% raw %} Foobar {% invalid {% endraw %}", " Foobar {% invalid "},
			{"{% raw %} Foobar invalid %} {% endraw %}", " Foobar invalid %} "},
			{"{% raw %} Foobar {{ invalid {% endraw %}", " Foobar {{ invalid "},
			{"{% raw %} Foobar invalid }} {% endraw %}", " Foobar invalid }} "},
			{"{% raw %} Foobar {% invalid {% {% endraw {% endraw %}", " Foobar {% invalid {% {% endraw "},
			{"{% raw %} Foobar {% {% {% {% endraw %}", " Foobar {% {% {% "},
			{"{% raw %} test {% raw %} {% {% endraw %}endraw %}", " test {% raw %} {% endraw %}"},
			{"{% raw %} Foobar {{ invalid {% endraw %}{{ 1 }}", " Foobar {{ invalid 1"},
			{"{% raw %} Foobar {% foo {% bar %}{% endraw %}", " Foobar {% foo {% bar %}"},
		}
		for i, c := range cases {
			out, err := Render(c[0], nil)
			if err != nil {
				t.Errorf("case %d: unexpected error: %v\nsrc: %s", i, err, c[0])
				continue
			}
			if out != c[1] {
				t.Errorf("case %d:\n src: %s\n want %q\n got  %q", i, c[0], c[1], out)
			}
		}
	})

	t.Run("test_invalid_raw_unclosed", func(t *testing.T) {
		_, err := Parse("{% raw %} foo")
		if err == nil {
			t.Fatal("expected error for unclosed raw")
		}
		if !strings.Contains(strings.ToLower(err.Error()), "raw") &&
			!strings.Contains(strings.ToLower(err.Error()), "never closed") &&
			!strings.Contains(strings.ToLower(err.Error()), "unterminated") {
			t.Errorf("error should reference raw/closure; got %q", err.Error())
		}
	})

	t.Run("test_invalid_raw_syntax", func(t *testing.T) {
		// `{% raw }` is malformed: the bang-brace mismatch should be a parse error
		// (Ruby returns "Valid syntax: …").
		if _, err := Parse("{% raw } foo {% endraw %}"); err == nil {
			t.Fatal("expected error for `{% raw }` syntax")
		}
		if _, err := Parse("{% raw } foo %}{% endraw %}"); err == nil {
			t.Fatal("expected error for `{% raw }` syntax (2)")
		}
	})
}
