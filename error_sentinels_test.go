package liquid

import (
	"errors"
	"testing"
)

func TestErrUndefinedVariableSentinel(t *testing.T) {
	cases := []struct {
		name, src string
		data      any
	}{
		{"bare variable", `{{ missing }}`, map[string]any{}},
		{"property",      `{{ obj.missing }}`, map[string]any{"obj": map[string]any{}}},
		{"index",         `{{ obj["missing"] }}`, map[string]any{"obj": map[string]any{}}},
	}
	for _, tc := range cases {
		_, err := Render(tc.src, tc.data, StrictVariables())
		if err == nil {
			t.Errorf("%s: expected error, got nil", tc.name)
			continue
		}
		if !errors.Is(err, ErrUndefinedVariable) {
			t.Errorf("%s: expected ErrUndefinedVariable, got %v", tc.name, err)
		}
		// Wrapped in RenderError so source position is reachable.
		var re *RenderError
		if !errors.As(err, &re) {
			t.Errorf("%s: expected wrapped RenderError, got %T", tc.name, err)
		}
	}
}

func TestErrUndefinedFilterSentinel(t *testing.T) {
	_, err := Render(`{{ "x" | not_a_real_filter }}`, nil, StrictFilters())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrUndefinedFilter) {
		t.Fatalf("expected ErrUndefinedFilter, got %v", err)
	}
	var re *RenderError
	if !errors.As(err, &re) {
		t.Errorf("expected wrapped RenderError, got %T", err)
	}
}

// Without strict mode, both error conditions are silent — the sentinel
// must not leak into normal renders.
func TestSentinelsNotRaisedInLaxMode(t *testing.T) {
	if _, err := Render(`{{ missing }}`, nil); err != nil {
		t.Errorf("undefined variable should be silent: %v", err)
	}
	if _, err := Render(`{{ "x" | wat }}`, nil); err != nil {
		t.Errorf("unknown filter should be silent: %v", err)
	}
}
