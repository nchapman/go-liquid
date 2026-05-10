package liquid

import (
	"errors"
	"fmt"
	"io"
	"testing"
)

// TestIncludeDisabledInsideRender mirrors Ruby Liquid's
// Tag::Disabler / Tag::Disableable mixins: {% render %} disables
// {% include %} inside the partial it renders, so a malicious template
// can't bypass render's isolated-scope guarantee by chaining through
// include.
func TestIncludeDisabledInsideRender(t *testing.T) {
	tmpl := MustParse(`{% render "outer" %}`).WithLoader(MapLoader{
		"outer": `outer:{% include "inner" %}`,
		"inner": `inner`,
	})
	_, err := tmpl.Render(nil)
	if err == nil {
		t.Fatal("expected disabled-tag error, got nil")
	}
	if !errors.Is(err, ErrDisabledTag) {
		t.Fatalf("expected errors.Is(err, ErrDisabledTag), got %v", err)
	}
}

// Disabling propagates across the entire render subtree, not just one
// partial deep: render → render → include must still error.
func TestIncludeDisabledAcrossNestedRenders(t *testing.T) {
	tmpl := MustParse(`{% render "a" %}`).WithLoader(MapLoader{
		"a": `{% render "b" %}`,
		"b": `{% include "c" %}`,
		"c": `c`,
	})
	_, err := tmpl.Render(nil)
	if err == nil || !errors.Is(err, ErrDisabledTag) {
		t.Fatalf("expected ErrDisabledTag through nested renders, got %v", err)
	}
}

// {% include %} from the top level (no enclosing render) still works.
func TestIncludeWorksOutsideRender(t *testing.T) {
	out, err := MustParse(`{% include "x" %}`).WithLoader(MapLoader{
		"x": `hello`,
	}).Render(nil)
	if err != nil {
		t.Fatal(err)
	}
	if out != "hello" {
		t.Errorf("got %q, want %q", out, "hello")
	}
}

// {% include %} nested inside another {% include %} is allowed — only
// {% render %} disables include. Matches Ruby (include.rb does not
// declare disable_tags).
func TestIncludeNestedInIncludeIsAllowed(t *testing.T) {
	out, err := MustParse(`{% include "a" %}`).WithLoader(MapLoader{
		"a": `a:{% include "b" %}`,
		"b": `b`,
	}).Render(nil)
	if err != nil {
		t.Fatal(err)
	}
	if out != "a:b" {
		t.Errorf("got %q, want %q", out, "a:b")
	}
}

// After a {% render %} returns, the disable on include is lifted —
// the next sibling tag can still use {% include %}.
func TestIncludeReenabledAfterRender(t *testing.T) {
	out, err := MustParse(`{% render "r" %}-{% include "i" %}`).WithLoader(MapLoader{
		"r": `r`,
		"i": `i`,
	}).Render(nil)
	if err != nil {
		t.Fatal(err)
	}
	if out != "r-i" {
		t.Errorf("got %q, want %q", out, "r-i")
	}
}

// Custom tags can opt into the disable mechanism via TagContext.
// Mirrors the Ruby pattern where any Tag can mix in Disabler / Disableable.
func TestCustomTagDisableable(t *testing.T) {
	env := NewEnvironment()
	env.RegisterTag("siren", func(_ string) (TagRenderer, error) {
		return tagRendererFunc(func(w io.Writer, ctx TagContext) error {
			if ctx.TagDisabled("siren") {
				return fmt.Errorf("%w: siren is off", ErrDisabledTag)
			}
			_, err := io.WriteString(w, "WEEOO")
			return err
		}), nil
	})
	env.RegisterTag("hush", func(_ string) (TagRenderer, error) {
		return tagRendererFunc(func(w io.Writer, ctx TagContext) error {
			return ctx.WithDisabledTags([]string{"siren"}, func() error {
				return ctx.RenderPartial(w, "p")
			})
		}), nil
	})
	env = env.WithLoader(MapLoader{"p": `{% siren %}`})

	if out, err := env.Render(`{% siren %}`, nil); err != nil || out != "WEEOO" {
		t.Fatalf("bare siren: got %q err %v", out, err)
	}
	_, err := env.Render(`{% hush %}`, nil)
	if err == nil || !errors.Is(err, ErrDisabledTag) {
		t.Fatalf("expected ErrDisabledTag from hush, got %v", err)
	}
}

