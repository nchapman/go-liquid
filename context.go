package liquid

import "maps"

// context manages variable scopes during template evaluation.
type context struct {
	vars   map[string]any
	parent *context
}

func newContext(data map[string]any) *context {
	vars := make(map[string]any)
	maps.Copy(vars, data)
	return &context{vars: vars}
}

// push creates a new child scope. Most pushed scopes hold only a couple
// of keys (a loop variable plus `forloop`), so we leave vars nil and let
// the first set allocate a small map — saving an allocation on scopes
// that end up empty (e.g. partials that never assign).
func (c *context) push() *context {
	return &context{parent: c}
}

// get retrieves a variable from the current scope or any parent scope.
// Returns nil if not found (Liquid semantics: undefined = nil).
func (c *context) get(name string) any {
	val, _ := c.lookup(name)
	return val
}

// lookup is like get but reports whether the name was bound. Used by
// strict-variables mode to distinguish "undefined" from "explicitly nil".
func (c *context) lookup(name string) (any, bool) {
	if c.vars != nil {
		if val, ok := c.vars[name]; ok {
			return val, true
		}
	}
	if c.parent != nil {
		return c.parent.lookup(name)
	}
	return nil, false
}

// set sets a variable in the current scope.
func (c *context) set(name string, value any) {
	if c.vars == nil {
		// Most pushed scopes only need room for the loop variable and
		// `forloop`; size to that to avoid an immediate rehash.
		c.vars = make(map[string]any, 2)
	}
	c.vars[name] = value
}

// setGlobal sets a variable in the root scope.
func (c *context) setGlobal(name string, value any) {
	root := c
	for root.parent != nil {
		root = root.parent
	}
	root.vars[name] = value
}

// forloopState backs the `forloop` object exposed inside {% for %} bodies.
// It implements Drop so property access (forloop.index, forloop.first, …) is
// computed on demand rather than via a fresh 9-key map per iteration —
// allocating once per for-tag and mutating index0 each loop turn.
//
// `name` is what Shopify exposes as `forloop.name`; `parent` is the enclosing
// forloop (nil at the top level) so nested loops can walk
// {{ forloop.parentloop.parentloop.index }}.
type forloopState struct {
	index0 int
	length int
	name   string
	parent *forloopState
}

// LiquidLookup implements Drop.
func (f *forloopState) LiquidLookup(key string) (any, bool) {
	if f == nil {
		return nil, false
	}
	switch key {
	case "index":
		return f.index0 + 1, true
	case "index0":
		return f.index0, true
	case "rindex":
		return f.length - f.index0, true
	case "rindex0":
		return f.length - f.index0 - 1, true
	case "first":
		return f.index0 == 0, true
	case "last":
		return f.index0 == f.length-1, true
	case "length":
		return f.length, true
	case "name":
		return f.name, true
	case "parentloop":
		// Returning (nil, true) — "defined but no parent" — so strict-variables
		// mode does not error on {{ forloop.parentloop.index }} at the top level.
		// The nil propagates and chained property access yields nil per Liquid
		// semantics.
		if f.parent == nil {
			return nil, true
		}
		return f.parent, true
	}
	return nil, false
}

// tablerowloopState backs the `tablerowloop` Drop exposed inside
// {% tablerow %} bodies. Like forloopState, it's allocated once per tag and
// mutated each iteration to avoid per-row map allocations.
type tablerowloopState struct {
	index0   int
	length   int
	col      int
	row      int
	cols     int
	colFirst bool
	colLast  bool
}

func (t *tablerowloopState) LiquidLookup(key string) (any, bool) {
	if t == nil {
		return nil, false
	}
	switch key {
	case "length":
		return t.length, true
	case "index":
		return t.index0 + 1, true
	case "index0":
		return t.index0, true
	case "rindex":
		return t.length - t.index0, true
	case "rindex0":
		return t.length - t.index0 - 1, true
	case "col":
		return t.col, true
	case "col0":
		return t.col - 1, true
	case "row":
		return t.row, true
	case "first":
		return t.index0 == 0, true
	case "last":
		return t.index0 == t.length-1, true
	case "col_first":
		return t.colFirst, true
	case "col_last":
		return t.colLast, true
	}
	return nil, false
}
