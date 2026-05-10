package liquid

import "testing"

// Upstream parity: integration/variable_test.rb

func TestUpstream_Variable_SimpleVariable(t *testing.T) {
	renderEq(t, "1", "{{test}}", map[string]any{"test": "worked"}, "worked")
	renderEq(t, "2", "{{test}}", map[string]any{"test": "worked wonderfully"}, "worked wonderfully")
}

func TestUpstream_Variable_RenderCallsToLiquid(t *testing.T) {
	renderEq(t, "to_liquid", "{{ foo }}",
		map[string]any{"foo": thingWithToLiquid{}}, "foobar")
}

func TestUpstream_Variable_LookupCallsToLiquidValue(t *testing.T) {
	renderEq(t, "int-drop",
		"{{ foo }}",
		map[string]any{"foo": &integerDrop{v: 1}}, "1")
	renderEq(t, "int-index",
		"{{ list[foo] }}",
		map[string]any{"foo": &integerDrop{v: 1}, "list": []any{1, 2, 3}}, "2")
	renderEq(t, "bool-drop",
		"{{ foo }}",
		map[string]any{"foo": &booleanDrop{v: true}}, "Yay")
	renderEq(t, "bool-filtered",
		"{{ foo | upcase }}",
		map[string]any{"foo": &booleanDrop{v: true}}, "YAY")
}

func TestUpstream_Variable_IfTagCallsToLiquidValue(t *testing.T) {
	renderEq(t, "int-eq-literal",
		"{% if foo == 1 %}one{% endif %}",
		map[string]any{"foo": &integerDrop{v: 1}}, "one")
	renderEq(t, "int-eq-drop",
		"{% if foo == eqv %}one{% endif %}",
		map[string]any{"foo": &integerDrop{v: 1}, "eqv": &integerDrop{v: 1}}, "one")
	renderEq(t, "int-lt",
		"{% if 0 < foo %}one{% endif %}",
		map[string]any{"foo": &integerDrop{v: 1}}, "one")
	renderEq(t, "int-gt",
		"{% if foo > 0 %}one{% endif %}",
		map[string]any{"foo": &integerDrop{v: 1}}, "one")
	renderEq(t, "drop-gt-drop",
		"{% if b > a %}one{% endif %}",
		map[string]any{"a": &integerDrop{v: 0}, "b": &integerDrop{v: 1}}, "one")
	renderEq(t, "bool-eq-true",
		"{% if foo == true %}true{% endif %}",
		map[string]any{"foo": &booleanDrop{v: true}}, "true")
}

func TestUpstream_Variable_UnlessTagCallsToLiquidValue(t *testing.T) {
	renderEq(t, "true-suppressed",
		"{% unless foo %}true{% endunless %}",
		map[string]any{"foo": &booleanDrop{v: true}}, "")
	renderEq(t, "false-emits",
		"{% unless foo %}true{% endunless %}",
		map[string]any{"foo": &booleanDrop{v: false}}, "true")
}

func TestUpstream_Variable_CaseTagCallsToLiquidValue(t *testing.T) {
	renderEq(t, "case-drop-1",
		"{% case foo %}{% when 1 %}One{% endcase %}",
		map[string]any{"foo": &integerDrop{v: 1}}, "One")
}

func TestUpstream_Variable_SimpleWithWhitespaces(t *testing.T) {
	renderEq(t, "1", "  {{ test }}  ", map[string]any{"test": "worked"}, "  worked  ")
	renderEq(t, "2", "  {{ test }}  ", map[string]any{"test": "worked wonderfully"}, "  worked wonderfully  ")
}

func TestUpstream_Variable_ExpressionWithWhitespaceInSquareBrackets(t *testing.T) {
	renderEq(t, "1", "{{ a[ 'b' ] }}",
		map[string]any{"a": map[string]any{"b": "result"}}, "result")
}

func TestUpstream_Variable_ExpressionWithWhitespaceInSquareBrackets_Self(t *testing.T) {
	renderEq(t, "self", "{{ a[ self[ 'b' ] ] }}",
		map[string]any{"b": "c", "a": map[string]any{"c": "result"}}, "result")
}

func TestUpstream_Variable_IgnoreUnknown(t *testing.T) {
	renderEq(t, "missing", "{{ test }}", nil, "")
}

func TestUpstream_Variable_UsingBlankAsVariableName(t *testing.T) {
	renderEq(t, "blank", "{% assign foo = blank %}{{ foo }}", nil, "")
}

func TestUpstream_Variable_UsingEmptyAsVariableName(t *testing.T) {
	renderEq(t, "empty", "{% assign foo = empty %}{{ foo }}", nil, "")
}

func TestUpstream_Variable_HashScoping(t *testing.T) {
	d := map[string]any{"test": map[string]any{"test": "worked"}}
	renderEq(t, "1", "{{ test.test }}", d, "worked")
	renderEq(t, "2", "{{ test . test }}", d, "worked")
}

func TestUpstream_Variable_FalseRendersAsFalse(t *testing.T) {
	renderEq(t, "var", "{{ foo }}", map[string]any{"foo": false}, "false")
	renderEq(t, "lit", "{{ false }}", nil, "false")
}

func TestUpstream_Variable_NilRendersAsEmptyString(t *testing.T) {
	renderEq(t, "nil", "{{ nil }}", nil, "")
	renderEq(t, "nil-filter", "{{ nil | append: 'cat' }}", nil, "cat")
}

func TestUpstream_Variable_PresetAssigns(t *testing.T) {
	t.Skip("go-liquid does not expose Template.assigns; assigns flow through Render(data) only")
	// template.assigns['test'] = 'worked'; template.render! => "worked"
}

func TestUpstream_Variable_ReuseParsedTemplate(t *testing.T) {
	tmpl, err := Parse("{{ greeting }} {{ name }}")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	cases := []struct {
		data map[string]any
		want string
	}{
		{map[string]any{"greeting": "Hello", "name": "Tobi"}, "Hello Tobi"},
		{map[string]any{"greeting": "Hello", "unknown": "Tobi"}, "Hello "},
		{map[string]any{"greeting": "Hello", "name": "Brian"}, "Hello Brian"},
	}
	for _, c := range cases {
		got, err := tmpl.Render(c.data)
		if err != nil {
			t.Fatalf("render: %v", err)
		}
		if got != c.want {
			t.Fatalf("want %q got %q", c.want, got)
		}
	}
}

func TestUpstream_Variable_AssignsNotPollutedFromTemplate(t *testing.T) {
	tmpl, err := Parse(`{{ test }}{% assign test = 'bar' %}{{ test }}`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	got, _ := tmpl.Render(map[string]any{"test": "foo"})
	if got != "foobar" {
		t.Fatalf("first render: %q", got)
	}
	// Second render with same data: should still see 'foo' as initial value
	got, _ = tmpl.Render(map[string]any{"test": "foo"})
	if got != "foobar" {
		t.Fatalf("second render: %q", got)
	}
}

func TestUpstream_Variable_HashWithDefaultProc(t *testing.T) {
	t.Skip("Ruby Hash default proc throws on missing key; go-liquid uses plain map (no proc support)")
}

func TestUpstream_Variable_MultilineVariable(t *testing.T) {
	renderEq(t, "multiline", "{{\ntest\n}}", map[string]any{"test": "worked"}, "worked")
}

func TestUpstream_Variable_RenderSymbol(t *testing.T) {
	t.Skip("Go has no symbol type; Ruby :bar renders as 'bar' but no direct go-liquid equivalent")
	renderEq(t, "sym", "{{ foo }}", map[string]any{"foo": "bar"}, "bar")
}

func TestUpstream_Variable_NestedArray(t *testing.T) {
	renderEq(t, "nested", "{{ foo }}",
		map[string]any{"foo": []any{[]any{nil}}}, "")
}

func TestUpstream_Variable_DynamicFindVar(t *testing.T) {
	renderEq(t, "dyn", "{{ self[key] }}",
		map[string]any{"key": "foo", "foo": "bar"}, "bar")
}

func TestUpstream_Variable_RawValueVariable(t *testing.T) {
	renderEq(t, "raw", "{{ self[key] }}",
		map[string]any{"key": "foo", "foo": "bar"}, "bar")
}

func TestUpstream_Variable_DynamicFindVarWithDrop(t *testing.T) {
	renderEq(t, "drop-index-scalar", "{{ self[list[settings.zero]] }}",
		map[string]any{
			"list":     []any{"foo"},
			"settings": &settingsDrop{m: map[string]any{"zero": 0}},
			"foo":      "bar",
		}, "bar")
	renderEq(t, "drop-index-nested", "{{ self[list[settings.zero][\"foo\"]] }}",
		map[string]any{
			"list":     []any{map[string]any{"foo": "bar"}},
			"settings": &settingsDrop{m: map[string]any{"zero": 0}},
			"bar":      "foo",
		}, "foo")
}

func TestUpstream_Variable_DoubleNestedVariableLookup(t *testing.T) {
	renderEq(t, "double-nested", `{{ list[list[settings.zero]]["foo"] }}`,
		map[string]any{
			"list":     []any{1, map[string]any{"foo": "bar"}},
			"settings": &settingsDrop{m: map[string]any{"zero": 0}},
			"bar":      "foo",
		}, "bar")
}

func TestUpstream_Variable_LookupShouldNotHangWithInvalidSyntax(t *testing.T) {
	t.Skip("requires lax error mode that recovers from malformed lookups (e.g. '{{[\\'foo\\'}}')")
}

func TestUpstream_Variable_FilterWithSingleTrailingComma(t *testing.T) {
	t.Skip("requires strict2 mode (tolerates trailing comma in filter args)")
	// strict: parse error; strict2: 'helloworld'
}

func TestUpstream_Variable_MultipleFiltersWithTrailingCommas(t *testing.T) {
	t.Skip("requires strict2 mode")
}

func TestUpstream_Variable_FilterWithColonButNoArguments(t *testing.T) {
	t.Skip("requires strict2 mode (allows '| upcase:' with no args)")
}

func TestUpstream_Variable_FilterChainWithColonNoArgs(t *testing.T) {
	t.Skip("requires strict2 mode")
}

func TestUpstream_Variable_CombiningTrailingCommaAndEmptyArgs(t *testing.T) {
	t.Skip("requires strict2 mode")
}
