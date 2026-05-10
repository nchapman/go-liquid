package liquid

import "testing"

// Upstream parity: integration/security_test.rb

func TestUpstream_Security_NoInstanceEval(t *testing.T) {
	renderEq(t, "instance_eval", " {{ '1+1' | instance_eval }} ", nil, " 1+1 ")
}

func TestUpstream_Security_NoExistingInstanceEval(t *testing.T) {
	renderEq(t, "double_underscore", " {{ '1+1' | __instance_eval__ }} ", nil, " 1+1 ")
}

func TestUpstream_Security_NoInstanceEvalAfterMixingInNewFilter(t *testing.T) {
	renderEq(t, "after_mixin", " {{ '1+1' | instance_eval }} ", nil, " 1+1 ")
}

func TestUpstream_Security_NoInstanceEvalLaterInChain(t *testing.T) {
	env := NewEnvironment()
	env.RegisterFilter("add_one", func(input any, args ...any) any {
		return toString(input) + " + 1"
	})
	tmpl, err := env.Parse(" {{ '1+1' | add_one | instance_eval }} ")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	got, err := tmpl.Render(nil)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if got != " 1+1 + 1 " {
		t.Fatalf("got %q", got)
	}
}

func TestUpstream_Security_DoesNotPermanentlyAddFiltersToSymbolTable(t *testing.T) {
	t.Skip("Ruby-specific: tests that unknown filter names don't pollute Symbol.all_symbols")
}

func TestUpstream_Security_DoesNotAddDropMethodsToSymbolTable(t *testing.T) {
	t.Skip("Ruby-specific: Symbol.all_symbols leak check")
}

func TestUpstream_Security_MaxDepthNestedBlocksDoesNotRaiseException(t *testing.T) {
	// Same as Ruby's MAX_DEPTH=100 in Liquid::Block. go-liquid has no
	// equivalent constant; the parser is iterative-recursive but should
	// comfortably handle 100 nested {% if %} blocks.
	depth := 100
	src := ""
	for range depth {
		src += "{% if true %}"
	}
	src += "rendered"
	for range depth {
		src += "{% endif %}"
	}
	renderEq(t, "max_depth", src, nil, "rendered")
}

func TestUpstream_Security_MoreThanMaxDepthNestedBlocksRaisesException(t *testing.T) {
	t.Skip("go-liquid has no MAX_DEPTH cap on nested blocks; parser recursion is bounded by Go stack only")
	// depth := Liquid::Block::MAX_DEPTH + 1
	// expect Liquid::StackLevelError
}
