package liquid

import (
	"io"
	"testing"
)

// Upstream parity: integration/tag_test.rb

// helloTag is a minimal inline tag that always emits "hello".
type helloTag struct{}

func (helloTag) Render(w io.Writer, ctx TagContext) error {
	_, err := io.WriteString(w, "hello")
	return err
}

// wrappedHelloTag wraps a base renderer's output with "foo"/"bar". Mirrors
// the Ruby test's Class.new(klass1) override that does
// 'foo' + super + 'bar'.
type wrappedHelloTag struct{ base TagRenderer }

func (w wrappedHelloTag) Render(out io.Writer, ctx TagContext) error {
	if _, err := io.WriteString(out, "foo"); err != nil {
		return err
	}
	if err := w.base.Render(out, ctx); err != nil {
		return err
	}
	_, err := io.WriteString(out, "bar")
	return err
}

func TestUpstream_Tag_CustomTagsHaveDefaultRenderToOutputBuffer(t *testing.T) {
	env := NewEnvironment()
	env.RegisterTag("blabla", func(markup string) (TagRenderer, error) {
		return helloTag{}, nil
	})
	tmpl, err := env.Parse("{% blabla %}")
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

	// Subclass-like override: wrap "foo"+super+"bar".
	env2 := NewEnvironment()
	env2.RegisterTag("blabla", func(markup string) (TagRenderer, error) {
		return wrappedHelloTag{base: helloTag{}}, nil
	})
	tmpl2, err := env2.Parse("{% blabla %}")
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
