//go:build ruby_internals

package liquid

import "testing"

// Upstream parity: unit/template_unit_test.rb

func TestUpstream_TemplateUnit_SetsDefaultLocalizationInDocument(t *testing.T) {
	t.Skip("Ruby Liquid::I18n localization API not exposed in go-liquid")
}

func TestUpstream_TemplateUnit_SetsDefaultLocalizationInContextWithQuickInitialization(t *testing.T) {
	t.Skip("Ruby Liquid::I18n localization API not exposed")
}

func TestUpstream_TemplateUnit_TagsCanBeLoopedOver(t *testing.T) {
	t.Skip("Ruby Environment#tags / each enumerator not exposed")
}

func TestUpstream_TemplateUnit_TemplateInheritance(t *testing.T) {
	t.Skip("Ruby Template subclass / inheritance not modeled in go-liquid")
}
