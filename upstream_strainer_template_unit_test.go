package liquid

import "testing"

// Upstream parity: unit/strainer_template_unit_test.rb
//
// Ruby's StrainerTemplate is a dynamically generated Module subclass that
// holds filter methods for a given Environment. go-liquid stores filters
// in a plain map keyed by name; there's no Strainer class to test.

func TestUpstream_StrainerTemplateUnit_AddFilterWhenWrongFilterClass(t *testing.T) {
	t.Skip("Liquid::StrainerTemplate class not exposed")
}
func TestUpstream_StrainerTemplateUnit_AddFilterRaisesWhenModulePrivatelyOverridesRegisteredPublicMethods(t *testing.T) {
	t.Skip("Ruby module method-visibility semantics not modeled in Go")
}
func TestUpstream_StrainerTemplateUnit_AddFilterRaisesWhenModuleOverridesRegisteredPublicMethodAsProtected(t *testing.T) {
	t.Skip("Ruby module method-visibility semantics not modeled in Go")
}
func TestUpstream_StrainerTemplateUnit_AddFilterDoesNotRaiseWhenModuleOverridesPreviouslyRegisteredMethod(t *testing.T) {
	// Closest equivalent: registering the same filter twice — the latter
	// overrides the former without error.
	env := NewEnvironment()
	env.RegisterFilter("x", func(input any, args ...any) any { return "a" })
	env.RegisterFilter("x", func(input any, args ...any) any { return "b" })
	tmpl, err := env.Parse("{{ 'in' | x }}")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	got, err := tmpl.Render(nil)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if got != "b" {
		t.Fatalf("got %q", got)
	}
}
func TestUpstream_StrainerTemplateUnit_AddFilterDoesNotIncludeAlreadyIncludedModule(t *testing.T) {
	t.Skip("Ruby Module#include de-duplication; not relevant in Go (filters keyed by name)")
}
