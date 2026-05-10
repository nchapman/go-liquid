//go:build ruby_internals

package liquid

import "testing"

// Upstream parity: unit/regexp_unit_test.rb
//
// Tests target Ruby's Liquid::Lexer / Liquid::Tokenizer regex primitives
// (QuotedString, QuotedFragment, VariableParser constants). go-liquid
// does not use regex-based lexing — tokenization is hand-rolled — so
// the regex contract isn't exposed. Bodies retained for context.

func TestUpstream_RegexpUnit_Empty(t *testing.T)              { t.Skip("Liquid regex constants not exposed") }
func TestUpstream_RegexpUnit_Quote(t *testing.T)              { t.Skip("Liquid regex constants not exposed") }
func TestUpstream_RegexpUnit_Words(t *testing.T)              { t.Skip("Liquid regex constants not exposed") }
func TestUpstream_RegexpUnit_Tags(t *testing.T)               { t.Skip("Liquid regex constants not exposed") }
func TestUpstream_RegexpUnit_DoubleQuotedWords(t *testing.T)  { t.Skip("Liquid regex constants not exposed") }
func TestUpstream_RegexpUnit_SingleQuotedWords(t *testing.T)  { t.Skip("Liquid regex constants not exposed") }
func TestUpstream_RegexpUnit_QuotedWordsInTheMiddle(t *testing.T) {
	t.Skip("Liquid regex constants not exposed")
}
func TestUpstream_RegexpUnit_VariableParser(t *testing.T) {
	t.Skip("Liquid::VariableParser regex not exposed")
}
func TestUpstream_RegexpUnit_VariableParserWithLargeInput(t *testing.T) {
	t.Skip("Liquid::VariableParser regex not exposed")
}
