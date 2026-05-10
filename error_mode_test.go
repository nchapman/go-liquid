package liquid

import (
	"strings"
	"testing"
)

func TestErrorModeStrictRejectsUnknownTag(t *testing.T) {
	env := NewEnvironment()
	_, err := env.Parse(`hi {% mystery %}`)
	if err == nil {
		t.Fatal("expected parse error in strict mode")
	}
	if !strings.Contains(err.Error(), "unknown tag") {
		t.Errorf("error %q does not mention 'unknown tag'", err.Error())
	}
}

func TestErrorModeWarnEmitsRawAndRecordsWarning(t *testing.T) {
	env := NewEnvironment().WithErrorMode(ErrorModeWarn)
	tmpl, err := env.Parse(`hi {% mystery x=1 %} bye`)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	out, err := tmpl.Render(nil)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(out, "{% mystery x=1 %}") {
		t.Errorf("output %q lost the raw tag span", out)
	}
	ws := tmpl.Warnings()
	if len(ws) != 1 {
		t.Fatalf("warnings = %d, want 1: %v", len(ws), ws)
	}
	if !strings.Contains(ws[0].Message, "mystery") {
		t.Errorf("warning %q does not name the tag", ws[0].Message)
	}
	if ws[0].Line == 0 {
		t.Error("warning is missing line number")
	}
}

func TestErrorModeWarnPreservesWhitespaceControl(t *testing.T) {
	env := NewEnvironment().WithErrorMode(ErrorModeWarn)
	tmpl, err := env.Parse("a   {%- mystery foo -%}   b")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	out, _ := tmpl.Render(nil)
	// `{%-` strips trailing whitespace from "a   "; `-%}` strips
	// leading whitespace from "   b". The mystery tag itself is
	// reconstructed faithfully.
	if !strings.Contains(out, "{%- mystery foo -%}") {
		t.Errorf("output %q lost whitespace-control trim modifiers", out)
	}
	if strings.Contains(out, "a   {") {
		t.Errorf("output %q kept whitespace that {%%- should have stripped", out)
	}
}

func TestErrorModeLaxEmitsRawSilently(t *testing.T) {
	env := NewEnvironment().WithErrorMode(ErrorModeLax)
	tmpl, err := env.Parse(`{% mystery %}`)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if got := len(tmpl.Warnings()); got != 0 {
		t.Errorf("warnings in lax mode = %d, want 0", got)
	}
	out, _ := tmpl.Render(nil)
	if !strings.Contains(out, "mystery") {
		t.Errorf("output %q lost the raw tag span", out)
	}
}

func TestErrorModeStringRoundTrip(t *testing.T) {
	for _, m := range []ErrorMode{ErrorModeStrict, ErrorModeWarn, ErrorModeLax} {
		s := m.String()
		if s == "" {
			t.Errorf("ErrorMode(%d).String() empty", int(m))
		}
	}
}

func TestEnvironmentErrorModeAccessor(t *testing.T) {
	env := NewEnvironment()
	if got := env.ErrorMode(); got != ErrorModeStrict {
		t.Errorf("default = %v, want strict", got)
	}
	env.WithErrorMode(ErrorModeWarn)
	if got := env.ErrorMode(); got != ErrorModeWarn {
		t.Errorf("after set = %v, want warn", got)
	}
}

func TestErrorModeWarnPreservesKnownTags(t *testing.T) {
	// Switching to warn must not change behavior for built-in/registered tags.
	env := NewEnvironment().WithErrorMode(ErrorModeWarn)
	tmpl, err := env.Parse(`{% if x %}A{% else %}B{% endif %}`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	out, _ := tmpl.Render(map[string]any{"x": true})
	if out != "A" {
		t.Errorf("got %q want %q", out, "A")
	}
	if got := len(tmpl.Warnings()); got != 0 {
		t.Errorf("warnings on valid template = %d, want 0", got)
	}
}

func TestWarningStringWithName(t *testing.T) {
	w := Warning{Message: "oops", Line: 3, Column: 5, TemplateName: "page.liquid"}
	got := w.String()
	if !strings.Contains(got, "page.liquid") || !strings.Contains(got, "3") {
		t.Errorf("String() = %q, missing name or line", got)
	}
}
