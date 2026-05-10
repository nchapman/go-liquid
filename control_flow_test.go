package liquid

import (
	"errors"
	"fmt"
	"testing"
)

// TestControlFlowErrorsArePreservedThroughWrapping locks down the
// invariant that {% break %} and {% continue %} keep working even if a
// future caller wraps the sentinel — we use errors.Is, not pointer
// equality, in the for/tablerow loops so the control-flow contract
// survives error wrapping.
func TestControlFlowErrorsArePreservedThroughWrapping(t *testing.T) {
	// Sanity: the unwrapped sentinels still match themselves.
	if !errors.Is(errBreak, errBreak) {
		t.Fatal("errBreak does not match itself via errors.Is")
	}
	if !errors.Is(errContinue, errContinue) {
		t.Fatal("errContinue does not match itself via errors.Is")
	}

	// And a wrapped sentinel still matches via errors.Is.
	wrapped := fmt.Errorf("inside hypothetical filter: %w", errBreak)
	if !errors.Is(wrapped, errBreak) {
		t.Fatal("wrapped errBreak should match via errors.Is")
	}

	// End-to-end: real templates that depend on break/continue still work.
	cases := []struct {
		name, tmpl, want string
	}{
		{"break in for", `{% for n in (1..5) %}{% if n == 3 %}{% break %}{% endif %}{{ n }}{% endfor %}`, "12"},
		{"continue in for", `{% for n in (1..5) %}{% if n == 3 %}{% continue %}{% endif %}{{ n }}{% endfor %}`, "1245"},
		{"break in tablerow", `{% tablerow n in items cols: 2 %}{% if n == 3 %}{% break %}{% endif %}{{ n }}{% endtablerow %}`, "<tr class=\"row1\">\n<td class=\"col1\">1</td><td class=\"col2\">2</td></tr>\n<tr class=\"row2\"><td class=\"col1\"></td></tr>\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Render(tc.tmpl, map[string]any{"items": []any{1, 2, 3, 4, 5}})
			if err != nil {
				t.Fatalf("render: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %q want %q", got, tc.want)
			}
		})
	}
}

// methodSurfaceModel exercises callTemplateMethod's signature filter: only
// (T) and (T, error) shapes are auto-invoked; bare error and multi-value
// returns are skipped. The contract is documented at the package level;
// this test pins it down so a refactor can't quietly change the surface.
type methodSurfaceModel struct{ Name string }

func (m methodSurfaceModel) DisplayName() string             { return "DISPLAY:" + m.Name }
func (m methodSurfaceModel) WithError() (string, error)      { return "ok-" + m.Name, nil }
func (m methodSurfaceModel) WithErrorFails() (string, error) { return "x", errors.New("nope") }
func (m methodSurfaceModel) Mutator() error                  { panic("must not be called from template") }
func (m methodSurfaceModel) Multi() (string, string)         { return "a", "b" }

func TestMethodAutoInvocationSurface(t *testing.T) {
	data := map[string]any{"m": methodSurfaceModel{Name: "Ada"}}
	cases := []struct {
		name, tmpl, want string
	}{
		{"single string return is invoked", `{{ m.DisplayName }}`, "DISPLAY:Ada"},
		{"(T, error) with nil error is invoked", `{{ m.WithError }}`, "ok-Ada"},
		{"(T, error) with non-nil error is treated as undefined", `[{{ m.WithErrorFails }}]`, "[]"},
		{"bare error return is NOT invoked", `[{{ m.Mutator }}]`, "[]"},
		{"multi-value return is NOT invoked", `[{{ m.Multi }}]`, "[]"},
		{"plain field is unaffected", `{{ m.Name }}`, "Ada"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Render(tc.tmpl, data)
			if err != nil {
				t.Fatalf("render: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %q want %q", got, tc.want)
			}
		})
	}
}

// hiddenDrop demonstrates the Drop opt-out path: Mutator-style methods
// stay unreachable, and only what LiquidLookup exposes is visible.
type hiddenDrop struct{ token string }

func (h hiddenDrop) Token() string                    { return h.token }
func (h hiddenDrop) LiquidLookup(key string) (any, bool) {
	if key == "name" {
		return "public-name", true
	}
	return nil, false
}

func TestDropHidesMethodAutoInvocation(t *testing.T) {
	got, err := Render(`name=[{{ d.name }}] token=[{{ d.Token }}]`, map[string]any{"d": hiddenDrop{token: "SECRET"}})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	want := `name=[public-name] token=[]`
	if got != want {
		t.Errorf("Drop should hide Token() — got %q want %q", got, want)
	}
}
