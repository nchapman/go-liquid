package liquid

// Ported from upstream Ruby Liquid test/integration/blank_test.rb.

import (
	"io"
	"strings"
	"testing"
)

// blankN matches Ruby's N=10 — the inner-for repetition count.
const blankN = 10

func wrapInFor(body string) string {
	return "{% for i in (1.." + itoaBlank(blankN) + ") %}" + body + "{% endfor %}"
}
func wrapInIf(body string) string {
	return "{% if true %}" + body + "{% endif %}"
}
func wrapBlank(body string) string {
	return wrapInFor(body) + wrapInIf(body)
}
func itoaBlank(n int) string {
	if n == 0 {
		return "0"
	}
	var s []byte
	for n > 0 {
		s = append([]byte{byte('0' + n%10)}, s...)
		n /= 10
	}
	return string(s)
}

func TestUpstreamBlank(t *testing.T) {
	t.Run("test_new_tags_are_not_blank_by_default", func(t *testing.T) {
		env := NewEnvironment()
		env.RegisterTag("foobar", func(markup string) (TagRenderer, error) {
			return foobarBlankTag{}, nil
		})
		out, err := env.Render(wrapInFor("{% foobar %}"), nil)
		if err != nil {
			t.Fatalf("render error: %v", err)
		}
		if want := strings.Repeat(" ", blankN); out != want {
			t.Errorf("\n want %q\n got  %q", want, out)
		}
	})

	t.Run("test_loops_are_blank", func(t *testing.T) {
		renderEq(t, "loops", wrapInFor(" "), nil, "")
	})

	t.Run("test_if_else_are_blank", func(t *testing.T) {
		renderEq(t, "if-elsif-else", "{% if true %} {% elsif false %} {% else %} {% endif %}", nil, "")
	})

	t.Run("test_unless_is_blank", func(t *testing.T) {
		renderEq(t, "unless", wrapBlank("{% unless true %} {% endunless %}"), nil, "")
	})

	t.Run("test_mark_as_blank_only_during_parsing", func(t *testing.T) {
		want := strings.Repeat(" ", blankN+1)
		renderEq(t, "mark-blank", wrapBlank(" {% if false %} this never happens, but still, this block is not blank {% endif %}"), nil, want)
	})

	t.Run("test_comments_are_blank", func(t *testing.T) {
		renderEq(t, "comments", wrapBlank(" {% comment %} whatever {% endcomment %} "), nil, "")
	})

	t.Run("test_captures_are_blank", func(t *testing.T) {
		renderEq(t, "captures", wrapBlank(" {% capture foo %} whatever {% endcapture %} "), nil, "")
	})

	t.Run("test_nested_blocks_are_blank_but_only_if_all_children_are", func(t *testing.T) {
		renderEq(t, "nested-blank", wrapBlank(wrapBlank(" ")), nil, "")

		nested := wrapBlank("{% if true %} {% comment %} this is blank {% endcomment %} {% endif %}\n      {% if true %} but this is not {% endif %}")
		want := strings.Repeat("\n       but this is not ", blankN+1)
		renderEq(t, "nested-mixed", nested, nil, want)
	})

	t.Run("test_assigns_are_blank", func(t *testing.T) {
		renderEq(t, "assigns", wrapBlank(` {% assign foo = "bar" %} `), nil, "")
	})

	t.Run("test_whitespace_is_blank", func(t *testing.T) {
		renderEq(t, "spaces", wrapBlank(" "), nil, "")
		renderEq(t, "tabs", wrapBlank("\t"), nil, "")
	})

	t.Run("test_whitespace_is_not_blank_if_other_stuff_is_present", func(t *testing.T) {
		body := "     x "
		want := strings.Repeat(body, blankN+1)
		renderEq(t, "not-blank", wrapBlank(body), nil, want)
	})

	t.Run("test_increment_is_not_blank", func(t *testing.T) {
		want := strings.Repeat(" 0", 2*(blankN+1))
		renderEq(t, "increment", wrapBlank("{% assign foo = 0 %} {% increment foo %} {% decrement foo %}"), nil, want)
	})

	t.Run("test_cycle_is_not_blank", func(t *testing.T) {
		want := strings.Repeat("  ", (blankN+1)/2) + " "
		renderEq(t, "cycle", wrapBlank("{% cycle ' ', ' ' %}"), nil, want)
	})

	t.Run("test_raw_is_not_blank", func(t *testing.T) {
		want := strings.Repeat("  ", blankN+1)
		renderEq(t, "raw", wrapBlank(" {% raw %} {% endraw %}"), nil, want)
	})

	t.Run("test_include_is_blank", func(t *testing.T) {
		loader := MapLoader{
			"foobar":   "foobar",
			" foobar ": " foobar ",
			" ":        " ",
		}
		env := NewEnvironment().WithLoader(loader)

		render := func(src string) string {
			out, err := env.Render(src, nil)
			if err != nil {
				t.Fatalf("render error for %q: %v", src, err)
			}
			return out
		}

		if got, want := render(wrapBlank("{% include 'foobar' %}")), strings.Repeat("foobar", blankN+1); got != want {
			t.Errorf("foobar:\n want %q\n got  %q", want, got)
		}
		if got, want := render(wrapBlank("{% include ' foobar ' %}")), strings.Repeat(" foobar ", blankN+1); got != want {
			t.Errorf("space-padded:\n want %q\n got  %q", want, got)
		}
		if got, want := render(wrapBlank(" {% include ' ' %} ")), strings.Repeat("   ", blankN+1); got != want {
			t.Errorf("space-only:\n want %q\n got  %q", want, got)
		}
	})

	t.Run("test_case_is_blank", func(t *testing.T) {
		renderEq(t, "case-blank-bar", wrapBlank(" {% assign foo = 'bar' %} {% case foo %} {% when 'bar' %} {% when 'whatever' %} {% else %} {% endcase %} "), nil, "")
		renderEq(t, "case-blank-else", wrapBlank(" {% assign foo = 'else' %} {% case foo %} {% when 'bar' %} {% when 'whatever' %} {% else %} {% endcase %} "), nil, "")
		want := strings.Repeat("   x  ", blankN+1)
		renderEq(t, "case-not-blank", wrapBlank(" {% assign foo = 'else' %} {% case foo %} {% when 'bar' %} {% when 'whatever' %} {% else %} x {% endcase %} "), nil, want)
	})
}

type foobarBlankTag struct{}

func (foobarBlankTag) Render(w io.Writer, _ TagContext) error {
	_, err := io.WriteString(w, " ")
	return err
}
