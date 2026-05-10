package liquid

import (
	"fmt"
	"io"
)

// TagRenderer renders a single invocation of a custom tag. Implementations
// are produced by a TagParser at template-parse time and must be safe for
// concurrent use across renders — a parsed Template is shared across
// goroutines, and so is every TagRenderer it contains.
type TagRenderer interface {
	Render(w io.Writer, ctx TagContext) error
}

// TagParser turns the raw markup between `{% NAME` and `%}` into a
// TagRenderer. It runs once, when the template is parsed. For block tags,
// the body between `{% NAME %}` and `{% endNAME %}` is parsed automatically
// — the parser receives only the markup string.
//
// The markup is handed to the plugin verbatim; the library's own expression
// evaluator is not exposed at parse time. Plugins that need to interpret
// Liquid expressions (variable references, filters) should either parse
// the markup themselves with regexp/strings or store the raw text and
// resolve it at render time via TagContext.Get.
type TagParser func(markup string) (TagRenderer, error)

// TagContext is the live state surface available to a custom-tag renderer.
// All mutations are scoped to the current render and never leak into the
// parsed Template.
//
// Lifetime: a TagContext is valid only for the duration of the Render call
// that produced it. Implementations MUST NOT retain it past return, store
// it in a struct field, or hand it to a goroutine — the underlying state
// is per-render scratch and is reused across concurrent renders of the
// same parsed Template.
type TagContext interface {
	// Get reads a variable using normal Liquid lookup semantics: walk
	// the active scope chain, return nil if the name is unbound.
	Get(name string) any
	// Assign sets a variable in the innermost (current) scope.
	Assign(name string, value any)
	// PushScope runs fn inside a fresh child scope. Variables assigned
	// inside fn are dropped when it returns. Mirrors Ruby Liquid's
	// `context.stack do … end`.
	PushScope(fn func() error) error
	// RenderBody renders the block body into w. For inline tags (no
	// body) it is a no-op that returns nil, so renderers can call it
	// unconditionally.
	RenderBody(w io.Writer) error
	// RenderPartial loads the partial named name through the active
	// loader (reusing the partial cache) and renders it into w against
	// the current scope, with the same shared-scope semantics as
	// {% include %} — variables assigned in the partial leak back into
	// the caller. Returns an error if no loader is configured, the
	// partial cannot be loaded or parsed, or the partial-recursion cap
	// is exceeded.
	//
	// Mirrors Ruby Liquid's PartialCache.load + the shared-scope render
	// path used by the built-in Include tag.
	RenderPartial(w io.Writer, name string) error
	// Eval evaluates a Liquid expression against the current scope.
	// Pair with ParseExpression at parse time so a custom tag can
	// accept Liquid syntax in its arguments.
	Eval(expr Expression) (any, error)
	// StrictVariables reports whether the active Render was started
	// with the StrictVariables option. Custom tags that interpret
	// missing values should consult this instead of choosing a fixed
	// behavior.
	StrictVariables() bool
	// WithDisabledTags runs fn with the named tags disabled in the
	// current scope. While disabled, any attempt to invoke a listed
	// tag (including from within partials rendered inside fn) returns
	// ErrDisabledTag. Disables are counted, so nested calls compose.
	// Mirrors Ruby Liquid's Tag::Disabler mixin.
	WithDisabledTags(names []string, fn func() error) error
	// TagDisabled reports whether the named tag is currently disabled
	// in the active scope. Custom tags that wish to participate in the
	// disable mechanism should check this on entry and return
	// ErrDisabledTag if true. Mirrors Tag::Disableable.
	TagDisabled(name string) bool
	// Registers returns the user-supplied state map attached via
	// WithRegisters, or nil if none was attached. Use it to thread
	// per-render state (request IDs, caches, ad-hoc counters) through
	// custom tags. Mirrors Ruby Liquid's Context#registers.
	Registers() map[string]any
}

// RegisterTag installs an inline custom tag on the default environment.
// The framework treats the tag as having no body: `{% NAME ... %}`.
// Registering a name that collides with a built-in tag (`if`, `for`, etc.)
// panics. Use Environment.RegisterTag for an isolated registry.
func RegisterTag(name string, parse TagParser) {
	Default().RegisterTag(name, parse)
}

// RegisterBlock installs a block custom tag on the default environment.
// The framework consumes the body between `{% NAME ... %}` and
// `{% endNAME %}`. Panics on collision with a built-in tag.
func RegisterBlock(name string, parse TagParser) {
	Default().RegisterBlock(name, parse)
}

// guardCustomTagName rejects names that would shadow a built-in tag. We
// keep this list small and explicit rather than reflecting on parser.go
// so additions are obvious.
func guardCustomTagName(name string) {
	if name == "" {
		panic("liquid: custom tag name must not be empty")
	}
	if _, taken := builtinTagNames[name]; taken {
		panic(fmt.Sprintf("liquid: %q is a built-in tag and cannot be overridden", name))
	}
}

var builtinTagNames = map[string]struct{}{
	"if": {}, "elsif": {}, "else": {}, "endif": {},
	"unless": {}, "endunless": {},
	"case": {}, "when": {}, "endcase": {},
	"for": {}, "endfor": {}, "break": {}, "continue": {},
	"assign":  {},
	"capture": {}, "endcapture": {},
	"comment": {}, "endcomment": {},
	"raw": {}, "endraw": {},
	"cycle":     {},
	"increment": {}, "decrement": {},
	"render": {}, "include": {},
	"echo":     {},
	"liquid":   {},
	"tablerow": {}, "endtablerow": {},
	"ifchanged": {}, "endifchanged": {},
	"doc": {}, "enddoc": {},
}

// customTagNode is the AST node for an inline custom tag.
type customTagNode struct {
	name     string
	renderer TagRenderer
	line     int
	column   int
}

func (n *customTagNode) node()                   {}
func (n *customTagNode) Pos() (line, column int) { return n.line, n.column }

// customBlockNode is the AST node for a custom block tag with its parsed body.
type customBlockNode struct {
	name     string
	renderer TagRenderer
	body     []Node
	line     int
	column   int
}

func (n *customBlockNode) node()                   {}
func (n *customBlockNode) Pos() (line, column int) { return n.line, n.column }

// tagCtx adapts an evaluator + body slice to the TagContext interface.
// Stack-allocated per invocation; never escapes outside the renderer call.
type tagCtx struct {
	ev   *evaluator
	body []Node
}

func (c *tagCtx) Get(name string) any           { return c.ev.ctx.get(name) }
func (c *tagCtx) Assign(name string, value any) { c.ev.ctx.set(name, value) }

func (c *tagCtx) PushScope(fn func() error) error {
	prev := c.ev.ctx
	c.ev.ctx = prev.push()
	defer func() { c.ev.ctx = prev }()
	return fn()
}

func (c *tagCtx) RenderBody(w io.Writer) error {
	if len(c.body) == 0 {
		return nil
	}
	return c.ev.evalNodes(w, c.body)
}

func (c *tagCtx) Eval(expr Expression) (any, error) {
	return c.ev.evalExpr(expr)
}

func (c *tagCtx) StrictVariables() bool { return c.ev.cfg.strictVariables }

func (c *tagCtx) WithDisabledTags(names []string, fn func() error) error {
	return c.ev.withDisabledTags(names, fn)
}

func (c *tagCtx) TagDisabled(name string) bool { return c.ev.tagDisabled(name) }

func (c *tagCtx) Registers() map[string]any { return c.ev.cfg.userRegisters }

func (c *tagCtx) RenderPartial(w io.Writer, name string) error {
	if c.ev.partialDepth >= maxPartialDepth {
		return fmt.Errorf("partial depth exceeded %d (possible cycle in %q)", maxPartialDepth, name)
	}
	partial, err := c.ev.loadPartial(name)
	if err != nil {
		return err
	}
	c.ev.partialDepth++
	defer func() { c.ev.partialDepth-- }()
	return c.ev.evalNodes(w, partial.ast.nodes)
}
