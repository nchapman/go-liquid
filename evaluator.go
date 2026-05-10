package liquid

import (
	"errors"
	"fmt"
	"io"
	"reflect"
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
// for {% include %}); ctx is the live scope chain.
type evaluator struct {
	cfg          *renderConfig
	regs         *registers
	ctx          *context
	templateName string // optional, surfaced on RenderError; varies per partial
	partialDepth int
}

func newEvaluator(data map[string]any) *evaluator {
	return &evaluator{
		cfg:  &renderConfig{env: Default()},
		regs: newRegisters(),
		ctx:  newContext(data),
	}
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
		return nil

	case *CaptureTag:
		captured, err := e.evalNodeToString(n.Body)
		if err != nil {
			return err
		}
		e.ctx.setGlobal(n.Variable, captured)
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
// modify caller variables.
func (e *evaluator) evalRenderTag(w io.Writer, tag *RenderTag) error {
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
		for i, item := range items {
			itemData := make(map[string]any, len(data)+2)
			for k, v := range data {
				itemData[k] = v
			}
			itemData[alias] = item
			// render is isolated → parentloop is nil. Shopify's render.rb sets
			// forloop.name to the partial's template name (not the iteration
			// alias), so {% render "card" for items as item %} exposes
			// forloop.name == "card".
			itemData["forloop"] = newForloop(i, length, tag.Template, nil)
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
		// becomes parentloop.
		parent, _ := e.ctx.get("forloop").(map[string]any)
		for i, item := range items {
			e.ctx.set(alias, item)
			e.ctx.set("forloop", newForloop(i, length, alias, parent))
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

// forloopName approximates Shopify's `forloop.name` ("{var}-{collection}").
// We synthesize the collection portion from the AST since there is no
// source-text reference; identifiers and ranges produce stable names, and
// other expressions fall back to the variable name alone.
func forloopName(tag *ForTag) string {
	switch c := tag.Collection.(type) {
	case *IdentExpr:
		return tag.Variable + "-" + c.Name
	case *DotExpr:
		// best effort: walk to the rightmost property
		if obj, ok := c.Object.(*IdentExpr); ok {
			return tag.Variable + "-" + obj.Name + "." + c.Property
		}
		return tag.Variable + "-" + c.Property
	case *RangeExpr:
		return tag.Variable + "-(range)"
	}
	return tag.Variable
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
		return e.evalNodes(w, tag.ThenBranch)
	}

	for _, elsif := range tag.ElsifBranches {
		cond, err := e.evalExpr(elsif.Condition)
		if err != nil {
			return err
		}
		if toBool(cond) {
			return e.evalNodes(w, elsif.Body)
		}
	}

	if tag.ElseBranch != nil {
		return e.evalNodes(w, tag.ElseBranch)
	}

	return nil
}

func (e *evaluator) evalUnlessTag(w io.Writer, tag *UnlessTag) error {
	cond, err := e.evalExpr(tag.Condition)
	if err != nil {
		return err
	}

	if !toBool(cond) {
		return e.evalNodes(w, tag.Body)
	}

	if tag.ElseBranch != nil {
		return e.evalNodes(w, tag.ElseBranch)
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
				return e.evalNodes(w, when.Body)
			}
		}
	}

	if tag.Else != nil {
		return e.evalNodes(w, tag.Else)
	}

	return nil
}

func (e *evaluator) evalForTag(w io.Writer, tag *ForTag) error {
	collection, err := e.evalExpr(tag.Collection)
	if err != nil {
		return err
	}

	// Handle range expression. The slice is materialized eagerly so that
	// limit/offset/reversed below can apply uniformly; cap the range size
	// to keep attacker-controlled endpoints from exhausting memory.
	if rangeExpr, ok := tag.Collection.(*RangeExpr); ok {
		startVal, err := e.evalExpr(rangeExpr.Start)
		if err != nil {
			return err
		}
		endVal, err := e.evalExpr(rangeExpr.End)
		if err != nil {
			return err
		}
		start := int(toInt(toNumber(startVal)))
		end := int(toInt(toNumber(endVal)))

		size := end - start
		if size < 0 {
			size = -size
		}
		if size+1 > maxRangeSize {
			return fmt.Errorf("for range size %d exceeds limit %d", size+1, maxRangeSize)
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
		collection = items
	}

	// Liquid treats a non-empty string as a single iteration item, not a
	// sequence of characters (toSlice splits strings for filter use). An
	// empty/blank string is non-iterable — zero iterations — to match
	// upstream's `blank_string_not_iterable` behavior.
	var items []any
	if s, ok := collection.(string); ok {
		if s != "" {
			items = []any{s}
		}
	} else {
		items = toSlice(collection)
	}
	if len(items) == 0 {
		if tag.ElseBody != nil {
			return e.evalNodes(w, tag.ElseBody)
		}
		return nil
	}

	// `offset: continue` resumes from where the previous for-tag iterating
	// the same {var,collection} stopped. The cursor lives in registers,
	// keyed by forloopName so distinct loops over the same collection (or
	// distinct collections under the same variable name) get independent
	// cursors. Registers are per-Render, so the cursor resets between
	// top-level renders.
	contKey := forloopName(tag)
	off := 0
	switch {
	case tag.OffsetContinue:
		off = e.regs.forContinue[contKey]
	case tag.Offset != nil:
		offset, err := e.evalExpr(tag.Offset)
		if err != nil {
			return err
		}
		off = int(toInt(toNumber(offset)))
	}
	if off > 0 && off < len(items) {
		items = items[off:]
	} else if off >= len(items) {
		items = nil
	}

	// Apply limit
	if tag.Limit != nil {
		limit, err := e.evalExpr(tag.Limit)
		if err != nil {
			return err
		}
		lim := int(toInt(toNumber(limit)))
		if lim > 0 && lim < len(items) {
			items = items[:lim]
		}
	}

	// Apply reversed
	if tag.Reversed {
		reversed := make([]any, len(items))
		for i, v := range items {
			reversed[len(items)-1-i] = v
		}
		items = reversed
	}

	if len(items) == 0 {
		if tag.ElseBody != nil {
			return e.evalNodes(w, tag.ElseBody)
		}
		return nil
	}

	// Capture the enclosing forloop (if any) BEFORE pushing a new scope so
	// the new forloop.parentloop reflects the outer iteration. Shopify's
	// `forloop.name` is "{var}-{collection}"; we render the collection
	// expression source where possible, falling back to the variable name.
	parent, _ := e.ctx.get("forloop").(map[string]any)
	loopName := forloopName(tag)

	e.ctx = e.ctx.push()
	defer func() { e.ctx = e.ctx.parent }()

	length := len(items)
	consumed := 0

	for i, item := range items {
		e.ctx.set(tag.Variable, item)
		e.ctx.set("forloop", newForloop(i, length, loopName, parent))

		// Body output streams directly to w as it goes. {% break %} and
		// {% continue %} do not unwind output already written this
		// iteration — matching the pre-Writer behavior, where evalNodes
		// returned (partial-output, errBreak) and the caller appended
		// the partial output before breaking.
		err := e.evalNodes(w, tag.Body)
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

func (e *evaluator) evalExpr(expr Expression) (any, error) {
	switch x := expr.(type) {
	case *IdentExpr:
		val := e.ctx.get(x.Name)
		if val == nil && e.cfg.strictVariables {
			if _, ok := e.ctx.lookup(x.Name); !ok {
				return nil, wrapAtNode(x, fmt.Errorf("undefined variable %q", x.Name), e.templateName)
			}
		}
		return val, nil

	case *LiteralExpr:
		return x.Value, nil

	case *DotExpr:
		obj, err := e.evalExpr(x.Object)
		if err != nil {
			return nil, err
		}
		val, ok := getPropertyOK(obj, x.Property)
		if !ok && e.cfg.strictVariables {
			return nil, wrapAtNode(x, fmt.Errorf("undefined property %q", x.Property), e.templateName)
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
		val, ok := getIndexOK(obj, idx)
		if !ok && e.cfg.strictVariables {
			return nil, wrapAtNode(x, fmt.Errorf("undefined index %v", idx), e.templateName)
		}
		return val, nil

	case *FilterExpr:
		input, err := e.evalExpr(x.Input)
		if err != nil {
			return nil, err
		}

		var args []any
		for _, arg := range x.Args {
			val, err := e.evalExpr(arg)
			if err != nil {
				return nil, err
			}
			args = append(args, val)
		}
		var kwargs map[string]any
		if len(x.Kwargs) > 0 {
			kwargs = make(map[string]any, len(x.Kwargs))
			for _, kv := range x.Kwargs {
				val, err := e.evalExpr(kv.Value)
				if err != nil {
					return nil, err
				}
				kwargs[kv.Name] = val
			}
		}

		// Single dispatch through the unified Filter interface. Positional
		// filters that don't care about kwargs ignore them via FilterFunc.Apply,
		// matching Shopify's "extra hash arg is a no-op" semantics.
		if fn, ok := e.cfg.env.lookupFilter(x.Name); ok {
			out, err := fn.Apply(input, args, kwargs)
			if err != nil {
				return nil, wrapAtNode(x, err, e.templateName)
			}
			return out, nil
		}
		if e.cfg.strictFilters {
			return nil, wrapAtNode(x, fmt.Errorf("unknown filter %q", x.Name), e.templateName)
		}
		// Unknown filter — return input unchanged (lax Liquid semantics).
		return input, nil

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
		// Relational operators against nil are always false in upstream
		// Liquid (incompatible types). Equality operators above still apply.
		if left == nil || right == nil {
			return false, nil
		}
		switch expr.Operator {
		case "<":
			return compare(left, right) < 0, nil
		case ">":
			return compare(left, right) > 0, nil
		case "<=":
			return compare(left, right) <= 0, nil
		default: // ">="
			return compare(left, right) >= 0, nil
		}
	case "contains":
		return contains(left, right), nil
	default:
		// Unknown operators return false (Liquid semantics)
		return false, nil
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
func getProperty(obj any, prop string) any {
	v, _ := getPropertyOK(obj, prop)
	return v
}

// getPropertyOK is like getProperty but reports whether the property was
// actually defined on the receiver. Used by strict-variables mode to
// distinguish "key is absent" from "key is present and explicitly nil".
//
// The Liquid built-ins `first`, `last`, and `size` are always considered
// present on any value that toSlice/filterSize can handle. Struct method
// dispatch is also considered present when a matching method is invoked.
func getPropertyOK(obj any, prop string) (any, bool) {
	if obj == nil {
		return nil, false
	}

	// Drop opts the type into custom property resolution (Shopify Liquid's
	// Drop equivalent). When implemented, methods on the underlying type
	// are NOT auto-dispatched — the Drop is the sole source of truth.
	if d, ok := obj.(Drop); ok {
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

// getIndex returns an element by index, or nil if absent / out of range.
func getIndex(obj any, idx any) any {
	v, _ := getIndexOK(obj, idx)
	return v
}

// getIndexOK is the strict-aware variant. Reports false when the
// requested index is out of range or the object isn't indexable.
func getIndexOK(obj any, idx any) (any, bool) {
	if obj == nil {
		return nil, false
	}

	if s, ok := idx.(string); ok {
		return getPropertyOK(obj, s)
	}

	i := int(toInt(toNumber(idx)))

	switch v := obj.(type) {
	case []any:
		if i >= 0 && i < len(v) {
			return v[i], true
		}
		if i < 0 && -i <= len(v) {
			return v[len(v)+i], true
		}
		return nil, false
	case string:
		if i >= 0 && i < len(v) {
			return string(v[i]), true
		}
		if i < 0 && -i <= len(v) {
			return string(v[len(v)+i]), true
		}
		return nil, false
	default:
		rv := reflect.ValueOf(obj)
		if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
			if i >= 0 && i < rv.Len() {
				return rv.Index(i).Interface(), true
			}
			if i < 0 && -i <= rv.Len() {
				return rv.Index(rv.Len() + i).Interface(), true
			}
		}
	}

	return nil, false
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
	}
	return false
}

func (e *evaluator) evalCycleTag(w io.Writer, tag *CycleTag) error {
	// Generate a unique key for this cycle
	// Use the group name if provided, otherwise create one from values
	key := tag.GroupName
	if key == "" {
		// Generate a key from the values for unnamed cycles
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
	// Increment outputs the current value, then increments
	val := e.regs.counter[tag.Variable]
	e.regs.counter[tag.Variable] = val + 1
	_, err := io.WriteString(w, toString(val))
	return err
}

func (e *evaluator) evalDecrementTag(w io.Writer, tag *DecrementTag) error {
	// Decrement decrements first, then outputs the value
	e.regs.counter[tag.Variable]--
	val := e.regs.counter[tag.Variable]
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
	items := toSlice(collection)

	if tag.Offset != nil {
		off, err := e.evalExpr(tag.Offset)
		if err != nil {
			return err
		}
		n := int(toInt(toNumber(off)))
		if n > 0 && n < len(items) {
			items = items[n:]
		} else if n >= len(items) {
			items = nil
		}
	}
	if tag.Limit != nil {
		lim, err := e.evalExpr(tag.Limit)
		if err != nil {
			return err
		}
		n := int(toInt(toNumber(lim)))
		if n >= 0 && n < len(items) {
			items = items[:n]
		}
	}

	cols := len(items)
	if tag.Cols != nil {
		v, err := e.evalExpr(tag.Cols)
		if err != nil {
			return err
		}
		if n := int(toInt(toNumber(v))); n > 0 {
			cols = n
		}
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
	length := len(items)
	// Per-cell scratch buffer: tablerow wraps each cell in <td>...</td>,
	// and break/continue inside the body must still emit the partial cell
	// with its closing tag — easier to evaluate body to scratch and then
	// emit the wrapped cell than to insert the closing tag from the
	// outside if break/continue fired mid-stream.
	var cell strings.Builder
	for i, item := range items {
		col := i%cols + 1
		row := i/cols + 1
		colFirst := col == 1
		colLast := col == cols || i == length-1
		isLast := i == length-1

		e.ctx.set(tag.Variable, item)
		e.ctx.set("tablerowloop", map[string]any{
			"length":    length,
			"index":     i + 1,
			"index0":    i,
			"rindex":    length - i,
			"rindex0":   length - i - 1,
			"col":       col,
			"col0":      col - 1,
			"row":       row,
			"first":     i == 0,
			"last":      isLast,
			"col_first": colFirst,
			"col_last":  colLast,
		})

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
			break
		}
		if errors.Is(bodyErr, errContinue) {
			if colLast && !isLast {
				fmt.Fprintf(w, "</tr>\n<tr class=\"row%d\">", row+1)
			}
			continue
		}
		if bodyErr != nil {
			return bodyErr
		}
		if colLast && !isLast {
			fmt.Fprintf(w, "</tr>\n<tr class=\"row%d\">", row+1)
		}
	}
	_, err = io.WriteString(w, "</tr>\n")
	return err
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
