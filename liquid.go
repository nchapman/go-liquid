// Package liquid is a pure-Go implementation of the Liquid template language.
//
// Liquid is a safe, customer-facing template language created by Shopify and
// widely used in static site generators (Jekyll, Hugo's Liquid mode), e-commerce
// platforms, and configuration tooling. This package implements the standard
// Liquid syntax — output ({{ ... }}), tags ({% ... %}), filters, control flow,
// and loops — with full whitespace control ({%- -%}, {{- -}}).
//
// Quick start:
//
//	out, err := liquid.Render("Hello, {{ name }}!", map[string]any{"name": "World"})
//	// out == "Hello, World!"
//
// Parse once, render many times:
//
//	tmpl, err := liquid.Parse("{% for x in items %}{{ x }}{% endfor %}")
//	if err != nil { ... }
//	out, _ := tmpl.Render(map[string]any{"items": []int{1, 2, 3}})
//
// Streaming output:
//
//	tmpl.RenderTo(w, data)
//
// Data may be a map[string]any, any other map keyed by a stringable type, or a
// struct. Struct fields and zero-arg methods are accessible via dot notation.
//
// # Method auto-invocation and the Drop interface
//
// When a template accesses obj.foo on a struct value, the engine first looks
// for a Foo field, then for a zero-argument method whose return signature
// looks like a data accessor — either a single non-error return, or
// (T, error). Methods whose only return is error (and other multi-value
// shapes) are NOT invoked, on the theory that they're side-effecting
// (Close, Save, Delete) and silent execution would be unsafe. This is a
// reasonable default for data DTOs but can surprise authors who expose
// methods like Token() string or APIKey() string on a model: those WILL
// be reachable from a template. To opt out per-type, implement the Drop
// interface — LiquidLookup is consulted instead of reflection.
package liquid

import (
	"io"
	"reflect"
	"sync/atomic"
)

// Template is a parsed Liquid template. Templates are safe to render
// concurrently from multiple goroutines, including across calls to
// WithLoader (the loader pointer is updated atomically).
type Template struct {
	ast      *templateAST
	partials atomic.Pointer[partialCache]
	name     string // optional; populated when loaded via a Loader
}

// WithLoader attaches a Loader so the template can resolve {% render %} and
// {% include %} partials. Returns the template for chaining. Subsequent
// calls swap the loader atomically and reset the partial cache; in-flight
// renders observe the loader they started with.
func (t *Template) WithLoader(l Loader) *Template {
	t.partials.Store(newPartialCache(l))
	return t
}

// WithName attaches a name (typically a file path) to the template so
// parse and render errors carry it on their TemplateName field. Useful
// when reporting errors from templates loaded by name. Returns the
// template for chaining.
func (t *Template) WithName(name string) *Template {
	t.name = name
	return t
}

// Parse parses a Liquid template source string.
func Parse(source string) (*Template, error) {
	p := newParser(source)
	ast, err := p.parse()
	if err != nil {
		return nil, err
	}
	return &Template{ast: ast}, nil
}

// MustParse is like Parse but panics on error. Use for templates known at
// program start (e.g. embedded with //go:embed).
func MustParse(source string) *Template {
	t, err := Parse(source)
	if err != nil {
		panic(err)
	}
	return t
}

// RenderOption configures a single Render call. Options are applied in
// order; later options override earlier ones.
type RenderOption func(*renderOpts)

type renderOpts struct {
	strictVariables bool
	strictFilters   bool
}

// StrictVariables makes Render fail when a referenced variable is
// undefined. By default Liquid silently treats undefined as nil.
func StrictVariables() RenderOption {
	return func(o *renderOpts) { o.strictVariables = true }
}

// StrictFilters makes Render fail when a filter name does not resolve.
// By default Liquid passes the input through unchanged.
func StrictFilters() RenderOption {
	return func(o *renderOpts) { o.strictFilters = true }
}

// Render executes the template against data and returns the rendered string.
// Pass StrictVariables and/or StrictFilters to opt into strict semantics.
func (t *Template) Render(data any, opts ...RenderOption) (string, error) {
	var o renderOpts
	for _, opt := range opts {
		opt(&o)
	}
	eval := newEvaluator(toStringMap(data))
	eval.cfg.partials = t.partials.Load()
	eval.cfg.strictVariables = o.strictVariables
	eval.cfg.strictFilters = o.strictFilters
	eval.templateName = t.name
	return eval.evaluate(t.ast)
}

// RenderTo executes the template against data and writes the result to w.
func (t *Template) RenderTo(w io.Writer, data any, opts ...RenderOption) error {
	out, err := t.Render(data, opts...)
	if err != nil {
		return err
	}
	_, err = io.WriteString(w, out)
	return err
}

// Render parses and executes a template in one call. For templates rendered
// repeatedly, prefer Parse + (*Template).Render to avoid re-parsing.
func Render(source string, data any, opts ...RenderOption) (string, error) {
	t, err := Parse(source)
	if err != nil {
		return "", err
	}
	return t.Render(data, opts...)
}

// MustRender is like Render but panics on parse or render error.
func MustRender(source string, data any, opts ...RenderOption) string {
	out, err := Render(source, data, opts...)
	if err != nil {
		panic(err)
	}
	return out
}

// RegisterFilter installs a custom filter under the given name. It
// overrides any built-in filter with the same name. Not safe to call
// concurrently with rendering.
func RegisterFilter(name string, fn FilterFunc) {
	filters[name] = fn
}

// RegisterKwargFilter installs a custom filter that accepts named
// arguments via kwargs. Use this when your filter takes options like
// `{{ x | my_filter: group_by: "name", limit: 10 }}`. Not safe to call
// concurrently with rendering. Overrides any built-in filter with the
// same name.
func RegisterKwargFilter(name string, fn KwargFilterFunc) {
	kwargFilters[name] = fn
}

// toStringMap normalizes user-provided data into the map[string]any shape the
// evaluator expects at the top level. Structs are passed through — getProperty
// handles them via reflection during evaluation.
func toStringMap(data any) map[string]any {
	if data == nil {
		return map[string]any{}
	}
	if m, ok := data.(map[string]any); ok {
		return m
	}
	rv := reflect.ValueOf(data)
	for rv.Kind() == reflect.Pointer {
		rv = rv.Elem()
	}
	switch rv.Kind() {
	case reflect.Map:
		out := make(map[string]any, rv.Len())
		iter := rv.MapRange()
		for iter.Next() {
			out[toString(iter.Key().Interface())] = iter.Value().Interface()
		}
		return out
	case reflect.Struct:
		out := make(map[string]any, rv.NumField())
		t := rv.Type()
		for i := 0; i < rv.NumField(); i++ {
			f := t.Field(i)
			if !f.IsExported() {
				continue
			}
			name := f.Name
			if tag := f.Tag.Get("liquid"); tag != "" {
				name = tag
			}
			out[name] = rv.Field(i).Interface()
		}
		return out
	}
	return map[string]any{}
}
