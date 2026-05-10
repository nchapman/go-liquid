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
func pos(fn FilterFunc) Filter      { return fn }
func kw(fn KwargFilterFunc) Filter  { return fn }

// filters is the global filter registry, keyed by template name. One
// table covers both positional and kwarg filters via the Filter
// interface; the evaluator dispatches through a single lookup.
var filters = map[string]Filter{
	// String filters
	"upcase":        pos(filterUpcase),
	"downcase":      pos(filterDowncase),
	"capitalize":    pos(filterCapitalize),
	"strip":         pos(filterStrip),
	"lstrip":        pos(filterLstrip),
	"rstrip":        pos(filterRstrip),
	"escape":        pos(filterEscape),
	"split":         pos(filterSplit),
	"append":        pos(filterAppend),
	"prepend":       pos(filterPrepend),
	"replace":       pos(filterReplace),
	"replace_first": pos(filterReplaceFirst),
	"remove":        pos(filterRemove),
	"remove_first":  pos(filterRemoveFirst),
	"truncate":      pos(filterTruncate),
	"truncatewords": pos(filterTruncateWords),
	"slice":         pos(filterSlice),
	"newline_to_br": pos(filterNewlineToBr),
	"escape_once":   pos(filterEscapeOnce),
	"url_encode":    pos(filterURLEncode),
	"url_decode":    pos(filterURLDecode),
	"strip_html":    pos(filterStripHTML),
	"strip_newlines": pos(filterStripNewlines),
	"squish":        pos(filterSquish),
	"replace_last":  pos(filterReplaceLast),
	"remove_last":   pos(filterRemoveLast),

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
	"flatten":      pos(filterFlatten),
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

func filterUpcase(input any, args ...any) any {
	return strings.ToUpper(toString(input))
}

func filterDowncase(input any, args ...any) any {
	return strings.ToLower(toString(input))
}

func filterCapitalize(input any, args ...any) any {
	s := toString(input)
	if len(s) == 0 {
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

func filterStrip(input any, args ...any) any {
	return strings.Trim(toString(input), stripChars)
}

func filterEscape(input any, args ...any) any {
	return html.EscapeString(toString(input))
}

func filterLstrip(input any, args ...any) any {
	return strings.TrimLeft(toString(input), stripChars)
}

func filterRstrip(input any, args ...any) any {
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
	old := toString(args[0])
	new := toString(args[1])
	return strings.ReplaceAll(s, old, new)
}

func filterReplaceFirst(input any, args ...any) any {
	s := toString(input)
	if len(args) < 2 {
		return s
	}
	old := toString(args[0])
	new := toString(args[1])
	return strings.Replace(s, old, new, 1)
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
	s := toString(input)
	length := 50
	ellipsis := "..."

	if len(args) > 0 {
		length = int(toInt(toNumber(args[0])))
	}
	if len(args) > 1 {
		ellipsis = toString(args[1])
	}

	if len(s) <= length {
		return s
	}
	// Keep `length - len(ellipsis)` characters then append the ellipsis.
	// Matches Shopify exactly, including the corner case where `length` is
	// smaller than the ellipsis: the kept portion clamps to 0 and the full
	// ellipsis is still appended (so output may exceed `length`).
	keep := length - len(ellipsis)
	if keep < 0 {
		keep = 0
	}
	return s[:keep] + ellipsis
}

func filterTruncateWords(input any, args ...any) any {
	s := toString(input)
	wordCount := 15
	ellipsis := "..."

	if len(args) > 0 {
		wordCount = int(toInt(toNumber(args[0])))
	}
	if len(args) > 1 {
		ellipsis = toString(args[1])
	}

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

func filterNewlineToBr(input any, args ...any) any {
	s := toString(input)
	// Shopify inserts the <br /> before the newline rather than replacing
	// it, so source line breaks survive into the rendered output.
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.ReplaceAll(s, "\n", "<br />\n")
}

// Array filters

func filterFirst(input any, args ...any) any {
	slice := toSlice(input)
	if len(slice) == 0 {
		return nil
	}
	return slice[0]
}

func filterLast(input any, args ...any) any {
	slice := toSlice(input)
	if len(slice) == 0 {
		return nil
	}
	return slice[len(slice)-1]
}

func filterSize(input any, args ...any) any {
	if input == nil {
		return 0
	}
	switch v := input.(type) {
	case string:
		return len(v)
	case []any:
		return len(v)
	default:
		rv := reflect.ValueOf(input)
		switch rv.Kind() {
		case reflect.Array, reflect.Slice, reflect.Map, reflect.String:
			return rv.Len()
		}
	}
	return 0
}

func filterJoin(input any, args ...any) any {
	slice := toSlice(input)
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

func filterReverse(input any, args ...any) any {
	slice := toSlice(input)
	result := make([]any, len(slice))
	for i, v := range slice {
		result[len(slice)-1-i] = v
	}
	return result
}

func filterSort(input any, args ...any) any {
	slice := toSlice(input)
	if slice == nil {
		return nil
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
	slices.SortFunc(result, func(a, b any) int {
		return compareValues(a, b)
	})
	return result
}

func filterSortNatural(input any, args ...any) any {
	slice := toSlice(input)
	if slice == nil {
		return nil
	}

	result := make([]any, len(slice))
	copy(result, slice)

	// If a property is specified, sort by that property (case-insensitive)
	if len(args) > 0 {
		prop := toString(args[0])
		slices.SortFunc(result, func(a, b any) int {
			aVal := strings.ToLower(toString(getProperty(a, prop)))
			bVal := strings.ToLower(toString(getProperty(b, prop)))
			return cmp.Compare(aVal, bVal)
		})
		return result
	}

	// Sort by value (case-insensitive)
	slices.SortFunc(result, func(a, b any) int {
		return cmp.Compare(strings.ToLower(toString(a)), strings.ToLower(toString(b)))
	})
	return result
}

func filterMap(input any, args ...any) any {
	slice := toSlice(input)
	if slice == nil || len(args) == 0 {
		return nil
	}

	prop := toString(args[0])
	result := make([]any, len(slice))
	for i, item := range slice {
		result[i] = getProperty(item, prop)
	}
	return result
}

func filterWhere(input any, args ...any) any {
	slice := toSlice(input)
	if slice == nil || len(args) == 0 {
		return nil
	}

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
	slice := toSlice(input)
	if slice == nil || len(args) == 0 {
		return nil
	}

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
	slice := toSlice(input)
	if slice == nil {
		return nil
	}

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
	slice := toSlice(input)
	if slice == nil {
		return nil
	}

	var result []any
	for _, item := range slice {
		if item != nil {
			result = append(result, item)
		}
	}
	return result
}

func filterConcat(input any, args ...any) any {
	slice := toSlice(input)
	if slice == nil {
		slice = []any{}
	}

	result := make([]any, len(slice))
	copy(result, slice)

	for _, arg := range args {
		if !isArrayLike(arg) {
			return filterErrorf("concat: argument is not an array")
		}
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

func filterFlatten(input any, args ...any) any {
	slice := toSlice(input)
	if slice == nil {
		return []any{}
	}
	var result []any
	var walk func([]any)
	walk = func(items []any) {
		for _, item := range items {
			// Mirror Ruby Array#flatten: recursively flatten any nested
			// arrays, but never descend into strings (toSlice splits them).
			if _, isString := item.(string); !isString {
				if sub := toSlice(item); sub != nil {
					walk(sub)
					continue
				}
			}
			result = append(result, item)
		}
	}
	walk(slice)
	return result
}

func filterSum(input any, args ...any) any {
	slice := toSlice(input)
	if slice == nil {
		return 0
	}

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

func filterAbs(input any, args ...any) any {
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

func filterCeil(input any, args ...any) any {
	f := toFloat(toNumber(input))
	return int64(math.Ceil(f))
}

func filterFloor(input any, args ...any) any {
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

	// Parse the input as a time
	var t time.Time
	switch v := input.(type) {
	case time.Time:
		t = v
	case string:
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

// strftimeToGo converts a strftime format string to Go's time format
func strftimeToGo(t time.Time, format string) string {
	// Map of strftime directives to Go format
	replacements := map[string]string{
		"%Y": "2006",
		"%y": "06",
		"%m": "01",
		"%d": "02",
		"%H": "15",
		"%I": "03",
		"%M": "04",
		"%S": "05",
		"%p": "PM",
		"%A": "Monday",
		"%a": "Mon",
		"%B": "January",
		"%b": "Jan",
		"%Z": "MST",
		"%z": "-0700",
		"%%": "%",
	}

	result := format
	for directive, goFormat := range replacements {
		result = strings.ReplaceAll(result, directive, goFormat)
	}

	return t.Format(result)
}

// Type conversion utilities

// toString converts any value to a string.
func toString(v any) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case bool:
		if val {
			return "true"
		}
		return "false"
	case int:
		return fmt.Sprintf("%d", val)
	case int64:
		return fmt.Sprintf("%d", val)
	case float64:
		// Format without trailing zeros
		s := fmt.Sprintf("%g", val)
		return s
	default:
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
		// Split string into characters
		result := make([]any, len(val))
		for i, c := range val {
			result[i] = string(c)
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
		}
	}
	return false
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
		}
	}
	return false
}

