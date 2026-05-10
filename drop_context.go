package liquid

// RenderContext is the read-only view of an in-progress render that a
// Drop can opt into receiving via the ContextAwareDrop interface. It
// exposes a useful subset of Ruby's Liquid::Context surface so a Drop
// can adapt its behavior to render state — for example, raising a
// "missing field" error only when StrictVariables is enabled, or
// charging an asset lookup against the active ResourceLimits budget.
//
// The context is bound to a single in-flight render and must not be
// retained beyond the LiquidLookup call that received it: subsequent
// renders may wire a different one.
type RenderContext interface {
	// Get reads a variable from the current scope chain. Returns the
	// value and whether the key was defined; mirrors Liquid's normal
	// variable resolution semantics, walking the scope chain outward.
	Get(name string) (any, bool)

	// StrictVariables reports whether the render was started with
	// StrictVariables(). A Drop may use this to decide whether to
	// silently return nil for unknown keys or surface an error.
	StrictVariables() bool

	// StrictFilters reports whether the render was started with
	// StrictFilters().
	StrictFilters() bool

	// ResourceLimits returns the active *ResourceLimits if one was
	// attached via WithLimits, or nil. Drops doing expensive lookups can
	// charge against assign_score to participate in the limit budget.
	ResourceLimits() *ResourceLimits

	// TemplateName returns the current template name (set via
	// WithName, or propagated from a partial). Useful for diagnostic
	// messages emitted from a Drop.
	TemplateName() string
}

// ContextAwareDrop is implemented by Drop types that need access to the
// current render context. The engine calls SetRenderContext on the drop
// just before its first LiquidLookup call within a render, mirroring
// Ruby's `Liquid::Drop#context=` setter.
//
// Implementing this interface is opt-in. A plain Drop that doesn't need
// to inspect render state can ignore it. The engine sets the context
// during variable resolution and during property/index descent; if your
// Drop returns nested Drops from LiquidLookup, propagate the context to
// them yourself.
//
// Filter limitation: a Drop that is reached only as an element of a
// collection passed through a filter (e.g. `{{ items | map: "title" }}`
// where each item is a Drop) will not have SetRenderContext invoked
// from inside the filter, because filters don't carry a RenderContext.
// Such drops will still observe the context that was set on them during
// the most recent variable lookup that materialized them, if any.
//
// Because a single Drop may receive many SetRenderContext calls during
// a render, implementations should treat the operation as a cheap
// pointer overwrite (no expensive re-initialization).
//
// CONCURRENCY: A single ContextAwareDrop must not be shared across
// concurrent Render calls. SetRenderContext is called without
// synchronization, so two concurrent renders sharing a Drop pointer
// will race on the context field. Construct a fresh Drop per render —
// or guard SetRenderContext yourself — if you need concurrent use.
type ContextAwareDrop interface {
	Drop
	SetRenderContext(ctx RenderContext)
}

// evaluator implements RenderContext.

func (e *evaluator) Get(name string) (any, bool) {
	return e.ctx.lookup(name)
}

func (e *evaluator) StrictVariables() bool { return e.cfg.strictVariables }

func (e *evaluator) StrictFilters() bool { return e.cfg.strictFilters }

func (e *evaluator) ResourceLimits() *ResourceLimits { return e.cfg.limits }

func (e *evaluator) TemplateName() string { return e.templateName }

// setDropContext installs ctx on obj if it's a ContextAwareDrop. No-op
// otherwise. Called at every Drop lookup site so nested drops returned
// from LiquidLookup also receive context as the engine descends into
// them.
func setDropContext(obj any, ctx RenderContext) {
	if ctx == nil {
		return
	}
	if d, ok := obj.(ContextAwareDrop); ok {
		d.SetRenderContext(ctx)
	}
}
