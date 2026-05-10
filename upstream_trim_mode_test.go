package liquid

import "testing"

// Upstream parity: integration/trim_mode_test.rb
//
// Tests use Ruby's HEREDOC with `<<-END_TEMPLATE` (preserves leading spaces).
// The equivalent Go-side string literals reproduce the same input. All
// templates are indented six spaces to match Ruby's literal indentation.

func TestUpstream_TrimMode_StandardOutput(t *testing.T) {
	text := "      <div>\n" +
		"        <p>\n" +
		"          {{ 'John' }}\n" +
		"        </p>\n" +
		"      </div>\n"
	expected := "      <div>\n" +
		"        <p>\n" +
		"          John\n" +
		"        </p>\n" +
		"      </div>\n"
	renderEq(t, "std", text, nil, expected)
}

func TestUpstream_TrimMode_VariableOutputWithMultipleBlankLines(t *testing.T) {
	text := "      <div>\n" +
		"        <p>\n" +
		"\n" +
		"\n" +
		"          {{- 'John' -}}\n" +
		"\n" +
		"\n" +
		"        </p>\n" +
		"      </div>\n"
	expected := "      <div>\n" +
		"        <p>John</p>\n" +
		"      </div>\n"
	renderEq(t, "blank-lines", text, nil, expected)
}

func TestUpstream_TrimMode_TagOutputWithMultipleBlankLines(t *testing.T) {
	text := "      <div>\n" +
		"        <p>\n" +
		"\n" +
		"\n" +
		"          {%- if true -%}\n" +
		"          yes\n" +
		"          {%- endif -%}\n" +
		"\n" +
		"\n" +
		"        </p>\n" +
		"      </div>\n"
	expected := "      <div>\n" +
		"        <p>yes</p>\n" +
		"      </div>\n"
	renderEq(t, "tag-blank", text, nil, expected)
}

func TestUpstream_TrimMode_StandardTags(t *testing.T) {
	ws := "          "
	text := "      <div>\n" +
		"        <p>\n" +
		"          {% if true %}\n" +
		"          yes\n" +
		"          {% endif %}\n" +
		"        </p>\n" +
		"      </div>\n"
	expected := "      <div>\n" +
		"        <p>\n" +
		ws + "\n" +
		"          yes\n" +
		ws + "\n" +
		"        </p>\n" +
		"      </div>\n"
	renderEq(t, "std-true", text, nil, expected)

	text2 := "      <div>\n" +
		"        <p>\n" +
		"          {% if false %}\n" +
		"          no\n" +
		"          {% endif %}\n" +
		"        </p>\n" +
		"      </div>\n"
	expected2 := "      <div>\n" +
		"        <p>\n" +
		ws + "\n" +
		"        </p>\n" +
		"      </div>\n"
	renderEq(t, "std-false", text2, nil, expected2)
}

func TestUpstream_TrimMode_NoTrimOutput(t *testing.T) {
	renderEq(t, "no-trim", `<p>{{- 'John' -}}</p>`, nil, "<p>John</p>")
}

func TestUpstream_TrimMode_NoTrimTags(t *testing.T) {
	renderEq(t, "true", `<p>{%- if true -%}yes{%- endif -%}</p>`, nil, "<p>yes</p>")
	renderEq(t, "false", `<p>{%- if false -%}no{%- endif -%}</p>`, nil, "<p></p>")
}

func TestUpstream_TrimMode_SingleLineOuterTag(t *testing.T) {
	renderEq(t, "true", `<p> {%- if true %} yes {% endif -%} </p>`, nil, "<p> yes </p>")
	renderEq(t, "false", `<p> {%- if false %} no {% endif -%} </p>`, nil, "<p></p>")
}

func TestUpstream_TrimMode_SingleLineInnerTag(t *testing.T) {
	renderEq(t, "true", `<p> {% if true -%} yes {%- endif %} </p>`, nil, "<p> yes </p>")
	renderEq(t, "false", `<p> {% if false -%} no {%- endif %} </p>`, nil, "<p>  </p>")
}

func TestUpstream_TrimMode_SingleLinePostTag(t *testing.T) {
	renderEq(t, "true", `<p> {% if true -%} yes {% endif -%} </p>`, nil, "<p> yes </p>")
	renderEq(t, "false", `<p> {% if false -%} no {% endif -%} </p>`, nil, "<p> </p>")
}

func TestUpstream_TrimMode_SingleLinePreTag(t *testing.T) {
	renderEq(t, "true", `<p> {%- if true %} yes {%- endif %} </p>`, nil, "<p> yes </p>")
	renderEq(t, "false", `<p> {%- if false %} no {%- endif %} </p>`, nil, "<p> </p>")
}

func TestUpstream_TrimMode_PreTrimOutput(t *testing.T) {
	text := "      <div>\n        <p>\n          {{- 'John' }}\n        </p>\n      </div>\n"
	expected := "      <div>\n        <p>John\n        </p>\n      </div>\n"
	renderEq(t, "pre-trim", text, nil, expected)
}

func TestUpstream_TrimMode_PreTrimTags(t *testing.T) {
	text := "      <div>\n        <p>\n          {%- if true %}\n          yes\n          {%- endif %}\n        </p>\n      </div>\n"
	expected := "      <div>\n        <p>\n          yes\n        </p>\n      </div>\n"
	renderEq(t, "true", text, nil, expected)

	text2 := "      <div>\n        <p>\n          {%- if false %}\n          no\n          {%- endif %}\n        </p>\n      </div>\n"
	expected2 := "      <div>\n        <p>\n        </p>\n      </div>\n"
	renderEq(t, "false", text2, nil, expected2)
}

func TestUpstream_TrimMode_PostTrimOutput(t *testing.T) {
	text := "      <div>\n        <p>\n          {{ 'John' -}}\n        </p>\n      </div>\n"
	expected := "      <div>\n        <p>\n          John</p>\n      </div>\n"
	renderEq(t, "post-trim", text, nil, expected)
}

func TestUpstream_TrimMode_PostTrimTags(t *testing.T) {
	text := "      <div>\n        <p>\n          {% if true -%}\n          yes\n          {% endif -%}\n        </p>\n      </div>\n"
	expected := "      <div>\n        <p>\n          yes\n          </p>\n      </div>\n"
	renderEq(t, "true", text, nil, expected)

	text2 := "      <div>\n        <p>\n          {% if false -%}\n          no\n          {% endif -%}\n        </p>\n      </div>\n"
	expected2 := "      <div>\n        <p>\n          </p>\n      </div>\n"
	renderEq(t, "false", text2, nil, expected2)
}

func TestUpstream_TrimMode_PreAndPostTrimTags(t *testing.T) {
	text := "      <div>\n        <p>\n          {%- if true %}\n          yes\n          {% endif -%}\n        </p>\n      </div>\n"
	expected := "      <div>\n        <p>\n          yes\n          </p>\n      </div>\n"
	renderEq(t, "true", text, nil, expected)

	text2 := "      <div>\n        <p>\n          {%- if false %}\n          no\n          {% endif -%}\n        </p>\n      </div>\n"
	expected2 := "      <div>\n        <p></p>\n      </div>\n"
	renderEq(t, "false", text2, nil, expected2)
}

func TestUpstream_TrimMode_PostAndPreTrimTags(t *testing.T) {
	text := "      <div>\n        <p>\n          {% if true -%}\n          yes\n          {%- endif %}\n        </p>\n      </div>\n"
	expected := "      <div>\n        <p>\n          yes\n        </p>\n      </div>\n"
	renderEq(t, "true", text, nil, expected)

	ws := "          "
	text2 := "      <div>\n        <p>\n          {% if false -%}\n          no\n          {%- endif %}\n        </p>\n      </div>\n"
	expected2 := "      <div>\n        <p>\n" + ws + "\n        </p>\n      </div>\n"
	renderEq(t, "false", text2, nil, expected2)
}

func TestUpstream_TrimMode_TrimOutput(t *testing.T) {
	text := "      <div>\n        <p>\n          {{- 'John' -}}\n        </p>\n      </div>\n"
	expected := "      <div>\n        <p>John</p>\n      </div>\n"
	renderEq(t, "trim", text, nil, expected)
}

func TestUpstream_TrimMode_TrimTags(t *testing.T) {
	text := "      <div>\n        <p>\n          {%- if true -%}\n          yes\n          {%- endif -%}\n        </p>\n      </div>\n"
	expected := "      <div>\n        <p>yes</p>\n      </div>\n"
	renderEq(t, "true", text, nil, expected)

	text2 := "      <div>\n        <p>\n          {%- if false -%}\n          no\n          {%- endif -%}\n        </p>\n      </div>\n"
	expected2 := "      <div>\n        <p></p>\n      </div>\n"
	renderEq(t, "false", text2, nil, expected2)
}

func TestUpstream_TrimMode_WhitespaceTrimOutput(t *testing.T) {
	text := "      <div>\n        <p>\n          {{- 'John' -}},\n          {{- '30' -}}\n        </p>\n      </div>\n"
	expected := "      <div>\n        <p>John,30</p>\n      </div>\n"
	renderEq(t, "ws-trim", text, nil, expected)
}

func TestUpstream_TrimMode_WhitespaceTrimTags(t *testing.T) {
	text := "      <div>\n        <p>\n          {%- if true -%}\n          yes\n          {%- endif -%}\n        </p>\n      </div>\n"
	expected := "      <div>\n        <p>yes</p>\n      </div>\n"
	renderEq(t, "true", text, nil, expected)

	text2 := "      <div>\n        <p>\n          {%- if false -%}\n          no\n          {%- endif -%}\n        </p>\n      </div>\n"
	expected2 := "      <div>\n        <p></p>\n      </div>\n"
	renderEq(t, "false", text2, nil, expected2)
}

func TestUpstream_TrimMode_ComplexTrimOutput(t *testing.T) {
	text := "      <div>\n        <p>\n          {{- 'John' -}}\n          {{- '30' -}}\n        </p>\n" +
		"        <b>\n          {{ 'John' -}}\n          {{- '30' }}\n        </b>\n" +
		"        <i>\n          {{- 'John' }}\n          {{ '30' -}}\n        </i>\n      </div>\n"
	expected := "      <div>\n        <p>John30</p>\n" +
		"        <b>\n          John30\n        </b>\n" +
		"        <i>John\n          30</i>\n      </div>\n"
	renderEq(t, "complex", text, nil, expected)
}

func TestUpstream_TrimMode_ComplexTrim(t *testing.T) {
	text := "      <div>\n        {%- if true -%}\n          {%- if true -%}\n            <p>\n              {{- 'John' -}}\n            </p>\n          {%- endif -%}\n        {%- endif -%}\n      </div>\n"
	expected := "      <div><p>John</p></div>\n"
	renderEq(t, "nested", text, nil, expected)
}

func TestUpstream_TrimMode_RightTrimFollowedByTag(t *testing.T) {
	renderEq(t, "rtrim", `{{ "a" -}}{{ "b" }} c`, nil, "ab c")
}

func TestUpstream_TrimMode_RawOutput(t *testing.T) {
	ws := "        "
	text := "      <div>\n        {% raw %}\n          {%- if true -%}\n            <p>\n              {{- 'John' -}}\n            </p>\n          {%- endif -%}\n        {% endraw %}\n      </div>\n"
	expected := "      <div>\n" + ws + "\n" +
		"          {%- if true -%}\n            <p>\n              {{- 'John' -}}\n            </p>\n          {%- endif -%}\n" + ws + "\n      </div>\n"
	renderEq(t, "raw", text, nil, expected)
}

func TestUpstream_TrimMode_PreTrimBlankPrecedingText(t *testing.T) {
	renderEq(t, "raw-after-nl", "\n{%- raw %}{% endraw %}", nil, "")
	renderEq(t, "if-after-nl", "\n{%- if true %}{% endif %}", nil, "")
	renderEq(t, "B-then-if", "{{ 'B' }} \n{%- if true %}C{% endif %}", nil, "BC")
}

func TestUpstream_TrimMode_BugCompatiblePreTrim(t *testing.T) {
	t.Skip("go-liquid does not expose Ruby's bug_compatible_whitespace_trimming option")
}

func TestUpstream_TrimMode_TrimBlank(t *testing.T) {
	renderEq(t, "trim-empty", "foo {{-}} bar", nil, "foobar")
}
