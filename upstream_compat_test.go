package liquid

// This file once held a minimal first-pass port of selected upstream
// suites. Those tests have since been moved into per-file ports
// (upstream_parsing_quirks_test.go, upstream_variable_test.go,
// upstream_for_tag_test.go, upstream_if_else_test.go, upstream_raw_test.go,
// upstream_inline_comment_test.go, upstream_cycle_test.go,
// upstream_table_row_test.go, upstream_break_test.go,
// upstream_continue_test.go, upstream_statements_test.go,
// upstream_standard_tag_test.go, upstream_unless_else_test.go).
//
// What remains here is the shared `renderEq` helper used across all the
// per-file ports plus a small case/when fixture that has no dedicated
// Ruby file of its own (Ruby covers it inside standard_tag/statements).

import (
	"testing"
)

// renderEq is the shared assertion used across every upstream_*_test.go
// file: parse + render `src` against `data`, fail if the output differs
// from `want`. Lives in this file (not a dedicated _helpers file) so it
// is unmistakably testing-scoped.
func renderEq(t *testing.T, name, src string, data map[string]any, want string) {
	t.Helper()
	got, err := Render(src, data)
	if err != nil {
		t.Fatalf("%s: unexpected error: %v", name, err)
	}
	if got != want {
		t.Errorf("%s: template %q\n  want %q\n  got  %q", name, src, want, got)
	}
}

// ---- case/when: no dedicated Ruby file (covered inside Ruby's
// standard_tag/statements suites) ----

func TestUpstream_CaseWhen(t *testing.T) {
	t.Run("when_with_comma", func(t *testing.T) {
		src := "{% case x %}{% when 1, 2, 3 %}low{% when 4 %}four{% endcase %}"
		renderEq(t, "comma-1", src, map[string]any{"x": 1}, "low")
		renderEq(t, "comma-3", src, map[string]any{"x": 3}, "low")
		renderEq(t, "comma-4", src, map[string]any{"x": 4}, "four")
	})

	t.Run("when_with_or", func(t *testing.T) {
		src := "{% case x %}{% when 1 or 2 or 3 %}low{% when 4 %}four{% endcase %}"
		renderEq(t, "or-1", src, map[string]any{"x": 1}, "low")
		renderEq(t, "or-2", src, map[string]any{"x": 2}, "low")
		renderEq(t, "or-3", src, map[string]any{"x": 3}, "low")
		renderEq(t, "or-4", src, map[string]any{"x": 4}, "four")
	})

	t.Run("case_on_size", func(t *testing.T) {
		src := "{% case a.size %}{% when 1 %}1{% when 2 %}2{% endcase %}"
		renderEq(t, "size-0", src, map[string]any{"a": []int{}}, "")
		renderEq(t, "size-1", src, map[string]any{"a": []int{10}}, "1")
		renderEq(t, "size-2", src, map[string]any{"a": []int{10, 20}}, "2")
	})
}
