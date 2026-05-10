package liquid

import (
	"cmp"
	"fmt"
	"html"
	"math"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// Filter is the unified filter interface. The evaluator dispatches every
// {{ x | name: ... }} call through Apply: positional arguments arrive in
// args, named arguments (`name: x, opt: y`) arrive in kwargs (nil if the
// template supplied none). Returning a non-nil error aborts the render
// with a positioned RenderError.
//
// Most built-in filters are infallible and care only about positional
// args, so they're written against the legacy FilterFunc signature and
// adapted via FilterFunc.Apply. Filters that need kwargs (currently just
// `default`) use KwargFilterFunc. New filters that want first-class error
// returns can implement Filter directly or use FilterFuncE.
type Filter interface {
	Apply(input any, args []any, kwargs map[string]any) (any, error)
}

// FilterFunc is the legacy positional-only signature. Adapted to Filter
// by ignoring kwargs and unwrapping the filterError sentinel.
type FilterFunc func(input any, args ...any) any

// Apply implements Filter for the positional-only signature.
func (f FilterFunc) Apply(input any, args []any, _ map[string]any) (any, error) {
	out := f(input, args...)
	if fe, ok := out.(filterError); ok {
		return nil, fe.err
	}
	return out, nil
}

// FilterFuncE is the recommended signature for new filters that may fail
// or want named-argument support. Returning a non-nil error aborts the
// render at this filter's source position.
type FilterFuncE func(input any, args []any, kwargs map[string]any) (any, error)

// Apply implements Filter.
func (f FilterFuncE) Apply(input any, args []any, kwargs map[string]any) (any, error) {
	return f(input, args, kwargs)
}

// filterError is the sentinel returned by filterErrorf so legacy
// FilterFunc/KwargFilterFunc filters can signal errors through the `any`
// return without the boilerplate of `(any, error)`. The Apply adapters
// unwrap it transparently. New filters should implement Filter directly
// and return real errors.
type filterError struct{ err error }

func filterErrorf(format string, args ...any) any {
	return filterError{err: fmt.Errorf(format, args...)}
}

// pos and kw wrap raw filter functions into the Filter interface so the
// registry literal below stays tidy.
func pos(fn FilterFunc) Filter     { return fn }
func kw(fn KwargFilterFunc) Filter { return fn }

// registerStandardFilters writes the built-in filter set into e. Called by
// NewEnvironment so every fresh environment ships with the Shopify/Ruby
// Liquid standard library.
func registerStandardFilters(e *Environment) {
	for name, fn := range standardFilters {
		e.filters[name] = fn
	}
}

// standardFilters is the canonical built-in filter set. Treated as
// effectively immutable after package init: registerStandardFilters copies
// each entry into a fresh per-Environment map, and nothing else writes to
// it. Do not mutate after init — a write here would silently affect every
// future environment.
var standardFilters = map[string]Filter{
	// String filters
	"upcase":         pos(filterUpcase),
	"downcase":       pos(filterDowncase),
	"capitalize":     pos(filterCapitalize),
	"strip":          pos(filterStrip),
	"lstrip":         pos(filterLstrip),
	"rstrip":         pos(filterRstrip),
	"escape":         pos(filterEscape),
	"h":              pos(filterEscape), // Ruby alias for `escape`.
	"split":          pos(filterSplit),
	"append":         pos(filterAppend),
	"prepend":        pos(filterPrepend),
	"replace":        pos(filterReplace),
	"replace_first":  pos(filterReplaceFirst),
	"remove":         pos(filterRemove),
	"remove_first":   pos(filterRemoveFirst),
	"truncate":       pos(filterTruncate),
	"truncatewords":  pos(filterTruncateWords),
	"slice":          pos(filterSlice),
	"newline_to_br":  pos(filterNewlineToBr),
	"escape_once":    pos(filterEscapeOnce),
	"url_encode":     pos(filterURLEncode),
	"url_decode":     pos(filterURLDecode),
	"strip_html":     pos(filterStripHTML),
	"strip_newlines": pos(filterStripNewlines),
	"squish":         pos(filterSquish),
	"replace_last":   pos(filterReplaceLast),
	"remove_last":    pos(filterRemoveLast),

	// Base64
	"base64_encode":          pos(filterBase64Encode),
	"base64_decode":          pos(filterBase64Decode),
	"base64_url_safe_encode": pos(filterBase64URLSafeEncode),
	"base64_url_safe_decode": pos(filterBase64URLSafeDecode),

	// Array filters
	"first":        pos(filterFirst),
	"last":         pos(filterLast),
	"size":         pos(filterSize),
	"join":         pos(filterJoin),
	"reverse":      pos(filterReverse),
	"sort":         pos(filterSort),
	"sort_natural": pos(filterSortNatural),
	"map":          pos(filterMap),
	"where":        pos(filterWhere),
	"reject":       pos(filterReject),
	"find":         pos(filterFind),
	"find_index":   pos(filterFindIndex),
	"has":          pos(filterHas),
	"uniq":         pos(filterUniq),
	"compact":      pos(filterCompact),
	"concat":       pos(filterConcat),
	"sum":          pos(filterSum),

	// `default` accepts the `allow_false:` named arg.
	"default": kw(filterDefaultKw),

	// Math filters
	"plus":       pos(filterPlus),
	"minus":      pos(filterMinus),
	"times":      pos(filterTimes),
	"divided_by": pos(filterDividedBy),
	"modulo":     pos(filterModulo),
	"abs":        pos(filterAbs),
	"round":      pos(filterRound),
	"ceil":       pos(filterCeil),
	"floor":      pos(filterFloor),
	"at_least":   pos(filterAtLeast),
	"at_most":    pos(filterAtMost),

	// Date filter
	"date": pos(filterDate),
}

// String filters

func filterUpcase(input any, _ ...any) any {
	return strings.ToUpper(toString(input))
}

func filterDowncase(input any, _ ...any) any {
	return strings.ToLower(toString(input))
}

func filterCapitalize(input any, _ ...any) any {
	s := toString(input)
	if s == "" {
		return s
	}
	r, size := utf8.DecodeRuneInString(s)
	return strings.ToUpper(string(r)) + strings.ToLower(s[size:])
}

// stripChars is the cutset for strip/lstrip/rstrip. Matches Ruby's
// String#strip (ASCII whitespace plus null), which is what Shopify uses.
// Unicode whitespace like U+00A0 is intentionally NOT stripped so that
// templates relying on Shopify's behavior produce identical output.
const stripChars = " \t\n\r\v\f\x00"

func filterStrip(input any, _ ...any) any {
	return strings.Trim(toString(input), stripChars)
}

func filterEscape(input any, _ ...any) any {
	if input == nil {
		return nil
	}
	return html.EscapeString(toString(input))
}

func filterLstrip(input any, _ ...any) any {
	return strings.TrimLeft(toString(input), stripChars)
}

func filterRstrip(input any, _ ...any) any {
	return strings.TrimRight(toString(input), stripChars)
}

func filterSplit(input any, args ...any) any {
	s := toString(input)
	sep := " "
	if len(args) > 0 {
		sep = toString(args[0])
	}
	parts := strings.Split(s, sep)
	result := make([]any, len(parts))
	for i, p := range parts {
		result[i] = p
	}
	return result
}

func filterAppend(input any, args ...any) any {
	s := toString(input)
	if len(args) > 0 {
		s += toString(args[0])
	}
	return s
}

func filterPrepend(input any, args ...any) any {
	s := toString(input)
	if len(args) > 0 {
		s = toString(args[0]) + s
	}
	return s
}

func filterReplace(input any, args ...any) any {
	s := toString(input)
	if len(args) < 2 {
		return s
	}
	oldStr := toString(args[0])
	newStr := toString(args[1])
	return strings.ReplaceAll(s, oldStr, newStr)
}

func filterReplaceFirst(input any, args ...any) any {
	s := toString(input)
	if len(args) < 2 {
		return s
	}
	oldStr := toString(args[0])
	newStr := toString(args[1])
	return strings.Replace(s, oldStr, newStr, 1)
}

func filterRemove(input any, args ...any) any {
	s := toString(input)
	if len(args) == 0 {
		return s
	}
	old := toString(args[0])
	return strings.ReplaceAll(s, old, "")
}

func filterRemoveFirst(input any, args ...any) any {
	s := toString(input)
	if len(args) == 0 {
		return s
	}
	old := toString(args[0])
	return strings.Replace(s, old, "", 1)
}

func filterTruncate(input any, args ...any) any {
	if input == nil {
		return nil
	}
	s := toString(input)
	length := 50
	ellipsis := "..."

	if len(args) > 0 {
		length = int(toInt(toNumber(args[0])))
	}
	if len(args) > 1 {
		ellipsis = toString(args[1])
	}

	// Shopify counts characters (Ruby String#length), not bytes, so multi-byte
	// UTF-8 inputs slice cleanly. Same applies to the ellipsis budget.
	runes := []rune(s)
	ellRunes := []rune(ellipsis)
	if len(runes) <= length {
		return s
	}
	keep := length - len(ellRunes)
	if keep < 0 {
		keep = 0
	}
	return string(runes[:keep]) + ellipsis
}

func filterTruncateWords(input any, args ...any) any {
	if input == nil {
		return nil
	}
	s := toString(input)
	wordCount := 15
	ellipsis := "..."

	if len(args) > 0 {
		wordCount = int(toInt(toNumber(args[0])))
	}
	if len(args) > 1 {
		ellipsis = toString(args[1])
	}
	// Ruby clamps wordCount to a minimum of 1 (standardfilters.rb#truncatewords).
	if wordCount <= 0 {
		wordCount = 1
	}

	// Ruby's `split(" ", n)` is the special single-space form that
	// collapses runs of whitespace and strips the lead — strings.Fields
	// matches that exactly.
	words := strings.Fields(s)
	if len(words) <= wordCount {
		return s
	}
	return strings.Join(words[:wordCount], " ") + ellipsis
}

func filterSlice(input any, args ...any) any {
	if len(args) == 0 {
		return input
	}

	offset := int(toInt(toNumber(args[0])))
	length := 1
	if len(args) > 1 {
		length = int(toInt(toNumber(args[1])))
	}

	// Handle string slicing — Shopify slices by character (rune), not byte,
	// so multi-byte UTF-8 inputs slice cleanly.
	if s, ok := input.(string); ok {
		runes := []rune(s)
		if offset < 0 {
			offset = len(runes) + offset
		}
		if offset < 0 {
			offset = 0
		}
		if offset >= len(runes) {
			return ""
		}
		end := min(offset+length, len(runes))
		return string(runes[offset:end])
	}

	// Handle array slicing. Always return a (possibly empty) []any so that
	// downstream filters like `default` and `size` see an array, matching
	// Shopify's behavior.
	slice := toSlice(input)
	if slice == nil {
		return []any{}
	}

	if offset < 0 {
		offset = len(slice) + offset
	}
	if offset < 0 {
		offset = 0
	}
	if offset >= len(slice) {
		return []any{}
	}
	end := min(offset+length, len(slice))
	return slice[offset:end]
}

func filterNewlineToBr(input any, _ ...any) any {
	s := toString(input)
	// Shopify inserts the <br /> before the newline rather than replacing
	// it, so source line breaks survive into the rendered output.
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.ReplaceAll(s, "\n", "<br />\n")
}

// Array filters

func filterFirst(input any, _ ...any) any {
	// Ruby Liquid returns "" (not nil) for first/last on an empty string,
	// matching String#[]. Treat the string case explicitly so empties round-
	// trip predictably and so the result is a proper character (rune), not
	// a byte sliced mid-codepoint.
	if s, ok := input.(string); ok {
		if s == "" {
			return ""
		}
		runes := []rune(s)
		return string(runes[0])
	}
	slice := toSlice(input)
	if len(slice) == 0 {
		return nil
	}
	return slice[0]
}

func filterLast(input any, _ ...any) any {
	if s, ok := input.(string); ok {
		if s == "" {
			return ""
		}
		runes := []rune(s)
		return string(runes[len(runes)-1])
	}
	slice := toSlice(input)
	if len(slice) == 0 {
		return nil
	}
	return slice[len(slice)-1]
}

func filterSize(input any, _ ...any) any {
	if input == nil {
		return 0
	}
	switch v := input.(type) {
	case string:
		// Ruby's String#size counts characters, not bytes; match that for
		// any UTF-8 input. utf8.RuneCountInString is allocation-free.
		return utf8.RuneCountInString(v)
	case []any:
		return len(v)
	default:
		rv := reflect.ValueOf(input)
		switch rv.Kind() {
		case reflect.String:
			return utf8.RuneCountInString(rv.String())
		case reflect.Array, reflect.Slice, reflect.Map:
			return rv.Len()
		default:
			return 0
		}
	}
}

func filterJoin(input any, args ...any) any {
	slice := toFilterInput(input)
	sep := " " // default separator
	if len(args) > 0 {
		sep = toString(args[0])
	}

	strs := make([]string, len(slice))
	for i, v := range slice {
		strs[i] = toString(v)
	}
	return strings.Join(strs, sep)
}

func filterReverse(input any, _ ...any) any {
	slice := toFilterInput(input)
	result := make([]any, len(slice))
	for i, v := range slice {
		result[len(slice)-1-i] = v
	}
	return result
}

func filterSort(input any, args ...any) any {
	slice := toFilterInput(input)
	if len(slice) == 0 {
		// Ruby returns [] for empty input (standardfilters.rb#sort).
		return []any{}
	}

	result := make([]any, len(slice))
	copy(result, slice)

	// If a property is specified, sort by that property
	if len(args) > 0 {
		prop := toString(args[0])
		slices.SortFunc(result, func(a, b any) int {
			aVal := getProperty(a, prop)
			bVal := getProperty(b, prop)
			return compareValues(aVal, bVal)
		})
		return result
	}

	// Sort by value
	slices.SortFunc(result, compareValues)
	return result
}

func filterSortNatural(input any, args ...any) any {
	slice := toFilterInput(input)
	if len(slice) == 0 {
		// Ruby returns [] for empty input (standardfilters.rb#sort_natural).
		return []any{}
	}

	result := make([]any, len(slice))
	copy(result, slice)

	// nilSafeCaseCmp: nil sorts last, otherwise case-insensitive string compare.
	cmpStr := func(a, b any) int {
		switch {
		case a == nil && b == nil:
			return 0
		case a == nil:
			return 1
		case b == nil:
			return -1
		}
		return cmp.Compare(strings.ToLower(toString(a)), strings.ToLower(toString(b)))
	}
	if len(args) > 0 {
		prop := toString(args[0])
		slices.SortFunc(result, func(a, b any) int {
			return cmpStr(getProperty(a, prop), getProperty(b, prop))
		})
		return result
	}
	slices.SortFunc(result, cmpStr)
	return result
}

func filterMap(input any, args ...any) any {
	if len(args) == 0 {
		return nil
	}
	slice := toFilterInput(input)

	prop := toString(args[0])
	result := make([]any, len(slice))
	for i, item := range slice {
		result[i] = getProperty(item, prop)
	}
	return result
}

func filterWhere(input any, args ...any) any {
	if len(args) == 0 {
		return nil
	}
	slice := toFilterInput(input)

	prop := toString(args[0])
	var targetValue any = true // default is to check for truthy
	if len(args) > 1 {
		targetValue = args[1]
	}

	var result []any
	for _, item := range slice {
		val := getProperty(item, prop)
		if equalValues(val, targetValue) {
			result = append(result, item)
		}
	}
	return result
}

func filterFind(input any, args ...any) any {
	if len(args) == 0 {
		return nil
	}
	slice := toFilterInput(input)

	prop := toString(args[0])
	var targetValue any = true
	if len(args) > 1 {
		targetValue = args[1]
	}

	for _, item := range slice {
		val := getProperty(item, prop)
		if equalValues(val, targetValue) {
			return item
		}
	}
	return nil
}

// filterUniq returns input with duplicate elements removed, preserving order.
// Equality is the same Liquid equal() used elsewhere (numeric, string, or
// reflect.DeepEqual fallback), so structurally-equal maps and slices dedupe
// correctly. The optional `property` argument compares items by that
// property's value rather than the items themselves.
func filterUniq(input any, args ...any) any {
	slice := toFilterInput(input)

	var prop string
	if len(args) > 0 {
		prop = toString(args[0])
	}

	keyOf := func(item any) any {
		if prop != "" {
			return getProperty(item, prop)
		}
		return item
	}

	seen := make([]any, 0, len(slice))
	var result []any
	for _, item := range slice {
		k := keyOf(item)
		dup := false
		for _, s := range seen {
			if equal(k, s) {
				dup = true
				break
			}
		}
		if !dup {
			seen = append(seen, k)
			result = append(result, item)
		}
	}
	return result
}

func filterCompact(input any, args ...any) any {
	slice := toFilterInput(input)
	// `compact: "property"` (Ruby standardfilters.rb#compact) keeps items
	// whose property is non-nil. Without an argument it keeps non-nil items.
	var prop string
	if len(args) > 0 {
		prop = toString(args[0])
	}
	var result []any
	for _, item := range slice {
		if prop != "" {
			if getProperty(item, prop) != nil {
				result = append(result, item)
			}
			continue
		}
		if item != nil {
			result = append(result, item)
		}
	}
	return result
}

func filterConcat(input any, args ...any) any {
	slice := toFilterInput(input)
	result := append(make([]any, 0, len(slice)), slice...)

	for _, arg := range args {
		if !isArrayLike(arg) {
			return filterErrorf("concat: argument is not an array")
		}
		// Argument bypasses InputIterator in Ruby — no flattening here.
		result = append(result, toSlice(arg)...)
	}
	return result
}

// isArrayLike reports whether v is a slice or array. Unlike toSlice, it
// rejects strings (which toSlice splits into characters).
func isArrayLike(v any) bool {
	if v == nil {
		return false
	}
	if _, ok := v.([]any); ok {
		return true
	}
	rv := reflect.ValueOf(v)
	return rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array
}

// toFilterInput mirrors Ruby Liquid's InputIterator: most array filters
// (join, sort, map, where, uniq, ...) flatten nested arrays before iterating
// and wrap a Hash input as a single-element array. Strings are wrapped (not
// split into runes), and nil becomes the empty slice. Filters that don't go
// through InputIterator in Ruby (first, last, size, slice) keep using
// toSlice directly.
//
// Drops are not iterable here — they fall through to the scalar-wrap branch
// as `[drop]`, matching Ruby's `Array(hash_like)`. A Drop intended to act as
// a collection should be exposed as a real slice/map by the host code.
func toFilterInput(v any) []any {
	if v == nil {
		return []any{}
	}
	if s, ok := v.(string); ok {
		return []any{s}
	}
	if rv := reflect.ValueOf(v); rv.Kind() == reflect.Map {
		return []any{v}
	}
	if !isArrayLike(v) {
		return []any{v}
	}
	base := toSlice(v)
	out := make([]any, 0, len(base))
	var walk func([]any)
	walk = func(items []any) {
		for _, it := range items {
			if _, isStr := it.(string); !isStr && isArrayLike(it) {
				if sub := toSlice(it); sub != nil {
					walk(sub)
					continue
				}
			}
			out = append(out, it)
		}
	}
	walk(base)
	return out
}

func filterSum(input any, args ...any) any {
	slice := toFilterInput(input)

	var sum float64
	hasFloat := false

	for _, item := range slice {
		var val any
		if len(args) > 0 {
			// Sum by property
			prop := toString(args[0])
			val = getProperty(item, prop)
		} else {
			val = item
		}

		num := toNumber(val)
		if _, ok := num.(float64); ok {
			hasFloat = true
			sum += toFloat(num)
		} else {
			sum += float64(toInt(num))
		}
	}

	if hasFloat {
		return sum
	}
	return int64(sum)
}

// compareValues compares two values for sorting
func compareValues(a, b any) int {
	// Nil sorts last (matches Ruby's nil_safe_compare in standardfilters.rb).
	switch {
	case a == nil && b == nil:
		return 0
	case a == nil:
		return 1
	case b == nil:
		return -1
	}
	// Try numeric comparison first
	aNum := toNumber(a)
	bNum := toNumber(b)
	if aNum != int64(0) || bNum != int64(0) {
		aFloat := toFloat(aNum)
		bFloat := toFloat(bNum)
		return cmp.Compare(aFloat, bFloat)
	}
	// Fall back to string comparison
	return cmp.Compare(toString(a), toString(b))
}

// equalValues checks if two values are equal
func equalValues(a, b any) bool {
	// Handle boolean comparisons
	if bBool, ok := b.(bool); ok {
		if bBool {
			return !isFalsy(a)
		}
		return isFalsy(a)
	}

	// Try numeric equality
	aNum := toNumber(a)
	bNum := toNumber(b)
	if aNum != int64(0) || bNum != int64(0) {
		return toFloat(aNum) == toFloat(bNum)
	}

	// Fall back to string equality
	return toString(a) == toString(b)
}

// Utility filters

// KwargFilterFunc is the legacy signature for filters that accept named
// arguments. Adapted to Filter via its Apply method, which unwraps the
// filterError sentinel just like FilterFunc.Apply. New kwarg-aware
// filters should implement Filter directly (see FilterFuncE).
type KwargFilterFunc func(input any, args []any, kwargs map[string]any) any

// Apply implements Filter for the kwarg-aware legacy signature.
func (f KwargFilterFunc) Apply(input any, args []any, kwargs map[string]any) (any, error) {
	out := f(input, args, kwargs)
	if fe, ok := out.(filterError); ok {
		return nil, fe.err
	}
	return out, nil
}

// filterDefaultKw replaces the basic default filter when called with named
// arguments. Without `allow_false: true`, falsy and blank inputs fall back
// to the default value; with it, only nil and empty (but not false) do.
// Mirrors Shopify standardfilters.rb#default.
func filterDefaultKw(input any, args []any, kwargs map[string]any) any {
	allowFalse, _ := kwargs["allow_false"].(bool)
	fallback := func() any {
		if len(args) > 0 {
			return args[0]
		}
		return ""
	}

	if input == nil {
		return fallback()
	}
	if b, ok := input.(bool); ok {
		if !b && !allowFalse {
			return fallback()
		}
		// false with allow_false:true OR true → fall through to "non-empty"
		return input
	}
	if isEmpty(input) {
		return fallback()
	}
	return input
}

// Math filters

func filterPlus(input any, args ...any) any {
	if len(args) == 0 {
		return input
	}

	a := toNumber(input)
	b := toNumber(args[0])

	// If either is a float, return float
	if _, ok := a.(float64); ok {
		return toFloat(a) + toFloat(b)
	}
	if _, ok := b.(float64); ok {
		return toFloat(a) + toFloat(b)
	}
	return toInt(a) + toInt(b)
}

func filterMinus(input any, args ...any) any {
	if len(args) == 0 {
		return input
	}

	a := toNumber(input)
	b := toNumber(args[0])

	// If either is a float, return float
	if _, ok := a.(float64); ok {
		return toFloat(a) - toFloat(b)
	}
	if _, ok := b.(float64); ok {
		return toFloat(a) - toFloat(b)
	}
	return toInt(a) - toInt(b)
}

func filterTimes(input any, args ...any) any {
	if len(args) == 0 {
		return input
	}

	a := toNumber(input)
	b := toNumber(args[0])

	// If either is a float, return float
	if _, ok := a.(float64); ok {
		return toFloat(a) * toFloat(b)
	}
	if _, ok := b.(float64); ok {
		return toFloat(a) * toFloat(b)
	}
	return toInt(a) * toInt(b)
}

func filterDividedBy(input any, args ...any) any {
	if len(args) == 0 {
		return input
	}

	a := toNumber(input)
	b := toNumber(args[0])

	bFloat := toFloat(b)
	if bFloat == 0 {
		return filterErrorf("divided_by: division by zero")
	}

	// Float result if the divisor has a fractional part; otherwise integer
	// division (matches the previous spec fixtures and is the most common
	// Liquid expectation, since JSON-unmarshalled integers arrive as float64).
	if bFloat != float64(int64(bFloat)) {
		return toFloat(a) / bFloat
	}
	return toInt(a) / toInt(b)
}

func filterModulo(input any, args ...any) any {
	if len(args) == 0 {
		return input
	}

	a := toInt(toNumber(input))
	b := toInt(toNumber(args[0]))

	if b == 0 {
		return filterErrorf("modulo: division by zero")
	}
	return a % b
}

func filterAbs(input any, _ ...any) any {
	num := toNumber(input)
	if f, ok := num.(float64); ok {
		return math.Abs(f)
	}
	i := toInt(num)
	if i < 0 {
		return -i
	}
	return i
}

func filterRound(input any, args ...any) any {
	f := toFloat(toNumber(input))

	if len(args) > 0 {
		precision := int(toInt(toNumber(args[0])))
		multiplier := math.Pow(10, float64(precision))
		return math.Round(f*multiplier) / multiplier
	}

	return int64(math.Round(f))
}

func filterCeil(input any, _ ...any) any {
	f := toFloat(toNumber(input))
	return int64(math.Ceil(f))
}

func filterFloor(input any, _ ...any) any {
	f := toFloat(toNumber(input))
	return int64(math.Floor(f))
}

func filterAtLeast(input any, args ...any) any {
	if len(args) == 0 {
		return input
	}

	a := toNumber(input)
	b := toNumber(args[0])

	aFloat := toFloat(a)
	bFloat := toFloat(b)

	if aFloat < bFloat {
		return b
	}
	return a
}

func filterAtMost(input any, args ...any) any {
	if len(args) == 0 {
		return input
	}

	a := toNumber(input)
	b := toNumber(args[0])

	aFloat := toFloat(a)
	bFloat := toFloat(b)

	if aFloat > bFloat {
		return b
	}
	return a
}

// Date filter

func filterDate(input any, args ...any) any {
	if len(args) == 0 {
		return input
	}

	format := toString(args[0])

	// Parse the input as a time. Mirrors Ruby's Utils.to_date: accepts
	// time values, the literal strings "now"/"today" (current time),
	// integer or numeric-string Unix timestamps, and a handful of common
	// date/time string formats.
	var t time.Time
	switch v := input.(type) {
	case time.Time:
		t = v
	case int:
		t = time.Unix(int64(v), 0)
	case int32:
		t = time.Unix(int64(v), 0)
	case int64:
		t = time.Unix(v, 0)
	case uint, uint32, uint64:
		t = time.Unix(toInt(toNumber(v)), 0)
	case float32:
		t = time.Unix(int64(v), 0)
	case float64:
		t = time.Unix(int64(v), 0)
	case string:
		if strings.EqualFold(v, "now") || strings.EqualFold(v, "today") {
			t = time.Now()
			break
		}
		// Numeric string → Unix timestamp (matches Ruby's UNIX_TIMESTAMP_REGEX).
		if n, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64); err == nil {
			t = time.Unix(n, 0)
			break
		}
		// Try common formats
		formats := []string{
			time.RFC3339,
			"2006-01-02T15:04:05Z07:00",
			"2006-01-02 15:04:05",
			"2006-01-02",
		}
		var err error
		for _, f := range formats {
			t, err = time.Parse(f, v)
			if err == nil {
				break
			}
		}
		if err != nil {
			return input
		}
	default:
		return input
	}

	// Convert strftime format to Go format
	return strftimeToGo(t, format)
}

// strftimeToGo renders a Ruby-strftime format string against t. Each
// directive is expanded individually rather than via global replace so
// that escapes (`%%`) and literal text containing reference patterns
// (e.g. "01" inside output text vs. "%m") cannot collide. Directives that
// don't have a 1:1 Go layout-string analogue (week numbers, year-day,
// unix timestamp) are computed from t directly.
func strftimeToGo(t time.Time, format string) string {
	var sb strings.Builder
	sb.Grow(len(format) + 16)
	for i := 0; i < len(format); i++ {
		if format[i] != '%' || i+1 == len(format) {
			sb.WriteByte(format[i])
			continue
		}
		flag, width, j := parseStrftimeModifiers(format, i+1)
		if j >= len(format) {
			// Trailing `%` with flags but no directive — emit verbatim.
			sb.WriteString(format[i:])
			i = len(format) - 1
			continue
		}
		writeStrftimeDirective(&sb, t, format[j], flag, width)
		i = j
	}
	return sb.String()
}

// parseStrftimeModifiers reads the flag + width modifiers between `%` and the
// directive byte, starting at start. Returns the flag (0 if none), the
// explicit width (-1 if none), and the index of the directive byte.
//
// Ruby strftime flags: `-` strips zero-padding, `_` pads with spaces, `0`
// forces zero-padding. Multiple flags are tolerated; the last one wins.
func parseStrftimeModifiers(format string, start int) (flag byte, width, end int) {
	width = -1
	j := start
	for j < len(format) {
		c := format[j]
		if c != '-' && c != '_' && c != '0' {
			break
		}
		flag = c
		j++
	}
	for j < len(format) && format[j] >= '0' && format[j] <= '9' {
		if width < 0 {
			width = 0
		}
		width = width*10 + int(format[j]-'0')
		j++
	}
	return flag, width, j
}

// writeStrftimeDirective writes the formatted output of a single strftime
// directive into sb. The width is intrinsic to the Ruby strftime spec —
// every supported directive byte appears as one case — and grouping cases
// into per-category helpers would scatter the shared flag/width handling.
//
//nolint:gocyclo,cyclop,funlen // strftime directive dispatch; width matches the Ruby spec.
func writeStrftimeDirective(sb *strings.Builder, t time.Time, directive, flag byte, width int) {
	// spacePad returns flag, defaulting to '_' (space-pad) for directives
	// whose Ruby default is space-padded (e.g. %e, %k, %l).
	spacePad := func() byte {
		if flag == 0 {
			return '_'
		}
		return flag
	}

	switch directive {
	case 'Y':
		writeStrftimeNum(sb, t.Year(), 4, flag, width)
	case 'y':
		writeStrftimeNum(sb, t.Year()%100, 2, flag, width)
	case 'm':
		writeStrftimeNum(sb, int(t.Month()), 2, flag, width)
	case 'd':
		writeStrftimeNum(sb, t.Day(), 2, flag, width)
	case 'e':
		writeStrftimeNum(sb, t.Day(), 2, spacePad(), width)
	case 'H':
		writeStrftimeNum(sb, t.Hour(), 2, flag, width)
	case 'k':
		writeStrftimeNum(sb, t.Hour(), 2, spacePad(), width)
	case 'I':
		writeStrftimeNum(sb, hour12(t), 2, flag, width)
	case 'l':
		writeStrftimeNum(sb, hour12(t), 2, spacePad(), width)
	case 'M':
		writeStrftimeNum(sb, t.Minute(), 2, flag, width)
	case 'S':
		writeStrftimeNum(sb, t.Second(), 2, flag, width)
	case 'p':
		if t.Hour() < 12 {
			sb.WriteString("AM")
		} else {
			sb.WriteString("PM")
		}
	case 'P':
		if t.Hour() < 12 {
			sb.WriteString("am")
		} else {
			sb.WriteString("pm")
		}
	case 'A':
		sb.WriteString(t.Weekday().String())
	case 'a':
		sb.WriteString(t.Weekday().String()[:3])
	case 'B':
		sb.WriteString(t.Month().String())
	case 'b', 'h':
		sb.WriteString(t.Month().String()[:3])
	case 'j':
		writeStrftimeNum(sb, t.YearDay(), 3, flag, width)
	case 'w':
		writeStrftimeNum(sb, int(t.Weekday()), 1, flag, width) // Sunday=0
	case 'u':
		d := int(t.Weekday())
		if d == 0 {
			d = 7
		}
		writeStrftimeNum(sb, d, 1, flag, width) // ISO Monday=1..Sunday=7
	case 'U':
		writeStrftimeNum(sb, weekOfYearSundayStart(t), 2, flag, width)
	case 'W':
		writeStrftimeNum(sb, weekOfYearMondayStart(t), 2, flag, width)
	case 's':
		// Unix epoch can exceed 32 bits past 2038, so format the int64
		// directly rather than narrowing through writeStrftimeNum.
		fmt.Fprintf(sb, "%d", t.Unix())
	case 'Z':
		sb.WriteString(t.Format("MST"))
	case 'z':
		sb.WriteString(t.Format("-0700"))
	case 'c':
		sb.WriteString(strftimeToGo(t, "%a %b %e %H:%M:%S %Y"))
	case 'x', 'D':
		sb.WriteString(strftimeToGo(t, "%m/%d/%y"))
	case 'X', 'T':
		sb.WriteString(strftimeToGo(t, "%H:%M:%S"))
	case 'F':
		sb.WriteString(strftimeToGo(t, "%Y-%m-%d"))
	case 'R':
		sb.WriteString(strftimeToGo(t, "%H:%M"))
	case 'r':
		sb.WriteString(strftimeToGo(t, "%I:%M:%S %p"))
	case 'n':
		sb.WriteByte('\n')
	case 't':
		sb.WriteByte('\t')
	case '%':
		sb.WriteByte('%')
	default:
		// Unknown directive: emit `%`, any modifiers, and the directive
		// verbatim so authors can spot the typo.
		sb.WriteByte('%')
		if flag != 0 {
			sb.WriteByte(flag)
		}
		if width >= 0 {
			fmt.Fprintf(sb, "%d", width)
		}
		sb.WriteByte(directive)
	}
}

// hour12 returns t's hour in 12-hour format (1..12).
func hour12(t time.Time) int {
	h := t.Hour() % 12
	if h == 0 {
		return 12
	}
	return h
}

// writeStrftimeNum formats n into sb using strftime-style flag and width
// modifiers. defaultWidth is the directive's natural padding width when no
// explicit width was given. flag is one of 0 (default zero-pad),
// '-' (no padding), '_' (space-pad), '0' (zero-pad).
func writeStrftimeNum(sb *strings.Builder, n, defaultWidth int, flag byte, width int) {
	w := width
	if w < 0 {
		w = defaultWidth
	}
	switch flag {
	case '-':
		fmt.Fprintf(sb, "%d", n)
	case '_':
		fmt.Fprintf(sb, "%*d", w, n)
	default: // 0 or unset → zero-pad
		fmt.Fprintf(sb, "%0*d", w, n)
	}
}

// weekOfYearSundayStart implements strftime %U: the week number of the
// year (00-53), with the first Sunday being the first day of week 01.
// Days before the first Sunday are in week 00.
func weekOfYearSundayStart(t time.Time) int {
	yday := t.YearDay()
	jan1Weekday := int(time.Date(t.Year(), 1, 1, 0, 0, 0, 0, t.Location()).Weekday())
	return (yday + jan1Weekday - 1) / 7
}

// weekOfYearMondayStart implements strftime %W: same as %U but the first
// Monday begins week 01.
func weekOfYearMondayStart(t time.Time) int {
	yday := t.YearDay()
	jan1Weekday := int(time.Date(t.Year(), 1, 1, 0, 0, 0, 0, t.Location()).Weekday())
	// Shift Sunday=0 to Sunday=6 so Monday=0.
	jan1Weekday = (jan1Weekday + 6) % 7
	return (yday + jan1Weekday - 1) / 7
}

// Type conversion utilities

// toString converts any value to a string.
func toString(v any) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case emptyValue, blankValue:
		// `empty` / `blank` literals render as "" (matches upstream).
		// They retain identity for == comparisons (see `equal`).
		_ = val
		return ""
	case string:
		return val
	case bool:
		if val {
			return "true"
		}
		return "false"
	case int:
		return strconv.Itoa(val)
	case int64:
		return strconv.FormatInt(val, 10)
	case int32:
		return strconv.FormatInt(int64(val), 10)
	case uint:
		return strconv.FormatUint(uint64(val), 10)
	case uint64:
		return strconv.FormatUint(val, 10)
	case float64:
		return strconv.FormatFloat(val, 'g', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(val), 'g', -1, 32)
	case []byte:
		return string(val)
	case []any:
		// Matches Ruby's Array#to_s in Liquid render context: each element
		// stringified and concatenated with no separator (`[1,2,3]` → "123").
		var b strings.Builder
		for _, e := range val {
			b.WriteString(toString(e))
		}
		return b.String()
	case fmt.Stringer:
		return val.String()
	default:
		rv := reflect.ValueOf(val)
		if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
			var b strings.Builder
			for i := range rv.Len() {
				b.WriteString(toString(rv.Index(i).Interface()))
			}
			return b.String()
		}
		return fmt.Sprintf("%v", val)
	}
}

// toSlice converts any value to a slice.
func toSlice(v any) []any {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case []any:
		return val
	case []string:
		result := make([]any, len(val))
		for i, s := range val {
			result[i] = s
		}
		return result
	case []int:
		result := make([]any, len(val))
		for i, n := range val {
			result[i] = n
		}
		return result
	case string:
		// Split into characters (runes). The previous form preallocated
		// len(val) — the BYTE count — and then range-iterated runes,
		// leaving trailing nils on every multi-byte input.
		runes := []rune(val)
		result := make([]any, len(runes))
		for i, r := range runes {
			result[i] = string(r)
		}
		return result
	default:
		rv := reflect.ValueOf(v)
		if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
			result := make([]any, rv.Len())
			for i := range rv.Len() {
				result[i] = rv.Index(i).Interface()
			}
			return result
		}
	}
	return nil
}

// toNumber converts any value to a number (int64 or float64).
func toNumber(v any) any {
	if v == nil {
		return int64(0)
	}
	switch val := v.(type) {
	case int:
		return int64(val)
	case int64:
		return val
	case float64:
		return val
	case string:
		// strconv requires full consumption — Sscanf("%d", "5.5") would
		// silently succeed with 5 and lose the fractional part.
		s := strings.TrimSpace(val)
		if i, err := strconv.ParseInt(s, 10, 64); err == nil {
			return i
		}
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return f
		}
		return int64(0)
	case bool:
		if val {
			return int64(1)
		}
		return int64(0)
	default:
		return int64(0)
	}
}

// toInt converts any value to int64.
func toInt(v any) int64 {
	switch val := v.(type) {
	case int64:
		return val
	case float64:
		return int64(val)
	default:
		return 0
	}
}

// toFloat converts any value to float64.
func toFloat(v any) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case int64:
		return float64(val)
	default:
		return 0
	}
}

// toBool converts any value to a boolean.
func toBool(v any) bool {
	return !isFalsy(v)
}

// isFalsy returns true if the value is considered falsy in Liquid.
//
// Liquid is famously narrow: only nil and false are falsy. Everything else —
// including 0, "", [], and {} — is truthy. (The `empty` and `blank`
// expressions compare equal to their zero values via `==`, but as bare
// conditions they're sentinel objects, hence truthy.) See Shopify's
// condition.rb interpret_condition.
func isFalsy(v any) bool {
	if v == nil {
		return true
	}
	if b, ok := v.(bool); ok {
		return !b
	}
	return false
}

// isEmpty checks if a value matches the "empty" special value.
// Empty: nil, empty string, empty array, empty map
func isEmpty(v any) bool {
	if v == nil {
		return true
	}
	switch val := v.(type) {
	case string:
		return val == ""
	case []any:
		return len(val) == 0
	case map[string]any:
		return len(val) == 0
	default:
		rv := reflect.ValueOf(v)
		switch rv.Kind() {
		case reflect.Array, reflect.Slice, reflect.Map, reflect.String:
			return rv.Len() == 0
		default:
			return false
		}
	}
}

// isBlank checks if a value matches the "blank" special value.
// Blank: nil, false, empty string, whitespace-only string, empty array, empty map
func isBlank(v any) bool {
	if v == nil {
		return true
	}
	switch val := v.(type) {
	case bool:
		return !val
	case string:
		return strings.TrimSpace(val) == ""
	case []any:
		return len(val) == 0
	case map[string]any:
		return len(val) == 0
	default:
		rv := reflect.ValueOf(v)
		switch rv.Kind() {
		case reflect.Array, reflect.Slice, reflect.Map:
			return rv.Len() == 0
		case reflect.String:
			return strings.TrimSpace(rv.String()) == ""
		default:
			return false
		}
	}
}
