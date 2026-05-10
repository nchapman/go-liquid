package liquid

import (
	"fmt"
	"io"
	"strings"
	"testing"
)

// renderFunc adapts a plain func to the TagRenderer interface so each test
// can inline its tag behavior without declaring a one-method type.
type renderFunc func(w io.Writer, ctx TagContext) error

func (f renderFunc) Render(w io.Writer, ctx TagContext) error { return f(w, ctx) }

// TestRegisterTag_Inline registers an inline tag and verifies the parser
// and renderer cooperate end-to-end, including markup pass-through.
func TestRegisterTag_Inline(t *testing.T) {
	const name = "shout_inline_test"
	t.Cleanup(func() { delete(Default().inlineTags, name) })

	RegisterTag(name, func(markup string) (TagRenderer, error) {
		msg := strings.TrimSpace(markup)
		return renderFunc(func(w io.Writer, _ TagContext) error {
			_, err := io.WriteString(w, strings.ToUpper(msg)+"!")
			return err
		}), nil
	})

	got, err := Render(`a {% shout_inline_test hello world %} b`, nil)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if want := "a HELLO WORLD! b"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestRegisterBlock_BodyAndScope exercises the block-tag path: body
// rendering, scope reads via Get, and PushScope isolation.
func TestRegisterBlock_BodyAndScope(t *testing.T) {
	const name = "wrap_block_test"
	t.Cleanup(func() { delete(Default().blockTags, name) })

	RegisterBlock(name, func(markup string) (TagRenderer, error) {
		tag := strings.TrimSpace(markup)
		return renderFunc(func(w io.Writer, ctx TagContext) error {
			outer := ctx.Get("outer")
			fmt.Fprintf(w, "<%s outer=%v>", tag, outer)
			err := ctx.PushScope(func() error {
				ctx.Assign("inner", "scoped")
				return ctx.RenderBody(w)
			})
			if err != nil {
				return err
			}
			fmt.Fprintf(w, "</%s>", tag)
			return nil
		}), nil
	})

	src := `{% wrap_block_test box %}body inner={{ inner }}{% endwrap_block_test %}` +
		`|after inner={{ inner }}`
	got, err := Render(src, map[string]any{"outer": "OK"})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	want := "<box outer=OK>body inner=scoped</box>|after inner="
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestRegisterBlock_AssignVisibleAfterEnd confirms that Assign without
// PushScope writes into the caller's scope (escapes the block).
func TestRegisterBlock_AssignVisibleAfterEnd(t *testing.T) {
	const name = "leaky_block_test"
	t.Cleanup(func() { delete(Default().blockTags, name) })

	RegisterBlock(name, func(_ string) (TagRenderer, error) {
		return renderFunc(func(w io.Writer, ctx TagContext) error {
			ctx.Assign("leaked", "yes")
			return ctx.RenderBody(w)
		}), nil
	})

	got, err := Render(`{% leaky_block_test %}{% endleaky_block_test %}leaked={{ leaked }}`, nil)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if want := "leaked=yes"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestRegisterBlock_MismatchedEnd ensures an unrelated end-tag inside the
// body surfaces a parse error rather than silently closing the block.
func TestRegisterBlock_MismatchedEnd(t *testing.T) {
	const name = "needs_close_test"
	t.Cleanup(func() { delete(Default().blockTags, name) })

	RegisterBlock(name, func(_ string) (TagRenderer, error) {
		return renderFunc(func(w io.Writer, ctx TagContext) error { return ctx.RenderBody(w) }), nil
	})

	if _, err := Parse(`{% needs_close_test %}body{% endfor %}`); err == nil {
		t.Fatal("expected parse error for mismatched end tag, got nil")
	}
}

// TestRegisterTag_ParserErrorPropagates checks that an error from the
// user's TagParser is reported as a parse error, not a render error.
func TestRegisterTag_ParserErrorPropagates(t *testing.T) {
	const name = "rejects_test"
	t.Cleanup(func() { delete(Default().inlineTags, name) })

	RegisterTag(name, func(markup string) (TagRenderer, error) {
		return nil, fmt.Errorf("nope: %q", strings.TrimSpace(markup))
	})

	_, err := Parse(`{% rejects_test bad input %}`)
	if err == nil {
		t.Fatal("expected parse error, got nil")
	}
	if !strings.Contains(err.Error(), "nope") {
		t.Errorf("error %q does not mention plugin message", err)
	}
}

// TestRegisterTag_BuiltinNamePanics guards the documented contract that
// users cannot shadow a built-in tag.
func TestRegisterTag_BuiltinNamePanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic when registering a built-in tag name")
		}
	}()
	RegisterTag("if", func(string) (TagRenderer, error) { return nil, nil })
}

// TestRegisterTag_TrimRespected verifies whitespace control still works
// when a custom tag closes with -%}.
func TestRegisterTag_TrimRespected(t *testing.T) {
	const name = "trim_test"
	t.Cleanup(func() { delete(Default().inlineTags, name) })

	RegisterTag(name, func(string) (TagRenderer, error) {
		return renderFunc(func(w io.Writer, _ TagContext) error {
			_, err := io.WriteString(w, "X")
			return err
		}), nil
	})

	got, err := Render("a {% trim_test -%}   b", nil)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if want := "a Xb"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestRegisterBlock_Unterminated guards the EOF-before-end case: a block
// opened but never closed must surface a parse error mentioning the
// expected end tag.
func TestRegisterBlock_Unterminated(t *testing.T) {
	const name = "needs_eof_test"
	t.Cleanup(func() { delete(Default().blockTags, name) })

	RegisterBlock(name, func(_ string) (TagRenderer, error) {
		return renderFunc(func(w io.Writer, ctx TagContext) error { return ctx.RenderBody(w) }), nil
	})

	_, err := Parse(`{% needs_eof_test %}body and then nothing`)
	if err == nil {
		t.Fatal("expected parse error for unterminated block, got nil")
	}
	if !strings.Contains(err.Error(), "endneeds_eof_test") {
		t.Errorf("error %q does not mention expected end tag", err)
	}
}

// TestRegisterBlock_TrimAroundTags exercises whitespace control on both
// the opening and closing tags of a custom block.
func TestRegisterBlock_TrimAroundTags(t *testing.T) {
	const name = "trim_block_test"
	t.Cleanup(func() { delete(Default().blockTags, name) })

	RegisterBlock(name, func(_ string) (TagRenderer, error) {
		return renderFunc(func(w io.Writer, ctx TagContext) error { return ctx.RenderBody(w) }), nil
	})

	cases := []struct {
		src, want string
	}{
		{"a   {%- trim_block_test %}body{% endtrim_block_test %}   b", "abody   b"},
		{"a   {% trim_block_test -%}   body{% endtrim_block_test %}   b", "a   body   b"},
		{"a   {% trim_block_test %}body   {%- endtrim_block_test %}   b", "a   body   b"},
		{"a   {% trim_block_test %}body{% endtrim_block_test -%}   b", "a   bodyb"},
	}
	for _, tc := range cases {
		got, err := Render(tc.src, nil)
		if err != nil {
			t.Fatalf("render %q: %v", tc.src, err)
		}
		if got != tc.want {
			t.Errorf("\nsrc:  %q\ngot:  %q\nwant: %q", tc.src, got, tc.want)
		}
	}
}
