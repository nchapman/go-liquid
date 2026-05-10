package liquid

import "testing"

// Upstream parity: integration/template_test.rb
//
// Most upstream tests target Ruby's Template.new / .parse / .assigns /
// resource_limits / exception_renderer / strict_variables APIs. go-liquid's
// Parse → Render flow is intentionally simpler; those tests are skipped
// with bodies retained so they can be wired up if the API surface grows.

func TestUpstream_Template_InstanceAssignsPersistOnSameTemplateBetweenParses(t *testing.T) {
	t.Skip("Ruby Template.new + multiple .parse calls share instance @assigns; go-liquid templates are single-parse")
}

func TestUpstream_Template_WarningsNotExponentialTime(t *testing.T) {
	t.Skip("Ruby Template#warnings API not exposed in go-liquid")
}

func TestUpstream_Template_InstanceAssignsPersistOnSameTemplateBetweenRenders(t *testing.T) {
	tmpl, err := Parse("{{ foo }}{% assign foo = 'foo' %}{{ foo }}")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	got1, _ := tmpl.Render(nil)
	if got1 != "foo" {
		t.Fatalf("first render: %q", got1)
	}
	got2, _ := tmpl.Render(nil)
	// Ruby: second render sees the assigned foo from prior render. go-liquid:
	// each Render starts with a fresh scope, so still "foo" (not "foofoo").
	if got2 != "foo" {
		t.Fatalf("second render: %q (Ruby would be 'foofoo'; go-liquid intentionally has no inter-render state)", got2)
	}
}

func TestUpstream_Template_CustomAssignsDoNotPersistOnSameTemplate(t *testing.T) {
	tmpl, err := Parse("{{ foo }}")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	got1, _ := tmpl.Render(map[string]any{"foo": "from custom assigns"})
	if got1 != "from custom assigns" {
		t.Fatalf("first: %q", got1)
	}
	got2, _ := tmpl.Render(nil)
	if got2 != "" {
		t.Fatalf("second: %q", got2)
	}
}

func TestUpstream_Template_CustomAssignsSquashInstanceAssigns(t *testing.T) {
	t.Skip("requires Template.new + multi-parse instance assigns")
}

func TestUpstream_Template_PersistentAssignsSquashInstanceAssigns(t *testing.T) {
	t.Skip("Ruby Template#assigns API not exposed in go-liquid")
}

func TestUpstream_Template_LambdaCalledOnceFromPersistentAssigns(t *testing.T) {
	t.Skip("Ruby lambda assigns not modeled")
}

func TestUpstream_Template_LambdaCalledOnceFromCustomAssigns(t *testing.T) {
	t.Skip("Ruby lambda assigns not modeled")
}

func TestUpstream_Template_ResourceLimitsWorksWithCustomLengthMethod(t *testing.T) {
	t.Skip("Ruby ResourceLimits API differs; go-liquid uses Environment options")
}

func TestUpstream_Template_ResourceLimitsRenderLength(t *testing.T) {
	t.Skip("Ruby ResourceLimits API differs")
}

func TestUpstream_Template_ResourceLimitsRenderScore(t *testing.T) {
	t.Skip("Ruby ResourceLimits API differs")
}

func TestUpstream_Template_ResourceLimitsAbortsRenderingAfterFirstError(t *testing.T) {
	t.Skip("Ruby ResourceLimits API differs")
}

func TestUpstream_Template_ResourceLimitsHashInTemplateGetsUpdatedEvenIfNoLimitsAreSet(t *testing.T) {
	t.Skip("Ruby ResourceLimits API differs")
}

func TestUpstream_Template_RenderLengthPersistsBetweenBlocks(t *testing.T) {
	t.Skip("Ruby ResourceLimits API differs")
}

func TestUpstream_Template_RenderLengthUsesNumberOfBytesNotCharacters(t *testing.T) {
	t.Skip("Ruby ResourceLimits API differs")
}

func TestUpstream_Template_CumulativeRenderScoreLimitAcrossRenderTags(t *testing.T) {
	t.Skip("Ruby ResourceLimits cumulative tracking API not exposed")
}

func TestUpstream_Template_CumulativeRenderScoreLimitRaisesOnRenderBang(t *testing.T) {
	t.Skip("Ruby render! / ResourceLimits API not exposed")
}

func TestUpstream_Template_CumulativeAssignScoreLimitAcrossIncludeTags(t *testing.T) {
	t.Skip("Ruby ResourceLimits across includes API not exposed")
}

func TestUpstream_Template_CumulativeRenderScoreTracksAcrossPartialsWithoutLimit(t *testing.T) {
	t.Skip("Ruby ResourceLimits API not exposed")
}

func TestUpstream_Template_DefaultResourceLimitsUnaffectedByRenderWithContext(t *testing.T) {
	t.Skip("Ruby ResourceLimits API not exposed")
}

func TestUpstream_Template_CanUseDropAsContext(t *testing.T) {
	t.Skip("Ruby Liquid::Drop as render context not modeled")
}

func TestUpstream_Template_RenderBangForceRethrowErrorsOnPassedContext(t *testing.T) {
	t.Skip("Ruby render! / Context rethrow API not exposed")
}

func TestUpstream_Template_ExceptionRendererThatReturnsString(t *testing.T) {
	t.Skip("Ruby exception_renderer API not exposed in go-liquid")
}

func TestUpstream_Template_ExceptionRendererThatRaises(t *testing.T) {
	t.Skip("Ruby exception_renderer API not exposed")
}

func TestUpstream_Template_GlobalFilterOptionOnRender(t *testing.T) {
	t.Skip("Ruby render(global_filter: ...) per-call option not exposed")
}

func TestUpstream_Template_GlobalFilterOptionWhenNativeFiltersExist(t *testing.T) {
	t.Skip("Ruby render(global_filter: ...) per-call option not exposed")
}

func TestUpstream_Template_UndefinedVariables(t *testing.T) {
	t.Skip("Ruby strict_variables / Template#errors API not exposed in go-liquid")
}

func TestUpstream_Template_NilValueDoesNotRaise(t *testing.T) {
	// Even in strict_variables, nil-valued keys do not raise. go-liquid
	// always treats nil as empty; verify the same render output.
	renderEq(t, "nil", "some{{x}}thing", map[string]any{"x": nil}, "something")
}

func TestUpstream_Template_UndefinedVariablesRaise(t *testing.T) {
	t.Skip("Ruby strict_variables mode not modeled")
}

func TestUpstream_Template_UndefinedDropMethods(t *testing.T) {
	t.Skip("Ruby Liquid::Drop with strict_variables / UndefinedDropMethod not modeled")
}

func TestUpstream_Template_UndefinedDropMethodsRaise(t *testing.T) {
	t.Skip("Ruby Liquid::Drop with strict_variables / UndefinedDropMethod not modeled")
}

func TestUpstream_Template_UndefinedFilters(t *testing.T) {
	t.Skip("Ruby strict_filters + Template#errors API not exposed")
}

func TestUpstream_Template_UndefinedFiltersRaise(t *testing.T) {
	t.Skip("Ruby strict_filters mode not modeled (default behavior is silent passthrough)")
}

func TestUpstream_Template_UsingRangeLiteralWorksAsExpected(t *testing.T) {
	// Iteration over a (x..y) range works; only the bare {{ foo }} → "1..5"
	// rendering differs (go-liquid renders the materialized slice).
	renderEq(t, "for-range", "{% assign nums = (x..y) %}{% for num in nums %}{{ num }}{% endfor %}",
		map[string]any{"x": 1, "y": 5}, "12345")
}

func TestUpstream_Template_UsingRangeLiteralWorksAsExpected_AssignToString(t *testing.T) {
	t.Skip("Ruby renders an assigned range as '1..5'; go-liquid renders the materialized slice as '12345'")
	renderEq(t, "assign", "{% assign foo = (x..y) %}{{ foo }}",
		map[string]any{"x": 1, "y": 5}, "1..5")
}

func TestUpstream_Template_SourceStringSubclass(t *testing.T) {
	t.Skip("Ruby-specific: subclass-of-String source. go-liquid Parse takes Go string only")
}

func TestUpstream_Template_RaisesErrorWithInvalidUtf8(t *testing.T) {
	t.Skip("Ruby raises TemplateEncodingError for invalid UTF-8; go-liquid does not validate source encoding")
}

func TestUpstream_Template_AllowsNonStringValuesAsSource(t *testing.T) {
	t.Skip("Ruby Template.parse coerces nil/1/true to source string; go-liquid Parse signature is `string`")
}
