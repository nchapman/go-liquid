package liquid

import "fmt"

// LiquidErrorKind categorizes a Drop-raised render error so the output
// renderer can pick the prefix matching Ruby Liquid: "Liquid error"
// for standard/argument errors and "Liquid syntax error" for syntax
// errors. The kind has no behavioral effect beyond the rendered prefix.
type LiquidErrorKind int

const (
	// LiquidStandardError matches Ruby's Liquid::StandardError. Renders
	// as "Liquid error: <message>".
	LiquidStandardError LiquidErrorKind = iota
	// LiquidArgumentError matches Ruby's Liquid::ArgumentError. Renders
	// as "Liquid error: <message>" (same prefix as standard error in
	// upstream output).
	LiquidArgumentError
	// LiquidSyntaxError matches Ruby's Liquid::SyntaxError. Renders as
	// "Liquid syntax error: <message>".
	LiquidSyntaxError
)

// LiquidError is a value a Drop may return from LiquidLookup (or a
// filter from its result) in place of a regular value. Instead of
// aborting the render, the engine formats it inline ("Liquid error:
// <message>") and continues with the rest of the template. This
// mirrors Ruby Liquid's behavior of rescuing tagged exceptions inside
// `{{ }}` evaluation.
//
// A *LiquidError propagates through filter chains unchanged — it is
// not stringified, upcased, etc. — and is rendered only at the output
// boundary that produced it.
//
// In conditional contexts (`{% if drop.key %}`) a non-nil *LiquidError
// is truthy, matching Ruby's behavior of rescued exception objects
// evaluating as truthy. Use `nil` plus the second return of false from
// LiquidLookup for keys that should be falsy.
//
// LiquidError implements the error interface so it can also be used as
// a Go error if the caller wants to bubble it up.
type LiquidError struct {
	Kind    LiquidErrorKind
	Message string
}

// Error returns the formatted inline message (without the line-number
// suffix). Implements the error interface.
func (e *LiquidError) Error() string { return e.renderPrefix() + ": " + e.Message }

// renderPrefix returns "Liquid error" or "Liquid syntax error"
// depending on the error kind.
func (e *LiquidError) renderPrefix() string {
	if e.Kind == LiquidSyntaxError {
		return "Liquid syntax error"
	}
	return "Liquid error"
}

// NewStandardError constructs a tagged standard error a Drop or filter
// can return to render "Liquid error: <message>" inline.
func NewStandardError(format string, args ...any) *LiquidError {
	return &LiquidError{Kind: LiquidStandardError, Message: fmt.Sprintf(format, args...)}
}

// NewArgumentError constructs a tagged argument error.
func NewArgumentError(format string, args ...any) *LiquidError {
	return &LiquidError{Kind: LiquidArgumentError, Message: fmt.Sprintf(format, args...)}
}

// NewSyntaxError constructs a tagged syntax error, which renders with
// the "Liquid syntax error: " prefix.
func NewSyntaxError(format string, args ...any) *LiquidError {
	return &LiquidError{Kind: LiquidSyntaxError, Message: fmt.Sprintf(format, args...)}
}
