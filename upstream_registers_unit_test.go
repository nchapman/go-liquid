package liquid

import "testing"

// Upstream parity: unit/registers_unit_test.rb
//
// Ruby's Liquid::Registers is a per-render scratch store keyed by symbol
// (e.g. registers[:file_system]). go-liquid exposes a similar concept
// via TagContext.Registers, but this unit-test suite targets the
// Registers class API directly (set/get/delete/fetch/key?/static),
// which is not part of the public Go surface.

func TestUpstream_RegistersUnit_Set(t *testing.T)    { t.Skip("Liquid::Registers class not exposed") }
func TestUpstream_RegistersUnit_GetMissingKey(t *testing.T) {
	t.Skip("Liquid::Registers class not exposed")
}
func TestUpstream_RegistersUnit_Delete(t *testing.T) { t.Skip("Liquid::Registers class not exposed") }
func TestUpstream_RegistersUnit_Fetch(t *testing.T)  { t.Skip("Liquid::Registers class not exposed") }
func TestUpstream_RegistersUnit_Key(t *testing.T)    { t.Skip("Liquid::Registers class not exposed") }
func TestUpstream_RegistersUnit_StaticRegisterCanBeFrozen(t *testing.T) {
	t.Skip("Liquid::Registers#static not exposed")
}
func TestUpstream_RegistersUnit_NewStaticRetainsStatic(t *testing.T) {
	t.Skip("Liquid::Registers#static not exposed")
}
func TestUpstream_RegistersUnit_MultipleInstancesAreUnique(t *testing.T) {
	t.Skip("Liquid::Registers class not exposed")
}
func TestUpstream_RegistersUnit_InitializationReusedStaticSameMemoryObject(t *testing.T) {
	t.Skip("Liquid::Registers#static identity not exposed")
}
