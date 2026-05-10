package liquid

import "testing"

// Upstream parity: unit/environment_filter_test.rb
//
// Tests target Ruby's Liquid::StrainerTemplate (the filter-dispatch class).
// go-liquid uses Environment.RegisterFilter / RegisterFilterE / FilterFunc
// directly — there is no Strainer class to test. Bodies retained for the
// API points that do map.

func TestUpstream_EnvironmentFilter_Strainer(t *testing.T) {
	t.Skip("Ruby Liquid::StrainerTemplate class not modeled; go-liquid uses Environment.RegisterFilter")
}

func TestUpstream_EnvironmentFilter_StrainerRaisesArgumentError(t *testing.T) {
	t.Skip("Ruby Liquid::ArgumentError surfacing not modeled")
}

func TestUpstream_EnvironmentFilter_StrainerArgumentErrorContainsBacktrace(t *testing.T) {
	t.Skip("Ruby exception backtraces not modeled")
}

func TestUpstream_EnvironmentFilter_StrainerOnlyInvokesPublicFilterMethods(t *testing.T) {
	t.Skip("Ruby module-method visibility not modeled (Go has no public/private method distinction)")
}

func TestUpstream_EnvironmentFilter_StrainerReturnsNilIfNoFilterMethodFound(t *testing.T) {
	// Unknown filter is silently a no-op in go-liquid (matches Ruby's lax
	// behavior where strainer.invoke(:unknown, x) returns x).
	renderEq(t, "unknown", "{{ x | unknown_filter }}",
		map[string]any{"x": "hi"}, "hi")
}

func TestUpstream_EnvironmentFilter_StrainerReturnsFirstArgumentIfNoMethodAndArgumentsGiven(t *testing.T) {
	renderEq(t, "passthrough", "{{ x | unknown_filter: 'arg' }}",
		map[string]any{"x": "hi"}, "hi")
}

func TestUpstream_EnvironmentFilter_StrainerOnlyAllowsMethodsDefinedInFilters(t *testing.T) {
	t.Skip("Ruby Object#tap / Kernel methods not callable as filters by default; go-liquid filter registry is explicit")
}

func TestUpstream_EnvironmentFilter_StrainerUsesAClassCacheToAvoidMethodCacheInvalidation(t *testing.T) {
	t.Skip("Ruby method-cache invalidation concern not relevant to Go")
}

func TestUpstream_EnvironmentFilter_AddGlobalFilterClearsCache(t *testing.T) {
	t.Skip("Ruby Liquid::Environment.default's class cache not exposed in go-liquid")
}
