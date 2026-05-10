package liquid

import "testing"

// Upstream parity: unit/i18n_unit_test.rb
//
// Ruby's Liquid::I18n (.translate) is for error-message localization
// (e.g. "Liquid syntax error: ..." translated). Not modeled in go-liquid.

func TestUpstream_I18nUnit_SimpleTranslateString(t *testing.T) {
	t.Skip("Ruby Liquid::I18n.translate API not exposed in go-liquid")
}

func TestUpstream_I18nUnit_NestedTranslateString(t *testing.T) {
	t.Skip("Ruby Liquid::I18n.translate API not exposed")
}

func TestUpstream_I18nUnit_SingleStringInterpolation(t *testing.T) {
	t.Skip("Ruby Liquid::I18n.translate interpolation not exposed")
}

func TestUpstream_I18nUnit_RaisesUnknownTranslation(t *testing.T) {
	t.Skip("Ruby Liquid::I18n.translate / TranslationError not exposed")
}

func TestUpstream_I18nUnit_SetsDefaultPathToEn(t *testing.T) {
	t.Skip("Ruby Liquid::I18n DEFAULT_LOCALE_PATH not exposed")
}
