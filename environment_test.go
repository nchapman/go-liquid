package liquid

import (
	"io"
	"strings"
	"testing"
)

// TestEnvironmentIsolation verifies that filters/tags registered on a
// custom Environment do not leak into Default(), and vice versa. This is
// the central guarantee of the Environment API.
func TestEnvironmentIsolation(t *testing.T) {
	env := NewEnvironment()
	env.RegisterFilter("scoped_only", func(input any, _ ...any) any {
		return "[scoped]" + toString(input)
	})

	got, err := env.Render(`{{ "x" | scoped_only }}`, nil)
	if err != nil {
		t.Fatalf("env.Render: %v", err)
	}
	if got != "[scoped]x" {
		t.Errorf("env saw filter wrong: %q", got)
	}

	// Default does NOT see the env-scoped filter.
	got, err = Render(`{{ "x" | scoped_only }}`, nil)
	if err != nil {
		t.Fatalf("default render: %v", err)
	}
	if got != "x" {
		t.Errorf("default leaked custom filter: %q", got)
	}

	// And in strict mode, default reports the missing filter.
	tpl := MustParse(`{{ "x" | scoped_only }}`)
	if _, err := tpl.Render(nil, StrictFilters()); err == nil {
		t.Error("expected strict-filter error from default env")
	}
}

// TestEnvironmentBuiltins confirms NewEnvironment ships with the standard
// filter set — matching Ruby's Environment.build, which inherits
// standardfilters by default.
func TestEnvironmentBuiltins(t *testing.T) {
	env := NewEnvironment()
	got, err := env.Render(`{{ "hello" | upcase }}`, nil)
	if err != nil {
		t.Fatalf("env.Render: %v", err)
	}
	if got != "HELLO" {
		t.Errorf("builtin upcase missing from env: %q", got)
	}
}

// TestEnvironmentTagIsolation does the same check for custom tags.
func TestEnvironmentTagIsolation(t *testing.T) {
	env := NewEnvironment()
	env.RegisterTag("env_marker", func(_ string) (TagRenderer, error) {
		return tagRendererFunc(func(w io.Writer, _ TagContext) error {
			_, err := w.Write([]byte("[env]"))
			return err
		}), nil
	})

	got, err := env.Render(`{% env_marker %}`, nil)
	if err != nil {
		t.Fatalf("env render: %v", err)
	}
	if got != "[env]" {
		t.Errorf("env tag missing: %q", got)
	}

	// The default environment's parser should not recognize the tag.
	if _, err := Parse(`{% env_marker %}`); err == nil {
		t.Error("default should reject env-scoped tag at parse time")
	}
}

// TestEnvironmentWithLoader confirms a default loader on the environment
// flows into templates parsed through it, so {% include %} works without
// an explicit per-template WithLoader.
func TestEnvironmentWithLoader(t *testing.T) {
	env := NewEnvironment().WithLoader(MapLoader{
		"greeting": "hello, {{ name }}",
	})
	got, err := env.Render(`{% include "greeting" %}`, map[string]any{"name": "world"})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(got, "hello, world") {
		t.Errorf("include via env loader: %q", got)
	}
}

type tagRendererFunc func(w io.Writer, ctx TagContext) error

func (f tagRendererFunc) Render(w io.Writer, ctx TagContext) error { return f(w, ctx) }

// TestDateNowToday confirms the "now" / "today" magic strings render the
// current time, matching Ruby's Utils.to_date special-case.
func TestDateNowToday(t *testing.T) {
	got, err := Render(`{{ "now" | date: "%Y" }}`, nil)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	// The current year as 4 digits (won't change within a test run).
	wantPrefix := "20"
	if !strings.HasPrefix(got, wantPrefix) || len(got) != 4 {
		t.Errorf("date now: got %q, want a 4-digit year starting with 20", got)
	}

	got, err = Render(`{{ "today" | date: "%Y" }}`, nil)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.HasPrefix(got, wantPrefix) || len(got) != 4 {
		t.Errorf("date today: got %q", got)
	}
}

// TestTagContextRenderPartial confirms that custom tags can resolve and
// render partials through the active loader, matching Ruby's
// PartialCache.load primitive. This unblocks layout/macro patterns built
// outside the package.
func TestTagContextRenderPartial(t *testing.T) {
	env := NewEnvironment().WithLoader(MapLoader{
		"layout": "<wrap>{{ inner }}</wrap>",
	})
	env.RegisterBlock("with_layout", func(markup string) (TagRenderer, error) {
		layout := strings.Trim(strings.TrimSpace(markup), `"`)
		return tagRendererFunc(func(w io.Writer, ctx TagContext) error {
			var body strings.Builder
			if err := ctx.RenderBody(&body); err != nil {
				return err
			}
			ctx.Assign("inner", body.String())
			return ctx.RenderPartial(w, layout)
		}), nil
	})

	got, err := env.Render(`{% with_layout "layout" %}hi{% endwith_layout %}`, nil)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if got != "<wrap>hi</wrap>" {
		t.Errorf("got %q, want <wrap>hi</wrap>", got)
	}

	// Without a loader, RenderPartial reports a useful error rather than
	// rendering empty.
	envNoLoader := NewEnvironment()
	envNoLoader.RegisterTag("must_load", func(_ string) (TagRenderer, error) {
		return tagRendererFunc(func(w io.Writer, ctx TagContext) error {
			return ctx.RenderPartial(w, "missing")
		}), nil
	})
	if _, err := envNoLoader.Render(`{% must_load %}`, nil); err == nil {
		t.Error("expected error when no loader configured")
	}
}
