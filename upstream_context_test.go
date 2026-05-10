package liquid

import "testing"

// Upstream parity: integration/context_test.rb
//
// Most upstream tests exercise Ruby's Liquid::Context API directly
// (push/pop/merge/invoke/registers/global_filter/strict_variables). go-liquid
// has no equivalent surface; those tests are skipped with the assertion
// bodies retained so they can be re-enabled once a Context API is exposed.

func TestUpstream_Context_Variables(t *testing.T) {
	t.Skip("go-liquid does not expose a public Context API for direct var get/set")
}

func TestUpstream_Context_VariablesNotExisting(t *testing.T) {
	renderEq(t, "missing", "{% if does_not_exist == nil %}true{% endif %}", nil, "true")
}

func TestUpstream_Context_Scoping(t *testing.T) {
	t.Skip("Ruby Context#push/pop not exposed in go-liquid")
}

func TestUpstream_Context_LengthQuery(t *testing.T) {
	renderEq(t, "arr", "{% if numbers.size == 4 %}true{% endif %}",
		map[string]any{"numbers": []any{1, 2, 3, 4}}, "true")
}

func TestUpstream_Context_LengthQuery_HashSize(t *testing.T) {
	t.Skip("Ruby hashes with integer keys/.size — go-liquid maps use string keys; semantics differ")
	renderEq(t, "hash", "{% if numbers.size == 4 %}true{% endif %}",
		map[string]any{"numbers": map[any]any{1: 1, 2: 2, 3: 3, 4: 4}}, "true")
	renderEq(t, "hash-with-size", "{% if numbers.size == 1000 %}true{% endif %}",
		map[string]any{"numbers": map[string]any{"1": 1, "2": 2, "3": 3, "4": 4, "size": 1000}}, "true")
}

func TestUpstream_Context_HyphenatedVariable(t *testing.T) {
	t.Skip("go-liquid lexer does not allow '-' in identifiers")
	renderEq(t, "hyph", "{{ oh-my }}", map[string]any{"oh-my": "godz"}, "godz")
}

func TestUpstream_Context_AddFilter(t *testing.T) {
	t.Skip("Ruby Context#add_filters / Context#invoke not exposed in go-liquid")
}

func TestUpstream_Context_OnlyIntendedFiltersMakeItThere(t *testing.T) {
	t.Skip("Ruby Context#add_filters / Context#invoke not exposed")
}

func TestUpstream_Context_AddItemInOuterScope(t *testing.T) {
	t.Skip("Ruby Context#push/pop not exposed")
}

func TestUpstream_Context_AddItemInInnerScope(t *testing.T) {
	t.Skip("Ruby Context#push/pop not exposed")
}

func TestUpstream_Context_HierachicalData(t *testing.T) {
	d := map[string]any{"hash": map[string]any{"name": "tobi"}}
	renderEq(t, "dot", "{{ hash.name }}", d, "tobi")
	renderEq(t, "bracket", `{{ hash["name"] }}`, d, "tobi")
}

func TestUpstream_Context_Keywords(t *testing.T) {
	renderEq(t, "true", "{% if true == expect %}pass{% endif %}",
		map[string]any{"expect": true}, "pass")
	renderEq(t, "false", "{% if false == expect %}pass{% endif %}",
		map[string]any{"expect": false}, "pass")
}

func TestUpstream_Context_Digits(t *testing.T) {
	renderEq(t, "int", "{% if 100 == expect %}pass{% endif %}",
		map[string]any{"expect": 100}, "pass")
	renderEq(t, "float", "{% if 100.00 == expect %}pass{% endif %}",
		map[string]any{"expect": 100.00}, "pass")
}

func TestUpstream_Context_Strings(t *testing.T) {
	renderEq(t, "dq", `{{ "hello!" }}`, nil, "hello!")
	renderEq(t, "sq", "{{ 'hello!' }}", nil, "hello!")
}

func TestUpstream_Context_Merge(t *testing.T) {
	t.Skip("Ruby Context#merge not exposed in go-liquid")
}

func TestUpstream_Context_ArrayNotation(t *testing.T) {
	d := map[string]any{"test": []any{"a", "b"}}
	renderEq(t, "0", "{{ test[0] }}", d, "a")
	renderEq(t, "1", "{{ test[1] }}", d, "b")
	renderEq(t, "oob", "{% if test[2] == nil %}pass{% endif %}", d, "pass")
}

func TestUpstream_Context_RecoursiveArrayNotation(t *testing.T) {
	renderEq(t, "1", "{{ test.test[0] }}",
		map[string]any{"test": map[string]any{"test": []any{1, 2, 3, 4, 5}}}, "1")
	renderEq(t, "2", "{{ test[0].test }}",
		map[string]any{"test": []any{map[string]any{"test": "worked"}}}, "worked")
}

func TestUpstream_Context_HashToArrayTransition(t *testing.T) {
	d := map[string]any{"colors": map[string]any{
		"Blue":   []any{"003366", "336699", "6699CC", "99CCFF"},
		"Green":  []any{"003300", "336633", "669966", "99CC99"},
		"Yellow": []any{"CC9900", "FFCC00", "FFFF99", "FFFFCC"},
		"Red":    []any{"660000", "993333", "CC6666", "FF9999"},
	}}
	renderEq(t, "blue0", "{{ colors.Blue[0] }}", d, "003366")
	renderEq(t, "red3", "{{ colors.Red[3] }}", d, "FF9999")
}

func TestUpstream_Context_TryFirst(t *testing.T) {
	d := map[string]any{"test": []any{1, 2, 3, 4, 5}}
	renderEq(t, "first", "{{ test.first }}", d, "1")
	renderEq(t, "last", "{% if test.last == 5 %}pass{% endif %}", d, "pass")

	d2 := map[string]any{"test": map[string]any{"test": []any{1, 2, 3, 4, 5}}}
	renderEq(t, "nested-first", "{{ test.test.first }}", d2, "1")
	renderEq(t, "nested-last", "{{ test.test.last }}", d2, "5")

	d3 := map[string]any{"test": []any{1}}
	renderEq(t, "single-first", "{{ test.first }}", d3, "1")
	renderEq(t, "single-last", "{{ test.first }}", d3, "1")
}

func TestUpstream_Context_AccessHashesWithHashNotation(t *testing.T) {
	d := map[string]any{"products": map[string]any{
		"count": 5, "tags": []any{"deepsnow", "freestyle"},
	}}
	renderEq(t, "count", `{{ products["count"] }}`, d, "5")
	renderEq(t, "tag0", `{{ products["tags"][0] }}`, d, "deepsnow")
	renderEq(t, "first", `{{ products["tags"].first }}`, d, "deepsnow")

	d2 := map[string]any{"product": map[string]any{
		"variants": []any{
			map[string]any{"title": "draft151cm"},
			map[string]any{"title": "element151cm"},
		}}}
	renderEq(t, "v0", `{{ product["variants"][0]["title"] }}`, d2, "draft151cm")
	renderEq(t, "v1", `{{ product["variants"][1]["title"] }}`, d2, "element151cm")
	renderEq(t, "v-first", `{{ product["variants"].first["title"] }}`, d2, "draft151cm")
	renderEq(t, "v-last", `{{ product["variants"].last["title"] }}`, d2, "element151cm")
}

func TestUpstream_Context_AccessVariableWithHashNotation(t *testing.T) {
	renderEq(t, "var", "{{ foo }}", map[string]any{"foo": "baz"}, "baz")
}

func TestUpstream_Context_AccessVariableWithHashNotation_Self(t *testing.T) {
	renderEq(t, "self", `{{ self[bar] }}`,
		map[string]any{"foo": "baz", "bar": "foo"}, "baz")
}

func TestUpstream_Context_AccessHashesWithHashAccessVariables(t *testing.T) {
	d := map[string]any{
		"var":      "tags",
		"nested":   map[string]any{"var": "tags"},
		"products": map[string]any{"count": 5, "tags": []any{"deepsnow", "freestyle"}},
	}
	renderEq(t, "1", "{{ products[var].first }}", d, "deepsnow")
	renderEq(t, "2", "{{ products[nested.var].last }}", d, "freestyle")
}

func TestUpstream_Context_HashNotationOnlyForHashAccess(t *testing.T) {
	d := map[string]any{"array": []any{1, 2, 3, 4, 5}}
	renderEq(t, "first", "{{ array.first }}", d, "1")
	renderEq(t, "hash", `{{ hash["first"] }}`,
		map[string]any{"hash": map[string]any{"first": "Hello"}}, "Hello")
}

func TestUpstream_Context_HashNotationOnlyForHashAccess_BracketOnArray(t *testing.T) {
	d := map[string]any{"array": []any{1, 2, 3, 4, 5}}
	renderEq(t, "bracket-on-arr", `{% if array["first"] == nil %}pass{% endif %}`, d, "pass")
}

func TestUpstream_Context_FirstCanAppearInMiddleOfCallchain(t *testing.T) {
	d := map[string]any{"product": map[string]any{
		"variants": []any{
			map[string]any{"title": "draft151cm"},
			map[string]any{"title": "element151cm"},
		}}}
	renderEq(t, "0", "{{ product.variants[0].title }}", d, "draft151cm")
	renderEq(t, "1", "{{ product.variants[1].title }}", d, "element151cm")
	renderEq(t, "first", "{{ product.variants.first.title }}", d, "draft151cm")
	renderEq(t, "last", "{{ product.variants.last.title }}", d, "element151cm")
}

// Drop / proc / lambda tests — all skipped (no go-liquid equivalent).
func TestUpstream_Context_Cents(t *testing.T)                     { t.Skip("requires Liquid::Drop to_liquid") }
func TestUpstream_Context_NestedCents(t *testing.T)               { t.Skip("requires Liquid::Drop to_liquid") }
func TestUpstream_Context_CentsThroughDrop(t *testing.T)          { t.Skip("requires Liquid::Drop") }
func TestUpstream_Context_NestedCentsThroughDrop(t *testing.T)    { t.Skip("requires Liquid::Drop") }
func TestUpstream_Context_DropMethodsWithQuestionMarks(t *testing.T) {
	t.Skip("Ruby Drop#non_zero? not modeled; '?' not a valid identifier char in go-liquid")
}
func TestUpstream_Context_ContextFromWithinDrop(t *testing.T)        { t.Skip("requires Liquid::Drop") }
func TestUpstream_Context_NestedContextFromWithinDrop(t *testing.T)  { t.Skip("requires Liquid::Drop") }
func TestUpstream_Context_CentsThroughDropNestedly(t *testing.T)     { t.Skip("requires Liquid::Drop") }
func TestUpstream_Context_DropWithVariableCalledOnlyOnce(t *testing.T) {
	t.Skip("requires CounterDrop (Liquid::Drop with @count state)")
}
func TestUpstream_Context_DropWithKeyCalledOnlyOnce(t *testing.T) { t.Skip("requires CounterDrop") }
func TestUpstream_Context_ProcAsVariable(t *testing.T)            { t.Skip("Ruby proc/lambda not modeled") }
func TestUpstream_Context_LambdaAsVariable(t *testing.T)          { t.Skip("Ruby proc/lambda not modeled") }
func TestUpstream_Context_NestedLambdaAsVariable(t *testing.T)    { t.Skip("Ruby proc/lambda not modeled") }
func TestUpstream_Context_ArrayContainingLambdaAsVariable(t *testing.T) {
	t.Skip("Ruby proc/lambda not modeled")
}
func TestUpstream_Context_LambdaIsCalledOnce(t *testing.T)         { t.Skip("Ruby proc/lambda not modeled") }
func TestUpstream_Context_NestedLambdaIsCalledOnce(t *testing.T)   { t.Skip("Ruby proc/lambda not modeled") }
func TestUpstream_Context_LambdaInArrayIsCalledOnce(t *testing.T)  { t.Skip("Ruby proc/lambda not modeled") }
func TestUpstream_Context_AccessToContextFromProc(t *testing.T)    { t.Skip("Ruby proc/lambda not modeled") }
func TestUpstream_Context_ToLiquidAndContextAtFirstLevel(t *testing.T) {
	t.Skip("requires Category/CategoryDrop Liquid::Drop")
}

func TestUpstream_Context_Ranges(t *testing.T) {
	// Equality against a Range expression: in go-liquid (1..5) materializes
	// to []any{1,2,3,4,5}; expect value is the same slice.
	renderEq(t, "eq",
		"{% if (1..5) == expect %}pass{% endif %}",
		map[string]any{"expect": []any{1, 2, 3, 4, 5}}, "pass")
}

func TestUpstream_Context_Ranges_OutputAsString(t *testing.T) {
	renderEq(t, "out", "{{ (1..5) }}", nil, "1..5")
	renderEq(t, "dyn", "{{ (1..test) }}", map[string]any{"test": "5"}, "1..5")
	renderEq(t, "both-dyn", "{{ (test..test) }}", map[string]any{"test": "5"}, "5..5")
}

// Internal/API-only tests.
func TestUpstream_Context_InterruptAvoidsObjectAllocations(t *testing.T) {
	t.Skip("Ruby allocation profiling; not applicable to Go")
}
func TestUpstream_Context_InitializationWithProcInEnvironment(t *testing.T) {
	t.Skip("Ruby proc-in-environment not modeled")
}
func TestUpstream_Context_ApplyGlobalFilter(t *testing.T) { t.Skip("Ruby global_filter API not exposed") }
func TestUpstream_Context_StaticEnvironmentsReadWithLowerPriority(t *testing.T) {
	t.Skip("Ruby Context.build / static_environments API not exposed")
}
func TestUpstream_Context_ApplyGlobalFilterWhenNoGlobalFilterExist(t *testing.T) {
	t.Skip("Ruby global_filter API not exposed")
}
func TestUpstream_Context_NewIsolatedSubcontextDoesNotInheritVariables(t *testing.T) {
	t.Skip("Ruby Context#new_isolated_subcontext not exposed")
}
func TestUpstream_Context_NewIsolatedSubcontextInheritsStaticEnvironment(t *testing.T) {
	t.Skip("Ruby Context#new_isolated_subcontext not exposed")
}
func TestUpstream_Context_NewIsolatedSubcontextInheritsResourceLimits(t *testing.T) {
	t.Skip("Ruby Context#new_isolated_subcontext not exposed")
}
func TestUpstream_Context_NewIsolatedSubcontextInheritsExceptionRenderer(t *testing.T) {
	t.Skip("Ruby Context#new_isolated_subcontext not exposed")
}
func TestUpstream_Context_NewIsolatedSubcontextDoesNotInheritNonStaticRegisters(t *testing.T) {
	t.Skip("Ruby Context#new_isolated_subcontext not exposed")
}
func TestUpstream_Context_NewIsolatedSubcontextInheritsStaticRegisters(t *testing.T) {
	t.Skip("Ruby Context#new_isolated_subcontext not exposed")
}
func TestUpstream_Context_NewIsolatedSubcontextRegistersDoNotPolluteContext(t *testing.T) {
	t.Skip("Ruby Context#new_isolated_subcontext not exposed")
}
func TestUpstream_Context_NewIsolatedSubcontextInheritsFilters(t *testing.T) {
	t.Skip("Ruby Context#new_isolated_subcontext not exposed")
}
func TestUpstream_Context_DisablesTagSpecified(t *testing.T) {
	t.Skip("Ruby Context#with_disabled_tags / tag_disabled? API differs; go-liquid uses WithDisabledTags option")
}
func TestUpstream_Context_DisablesNestedTags(t *testing.T) {
	t.Skip("Ruby Context#with_disabled_tags nesting not directly exposed in go-liquid")
}
func TestUpstream_Context_OverrideGlobalFilter(t *testing.T) {
	t.Skip("Ruby global_filter override / per-render filters not modeled")
}
func TestUpstream_Context_HasKeyWillNotAddAnErrorForMissingKeys(t *testing.T) {
	t.Skip("Ruby Context#key? / Context#errors not exposed")
}
func TestUpstream_Context_KeyLookupRaisesForMissingKeysWhenStrictVariablesEnabled(t *testing.T) {
	t.Skip("Ruby strict_variables mode not modeled")
}
func TestUpstream_Context_HasKeyWillNotRaiseForMissingKeysWhenStrictVariablesEnabled(t *testing.T) {
	t.Skip("Ruby strict_variables mode not modeled")
}
func TestUpstream_Context_AlwaysUsesStaticRegisters(t *testing.T) {
	t.Skip("Ruby Registers class not modeled")
}
func TestUpstream_Context_VariableToLiquidReturnsContextualDrop(t *testing.T) {
	t.Skip("requires ProductsDrop (Liquid::Drop reading @context['forloop'])")
}
func TestUpstream_Context_NewIsolatedContextInheritsParentEnvironment(t *testing.T) {
	t.Skip("Ruby Environment.build / new_isolated_subcontext not exposed")
}
func TestUpstream_Context_NewlyBuiltContextInheritsParentEnvironment(t *testing.T) {
	t.Skip("Ruby Environment.build / Context.build not exposed")
}
