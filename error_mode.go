package liquid

import "fmt"

// ErrorMode controls how the parser handles non-fatal anomalies — most
// notably an unrecognized tag name. Set per-Environment via
// Environment.WithErrorMode; defaults to ErrorModeStrict.
//
// Mirrors Ruby Liquid's parse-time error_mode (:strict, :warn, :lax),
// minus :strict2 which has no current go-liquid analogue (the Go
// parser is already strict about expression syntax everywhere).
type ErrorMode int

const (
	// ErrorModeStrict (the default) fails parsing on any anomaly. An
	// unknown tag returns a parse error.
	//
	// Note: Ruby Liquid defaults to :warn instead. Go-liquid prefers
	// fail-fast as its default since it matches Go conventions.
	ErrorModeStrict ErrorMode = iota
	// ErrorModeWarn collects a Warning instead of failing. The
	// offending source span is rendered as literal text so output
	// roughly resembles the input. Inspect Template.Warnings() after
	// parsing.
	ErrorModeWarn
	// ErrorModeLax silently degrades — the offending source span is
	// rendered as literal text and no Warning is recorded. Use sparingly:
	// it hides bugs.
	ErrorModeLax
)

func (m ErrorMode) String() string {
	switch m {
	case ErrorModeStrict:
		return "strict"
	case ErrorModeWarn:
		return "warn"
	case ErrorModeLax:
		return "lax"
	}
	return fmt.Sprintf("ErrorMode(%d)", int(m))
}

// Warning describes a non-fatal anomaly recorded during parse when
// the active ErrorMode is ErrorModeWarn. Call Template.Warnings()
// after Parse to inspect them.
type Warning struct {
	Message      string
	Line, Column int
	TemplateName string
}

func (w Warning) String() string {
	if w.TemplateName != "" {
		return fmt.Sprintf("%s:%d:%d: %s", w.TemplateName, w.Line, w.Column, w.Message)
	}
	return fmt.Sprintf("line %d, column %d: %s", w.Line, w.Column, w.Message)
}
