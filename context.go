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

// push creates a new child scope.
func (c *context) push() *context {
	return &context{
		vars:   make(map[string]any),
		parent: c,
	}
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
	if val, ok := c.vars[name]; ok {
		return val, true
	}
	if c.parent != nil {
		return c.parent.lookup(name)
	}
	return nil, false
}

// set sets a variable in the current scope.
func (c *context) set(name string, value any) {
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

// newForloop returns the map exposed as `forloop` inside a {% for %} body.
// `name` is the bound iteration variable (Shopify uses this as
// `forloop.name`); `parent` is the enclosing forloop's map (nil at the top
// level). Both fields are surfaced verbatim so nested loops can walk
// {{ forloop.parentloop.parentloop.index }}.
func newForloop(index0, length int, name string, parent map[string]any) map[string]any {
	return map[string]any{
		"index":      index0 + 1,
		"index0":     index0,
		"rindex":     length - index0,
		"rindex0":    length - index0 - 1,
		"first":      index0 == 0,
		"last":       index0 == length-1,
		"length":     length,
		"name":       name,
		"parentloop": parent,
	}
}
