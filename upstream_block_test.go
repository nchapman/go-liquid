package liquid

import (
	"io"
	"testing"
)

// Upstream parity: integration/block_test.rb

// passthroughBlock renders its body verbatim. Mirrors Ruby's default
// Liquid::Block#render (super-able for subclass overrides).
type passthroughBlock struct{}

func (passthroughBlock) Render(w io.Writer, ctx TagContext) error {
	return ctx.RenderBody(w)
}

// helloBlock ignores its body and emits "hello".
type helloBlock struct{}

func (helloBlock) Render(w io.Writer, ctx TagContext) error {
	_, err := io.WriteString(w, "hello")
	return err
}

// wrappedHelloBlock mirrors the Ruby subclass that does
// 'foo' + super + 'bar' — wraps the base block's output.
type wrappedHelloBlock struct{ base TagRenderer }

func (w wrappedHelloBlock) Render(out io.Writer, ctx TagContext) error {
	if _, err := io.WriteString(out, "foo"); err != nil {
		return err
	}
	if err := w.base.Render(out, ctx); err != nil {
		return err
	}
	_, err := io.WriteString(out, "bar")
	return err
}

func TestUpstream_Block_UnexpectedEndTag(t *testing.T) {
	_, err := Parse("{% if true %}{% endunless %}")
	if err == nil {
		t.Fatalf("expected parse error")
	}
}

func TestUpstream_Block_WithCustomTag(t *testing.T) {
	env := NewEnvironment()
	env.RegisterBlock("testtag", func(markup string) (TagRenderer, error) {
		return passthroughBlock{}, nil
	})
	if _, err := env.Parse("{% testtag %} {% endtesttag %}"); err != nil {
		t.Fatalf("parse: %v", err)
	}
}

func TestUpstream_Block_CustomBlockTagsHaveDefaultRenderToOutputBuffer(t *testing.T) {
	env := NewEnvironment()
	env.RegisterBlock("blabla", func(markup string) (TagRenderer, error) {
		return helloBlock{}, nil
	})
	tmpl, err := env.Parse("{% blabla %} bla {% endblabla %}")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	got, err := tmpl.Render(nil)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if got != "hello" {
		t.Fatalf("got %q", got)
	}

	env2 := NewEnvironment()
	env2.RegisterBlock("blabla", func(markup string) (TagRenderer, error) {
		return wrappedHelloBlock{base: helloBlock{}}, nil
	})
	tmpl2, err := env2.Parse("{% blabla %} foo {% endblabla %}")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	got2, err := tmpl2.Render(nil)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if got2 != "foohellobar" {
		t.Fatalf("got %q", got2)
	}
}
