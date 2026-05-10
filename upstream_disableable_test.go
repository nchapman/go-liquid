package liquid

// Ported from upstream Ruby Liquid test/integration/tag/disableable_test.rb.
//
// Ruby's test verifies the inline-error-message rendering convention:
// when a tag is disabled, BlockBody catches the exception and emits the
// text "Liquid error: NAME usage is not allowed in this context" inline.
// go-liquid surfaces ErrDisabledTag from Render instead. The scenario
// (custom tags disabled by a custom block tag via the Disabler API) is
// the same; only the error-surface differs.

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

// renderTagName writes its registered tag name (Ruby's RenderTagName module).
type renderTagName struct{ name string }

func (r *renderTagName) Render(w io.Writer, ctx TagContext) error {
	if ctx.TagDisabled(r.name) {
		return ErrDisabledTag
	}
	_, err := io.WriteString(w, r.name)
	return err
}

// disableBlockRenderer disables the named tags while rendering its body.
type disableBlockRenderer struct{ disabled []string }

func (d *disableBlockRenderer) Render(w io.Writer, ctx TagContext) error {
	return ctx.WithDisabledTags(d.disabled, func() error {
		return ctx.RenderBody(w)
	})
}

func TestUpstreamTagDisableable(t *testing.T) {
	build := func(disabled []string) *Template {
		env := NewEnvironment()
		env.RegisterTag("custom", func(string) (TagRenderer, error) {
			return &renderTagName{name: "custom"}, nil
		})
		env.RegisterTag("custom2", func(string) (TagRenderer, error) {
			return &renderTagName{name: "custom2"}, nil
		})
		env.RegisterBlock("disable", func(string) (TagRenderer, error) {
			return &disableBlockRenderer{disabled: disabled}, nil
		})
		tmpl, err := env.Parse("{% disable %}{% custom %};{% custom2 %}{% enddisable %}")
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		return tmpl
	}

	expectDisabled := func(t *testing.T, tmpl *Template) {
		t.Helper()
		var buf bytes.Buffer
		err := tmpl.RenderTo(&buf, nil)
		if err == nil {
			t.Fatalf("expected ErrDisabledTag, got output %q", buf.String())
		}
		if !errors.Is(err, ErrDisabledTag) {
			t.Fatalf("expected errors.Is(err, ErrDisabledTag), got %v", err)
		}
	}

	t.Run("test_block_tag_disabling_nested_tag", func(t *testing.T) {
		// Ruby disable_tags "custom" only; custom2 still renders.
		// go-liquid surfaces ErrDisabledTag from Render rather than inlining
		// the error message into the output.
		expectDisabled(t, build([]string{"custom"}))
	})

	t.Run("test_block_tag_disabling_multiple_nested_tags", func(t *testing.T) {
		// Ruby disable_tags "custom", "custom2" — both disabled.
		expectDisabled(t, build([]string{"custom", "custom2"}))
	})
}
