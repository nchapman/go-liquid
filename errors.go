package liquid

import "fmt"

// ParseError is returned when a template fails to parse. It carries the
// source position so callers can pinpoint the offending tag, and an
// optional TemplateName that hosts may set on partials so error messages
// say which file blew up.
type ParseError struct {
	Message      string
	Line         int
	Column       int
	TemplateName string // optional; set on partials by the loader
}

func (e *ParseError) Error() string {
	if e.TemplateName != "" {
		return fmt.Sprintf("parse error in %q at line %d, column %d: %s",
			e.TemplateName, e.Line, e.Column, e.Message)
	}
	return fmt.Sprintf("parse error at line %d, column %d: %s", e.Line, e.Column, e.Message)
}

func newParseError(line, column int, format string, args ...any) *ParseError {
	return &ParseError{
		Message: fmt.Sprintf(format, args...),
		Line:    line,
		Column:  column,
	}
}

// RenderError wraps an error that occurred during template rendering
// together with the source position of the AST node that triggered it.
// Callers can use errors.As to extract it and report a line/column-anchored
// message, or errors.Unwrap to get the underlying cause. TemplateName is
// optional and surfaces which partial the error came from when present.
type RenderError struct {
	Line         int
	Column       int
	TemplateName string
	Inner        error
}

func (e *RenderError) Error() string {
	if e.TemplateName != "" {
		return fmt.Sprintf("render error in %q at line %d, column %d: %s",
			e.TemplateName, e.Line, e.Column, e.Inner.Error())
	}
	return fmt.Sprintf("render error at line %d, column %d: %s", e.Line, e.Column, e.Inner.Error())
}

func (e *RenderError) Unwrap() error { return e.Inner }

// wrapAtNode attaches a source position to err if it isn't already a
// RenderError. Used at evaluation boundaries (output, tag) so the deepest
// originating site is preserved through nested calls. templateName is
// optional and surfaces in the resulting RenderError when non-empty.
func wrapAtNode(n Node, err error, templateName string) error {
	if err == nil {
		return nil
	}
	if _, ok := err.(*RenderError); ok {
		return err
	}
	line, col := n.Pos()
	return &RenderError{Line: line, Column: col, TemplateName: templateName, Inner: err}
}
