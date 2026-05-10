package liquid

// This file consolidates the Liquid Drop protocol — the set of opt-in
// interfaces a Go type can implement to control how it appears inside a
// template. Each one mirrors a specific Ruby Liquid hook.
//
// The interfaces are intentionally narrow and orthogonal:
//
//   - Drop              (LiquidLookup)     — property access, the core hook
//   - ToLiquidConverter (ToLiquid)         — domain → Liquid representation
//   - LiquidValuer      (ToLiquidValue)    — Drop → scalar at value boundaries
//   - LiquidIterator    (LiquidEach)       — Drop → element sequence
//   - ContextAwareDrop  (SetRenderContext) — render-state introspection
//
// Drop itself lives in evaluator.go for historical reasons; the rest are
// gathered here along with the small helpers (`liquify`, `liquidScalar`,
// `liquidIter`) the engine uses at every boundary to consult them.

// ToLiquidConverter implements Ruby Liquid's `to_liquid` hook. When the
// engine first observes a value implementing this interface at a data
// boundary — variable resolution, property/index descent, or filter
// input — it calls ToLiquid() and replaces the value with the result.
// The returned value can be a Drop, a primitive, a map, a slice, or
// anything the engine knows how to render or descend into.
//
// Typical use: hide a domain object behind a Liquid-safe surface that
// only exposes computed properties.
type ToLiquidConverter interface {
	ToLiquid() any
}

// LiquidValuer implements Ruby Liquid's `to_liquid_value` hook. When a
// value appears in a context that wants a scalar — output rendering,
// arithmetic, equality comparison, or truthiness — the engine first
// calls ToLiquidValue() and uses the returned primitive. Typical
// implementations return int, float64, string, bool, or nil.
//
// This lets an IntegerDrop arithmetic-compare against `5`, a
// BooleanDrop participate in `{% if %}`, etc., without leaking the
// underlying type into filter math or comparison.
type LiquidValuer interface {
	ToLiquidValue() any
}

// LiquidIterator implements Ruby Liquid's `each` hook for Drops. When a
// value appears as the collection in a {% for %} loop or as input to a
// filter that iterates (map/where/sort/etc.), the engine calls
// LiquidEach() to obtain the slice of elements. Returning nil and
// returning an empty slice are equivalent.
type LiquidIterator interface {
	LiquidEach() []any
}

// liquify converts a value at a data boundary by calling ToLiquid() if
// the value implements ToLiquidConverter. The conversion is shallow —
// callers that descend further (property/index access, iteration) are
// responsible for re-liquifying as they go, mirroring Ruby's per-access
// behavior.
func liquify(v any) any {
	if c, ok := v.(ToLiquidConverter); ok {
		return c.ToLiquid()
	}
	return v
}

// liquidScalar unwraps v to its scalar representation by calling
// ToLiquidValue() if v implements LiquidValuer. Returns v unchanged
// otherwise. Called at every scalar boundary (rendering, arithmetic,
// comparison, truthiness) so a Drop can act as its underlying value.
func liquidScalar(v any) any {
	if c, ok := v.(LiquidValuer); ok {
		return c.ToLiquidValue()
	}
	return v
}

// liquidIter returns the slice view of v if v implements LiquidIterator.
// The boolean reports whether the interface was used so callers can
// fall back to their own generic iteration (e.g. toSlice) when it
// wasn't.
func liquidIter(v any) ([]any, bool) {
	if c, ok := v.(LiquidIterator); ok {
		return c.LiquidEach(), true
	}
	return nil, false
}
