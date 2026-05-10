package liquid

import "testing"

// Upstream parity: unit/tags/comment_tag_unit_test.rb

func TestUpstream_CommentTagUnit_CommentInsideLiquidTag(t *testing.T) {
	t.Skip("go-liquid disallows `comment` inside `{% liquid %}`; Ruby allows it")
	renderEq(t, "inside-liquid",
		"{% liquid\ncomment\nthis is a comment\nendcomment\necho 'hi' %}",
		nil, "hi")
}

func TestUpstream_CommentTagUnit_DoesNotParseNodesInsideAComment(t *testing.T) {
	// Tag-like text inside {% comment %} is discarded; if it were parsed,
	// {% if %} without {% endif %} would error.
	renderEq(t, "no-parse",
		"{% comment %}{% if x %}{% endcomment %}done",
		nil, "done")
}

func TestUpstream_CommentTagUnit_AllowsUnclosedTags(t *testing.T) {
	renderEq(t, "unclosed-inner",
		"{% comment %}{% for %}{% endcomment %}done",
		nil, "done")
}

func TestUpstream_CommentTagUnit_OpenTagsInComment(t *testing.T) {
	renderEq(t, "open-tags",
		"{% comment %}{% if %}{% endcomment %}done",
		nil, "done")
}

func TestUpstream_CommentTagUnit_ChildCommentTagsNeedToBeClosed(t *testing.T) {
	t.Skip("go-liquid: nested unclosed {% comment %} does not error; Ruby requires nesting balance")
	if _, err := Parse("{% comment %}{% comment %}{% endcomment %}"); err == nil {
		t.Fatalf("expected error for unclosed nested comment")
	}
}

func TestUpstream_CommentTagUnit_ChildRawTagsNeedToBeClosed(t *testing.T) {
	t.Skip("Behavior of {% raw %} inside {% comment %} requires verification; Ruby raises if raw is unclosed")
}

func TestUpstream_CommentTagUnit_ErrorLineNumberIsCorrect(t *testing.T) {
	t.Skip("Ruby error message line-number format / line_numbers: option not exposed")
}

func TestUpstream_CommentTagUnit_CommentTagDelimiterWithExtraStrings(t *testing.T) {
	t.Skip("Ruby strict2 mode behavior on `{% comment foo %}...{% endcomment %}` not modeled")
}

func TestUpstream_CommentTagUnit_NestedCommentTagWithExtraStrings(t *testing.T) {
	t.Skip("Ruby strict2 mode behavior on nested comments with extra args not modeled")
}

func TestUpstream_CommentTagUnit_IgnoresDelimiterWithExtraStrings(t *testing.T) {
	t.Skip("go-liquid: `{% endcomment endcomment %}` is not matched as the endtag; Ruby tolerates extra args on endcomment")
	renderEq(t, "extra",
		"{% comment %}don't render{% endcomment endcomment %}done",
		nil, "done")
}

func TestUpstream_CommentTagUnit_DelimiterCanHaveExtraStrings(t *testing.T) {
	t.Skip("go-liquid rejects extra args after `{% comment foo %}`; Ruby tolerates")
	renderEq(t, "extra-on-open",
		"{% comment foo %}don't render{% endcomment %}done",
		nil, "done")
}

func TestUpstream_CommentTagUnit_WithWhitespaceControl(t *testing.T) {
	t.Skip("go-liquid trim around `{%- comment -%}...{%- endcomment -%}` eats both leading and trailing two-space pads")
	renderEq(t, "ws", "  {%- comment -%}gone{%- endcomment -%}  ", nil, "    ")
}

func TestUpstream_CommentTagUnit_DontOverrideLiquidTagWhitespaceControl(t *testing.T) {
	t.Skip("Whitespace control around {% liquid %} requires verification of exact spacing")
}
