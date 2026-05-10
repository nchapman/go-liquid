package liquid

import (
	"errors"
	"fmt"
	"io"
	"reflect"
)

// ErrResourceLimit is returned when a render exceeds any of the configured
// ResourceLimits. Use errors.Is to detect: errors.Is(err, ErrResourceLimit).
// Mirrors Ruby's Liquid::MemoryError.
var ErrResourceLimit = errors.New("liquid: resource limits exceeded")

// ResourceLimits caps the work and output a single render is allowed to
// perform. It mirrors Ruby's Liquid::ResourceLimits and is intended for
// hosting untrusted templates where a malicious author could otherwise
// trigger pathological allocation or output.
//
// Each limit is opt-in: a zero value means "unlimited". Per-render limits
// (RenderScoreLimit, AssignScoreLimit, RenderLengthLimit) are checked
// against counters that reset at the start of each Render call. The
// Cumulative* limits track totals across every Render performed against
// the same *ResourceLimits instance — useful for capping a long-lived
// template that's rendered many times against a shared resource budget.
//
// Scoring matches Ruby:
//   - render_score: incremented by the number of AST nodes a block
//     executes, charged once per evalNodes call.
//   - assign_score: bytes of strings, recursive sum for collections, 1
//     per other value. Also charged for {% capture %} output bytes.
//   - render_length: cumulative bytes written to the user's output writer.
//
// ResourceLimits is NOT safe for concurrent renders: a single
// *ResourceLimits is shared by pointer through renderConfig and into
// isolated partials, so two Renders sharing the same instance will race
// on the counters. Use one *ResourceLimits per concurrent render, or
// serialize access externally.
//
// Negative limit values are treated as unlimited (same as zero).
type ResourceLimits struct {
	// RenderLengthLimit caps total bytes written to the user's output
	// writer in a single Render. Capture output does not count here.
	RenderLengthLimit int
	// RenderScoreLimit caps the per-render render_score counter.
	RenderScoreLimit int
	// AssignScoreLimit caps the per-render assign_score counter.
	AssignScoreLimit int
	// CumulativeRenderScoreLimit caps render_score summed across every
	// Render performed with this *ResourceLimits.
	CumulativeRenderScoreLimit int
	// CumulativeAssignScoreLimit caps assign_score summed across every
	// Render performed with this *ResourceLimits.
	CumulativeAssignScoreLimit int

	renderScore           int64
	assignScore           int64
	cumulativeRenderScore int64
	cumulativeAssignScore int64
	outputBytes           int64
	reached               bool
}

// RenderScore returns the per-render render_score from the most recent
// Render. Reset to zero at the start of each Render.
func (l *ResourceLimits) RenderScore() int64 { return l.renderScore }

// AssignScore returns the per-render assign_score from the most recent
// Render. Reset to zero at the start of each Render.
func (l *ResourceLimits) AssignScore() int64 { return l.assignScore }

// CumulativeRenderScore returns render_score summed across every Render
// performed with this *ResourceLimits.
func (l *ResourceLimits) CumulativeRenderScore() int64 { return l.cumulativeRenderScore }

// CumulativeAssignScore returns assign_score summed across every Render
// performed with this *ResourceLimits.
func (l *ResourceLimits) CumulativeAssignScore() int64 { return l.cumulativeAssignScore }

// Reached reports whether any limit was hit on the most recent Render.
func (l *ResourceLimits) Reached() bool { return l.reached }

// reset clears the per-render counters at the start of a Render. Cumulative
// counters survive; if a cumulative limit is already exceeded the reset
// itself trips the limit (matches Ruby).
func (l *ResourceLimits) reset() error {
	l.renderScore = 0
	l.assignScore = 0
	l.outputBytes = 0
	l.reached = false
	if l.CumulativeRenderScoreLimit > 0 && l.cumulativeRenderScore > int64(l.CumulativeRenderScoreLimit) {
		return l.trip()
	}
	if l.CumulativeAssignScoreLimit > 0 && l.cumulativeAssignScore > int64(l.CumulativeAssignScoreLimit) {
		return l.trip()
	}
	return nil
}

func (l *ResourceLimits) trip() error {
	l.reached = true
	return ErrResourceLimit
}

func (l *ResourceLimits) incrementRenderScore(n int) error {
	l.renderScore += int64(n)
	l.cumulativeRenderScore += int64(n)
	if l.RenderScoreLimit > 0 && l.renderScore > int64(l.RenderScoreLimit) {
		return l.trip()
	}
	if l.CumulativeRenderScoreLimit > 0 && l.cumulativeRenderScore > int64(l.CumulativeRenderScoreLimit) {
		return l.trip()
	}
	return nil
}

func (l *ResourceLimits) incrementAssignScore(n int) error {
	l.assignScore += int64(n)
	l.cumulativeAssignScore += int64(n)
	if l.AssignScoreLimit > 0 && l.assignScore > int64(l.AssignScoreLimit) {
		return l.trip()
	}
	if l.CumulativeAssignScoreLimit > 0 && l.cumulativeAssignScore > int64(l.CumulativeAssignScoreLimit) {
		return l.trip()
	}
	return nil
}

// assignScoreOf mirrors Ruby's assign_score_of: bytesize for strings, 1 +
// recursive sum for arrays, 1 + sum of (key + value) for hashes, 1 for
// any other scalar. We extend with reflect so typed slices/maps coming
// from Go callers are scored consistently with map[string]any.
func assignScoreOf(v any) int {
	switch x := v.(type) {
	case nil:
		return 1
	case string:
		return len(x)
	case []any:
		sum := 1
		for _, c := range x {
			sum += assignScoreOf(c)
		}
		return sum
	case map[string]any:
		sum := 1
		for k, c := range x {
			sum += len(k)
			sum += assignScoreOf(c)
		}
		return sum
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		sum := 1
		for i := 0; i < rv.Len(); i++ {
			sum += assignScoreOf(rv.Index(i).Interface())
		}
		return sum
	case reflect.Map:
		sum := 1
		iter := rv.MapRange()
		for iter.Next() {
			sum += assignScoreOf(iter.Key().Interface())
			sum += assignScoreOf(iter.Value().Interface())
		}
		return sum
	}
	return 1
}

// limitsWriter wraps the user-supplied io.Writer and trips the
// RenderLengthLimit before forwarding the bytes downstream. Capture
// buffers do not flow through this writer, so {% capture %} output is
// scored against assign_score only — matching Ruby's with_capture.
type limitsWriter struct {
	w io.Writer
	l *ResourceLimits
}

func (lw *limitsWriter) Write(p []byte) (int, error) {
	lw.l.outputBytes += int64(len(p))
	if lw.l.RenderLengthLimit > 0 && lw.l.outputBytes > int64(lw.l.RenderLengthLimit) {
		_ = lw.l.trip()
		return 0, fmt.Errorf("%w: render length %d exceeds limit %d",
			ErrResourceLimit, lw.l.outputBytes, lw.l.RenderLengthLimit)
	}
	return lw.w.Write(p)
}

// captureWriter wraps a capture buffer so each Write increments
// assign_score incrementally — matching Ruby's increment_write_score
// inside with_capture. Charging per-write (not once at the end) means a
// runaway capture body trips the assign limit before producing all of
// its bytes, which is the security-relevant property for untrusted
// templates.
type captureWriter struct {
	w io.Writer
	l *ResourceLimits
}

func (cw *captureWriter) Write(p []byte) (int, error) {
	if err := cw.l.incrementAssignScore(len(p)); err != nil {
		return 0, err
	}
	return cw.w.Write(p)
}

// WithLimits attaches a ResourceLimits budget to a Render call. The same
// *ResourceLimits may be reused across renders to track Cumulative* totals.
// Counter values can be inspected after the call returns (or after a
// failed render) via the read-only accessor methods.
func WithLimits(l *ResourceLimits) RenderOption {
	return func(o *renderOpts) { o.limits = l }
}
