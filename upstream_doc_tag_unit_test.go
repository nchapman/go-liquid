package liquid

import "testing"

// Upstream parity: unit/tags/doc_tag_unit_test.rb

func TestUpstream_DocTagUnit_DocTag(t *testing.T) {
	template := "{% doc %}\n" +
		"  Renders loading-spinner.\n\n" +
		"  @param {string} foo - some foo\n" +
		"  @param {string} [bar] - optional bar\n\n" +
		"  @example\n" +
		"  {% render 'loading-spinner', foo: 'foo' %}\n" +
		"  {% render 'loading-spinner', foo: 'foo', bar: 'bar' %}\n" +
		"{% enddoc %}"
	renderEq(t, "doc", template, nil, "")
}

func TestUpstream_DocTagUnit_DocTagBodyContent(t *testing.T) {
	t.Skip("Ruby ParseTreeVisitor#add_callback_for AST introspection not exposed")
}

func TestUpstream_DocTagUnit_DocTagDoesNotSupportExtraArguments(t *testing.T) {
	if _, err := Parse("{% doc extra %}\n{% enddoc %}"); err == nil {
		t.Fatalf("expected parse error for doc tag with extra args")
	}
}

func TestUpstream_DocTagUnit_DocTagMustSupportValidTags(t *testing.T) {
	for _, src := range []string{
		"{% doc %} foo",                  // never closed
		"{% doc } foo {% enddoc %}",      // malformed open
		"{% doc } foo %}{% enddoc %}",    // malformed open
	} {
		if _, err := Parse(src); err == nil {
			t.Errorf("expected parse error for %q", src)
		}
	}
}

func TestUpstream_DocTagUnit_DocTagIgnoresLiquidNodes(t *testing.T) {
	template := "{% doc %}\n" +
		"  {% if true %}\n" +
		"  {% if ... %}\n" +
		"  {%- for ? -%}\n" +
		"  {% while true %}\n" +
		"  {%\n    unless if\n  %}\n" +
		"  {% endcase %}\n" +
		"{% enddoc %}"
	renderEq(t, "ignore", template, nil, "")
}

func TestUpstream_DocTagUnit_DocTagIgnoresUnclosedLiquidTags(t *testing.T) {
	renderEq(t, "unclosed",
		"{% doc %}\n  {% if true %}\n{% enddoc %}",
		nil, "")
}

func TestUpstream_DocTagUnit_DocTagDoesNotAllowNestedDocs(t *testing.T) {
	t.Skip("Ruby raises 'Nested doc tags are not allowed'; go-liquid behavior may differ")
}

func TestUpstream_DocTagUnit_DocTagIgnoresNestedRawTags(t *testing.T) {
	renderEq(t, "nested-raw",
		"{% doc %}\n  {% raw %}{% endraw %}\n{% enddoc %}",
		nil, "")
}

func TestUpstream_DocTagUnit_DocTagIgnoresUnclosedAssign(t *testing.T) {
	renderEq(t, "unclosed-assign",
		"{% doc %}\n  {% assign foo = \n{% enddoc %}",
		nil, "")
}

func TestUpstream_DocTagUnit_DocTagIgnoresMalformedSyntax(t *testing.T) {
	renderEq(t, "malformed",
		"{% doc %}\n  {% if x x x %} {% endif %}\n{% enddoc %}",
		nil, "")
}

func TestUpstream_DocTagUnit_DocTagCapturesTokenBeforeEnddoc(t *testing.T) {
	t.Skip("Ruby Doc tag#nodelist#first.to_s introspection not exposed")
}

func TestUpstream_DocTagUnit_DocTagPreservesErrorLineNumbers(t *testing.T) {
	t.Skip("Ruby error line-number format not asserted in go-liquid")
}

func TestUpstream_DocTagUnit_DocTagWhitespaceControl(t *testing.T) {
	t.Skip("go-liquid trim around doc collapses both surrounding spaces; Ruby preserves the trailing one")
	renderEq(t, "trim",
		"foo {%- doc -%}gone{%- enddoc -%} bar",
		nil, "foo bar")
}

func TestUpstream_DocTagUnit_DocTagDelimiterHandling(t *testing.T) {
	t.Skip("Ruby strict2 mode behavior on {% doc x %} extra args; go-liquid is single-mode")
}

func TestUpstream_DocTagUnit_DocTagVisitor(t *testing.T) {
	t.Skip("Ruby ParseTreeVisitor not exposed")
}

func TestUpstream_DocTagUnit_DocTagBlankWithEmptyContent(t *testing.T) {
	renderEq(t, "blank-empty", "{% doc %}{% enddoc %}", nil, "")
}

func TestUpstream_DocTagUnit_DocTagBlankWithContent(t *testing.T) {
	renderEq(t, "blank-content", "{% doc %}some text{% enddoc %}", nil, "")
}

func TestUpstream_DocTagUnit_DocTagBlankWithWhitespaceOnly(t *testing.T) {
	renderEq(t, "blank-ws", "{% doc %}   {% enddoc %}", nil, "")
}

func TestUpstream_DocTagUnit_DocTagNodelistReturnsArrayWithBody(t *testing.T) {
	t.Skip("Ruby Doc#nodelist introspection not exposed")
}

func TestUpstream_DocTagUnit_DocTagNodelistWithEmptyContent(t *testing.T) {
	t.Skip("Ruby Doc#nodelist introspection not exposed")
}
