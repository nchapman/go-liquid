package liquid

import (
	"reflect"
	"strings"
)

// evaluator executes a parsed template.
type evaluator struct {
	ctx           *context
	cycleCounters map[string]int // tracks cycle position for each group
	counterVars   map[string]int // tracks increment/decrement counters
}

func newEvaluator(data map[string]any) *evaluator {
	return &evaluator{
		ctx:           newContext(data),
		cycleCounters: make(map[string]int),
		counterVars:   make(map[string]int),
	}
}

// evaluate executes the template and returns the result.
func (e *evaluator) evaluate(tmpl *templateAST) (string, error) {
	return e.evalNodes(tmpl.nodes)
}

func (e *evaluator) evalNodes(nodes []Node) (string, error) {
	var sb strings.Builder
	for _, node := range nodes {
		result, err := e.evalNode(node)
		if err != nil {
			return "", err
		}
		sb.WriteString(result)
	}
	return sb.String(), nil
}

func (e *evaluator) evalNode(node Node) (string, error) {
	switch n := node.(type) {
	case *TextNode:
		return n.Text, nil

	case *OutputNode:
		val, err := e.evalExpr(n.Expr)
		if err != nil {
			return "", err
		}
		return toString(val), nil

	case *IfTag:
		return e.evalIfTag(n)

	case *UnlessTag:
		return e.evalUnlessTag(n)

	case *CaseTag:
		return e.evalCaseTag(n)

	case *ForTag:
		return e.evalForTag(n)

	case *BreakTag:
		return "", errBreak

	case *ContinueTag:
		return "", errContinue

	case *AssignTag:
		val, err := e.evalExpr(n.Value)
		if err != nil {
			return "", err
		}
		e.ctx.setGlobal(n.Variable, val)
		return "", nil

	case *CaptureTag:
		captured, err := e.evalNodes(n.Body)
		if err != nil {
			return "", err
		}
		e.ctx.setGlobal(n.Variable, captured)
		return "", nil

	case *CommentTag:
		return "", nil

	case *RawTag:
		return n.Content, nil

	case *CycleTag:
		return e.evalCycleTag(n)

	case *IncrementTag:
		return e.evalIncrementTag(n)

	case *DecrementTag:
		return e.evalDecrementTag(n)

	default:
		return "", nil
	}
}

func (e *evaluator) evalIfTag(tag *IfTag) (string, error) {
	cond, err := e.evalExpr(tag.Condition)
	if err != nil {
		return "", err
	}

	if toBool(cond) {
		return e.evalNodes(tag.ThenBranch)
	}

	for _, elsif := range tag.ElsifBranches {
		cond, err := e.evalExpr(elsif.Condition)
		if err != nil {
			return "", err
		}
		if toBool(cond) {
			return e.evalNodes(elsif.Body)
		}
	}

	if tag.ElseBranch != nil {
		return e.evalNodes(tag.ElseBranch)
	}

	return "", nil
}

func (e *evaluator) evalUnlessTag(tag *UnlessTag) (string, error) {
	cond, err := e.evalExpr(tag.Condition)
	if err != nil {
		return "", err
	}

	if !toBool(cond) {
		return e.evalNodes(tag.Body)
	}

	if tag.ElseBranch != nil {
		return e.evalNodes(tag.ElseBranch)
	}

	return "", nil
}

func (e *evaluator) evalCaseTag(tag *CaseTag) (string, error) {
	value, err := e.evalExpr(tag.Value)
	if err != nil {
		return "", err
	}

	for _, when := range tag.Whens {
		for _, whenVal := range when.Values {
			v, err := e.evalExpr(whenVal)
			if err != nil {
				return "", err
			}
			if equal(value, v) {
				return e.evalNodes(when.Body)
			}
		}
	}

	if tag.Else != nil {
		return e.evalNodes(tag.Else)
	}

	return "", nil
}

func (e *evaluator) evalForTag(tag *ForTag) (string, error) {
	collection, err := e.evalExpr(tag.Collection)
	if err != nil {
		return "", err
	}

	// Handle range expression
	if rangeExpr, ok := tag.Collection.(*RangeExpr); ok {
		startVal, err := e.evalExpr(rangeExpr.Start)
		if err != nil {
			return "", err
		}
		endVal, err := e.evalExpr(rangeExpr.End)
		if err != nil {
			return "", err
		}
		start := int(toInt(toNumber(startVal)))
		end := int(toInt(toNumber(endVal)))

		var items []any
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

	items := toSlice(collection)
	if len(items) == 0 {
		if tag.ElseBody != nil {
			return e.evalNodes(tag.ElseBody)
		}
		return "", nil
	}

	// Apply offset
	if tag.Offset != nil {
		offset, err := e.evalExpr(tag.Offset)
		if err != nil {
			return "", err
		}
		off := int(toInt(toNumber(offset)))
		if off > 0 && off < len(items) {
			items = items[off:]
		} else if off >= len(items) {
			items = nil
		}
	}

	// Apply limit
	if tag.Limit != nil {
		limit, err := e.evalExpr(tag.Limit)
		if err != nil {
			return "", err
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
			return e.evalNodes(tag.ElseBody)
		}
		return "", nil
	}

	// Create new scope for loop
	e.ctx = e.ctx.push()
	defer func() { e.ctx = e.ctx.parent }()

	var sb strings.Builder
	length := len(items)

	for i, item := range items {
		e.ctx.set(tag.Variable, item)
		e.ctx.set("forloop", newForloop(i, length).toMap())

		result, err := e.evalNodes(tag.Body)
		if err == errBreak {
			break
		}
		if err == errContinue {
			continue
		}
		if err != nil {
			return "", err
		}
		sb.WriteString(result)
	}

	return sb.String(), nil
}

func (e *evaluator) evalExpr(expr Expression) (any, error) {
	switch x := expr.(type) {
	case *IdentExpr:
		return e.ctx.get(x.Name), nil

	case *LiteralExpr:
		return x.Value, nil

	case *DotExpr:
		obj, err := e.evalExpr(x.Object)
		if err != nil {
			return nil, err
		}
		return getProperty(obj, x.Property), nil

	case *IndexExpr:
		obj, err := e.evalExpr(x.Object)
		if err != nil {
			return nil, err
		}
		idx, err := e.evalExpr(x.Index)
		if err != nil {
			return nil, err
		}
		return getIndex(obj, idx), nil

	case *FilterExpr:
		input, err := e.evalExpr(x.Input)
		if err != nil {
			return nil, err
		}

		filter, ok := getFilter(x.Name)
		if !ok {
			// Unknown filter - return input unchanged
			return input, nil
		}

		var args []any
		for _, arg := range x.Args {
			val, err := e.evalExpr(arg)
			if err != nil {
				return nil, err
			}
			args = append(args, val)
		}

		return filter(input, args...), nil

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
	case "<":
		return compare(left, right) < 0, nil
	case ">":
		return compare(left, right) > 0, nil
	case "<=":
		return compare(left, right) <= 0, nil
	case ">=":
		return compare(left, right) >= 0, nil
	case "and":
		return toBool(left) && toBool(right), nil
	case "or":
		return toBool(left) || toBool(right), nil
	case "contains":
		return contains(left, right), nil
	default:
		// Unknown operators return false (Liquid semantics)
		return false, nil
	}
}

// getProperty gets a property from an object.
func getProperty(obj any, prop string) any {
	if obj == nil {
		return nil
	}

	// Handle map[string]any (includes forloop objects)
	if m, ok := obj.(map[string]any); ok {
		return m[prop]
	}

	// Handle special properties on arrays
	switch prop {
	case "first":
		if slice := toSlice(obj); len(slice) > 0 {
			return slice[0]
		}
		return nil
	case "last":
		if slice := toSlice(obj); len(slice) > 0 {
			return slice[len(slice)-1]
		}
		return nil
	case "size":
		return filterSize(obj)
	}

	// Use reflection for struct fields
	rv := reflect.ValueOf(obj)
	if rv.Kind() == reflect.Pointer {
		rv = rv.Elem()
	}
	if rv.Kind() == reflect.Struct {
		// Try field
		field := rv.FieldByName(prop)
		if field.IsValid() && field.CanInterface() {
			return field.Interface()
		}
		// Try method (only if it takes no arguments)
		method := reflect.ValueOf(obj).MethodByName(prop)
		if method.IsValid() && method.Type().NumIn() == 0 {
			results := method.Call(nil)
			if len(results) > 0 {
				return results[0].Interface()
			}
		}
	}

	// Try map with any key type
	if rv.Kind() == reflect.Map {
		key := reflect.ValueOf(prop)
		val := rv.MapIndex(key)
		if val.IsValid() {
			return val.Interface()
		}
	}

	return nil
}

// getIndex gets an element by index from an object.
func getIndex(obj any, idx any) any {
	if obj == nil {
		return nil
	}

	// Handle string index
	if s, ok := idx.(string); ok {
		return getProperty(obj, s)
	}

	// Handle numeric index
	i := int(toInt(toNumber(idx)))

	switch v := obj.(type) {
	case []any:
		if i >= 0 && i < len(v) {
			return v[i]
		}
		// Negative index
		if i < 0 && -i <= len(v) {
			return v[len(v)+i]
		}
		return nil
	case string:
		if i >= 0 && i < len(v) {
			return string(v[i])
		}
		if i < 0 && -i <= len(v) {
			return string(v[len(v)+i])
		}
		return nil
	default:
		rv := reflect.ValueOf(obj)
		if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
			if i >= 0 && i < rv.Len() {
				return rv.Index(i).Interface()
			}
			if i < 0 && -i <= rv.Len() {
				return rv.Index(rv.Len() + i).Interface()
			}
		}
	}

	return nil
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

func (e *evaluator) evalCycleTag(tag *CycleTag) (string, error) {
	// Generate a unique key for this cycle
	// Use the group name if provided, otherwise create one from values
	key := tag.GroupName
	if key == "" {
		// Generate a key from the values for unnamed cycles
		var parts []string
		for _, v := range tag.Values {
			val, err := e.evalExpr(v)
			if err != nil {
				return "", err
			}
			parts = append(parts, toString(val))
		}
		key = strings.Join(parts, ",")
	}

	// Get current position in cycle
	pos := e.cycleCounters[key]

	// Evaluate the current value
	idx := pos % len(tag.Values)
	val, err := e.evalExpr(tag.Values[idx])
	if err != nil {
		return "", err
	}

	// Increment counter for next call
	e.cycleCounters[key] = pos + 1

	return toString(val), nil
}

func (e *evaluator) evalIncrementTag(tag *IncrementTag) (string, error) {
	// Increment outputs the current value, then increments
	val := e.counterVars[tag.Variable]
	result := toString(val)
	e.counterVars[tag.Variable] = val + 1
	return result, nil
}

func (e *evaluator) evalDecrementTag(tag *DecrementTag) (string, error) {
	// Decrement decrements first, then outputs the value
	e.counterVars[tag.Variable]--
	val := e.counterVars[tag.Variable]
	return toString(val), nil
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
