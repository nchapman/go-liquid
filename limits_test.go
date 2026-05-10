package liquid

import (
	"errors"
	"strings"
	"testing"
)

func TestResourceLimitsRenderLength(t *testing.T) {
	tmpl := MustParse(`{% for i in (1..100) %}{{ i }}{% endfor %}`)
	limits := &ResourceLimits{RenderLengthLimit: 10}
	_, err := tmpl.Render(nil, WithLimits(limits))
	if !errors.Is(err, ErrResourceLimit) {
		t.Fatalf("expected ErrResourceLimit, got %v", err)
	}
	if !limits.Reached() {
		t.Errorf("expected Reached() true")
	}
}

func TestResourceLimitsRenderScore(t *testing.T) {
	// Each {{ ... }} and text node bumps render_score by 1 per evalNodes call.
	tmpl := MustParse(`{% for i in (1..50) %}{{ i }} {% endfor %}`)
	limits := &ResourceLimits{RenderScoreLimit: 5}
	_, err := tmpl.Render(nil, WithLimits(limits))
	if !errors.Is(err, ErrResourceLimit) {
		t.Fatalf("expected ErrResourceLimit, got %v", err)
	}
}

func TestResourceLimitsAssignScore(t *testing.T) {
	tmpl := MustParse(`{% assign x = "this is a long string" %}`)
	limits := &ResourceLimits{AssignScoreLimit: 5}
	_, err := tmpl.Render(nil, WithLimits(limits))
	if !errors.Is(err, ErrResourceLimit) {
		t.Fatalf("expected ErrResourceLimit, got %v", err)
	}
	if got := limits.AssignScore(); got <= 5 {
		t.Errorf("AssignScore = %d, want > 5 (the trip)", got)
	}
}

func TestResourceLimitsCaptureCountsAsAssign(t *testing.T) {
	// Capture body produces 50 chars; output bytes stay 0 because capture
	// doesn't flow through the limits writer.
	tmpl := MustParse(`{% capture x %}{{ "abcdefghij" }}{{ "abcdefghij" }}{{ "abcdefghij" }}{{ "abcdefghij" }}{{ "abcdefghij" }}{% endcapture %}`)
	limits := &ResourceLimits{AssignScoreLimit: 10}
	_, err := tmpl.Render(nil, WithLimits(limits))
	if !errors.Is(err, ErrResourceLimit) {
		t.Fatalf("expected ErrResourceLimit, got %v", err)
	}
}

func TestResourceLimitsCaptureChargedIncrementally(t *testing.T) {
	// A capture body that loops past the limit must trip mid-iteration,
	// not after the body completes — matching Ruby's increment_write_score.
	tmpl := MustParse(`{% capture x %}{% for i in (1..100) %}{{ i }}{% endfor %}{% endcapture %}`)
	limits := &ResourceLimits{AssignScoreLimit: 5}
	_, err := tmpl.Render(nil, WithLimits(limits))
	if !errors.Is(err, ErrResourceLimit) {
		t.Fatalf("expected ErrResourceLimit, got %v", err)
	}
	// Should trip before producing all 100 numbers worth of bytes.
	if got := limits.AssignScore(); got > 50 {
		t.Errorf("AssignScore = %d, expected to trip well before completion", got)
	}
}

func TestResourceLimitsCumulative(t *testing.T) {
	tmpl := MustParse(`{% assign x = "abcdefghij" %}`)
	limits := &ResourceLimits{CumulativeAssignScoreLimit: 25}
	for i := range 2 {
		if _, err := tmpl.Render(nil, WithLimits(limits)); err != nil {
			t.Fatalf("render %d: %v", i, err)
		}
	}
	// Third render pushes cumulative to 30 > 25.
	_, err := tmpl.Render(nil, WithLimits(limits))
	if !errors.Is(err, ErrResourceLimit) {
		t.Fatalf("expected ErrResourceLimit on 3rd render, got %v", err)
	}
	if got, want := limits.CumulativeAssignScore(), int64(30); got != want {
		t.Errorf("CumulativeAssignScore = %d, want %d", got, want)
	}
}

func TestResourceLimitsCumulativeResetTrip(t *testing.T) {
	// A reset that finds cumulative already over the limit must trip
	// before any rendering happens (Ruby parity).
	limits := &ResourceLimits{CumulativeRenderScoreLimit: 1}
	limits.cumulativeRenderScore = 100
	tmpl := MustParse(`hi`)
	_, err := tmpl.Render(nil, WithLimits(limits))
	if !errors.Is(err, ErrResourceLimit) {
		t.Fatalf("expected immediate trip on reset, got %v", err)
	}
}

func TestResourceLimitsResetsPerRender(t *testing.T) {
	tmpl := MustParse(`{% assign x = "abcdefghij" %}`)
	limits := &ResourceLimits{}
	_, _ = tmpl.Render(nil, WithLimits(limits))
	if got := limits.AssignScore(); got != 10 {
		t.Errorf("AssignScore after 1st = %d, want 10", got)
	}
	_, _ = tmpl.Render(nil, WithLimits(limits))
	if got := limits.AssignScore(); got != 10 {
		t.Errorf("AssignScore after 2nd = %d, want 10 (per-render reset)", got)
	}
	if got := limits.CumulativeAssignScore(); got != 20 {
		t.Errorf("CumulativeAssignScore = %d, want 20", got)
	}
}

func TestResourceLimitsZeroIsUnlimited(t *testing.T) {
	tmpl := MustParse(`{% for i in (1..1000) %}{{ i }}{% endfor %}`)
	limits := &ResourceLimits{}
	out, err := tmpl.Render(nil, WithLimits(limits))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(out, "12345") {
		t.Errorf("unexpected output prefix: %q", out[:10])
	}
	if !strings.HasSuffix(out, "9991000") {
		t.Errorf("unexpected output suffix")
	}
}

func TestAssignScoreOf(t *testing.T) {
	cases := []struct {
		name string
		v    any
		want int
	}{
		{"nil", nil, 1},
		{"empty string", "", 0},
		{"string bytes", "héllo", 6}, // bytesize, not runes — Ruby parity
		{"int", 42, 1},
		{"empty array", []any{}, 1},
		{"array of strings", []any{"ab", "cd"}, 1 + 2 + 2},
		{"empty hash", map[string]any{}, 1},
		{"hash one entry", map[string]any{"k": "vv"}, 1 + 1 + 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := assignScoreOf(tc.v); got != tc.want {
				t.Errorf("assignScoreOf(%v) = %d, want %d", tc.v, got, tc.want)
			}
		})
	}
}
