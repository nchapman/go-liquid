package liquid

import (
	"cmp"
	"fmt"
	"html"
	"math"
	"reflect"
	"slices"
	"strings"
	"time"
	"unicode/utf8"
)

// FilterFunc is the signature for filter functions.
// Input is the value being filtered, args are any additional arguments.
type FilterFunc func(input any, args ...any) any

// filters is the global filter registry.
var filters = map[string]FilterFunc{
	// String filters
	"upcase":        filterUpcase,
	"downcase":      filterDowncase,
	"capitalize":    filterCapitalize,
	"strip":         filterStrip,
	"lstrip":        filterLstrip,
	"rstrip":        filterRstrip,
	"escape":        filterEscape,
	"split":         filterSplit,
	"append":        filterAppend,
	"prepend":       filterPrepend,
	"replace":       filterReplace,
	"replace_first": filterReplaceFirst,
	"remove":        filterRemove,
	"remove_first":  filterRemoveFirst,
	"truncate":      filterTruncate,
	"truncatewords": filterTruncateWords,
	"slice":         filterSlice,
	"newline_to_br": filterNewlineToBr,

	// Array filters
	"first":        filterFirst,
	"last":         filterLast,
	"size":         filterSize,
	"join":         filterJoin,
	"reverse":      filterReverse,
	"sort":         filterSort,
	"sort_natural": filterSortNatural,
	"map":          filterMap,
	"where":        filterWhere,
	"find":         filterFind,
	"uniq":         filterUniq,
	"compact":      filterCompact,
	"concat":       filterConcat,
	"flatten":      filterFlatten,
	"sum":          filterSum,

	// Utility filters
	"default": filterDefault,

	// Math filters
	"plus":       filterPlus,
	"minus":      filterMinus,
	"times":      filterTimes,
	"divided_by": filterDividedBy,
	"modulo":     filterModulo,
	"abs":        filterAbs,
	"round":      filterRound,
	"ceil":       filterCeil,
	"floor":      filterFloor,
	"at_least":   filterAtLeast,
	"at_most":    filterAtMost,

	// Date filter
	"date": filterDate,
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

func filterStrip(input any, args ...any) any {
	return strings.TrimSpace(toString(input))
}

func filterEscape(input any, args ...any) any {
	return html.EscapeString(toString(input))
}

func filterLstrip(input any, args ...any) any {
	return strings.TrimLeft(toString(input), " \t\n\r")
}

func filterRstrip(input any, args ...any) any {
	return strings.TrimRight(toString(input), " \t\n\r")
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
	// Truncate to length minus ellipsis length
	truncLen := max(0, length-len(ellipsis))
	return s[:truncLen] + ellipsis
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

	// Handle string slicing
	if s, ok := input.(string); ok {
		// Handle negative offset
		if offset < 0 {
			offset = len(s) + offset
		}
		if offset < 0 {
			offset = 0
		}
		if offset >= len(s) {
			return ""
		}
		end := min(offset+length, len(s))
		return s[offset:end]
	}

	// Handle array slicing
	slice := toSlice(input)
	if slice == nil {
		return nil
	}

	// Handle negative offset
	if offset < 0 {
		offset = len(slice) + offset
	}
	if offset < 0 {
		offset = 0
	}
	if offset >= len(slice) {
		return nil
	}
	end := min(offset+length, len(slice))
	return slice[offset:end]
}

func filterNewlineToBr(input any, args ...any) any {
	s := toString(input)
	return strings.ReplaceAll(s, "\n", "<br />")
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

func filterUniq(input any, args ...any) any {
	slice := toSlice(input)
	if slice == nil {
		return nil
	}

	seen := make(map[string]bool)
	var result []any
	for _, item := range slice {
		key := toString(item)
		if !seen[key] {
			seen[key] = true
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
		argSlice := toSlice(arg)
		result = append(result, argSlice...)
	}
	return result
}

func filterFlatten(input any, args ...any) any {
	slice := toSlice(input)
	if slice == nil {
		return []any{}
	}

	var result []any
	for _, item := range slice {
		// If item is a slice, flatten it
		if itemSlice := toSlice(item); itemSlice != nil {
			result = append(result, itemSlice...)
		} else {
			result = append(result, item)
		}
	}
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

func filterDefault(input any, args ...any) any {
	if isFalsy(input) {
		if len(args) > 0 {
			return args[0]
		}
		return ""
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

	// Check for division by zero
	bFloat := toFloat(b)
	if bFloat == 0 {
		return 0
	}

	// If divisor has a fractional part, return float
	// Otherwise do integer division
	if bFloat != float64(int64(bFloat)) {
		return toFloat(a) / bFloat
	}

	// Integer division
	return toInt(a) / toInt(b)
}

func filterModulo(input any, args ...any) any {
	if len(args) == 0 {
		return input
	}

	a := toInt(toNumber(input))
	b := toInt(toNumber(args[0]))

	if b == 0 {
		return 0
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
		// Try parsing as int first, then float
		var i int64
		if _, err := fmt.Sscanf(val, "%d", &i); err == nil {
			return i
		}
		var f float64
		if _, err := fmt.Sscanf(val, "%f", &f); err == nil {
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
// Falsy: nil, false, empty string
func isFalsy(v any) bool {
	if v == nil {
		return true
	}
	switch val := v.(type) {
	case bool:
		return !val
	case string:
		return val == ""
	case emptyValue:
		return true
	case blankValue:
		return true
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

// getFilter returns a filter function by name.
func getFilter(name string) (FilterFunc, bool) {
	f, ok := filters[name]
	return f, ok
}
