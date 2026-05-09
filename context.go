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
	if val, ok := c.vars[name]; ok {
		return val
	}
	if c.parent != nil {
		return c.parent.get(name)
	}
	return nil
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

// forloopObject holds metadata about a for loop iteration.
type forloopObject struct {
	Index   int  // 1-indexed position
	Index0  int  // 0-indexed position
	RIndex  int  // reverse 1-indexed position (length - index0)
	RIndex0 int  // reverse 0-indexed position (length - index0 - 1)
	First   bool // true if first iteration
	Last    bool // true if last iteration
	Length  int  // total number of items
}

// newForloop creates a forloop object for the given iteration.
func newForloop(index0, length int) *forloopObject {
	return &forloopObject{
		Index:   index0 + 1,
		Index0:  index0,
		RIndex:  length - index0,
		RIndex0: length - index0 - 1,
		First:   index0 == 0,
		Last:    index0 == length-1,
		Length:  length,
	}
}

// toMap converts the forloop object to a map for template access.
func (f *forloopObject) toMap() map[string]any {
	return map[string]any{
		"index":   f.Index,
		"index0":  f.Index0,
		"rindex":  f.RIndex,
		"rindex0": f.RIndex0,
		"first":   f.First,
		"last":    f.Last,
		"length":  f.Length,
	}
}
