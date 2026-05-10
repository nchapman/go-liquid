package liquid

import (
	"errors"
	"fmt"
	"io"
	"reflect"
	"slices"
	"strconv"
	"strings"
)

// maxPartialDepth caps recursion through {% render %} and {% include %} so
// a self-referential or mutually-recursive partial returns an error rather
// than overflowing the goroutine stack.
const maxPartialDepth = 100

// maxRangeSize bounds eager expansion of {% for i in (a..b) %} so that
// attacker-controlled endpoints cannot trigger an unbounded allocation.
const maxRangeSize = 1_000_000

// renderConfig is the immutable-per-render configuration: the loader's
// partial cache plus the strict-mode flags. Inherited verbatim across
// {% render %} and {% include %} so a partial sees the same loader and
// strictness as the root template.
type renderConfig struct {
	env             *Environment
	partials        *partialCache
	strictVariables bool
	strictFilters   bool
	limits          *ResourceLimits
	// userRegisters is the caller-supplied state bag attached via
	// WithRegisters. Shared by reference with partials so mutations made
	// inside a custom tag are visible to the caller after Render returns.
	// Mirrors Ruby Liquid's Context#registers.
	userRegisters map[string]any
}

// registers holds per-render scratch state — the Shopify-Liquid concept
// of context.registers. {% include %} shares the caller's registers (so
// {% cycle %} positions, {% increment %}/{% decrement %} counters, and
// the {% ifchanged %} slot all propagate). {% render %} starts with a
// fresh set: an isolated partial cannot mutate caller state.
//
// partialDepth is intentionally NOT a register: it tracks recursion depth
// for the cycle-detection cap and must propagate across both shared
// (include) and isolated (render) boundaries, so it lives on the
// evaluator directly.
type registers struct {
	cycle         map[string]int
	counter       map[string]int
	ifchangedLast string
	ifchangedSet  bool
	// forContinue tracks the next index a `for ... offset: continue` loop
	// should resume from. Ruby keys this by "{var}-{collection}", so two
	// for-tags walking the same collection with the same loop variable
	// share a cursor — that's exactly the pagination shape.
	forContinue map[string]int
}

func newRegisters() *registers {
	return &registers{
		cycle:       map[string]int{},
		counter:     map[string]int{},
		forContinue: map[string]int{},
	}
}

// evaluator executes a parsed template. cfg is per-Render and shared
// with partials; regs is per-evaluator (fresh for {% render %}, shared
// for {% include %}); ctx is the live scope chain. disabledTags is a
// counter map keyed by tag name — non-zero means the tag is disabled in
// the current scope. Shared by pointer with sub-evaluators so that
// {% render %}'s disable of {% include %} survives partial boundaries,
// matching Ruby's Context#with_disabled_tags semantics.
type evaluator struct {
	cfg          *renderConfig
	regs         *registers
	ctx          *context
	disabledTags map[string]int
	templateName string // optional, surfaced on RenderError; varies per partial
	partialDepth int
}

func newEvaluator(data map[string]any) *evaluator {
	return &evaluator{
		cfg:          &renderConfig{env: Default()},
		regs:         newRegisters(),
		ctx:          newContext(data),
		disabledTags: map[string]int{},
	}
}

// withDisabledTags increments the disable counter for each name in names,
// runs fn, then decrements. Counters mean nested disables compose: a tag
// can disable "include" while the caller has also disabled it, and the
// outer disable survives the inner block's exit.
func (e *evaluator) withDisabledTags(names []string, fn func() error) error {
	for _, name := range names {
		e.disabledTags[name]++
	}
	defer func() {
		for _, name := range names {
			e.disabledTags[name]--
		}
	}()
	return fn()
}

// tagDisabled reports whether the named tag is currently disabled.
func (e *evaluator) tagDisabled(name string) bool {
	return e.disabledTags[name] > 0
}

// evaluate writes the template's rendered output to w.
func (e *evaluator) evaluate(w io.Writer, tmpl *templateAST) error {
	err := e.evalNodes(w, tmpl.nodes)
	// `{% break %}` / `{% continue %}` outside any loop are no-ops at the top
	// level: break short-circuits the rest of the template (output already
	// written is preserved), continue is silently swallowed. Matches upstream.
	if errors.Is(err, errBreak) || errors.Is(err, errContinue) {
		return nil
	}
	return err
}

// evalNodes renders each node into w in order, propagating the first
// error (or break/continue control signal) back to the caller.
func (e *evaluator) evalNodes(w io.Writer, nodes []Node) error {
	if e.cfg.limits != nil {
		if err := e.cfg.limits.incrementRenderScore(len(nodes)); err != nil {
			return err
		}
	}
	for _, node := range nodes {
		if err := e.evalNode(w, node); err != nil {
			return err
		}
	}
	return nil
}

// evalNodeToString is the scratch-buffer escape hatch for tags whose
// output must be inspected before being emitted: {% capture %} stores
// the result in a variable; {% ifchanged %} compares it against the
// last emission; tablerow wraps each cell's body in <td>...</td>.
func (e *evaluator) evalNodeToString(nodes []Node) (string, error) {
	var sb strings.Builder
	if err := e.evalNodes(&sb, nodes); err != nil {
		return sb.String(), err
	}
	return sb.String(), nil
}

func (e *evaluator) evalNode(w io.Writer, node Node) error {
	err := e.evalNodeInner(w, node)
	// Anchor the error to this node's position if the inner call returned a
	// bare error; deeper sites (filter, partial) already wrap at finer
	// positions and wrapAtNode is a no-op for *RenderError. errBreak and
	// errContinue are control-flow signals, not errors — leave them alone.
	// Use errors.Is so a future wrapper can't accidentally swallow them.
	if err != nil && !errors.Is(err, errBreak) && !errors.Is(err, errContinue) {
		err = wrapAtNode(node, err, e.templateName)
	}
	return err
}

// evalNodeInner dispatches to the per-tag evaluator for a single AST node.
// The width is intrinsic to the Liquid spec — every tag type appears here as
// a single case — and splitting it into per-tag dispatch tables would just
// trade locality for indirection. The evaluator hot path benefits from
// keeping the switch flat.
//
//nolint:gocyclo,cyclop,funlen // Tag-type dispatch; width matches the Liquid spec.
func (e *evaluator) evalNodeInner(w io.Writer, node Node) error {
	switch n := node.(type) {
	case *TextNode:
		_, err := io.WriteString(w, n.Text)
		return err

	case *OutputNode:
		if n.Expr == nil {
			return nil // {{}} renders the empty string
		}
		val, err := e.evalExpr(n.Expr)
		if err != nil {
			return err
		}
		_, err = io.WriteString(w, toString(val))
		return err

	case *IfTag:
		return e.evalIfTag(w, n)

	case *UnlessTag:
		return e.evalUnlessTag(w, n)

	case *CaseTag:
		return e.evalCaseTag(w, n)

	case *ForTag:
		return e.evalForTag(w, n)

	case *BreakTag:
		return errBreak

	case *ContinueTag:
		return errContinue

	case *AssignTag:
		val, err := e.evalExpr(n.Value)
		if err != nil {
			return err
		}
		e.ctx.setGlobal(n.Variable, val)
		if e.cfg.limits != nil {
			if err := e.cfg.limits.incrementAssignScore(assignScoreOf(val)); err != nil {
				return err
			}
		}
		return nil

	case *CaptureTag:
		var sb strings.Builder
		var cw io.Writer = &sb
		if e.cfg.limits != nil {
			cw = &captureWriter{w: &sb, l: e.cfg.limits}
		}
		if err := e.evalNodes(cw, n.Body); err != nil {
			return err
		}
		e.ctx.setGlobal(n.Variable, sb.String())
		return nil

	case *CommentTag:
		return nil

	case *RawTag:
		_, err := io.WriteString(w, n.Content)
		return err

	case *CycleTag:
		return e.evalCycleTag(w, n)

	case *IncrementTag:
		return e.evalIncrementTag(w, n)

	case *DecrementTag:
		return e.evalDecrementTag(w, n)

	case *RenderTag:
		return e.evalRenderTag(w, n)

	case *IncludeTag:
		return e.evalIncludeTag(w, n)

	case *LiquidTag:
		return e.evalNodes(w, n.Body)

	case *TablerowTag:
		return e.evalTablerowTag(w, n)

	case *IfchangedTag:
		return e.evalIfchangedTag(w, n)

	case *DocTag:
		return nil

	case *customTagNode:
		return n.renderer.Render(w, &tagCtx{ev: e})

	case *customBlockNode:
		return n.renderer.Render(w, &tagCtx{ev: e, body: n.body})

	default:
		return nil
	}
}

// evalRenderTag evaluates {% render %} in an isolated scope. Only explicitly
// bound variables are visible to the partial; the partial cannot see or
// modify caller variables. {% include %} is disabled inside a {% render %}
// to match Ruby's `disable_tags "include"` declaration on Tags::Render.
func (e *evaluator) evalRenderTag(w io.Writer, tag *RenderTag) error {
	return e.withDisabledTags([]string{"include"}, func() error {
		return e.evalRenderTagInner(w, tag)
	})
}

func (e *evaluator) evalRenderTagInner(w io.Writer, tag *RenderTag) error {
	partial, err := e.loadPartial(tag.Template)
	if err != nil {
		return err
	}

	// Build the data map the partial sees. `with` is bound first so that
	// named args with the same key explicitly override it.
	data := make(map[string]any)
	if tag.With != nil {
		val, err := e.evalExpr(tag.With)
		if err != nil {
			return err
		}
		// Skip nil bindings so the partial's `default:` filter applies, matching
		// Shopify Liquid's behavior for unset `with` operands.
		if val != nil {
			data[partialAlias(tag.WithAlias, tag.Template)] = val
		}
	}
	args, err := e.evalNamedArgs(tag.Args)
	if err != nil {
		return err
	}
	for k, v := range args {
		data[k] = v
	}

	// `for collection [as alias]`: render once per item with forloop.
	if tag.For != nil {
		coll, err := e.evalExpr(tag.For)
		if err != nil {
			return err
		}
		items := toSlice(coll)
		alias := partialAlias(tag.ForAlias, tag.Template)
		length := len(items)
		// Reuse one itemData map and one forloopState across iterations:
		// renderPartialIsolated copies into a fresh context, so the partial
		// can't observe later mutations.
		itemData := make(map[string]any, len(data)+2)
		for k, v := range data {
			itemData[k] = v
		}
		// render is isolated → parentloop is nil. Shopify's render.rb sets
		// forloop.name to the partial's template name (not the iteration
		// alias), so {% render "card" for items as item %} exposes
		// forloop.name == "card".
		fl := &forloopState{length: length, name: tag.Template}
		itemData["forloop"] = fl
		for i, item := range items {
			fl.index0 = i
			itemData[alias] = item
			if err := e.renderPartialIsolated(w, partial, itemData); err != nil {
				return err
			}
		}
		return nil
	}

	return e.renderPartialIsolated(w, partial, data)
}

// evalIncludeTag is the legacy form: shares the parent scope. Variables
// assigned in the partial leak into the caller.
func (e *evaluator) evalIncludeTag(w io.Writer, tag *IncludeTag) error {
	if e.tagDisabled("include") {
		return fmt.Errorf("%w: include is disabled inside {%% render %%}", ErrDisabledTag)
	}
	partial, err := e.loadPartial(tag.Template)
	if err != nil {
		return err
	}
	if e.partialDepth >= maxPartialDepth {
		return fmt.Errorf("partial depth exceeded %d (possible cycle in %q)", maxPartialDepth, tag.Template)
	}
	e.partialDepth++
	defer func() { e.partialDepth-- }()

	if tag.With != nil {
		val, err := e.evalExpr(tag.With)
		if err != nil {
			return err
		}
		if val != nil {
			e.ctx.set(partialAlias(tag.WithAlias, tag.Template), val)
		}
	}
	args, err := e.evalNamedArgs(tag.Args)
	if err != nil {
		return err
	}
	for k, v := range args {
		e.ctx.set(k, v)
	}

	if tag.For != nil {
		coll, err := e.evalExpr(tag.For)
		if err != nil {
			return err
		}
		items := toSlice(coll)
		alias := partialAlias(tag.ForAlias, tag.Template)
		length := len(items)
		// include shares parent scope, so the enclosing forloop (if any)
		// becomes parentloop. The forloop slot is re-set each iteration
		// because a nested {% include … for … %} inside the partial would
		// also write to this same shared scope and clobber our pointer.
		parent, _ := e.ctx.get("forloop").(*forloopState)
		fl := &forloopState{length: length, name: alias, parent: parent}
		for i, item := range items {
			fl.index0 = i
			e.ctx.set(alias, item)
			e.ctx.set("forloop", fl)
			if err := e.evalNodes(w, partial.ast.nodes); err != nil {
				return err
			}
		}
		return nil
	}

	return e.evalNodes(w, partial.ast.nodes)
}

func (e *evaluator) loadPartial(name string) (*Template, error) {
	if e.cfg.partials == nil {
		return nil, fmt.Errorf("no loader configured: cannot resolve partial %q", name)
	}
	return e.cfg.partials.get(name)
}

// computeForloopName approximates Shopify's `forloop.name`
// ("{var}-{collection}"). We synthesize the collection portion from the
// AST since there is no source-text reference; identifiers and ranges
// produce stable names, and other expressions fall back to the variable
// name alone. Result is cached on ForTag.LoopName at parse time.
func computeForloopName(variable string, collection Expression) string {
	switch c := collection.(type) {
	case *IdentExpr:
		return variable + "-" + c.Name
	case *DotExpr:
		if obj, ok := c.Object.(*IdentExpr); ok {
			return variable + "-" + obj.Name + "." + c.Property
		}
		return variable + "-" + c.Property
	case *RangeExpr:
		return variable + "-(range)"
	}
	return variable
}

// partialAlias returns the explicit alias if non-empty, otherwise the
// basename of the template path. Shopify Liquid binds `with`/`for` to the
// basename so that {% render "shared/card" with x %} exposes `card`, not
// `shared/card`.
func partialAlias(explicit, template string) string {
	if explicit != "" {
		return explicit
	}
	if i := strings.LastIndexByte(template, '/'); i >= 0 {
		return template[i+1:]
	}
	return template
}

func (e *evaluator) evalNamedArgs(args []NamedArg) (map[string]any, error) {
	out := make(map[string]any, len(args))
	for _, a := range args {
		v, err := e.evalExpr(a.Value)
		if err != nil {
			return nil, err
		}
		out[a.Name] = v
	}
	return out, nil
}

// renderPartialIsolated runs the partial with its own evaluator (no shared
// scope, fresh registers) and writes the result to w. The renderConfig is
// shared by pointer so the partial sees the same loader and strict-mode
// flags; the depth counter is inherited and incremented so the recursion
// cap still applies.
func (e *evaluator) renderPartialIsolated(w io.Writer, partial *Template, data map[string]any) error {
	if e.partialDepth >= maxPartialDepth {
		return fmt.Errorf("partial depth exceeded %d (possible cycle)", maxPartialDepth)
	}
	sub := &evaluator{
		cfg:          e.cfg,
		regs:         newRegisters(),
		ctx:          newContext(data),
		disabledTags: e.disabledTags, // shared by reference; counter increments survive the boundary
		templateName: partial.name,
		partialDepth: e.partialDepth + 1,
	}
	return sub.evaluate(w, partial.ast)
}

func (e *evaluator) evalIfTag(w io.Writer, tag *IfTag) error {
	cond, err := e.evalExpr(tag.Condition)
	if err != nil {
		return err
	}

	if toBool(cond) {
		return e.evalNodes(blankWriter(w, tag.ThenBranch), tag.ThenBranch)
	}

	for _, elsif := range tag.ElsifBranches {
		cond, err := e.evalExpr(elsif.Condition)
		if err != nil {
			return err
		}
		if toBool(cond) {
			return e.evalNodes(blankWriter(w, elsif.Body), elsif.Body)
		}
	}

	if tag.ElseBranch != nil {
		return e.evalNodes(blankWriter(w, tag.ElseBranch), tag.ElseBranch)
	}

	return nil
}

func (e *evaluator) evalUnlessTag(w io.Writer, tag *UnlessTag) error {
	cond, err := e.evalExpr(tag.Condition)
	if err != nil {
		return err
	}

	if !toBool(cond) {
		return e.evalNodes(blankWriter(w, tag.Body), tag.Body)
	}

	if tag.ElseBranch != nil {
		return e.evalNodes(blankWriter(w, tag.ElseBranch), tag.ElseBranch)
	}

	return nil
}

func (e *evaluator) evalCaseTag(w io.Writer, tag *CaseTag) error {
	value, err := e.evalExpr(tag.Value)
	if err != nil {
		return err
	}

	for _, when := range tag.Whens {
		for _, whenVal := range when.Values {
			v, err := e.evalExpr(whenVal)
			if err != nil {
				return err
			}
			if equal(value, v) {
				return e.evalNodes(blankWriter(w, when.Body), when.Body)
			}
		}
	}

	if tag.Else != nil {
		return e.evalNodes(blankWriter(w, tag.Else), tag.Else)
	}

	return nil
}

func (e *evaluator) evalForTag(w io.Writer, tag *ForTag) error {
	items, err := e.collectForItems(tag)
	if err != nil {
		return err
	}
	if len(items) == 0 {
		if tag.ElseBody != nil {
			return e.evalNodes(blankWriter(w, tag.ElseBody), tag.ElseBody)
		}
		return nil
	}

	off, err := e.resolveForOffset(tag)
	if err != nil {
		return err
	}
	switch {
	case off >= len(items):
		items = nil
	case off > 0:
		items = items[off:]
	}

	if tag.Limit != nil {
		lim, err := e.evalForInt(tag.Limit)
		if err != nil {
			return err
		}
		if lim > 0 && lim < len(items) {
			items = items[:lim]
		}
	}

	if tag.Reversed {
		// Clone before reversing: items may share a backing array with the
		// caller-supplied slice (toSlice returns []any by reference), and an
		// in-place reverse would corrupt the caller's data — visible on a
		// second render of the same template against the same context.
		items = slices.Clone(items)
		slices.Reverse(items)
	}

	if len(items) == 0 {
		if tag.ElseBody != nil {
			return e.evalNodes(blankWriter(w, tag.ElseBody), tag.ElseBody)
		}
		return nil
	}

	contKey := tag.LoopName

	// Capture the enclosing forloop (if any) BEFORE pushing a new scope so
	// the new forloop.parentloop reflects the outer iteration. Shopify's
	// `forloop.name` is "{var}-{collection}"; we render the collection
	// expression source where possible, falling back to the variable name.
	parent, _ := e.ctx.get("forloop").(*forloopState)
	loopName := tag.LoopName

	e.ctx = e.ctx.push()
	defer func() { e.ctx = e.ctx.parent }()

	length := len(items)
	consumed := 0
	// Allocate one forloopState per for-tag and mutate index0 each pass; the
	// re-set in the loop guards against an inner {% include … for … %} that
	// shares scope and would otherwise leave a stale pointer in our slot.
	fl := &forloopState{length: length, name: loopName, parent: parent}

	bw := blankWriter(w, tag.Body)
	for i, item := range items {
		fl.index0 = i
		e.ctx.set(tag.Variable, item)
		e.ctx.set("forloop", fl)

		// Body output streams directly to w as it goes. {% break %} and
		// {% continue %} do not unwind output already written this
		// iteration — matching the pre-Writer behavior, where evalNodes
		// returned (partial-output, errBreak) and the caller appended
		// the partial output before breaking.
		err := e.evalNodes(bw, tag.Body)
		consumed = i + 1
		if errors.Is(err, errBreak) {
			break
		}
		if errors.Is(err, errContinue) {
			continue
		}
		if err != nil {
			return err
		}
	}

	// Update the continue cursor so a sibling `for ... offset: continue`
	// resumes from the absolute position in the original collection.
	// Always record (even when this loop didn't use OffsetContinue) so a
	// later loop reading the same collection can resume past us.
	e.regs.forContinue[contKey] = off + consumed

	return nil
}

// collectForItems evaluates tag.Collection and produces the iteration slice,
// expanding range expressions and applying Liquid's "non-empty string is a
// single item; empty string is non-iterable" rule.
func (e *evaluator) collectForItems(tag *ForTag) ([]any, error) {
	if rangeExpr, ok := tag.Collection.(*RangeExpr); ok {
		return e.materializeRange(rangeExpr)
	}
	collection, err := e.evalExpr(tag.Collection)
	if err != nil {
		return nil, err
	}
	if s, ok := collection.(string); ok {
		if s == "" {
			return nil, nil
		}
		return []any{s}, nil
	}
	return toSlice(collection), nil
}

// materializeRange expands a RangeExpr into a concrete []any. The slice is
// materialized eagerly so that limit/offset/reversed apply uniformly; the
// range size is capped to keep attacker-controlled endpoints from
// exhausting memory.
func (e *evaluator) materializeRange(rangeExpr *RangeExpr) ([]any, error) {
	start, err := e.evalForInt(rangeExpr.Start)
	if err != nil {
		return nil, err
	}
	end, err := e.evalForInt(rangeExpr.End)
	if err != nil {
		return nil, err
	}
	size := end - start
	if size < 0 {
		size = -size
	}
	if size+1 > maxRangeSize {
		return nil, fmt.Errorf("for range size %d exceeds limit %d", size+1, maxRangeSize)
	}
	items := make([]any, 0, size+1)
	if start <= end {
		for i := start; i <= end; i++ {
			items = append(items, i)
		}
	} else {
		for i := start; i >= end; i-- {
			items = append(items, i)
		}
	}
	return items, nil
}

// resolveForOffset returns the iteration start offset, drawing from either
// the saved continue cursor or the explicit `offset:` operand.
//
// `offset: continue` resumes from where the previous for-tag iterating the
// same {var,collection} stopped. The cursor lives in per-Render registers
// keyed by tag.LoopName so distinct loops over the same collection (or
// distinct collections under the same variable name) get independent cursors.
func (e *evaluator) resolveForOffset(tag *ForTag) (int, error) {
	switch {
	case tag.OffsetContinue:
		return e.regs.forContinue[tag.LoopName], nil
	case tag.Offset != nil:
		return e.evalForInt(tag.Offset)
	}
	return 0, nil
}

// evalForInt evaluates expr and converts the result to a Go int via
// toInt(toNumber(...)). Used for {% for %}'s integer operands (range
// endpoints, offset, limit).
func (e *evaluator) evalForInt(expr Expression) (int, error) {
	v, err := e.evalExpr(expr)
	if err != nil {
		return 0, err
	}
	return int(toInt(toNumber(v))), nil
}

func (e *evaluator) evalExpr(expr Expression) (any, error) {
	switch x := expr.(type) {
	case *IdentExpr:
		val := e.ctx.get(x.Name)
		if val == nil && e.cfg.strictVariables {
			if _, ok := e.ctx.lookup(x.Name); !ok {
				return nil, wrapAtNode(x, fmt.Errorf("%w %q", ErrUndefinedVariable, x.Name), e.templateName)
			}
		}
		// A top-level Drop reached without a property descent (e.g.
		// `{% if drop %}` or `{{ drop }}`) still needs context — the
		// usual setDropContext fires inside getPropertyOK, which we
		// haven't called yet.
		setDropContext(val, e)
		return val, nil

	case *LiteralExpr:
		return x.Value, nil

	case *DotExpr:
		obj, err := e.evalExpr(x.Object)
		if err != nil {
			return nil, err
		}
		val, ok := getPropertyOK(obj, x.Property, e)
		if !ok && e.cfg.strictVariables {
			return nil, wrapAtNode(x, fmt.Errorf("%w: property %q", ErrUndefinedVariable, x.Property), e.templateName)
		}
		return val, nil

	case *IndexExpr:
		obj, err := e.evalExpr(x.Object)
		if err != nil {
			return nil, err
		}
		idx, err := e.evalExpr(x.Index)
		if err != nil {
			return nil, err
		}
		val, ok := getIndexOK(obj, idx, e)
		if !ok && e.cfg.strictVariables {
			return nil, wrapAtNode(x, fmt.Errorf("%w: index %v", ErrUndefinedVariable, idx), e.templateName)
		}
		return val, nil

	case *FilterExpr:
		return e.evalFilterExpr(x)

	case *BinaryExpr:
		return e.evalBinaryExpr(x)

	case *RangeExpr:
		// Ranges are evaluated in the context of for loops
		// Return empty slice here as they're handled specially in evalForTag
		return []any{}, nil

	default:
		// Unknown expression types return empty string (Liquid semantics)
		return "", nil
	}
}

// evalFilterExpr evaluates `input | name: arg, kw: val` by dispatching through
// the unified Filter interface. Positional filters that don't care about
// kwargs ignore them via FilterFunc.Apply, matching Shopify's "extra hash arg
// is a no-op" semantics. Unknown filters either error (strictFilters) or
// pass input through (lax mode).
func (e *evaluator) evalFilterExpr(x *FilterExpr) (any, error) {
	input, err := e.evalExpr(x.Input)
	if err != nil {
		return nil, err
	}
	args, err := e.evalExprList(x.Args)
	if err != nil {
		return nil, err
	}
	kwargs, err := e.evalNamedExprs(x.Kwargs)
	if err != nil {
		return nil, err
	}

	if fn, ok := e.cfg.env.lookupFilter(x.Name); ok {
		out, err := fn.Apply(input, args, kwargs)
		if err != nil {
			return nil, wrapAtNode(x, err, e.templateName)
		}
		return out, nil
	}
	if e.cfg.strictFilters {
		return nil, wrapAtNode(x, fmt.Errorf("%w %q", ErrUndefinedFilter, x.Name), e.templateName)
	}
	return input, nil
}

// evalExprList evaluates a positional argument list. Returns nil for an
// empty input so callers can pass directly to filter Apply without an
// allocated empty slice.
func (e *evaluator) evalExprList(exprs []Expression) ([]any, error) {
	if len(exprs) == 0 {
		return nil, nil
	}
	out := make([]any, len(exprs))
	for i, expr := range exprs {
		v, err := e.evalExpr(expr)
		if err != nil {
			return nil, err
		}
		out[i] = v
	}
	return out, nil
}

// evalNamedExprs evaluates a list of `name: value` pairs into a map. Returns
// nil for an empty input.
func (e *evaluator) evalNamedExprs(kvs []NamedArg) (map[string]any, error) {
	if len(kvs) == 0 {
		return nil, nil
	}
	out := make(map[string]any, len(kvs))
	for _, kv := range kvs {
		v, err := e.evalExpr(kv.Value)
		if err != nil {
			return nil, err
		}
		out[kv.Name] = v
	}
	return out, nil
}

// evalBinaryExpr dispatches over the binary-operator set. Logical operators
// short-circuit and preserve operand values; the rest evaluate both sides
// and delegate to a per-operator helper.
func (e *evaluator) evalBinaryExpr(expr *BinaryExpr) (any, error) {
	// Logical operators short-circuit AND preserve operand values:
	//
	//   `a or b`  → a if truthy, else b (verbatim)
	//   `a and b` → a if falsy,  else b (verbatim)
	//
	// Matches Shopify (and Ruby/JS). Conditional contexts wrap the result
	// in toBool, so {% if a or b %} still does the right thing; the value-
	// preservation matters when a chain is captured ({% assign x = a or b
	// %}) or compared.
	if expr.Operator == "and" || expr.Operator == "or" {
		left, err := e.evalExpr(expr.Left)
		if err != nil {
			return nil, err
		}
		leftBool := toBool(left)
		if expr.Operator == "and" && !leftBool {
			return left, nil
		}
		if expr.Operator == "or" && leftBool {
			return left, nil
		}
		return e.evalExpr(expr.Right)
	}

	left, err := e.evalExpr(expr.Left)
	if err != nil {
		return nil, err
	}
	right, err := e.evalExpr(expr.Right)
	if err != nil {
		return nil, err
	}

	switch expr.Operator {
	case "==":
		return equal(left, right), nil
	case "!=":
		return !equal(left, right), nil
	case "<", ">", "<=", ">=":
		return evalRelational(expr.Operator, left, right), nil
	case "contains":
		return contains(left, right), nil
	default:
		// Unknown operators return false (Liquid semantics)
		return false, nil
	}
}

// evalRelational implements <, >, <=, >=. Comparisons against nil are
// always false in upstream Liquid (incompatible types).
func evalRelational(op string, left, right any) bool {
	if left == nil || right == nil {
		return false
	}
	c := compare(left, right)
	switch op {
	case "<":
		return c < 0
	case ">":
		return c > 0
	case "<=":
		return c <= 0
	default: // ">="
		return c >= 0
	}
}

// Drop opts a Go type into Liquid's "drop" model: the type controls which
// properties templates can access. When a value passed to Render
// implements Drop, property lookups (e.g. {{ obj.x }} or {{ obj["x"] }})
// route through LiquidLookup instead of reflection on fields/methods.
//
// Use Drop when you want a Go object to expose computed properties to
// templates without auto-dispatching every public zero-arg method.
type Drop interface {
	// LiquidLookup returns the value for the given key and reports whether
	// the key is defined. Returning (nil, true) is meaningful: it means
	// "defined but no value" and prevents strict-variables errors.
	LiquidLookup(key string) (any, bool)
}

// errType is reflect.TypeOf((*error)(nil)).Elem(), used to detect
// error-typed return values from struct methods.
var errType = reflect.TypeOf((*error)(nil)).Elem()

// callTemplateMethod invokes a zero-arg method by name and returns the
// computed value if its return signature is consistent with a "data"
// accessor — a single non-error return, or (T, error). Returns ok=false
// for methods that don't fit (zero returns, error-only returns,
// multi-value returns), so side-effecting methods like Close() error and
// Stop() are not invoked from templates.
//
// Value structs whose method only exists on the pointer receiver are
// handled by copying the value to a fresh addressable location and
// calling on the pointer — so func (*T) Foo() is reachable even when a
// T was passed by value through a map/interface.
//
// To expose a side-effecting or non-conforming method, wrap the value in
// a Drop.
func callTemplateMethod(obj any, name string) (any, bool) {
	rv := reflect.ValueOf(obj)
	method := rv.MethodByName(name)
	if !method.IsValid() && rv.Kind() == reflect.Struct {
		// Promote to *T so pointer-receiver methods become visible.
		ptr := reflect.New(rv.Type())
		ptr.Elem().Set(rv)
		method = ptr.MethodByName(name)
	}
	if !method.IsValid() || method.Type().NumIn() != 0 {
		return nil, false
	}
	mt := method.Type()
	switch mt.NumOut() {
	case 1:
		if mt.Out(0) == errType {
			// Returns just an error — almost certainly a mutator (Close,
			// Save, Validate). Skip rather than discard the error.
			return nil, false
		}
		return method.Call(nil)[0].Interface(), true
	case 2:
		if mt.Out(1) != errType {
			return nil, false
		}
		results := method.Call(nil)
		if errVal, _ := results[1].Interface().(error); errVal != nil {
			// Method ran and reported an error. Treat the property as
			// undefined so strict-variables mode surfaces the failure
			// instead of silently rendering empty.
			return nil, false
		}
		return results[0].Interface(), true
	}
	return nil, false
}

// getProperty returns a property value, or nil if absent.
//
// Filter callers reach drop values through this helper; they don't have
// access to a RenderContext, so a ContextAwareDrop reached only through
// a filter (e.g. {{ obj | map: "field" }}) won't have its context
// re-set. The drop will still observe whatever context was set during
// the most recent variable lookup that materialized it.
func getProperty(obj any, prop string) any {
	v, _ := getPropertyOK(obj, prop, nil)
	return v
}

// getPropertyOK is like getProperty but reports whether the property was
// actually defined on the receiver. Used by strict-variables mode to
// distinguish "key is absent" from "key is present and explicitly nil".
//
// The Liquid built-ins `first`, `last`, and `size` are always considered
// present on any value that toSlice/filterSize can handle. Struct method
// dispatch is also considered present when a matching method is invoked.
func getPropertyOK(obj any, prop string, ctx RenderContext) (any, bool) {
	if obj == nil {
		return nil, false
	}

	// Drop opts the type into custom property resolution (Shopify Liquid's
	// Drop equivalent). When implemented, methods on the underlying type
	// are NOT auto-dispatched — the Drop is the sole source of truth.
	if d, ok := obj.(Drop); ok {
		setDropContext(d, ctx)
		v, present := d.LiquidLookup(prop)
		return v, present
	}

	if m, ok := obj.(map[string]any); ok {
		if v, present := m[prop]; present {
			return v, true
		}
		// Fall through to special-property handling so `hash.size` (etc.) works
		// when the map has no literal entry for that name. An explicit map entry
		// shadows the built-in.
		if prop == "size" {
			return len(m), true
		}
		return nil, false
	}

	switch prop {
	case "first":
		if slice := toSlice(obj); len(slice) > 0 {
			return slice[0], true
		}
		return nil, true
	case "last":
		if slice := toSlice(obj); len(slice) > 0 {
			return slice[len(slice)-1], true
		}
		return nil, true
	case "size":
		return filterSize(obj), true
	}

	rv := reflect.ValueOf(obj)
	if rv.Kind() == reflect.Pointer {
		rv = rv.Elem()
	}
	if rv.Kind() == reflect.Struct {
		field := rv.FieldByName(prop)
		if field.IsValid() && field.CanInterface() {
			return field.Interface(), true
		}
		if v, ok := callTemplateMethod(obj, prop); ok {
			return v, true
		}
	}

	if rv.Kind() == reflect.Map {
		key := reflect.ValueOf(prop)
		val := rv.MapIndex(key)
		if val.IsValid() {
			return val.Interface(), true
		}
		if prop == "size" {
			return rv.Len(), true
		}
	}

	return nil, false
}

// getIndexOK returns an element by index. Reports false when the
// requested index is out of range or the object isn't indexable.
func getIndexOK(obj, idx any, ctx RenderContext) (any, bool) {
	if obj == nil {
		return nil, false
	}
	if s, ok := idx.(string); ok {
		return getPropertyOK(obj, s, ctx)
	}
	if isFractionalFloat(idx) {
		// Reject fractional floats outright. Ruby's Utils.to_liquid_value
		// keeps floats as floats, so {{ h[1.9] }} cannot match the string
		// key "1" in a hash; truncating silently would invent a match
		// that upstream does not produce.
		return nil, false
	}

	i := int(toInt(toNumber(idx)))

	switch v := obj.(type) {
	case []any:
		return sliceAt(v, i)
	case string:
		if j, ok := normalizeIndex(i, len(v)); ok {
			return string(v[j]), true
		}
		return nil, false
	case map[string]any:
		return mapIntKey(obj, i, ctx)
	}

	rv := reflect.ValueOf(obj)
	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		if j, ok := normalizeIndex(i, rv.Len()); ok {
			return rv.Index(j).Interface(), true
		}
		return nil, false
	case reflect.Map:
		return mapIntKey(obj, i, ctx)
	default:
		return nil, false
	}
}

// isFractionalFloat reports whether v is a non-integer float.
func isFractionalFloat(v any) bool {
	switch f := v.(type) {
	case float32:
		return float32(int32(f)) != f
	case float64:
		return float64(int64(f)) != f
	}
	return false
}

// normalizeIndex resolves a possibly-negative index into a positive one,
// reporting false when out of range. Negative indices count from the end:
// -1 is the last element.
func normalizeIndex(i, length int) (int, bool) {
	switch {
	case i >= 0 && i < length:
		return i, true
	case i < 0 && -i <= length:
		return length + i, true
	}
	return 0, false
}

// sliceAt indexes an []any with optional negative-index support.
func sliceAt(v []any, i int) (any, bool) {
	if j, ok := normalizeIndex(i, len(v)); ok {
		return v[j], true
	}
	return nil, false
}

// mapIntKey looks up an integer key in a map, coercing it to its string form
// (Ruby coerces hash keys for lookup, so `obj[1]` finds entry "1"). Negative
// keys cannot match a stringified-int key, so the allocation is skipped.
func mapIntKey(obj any, i int, ctx RenderContext) (any, bool) {
	if i < 0 {
		return nil, false
	}
	return getPropertyOK(obj, strconv.Itoa(i), ctx)
}

// equal checks if two values are equal.
func equal(a, b any) bool {
	// Handle nil
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// Handle special empty/blank values
	if _, ok := a.(emptyValue); ok {
		return isEmpty(b)
	}
	if _, ok := b.(emptyValue); ok {
		return isEmpty(a)
	}
	if _, ok := a.(blankValue); ok {
		return isBlank(b)
	}
	if _, ok := b.(blankValue); ok {
		return isBlank(a)
	}

	// Compare numbers
	aNum := toNumber(a)
	bNum := toNumber(b)
	if isNumeric(a) && isNumeric(b) {
		return toFloat(aNum) == toFloat(bNum)
	}

	// Compare strings
	if _, ok := a.(string); ok {
		if _, ok := b.(string); ok {
			return a == b
		}
	}

	// Use reflect.DeepEqual for other types
	return reflect.DeepEqual(a, b)
}

// compare compares two values, returning -1, 0, or 1.
func compare(a, b any) int {
	if a == nil && b == nil {
		return 0
	}
	if a == nil {
		return -1
	}
	if b == nil {
		return 1
	}

	// Compare numbers
	if isNumeric(a) && isNumeric(b) {
		af := toFloat(toNumber(a))
		bf := toFloat(toNumber(b))
		if af < bf {
			return -1
		}
		if af > bf {
			return 1
		}
		return 0
	}

	// Compare strings
	as := toString(a)
	bs := toString(b)
	if as < bs {
		return -1
	}
	if as > bs {
		return 1
	}
	return 0
}

// contains checks if a contains b.
func contains(a, b any) bool {
	if a == nil {
		return false
	}

	// String contains
	if s, ok := a.(string); ok {
		return strings.Contains(s, toString(b))
	}

	// Array contains
	slice := toSlice(a)
	for _, item := range slice {
		if equal(item, b) {
			return true
		}
	}

	return false
}

// isNumeric checks if a value is a numeric type.
func isNumeric(v any) bool {
	if v == nil {
		return false
	}
	switch reflect.TypeOf(v).Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	default:
		return false
	}
}

func (e *evaluator) evalCycleTag(w io.Writer, tag *CycleTag) error {
	// Generate a unique key for this cycle.
	// Named cycles: evaluate the group expression at render time so that a
	// variable lookup as the group name works (Ruby parity).
	var key string
	if tag.GroupExpr != nil {
		val, err := e.evalExpr(tag.GroupExpr)
		if err != nil {
			return err
		}
		key = toString(val)
	} else if tag.GroupName != "" {
		key = tag.GroupName
	} else {
		// Generate a key from the values for unnamed cycles.
		var parts []string
		for _, v := range tag.Values {
			val, err := e.evalExpr(v)
			if err != nil {
				return err
			}
			parts = append(parts, toString(val))
		}
		key = strings.Join(parts, ",")
	}

	// Get current position in cycle
	pos := e.regs.cycle[key]

	// Evaluate the current value
	idx := pos % len(tag.Values)
	val, err := e.evalExpr(tag.Values[idx])
	if err != nil {
		return err
	}

	// Increment counter for next call
	e.regs.cycle[key] = pos + 1

	_, err = io.WriteString(w, toString(val))
	return err
}

func (e *evaluator) evalIncrementTag(w io.Writer, tag *IncrementTag) error {
	// Increment outputs the current counter, then advances it. The new
	// counter value is published into the scope so that a subsequent
	// {{ var }} sees the counter (Ruby Liquid semantics).
	val := e.regs.counter[tag.Variable]
	next := val + 1
	e.regs.counter[tag.Variable] = next
	e.ctx.setGlobal(tag.Variable, next)
	_, err := io.WriteString(w, toString(val))
	return err
}

func (e *evaluator) evalDecrementTag(w io.Writer, tag *DecrementTag) error {
	// Decrement decrements first, then outputs the new value. The new
	// counter value is also published into the scope so a subsequent
	// {{ var }} sees the counter (Ruby Liquid semantics).
	e.regs.counter[tag.Variable]--
	val := e.regs.counter[tag.Variable]
	e.ctx.setGlobal(tag.Variable, val)
	_, err := io.WriteString(w, toString(val))
	return err
}

// evalTablerowTag emits HTML <tr>/<td> markup over a collection. Output
// matches Shopify's exactly: `<tr class="row1">\n` opens, each item is
// wrapped in `<td class="colN">…</td>`, every `cols` items closes the
// current row and opens the next, and the final `</tr>\n` closes.
func (e *evaluator) evalTablerowTag(w io.Writer, tag *TablerowTag) error {
	collection, err := e.evalExpr(tag.Collection)
	if err != nil {
		return err
	}
	// Shopify short-circuits to "" when the collection itself is nil,
	// only emitting <tr>…</tr> markup for actual (possibly empty) arrays.
	if collection == nil {
		return nil
	}
	items, err := e.applyTablerowSlice(toSlice(collection), tag)
	if err != nil {
		return err
	}
	cols, err := e.tablerowCols(tag, len(items))
	if err != nil {
		return err
	}
	if cols == 0 {
		// Match Shopify: empty collection still emits a single empty row.
		_, err := io.WriteString(w, "<tr class=\"row1\">\n</tr>\n")
		return err
	}

	e.ctx = e.ctx.push()
	defer func() { e.ctx = e.ctx.parent }()

	if _, err := io.WriteString(w, "<tr class=\"row1\">\n"); err != nil {
		return err
	}
	if err := e.renderTablerowCells(w, items, cols, tag); err != nil {
		return err
	}
	_, err = io.WriteString(w, "</tr>\n")
	return err
}

// applyTablerowSlice applies tablerow's offset/limit operands. Tablerow has
// no `reversed`.
func (e *evaluator) applyTablerowSlice(items []any, tag *TablerowTag) ([]any, error) {
	if tag.Offset != nil {
		n, err := e.evalForInt(tag.Offset)
		if err != nil {
			return nil, err
		}
		switch {
		case n >= len(items):
			items = nil
		case n > 0:
			items = items[n:]
		}
	}
	if tag.Limit != nil {
		n, err := e.evalForInt(tag.Limit)
		if err != nil {
			return nil, err
		}
		if n >= 0 && n < len(items) {
			items = items[:n]
		}
	}
	return items, nil
}

// tablerowCols returns the configured column count, falling back to len(items)
// when the operand is missing or non-positive (matching Shopify).
func (e *evaluator) tablerowCols(tag *TablerowTag, defaultCols int) (int, error) {
	if tag.Cols == nil {
		return defaultCols, nil
	}
	n, err := e.evalForInt(tag.Cols)
	if err != nil {
		return 0, err
	}
	if n > 0 {
		return n, nil
	}
	return defaultCols, nil
}

// renderTablerowCells iterates items, emitting <td> per cell and <tr> row
// boundaries. The body is rendered to a scratch buffer per cell so that
// break/continue still produces a properly-closed <td> wrapper.
func (e *evaluator) renderTablerowCells(w io.Writer, items []any, cols int, tag *TablerowTag) error {
	length := len(items)
	tl := &tablerowloopState{length: length, cols: cols}
	var cell strings.Builder

	for i, item := range items {
		col := i%cols + 1
		row := i/cols + 1
		colLast := col == cols || i == length-1
		isLast := i == length-1

		tl.index0 = i
		tl.col = col
		tl.row = row
		tl.colFirst = col == 1
		tl.colLast = colLast
		e.ctx.set(tag.Variable, item)
		e.ctx.set("tablerowloop", tl)

		cell.Reset()
		bodyErr := e.evalNodes(&cell, tag.Body)
		fmt.Fprintf(w, "<td class=\"col%d\">", col)
		if _, err := io.WriteString(w, cell.String()); err != nil {
			return err
		}
		if _, err := io.WriteString(w, "</td>"); err != nil {
			return err
		}
		if errors.Is(bodyErr, errBreak) {
			return nil
		}
		if !errors.Is(bodyErr, errContinue) && bodyErr != nil {
			return bodyErr
		}
		if colLast && !isLast {
			fmt.Fprintf(w, "</tr>\n<tr class=\"row%d\">", row+1)
		}
	}
	return nil
}

// evalIfchangedTag emits the body only when its rendering differs from
// the most recent emission. Liquid uses a SINGLE shared register per
// render — every {% ifchanged %} block in the template compares against
// the same slot, so two distinct blocks emitting the same value will see
// the second suppressed. This matches Shopify's context.registers[:ifchanged].
func (e *evaluator) evalIfchangedTag(w io.Writer, tag *IfchangedTag) error {
	// Body must be evaluated to a scratch buffer first so we can compare
	// it against the last emission before deciding whether to write it.
	out, err := e.evalNodeToString(tag.Body)
	if err != nil {
		return err
	}
	if e.regs.ifchangedSet && e.regs.ifchangedLast == out {
		return nil
	}
	e.regs.ifchangedLast = out
	e.regs.ifchangedSet = true
	_, err = io.WriteString(w, out)
	return err
}

// Control flow errors for break/continue.
var (
	errBreak    = &controlFlowError{typ: "break"}
	errContinue = &controlFlowError{typ: "continue"}
)

type controlFlowError struct {
	typ string
}

func (e *controlFlowError) Error() string {
	return e.typ
}
