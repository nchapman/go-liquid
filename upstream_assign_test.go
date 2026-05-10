package liquid

// Ported from upstream Ruby Liquid test/integration/assign_test.rb.
// Each subtest names the originating Ruby test. Ruby strict2 mode is not
// implemented in go-liquid; tests originally written for strict2 are
// adapted to verify the same shape under the default (strict) parser.

import (
	"errors"
	"strings"
	"testing"
)

func TestUpstreamAssign(t *testing.T) {
	t.Run("test_assign_with_hyphen_in_variable_name", func(t *testing.T) {
		src := "{% assign this-thing = 'Print this-thing' -%}\n{{ this-thing -}}\n"
		renderEq(t, "hyphen", src, nil, "Print this-thing")
	})

	t.Run("test_assigned_variable", func(t *testing.T) {
		data := map[string]any{"values": []any{"foo", "bar", "baz"}}
		renderEq(t, "idx-0", "{% assign foo = values %}.{{ foo[0] }}.", data, ".foo.")
		renderEq(t, "idx-1", "{% assign foo = values %}.{{ foo[1] }}.", data, ".bar.")
	})

	t.Run("test_assign_with_filter", func(t *testing.T) {
		data := map[string]any{"values": "foo,bar,baz"}
		renderEq(t, "split", `{% assign foo = values | split: "," %}.{{ foo[1] }}.`, data, ".bar.")
	})

	t.Run("test_assign_syntax_error", func(t *testing.T) {
		_, err := Parse("{% assign foo not values %}.")
		if err == nil {
			t.Fatal("expected syntax error for invalid assign")
		}
		if !strings.Contains(strings.ToLower(err.Error()), "assign") {
			t.Errorf("expected error to mention 'assign'; got %q", err.Error())
		}
	})

	t.Run("test_assign_uses_error_mode_strict_fails", func(t *testing.T) {
		// In strict mode the parenthesized filter expression should fail.
		_, err := Parse("{% assign foo = ('X' | downcase) %}")
		if err == nil {
			t.Fatal("expected parse error in strict mode")
		}
	})

	t.Run("test_assign_uses_error_mode_lax_silent", func(t *testing.T) {
		// GAP: Ruby's :lax mode swallows malformed expressions inside
		// known tags by emitting the raw source span. go-liquid's lax
		// mode currently only recovers from unknown tag names — known
		// tags with bad bodies still surface a parse error. Expanding
		// lax recovery to cover known-tag bodies is its own change.
		t.Skip("TODO: extend ErrorModeLax to recover from malformed tag bodies")
	})

	t.Run("test_expression_with_whitespace_in_square_brackets", func(t *testing.T) {
		data := map[string]any{"a": map[string]any{"b": "result"}}
		renderEq(t, "wsbrk", "{% assign r = a[ 'b' ] %}{{ r }}", data, "result")
	})

	t.Run("test_assign_score_exceeding_resource_limit", func(t *testing.T) {
		tmpl := MustParse("{% assign foo = 42 %}{% assign bar = 23 %}")
		limits := &ResourceLimits{AssignScoreLimit: 1}
		_, err := tmpl.Render(nil, WithLimits(limits))
		if !errors.Is(err, ErrResourceLimit) {
			t.Fatalf("expected ErrResourceLimit, got %v", err)
		}
		if !limits.Reached() {
			t.Error("expected Reached() = true after trip")
		}

		limits2 := &ResourceLimits{AssignScoreLimit: 2}
		out, err := tmpl.Render(nil, WithLimits(limits2))
		if err != nil {
			t.Fatalf("unexpected error within larger budget: %v", err)
		}
		if out != "" {
			t.Errorf("expected empty render, got %q", out)
		}
		if limits2.AssignScore() == 0 {
			t.Error("expected AssignScore to be tracked")
		}
	})

	t.Run("test_assign_score_exceeding_limit_from_composite_object", func(t *testing.T) {
		tmpl := MustParse("{% assign foo = 'aaaa' | reverse %}")

		limits := &ResourceLimits{AssignScoreLimit: 3}
		_, err := tmpl.Render(nil, WithLimits(limits))
		if !errors.Is(err, ErrResourceLimit) {
			t.Fatalf("expected ErrResourceLimit at limit 3, got %v", err)
		}
		if !limits.Reached() {
			t.Error("expected Reached() = true")
		}

		limits2 := &ResourceLimits{AssignScoreLimit: 5}
		if _, err := tmpl.Render(nil, WithLimits(limits2)); err != nil {
			t.Fatalf("unexpected error at limit 5: %v", err)
		}
	})

	// Ruby's `assign_score_of` helper exercises the recursive scorer.
	// In Go this is the unexported `assignScoreOf` function.
	t.Run("test_assign_score_of_int", func(t *testing.T) {
		if got := assignScoreOf(123); got != 1 {
			t.Errorf("int score = %d, want 1", got)
		}
	})

	t.Run("test_assign_score_of_string_counts_bytes", func(t *testing.T) {
		if got := assignScoreOf("123"); got != 3 {
			t.Errorf("'123' score = %d, want 3", got)
		}
		if got := assignScoreOf("12345"); got != 5 {
			t.Errorf("'12345' score = %d, want 5", got)
		}
		if got := assignScoreOf("すごい"); got != 9 {
			t.Errorf("'すごい' score = %d, want 9 (bytes)", got)
		}
	})

	t.Run("test_assign_score_of_array", func(t *testing.T) {
		if got := assignScoreOf([]any{}); got != 1 {
			t.Errorf("[] score = %d, want 1", got)
		}
		if got := assignScoreOf([]any{123}); got != 2 {
			t.Errorf("[123] score = %d, want 2", got)
		}
		if got := assignScoreOf([]any{123, "abcd"}); got != 6 {
			t.Errorf("[123,'abcd'] score = %d, want 6", got)
		}
	})

	t.Run("test_assign_score_of_hash", func(t *testing.T) {
		if got := assignScoreOf(map[string]any{}); got != 1 {
			t.Errorf("{} score = %d, want 1", got)
		}
		if got := assignScoreOf(map[string]any{"int": 123}); got != 5 {
			t.Errorf("{int:123} score = %d, want 5", got)
		}
		if got := assignScoreOf(map[string]any{"int": 123, "str": "abcd"}); got != 12 {
			t.Errorf("{int:123,str:'abcd'} score = %d, want 12", got)
		}
	})

	// Ruby's strict2 tests assert the identifier rules. In go-liquid's
	// default strict parser, valid identifiers should parse and invalid
	// ones should fail at parse time.

	t.Run("test_assign_with_valid_identifier", func(t *testing.T) {
		renderEq(t, "valid-id", "{% assign my_var = 'hello' %}{{ my_var }}", nil, "hello")
	})

	t.Run("test_assign_with_hyphen", func(t *testing.T) {
		renderEq(t, "hyphen-id", "{% assign my-var = 'hello' %}{{ my-var }}", nil, "hello")
	})

	t.Run("test_assign_rejects_parentheses_in_variable_name", func(t *testing.T) {
		if _, err := Parse("{% assign (a(b(c) = 1234 %}"); err == nil {
			t.Fatal("expected parse error for parens in variable name")
		}
	})

	t.Run("test_assign_rejects_brackets_in_variable_name", func(t *testing.T) {
		if _, err := Parse("{% assign [x.y] = 'hello' %}"); err == nil {
			t.Fatal("expected parse error for brackets in variable name")
		}
	})

	t.Run("test_assign_rejects_dot_in_variable_name", func(t *testing.T) {
		if _, err := Parse("{% assign a.b = 'hello' %}"); err == nil {
			t.Fatal("expected parse error for dot in variable name")
		}
	})

	t.Run("test_assign_rejects_numeric_variable_name", func(t *testing.T) {
		if _, err := Parse("{% assign 1abc = 'hello' %}"); err == nil {
			t.Fatal("expected parse error for numeric-prefixed variable name")
		}
	})

	t.Run("test_assign_with_filter_basic", func(t *testing.T) {
		renderEq(t, "filter", "{% assign my_var = 'hello' | upcase %}{{ my_var }}", nil, "HELLO")
	})
}
