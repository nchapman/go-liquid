//go:build ruby_internals

package liquid

import "testing"

// Upstream parity: unit/partial_cache_unit_test.rb
//
// Ruby's Liquid::PartialCache is an internal cache for compiled partials
// (used by {% include %} / {% render %}). go-liquid caches loaded
// partials internally too, but doesn't expose the cache or its register
// keys. These tests are skipped; the user-observable caching behavior
// is covered by the loader tests.

func TestUpstream_PartialCacheUnit_UsesTheFileSystemRegisterIfPresent(t *testing.T) {
	t.Skip("Ruby file_system register slot not exposed; go-liquid uses Loader directly")
}
func TestUpstream_PartialCacheUnit_ReadsFromTheFileSystemOnlyOncePerFile(t *testing.T) {
	t.Skip("PartialCache internals not exposed; loader call count not surfaced")
}
func TestUpstream_PartialCacheUnit_CacheStateIsStoredPerContext(t *testing.T) {
	t.Skip("PartialCache internals not exposed")
}
func TestUpstream_PartialCacheUnit_CacheIsNotBrokenWhenADifferentParseContextIsUsed(t *testing.T) {
	t.Skip("PartialCache internals not exposed")
}
func TestUpstream_PartialCacheUnit_UsesDefaultTemplateFactoryWhenNoTemplateFactoryFoundInRegister(t *testing.T) {
	t.Skip("Ruby template_factory register slot not exposed")
}
func TestUpstream_PartialCacheUnit_UsesTemplateFactoryRegisterIfPresent(t *testing.T) {
	t.Skip("Ruby template_factory register slot not exposed")
}
func TestUpstream_PartialCacheUnit_CacheStateIsSharedForSubcontexts(t *testing.T) {
	t.Skip("PartialCache internals not exposed")
}
func TestUpstream_PartialCacheUnit_UsesTemplateNameFromTemplateFactory(t *testing.T) {
	t.Skip("Ruby template_factory not modeled")
}
func TestUpstream_PartialCacheUnit_IncludesErrorModeIntoTemplateCache(t *testing.T) {
	t.Skip("Ruby error_mode not modeled; go-liquid is single-mode")
}
