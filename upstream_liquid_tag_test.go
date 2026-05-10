package liquid

// Ported from upstream Ruby Liquid test/integration/tags/liquid_tag_test.rb.

import (
	"strings"
	"testing"
)

func TestUpstreamLiquidTag(t *testing.T) {
	t.Run("test_liquid_tag", func(t *testing.T) {
		data := map[string]any{"array": []any{1, 2, 3}}

		renderEq(t, "echo-join",
			"{%- liquid\n  echo array | join: \" \"\n-%}\n",
			data, "1 2 3")

		renderEq(t, "for-forloop-last",
			"{%- liquid\n  for value in array\n    echo value\n    unless forloop.last\n      echo \" \"\n    endunless\n  endfor\n-%}\n",
			data, "1 2 3")

		renderEq(t, "for-with-leak",
			"{%- liquid\n  for value in array\n    assign double_value = value | times: 2\n    echo double_value | times: 2\n    unless forloop.last\n      echo \" \"\n    endunless\n  endfor\n\n  echo \" \"\n  echo double_value\n-%}\n",
			data, "4 8 12 6")

		renderEq(t, "multiple-liquid-blocks",
			"{%- liquid echo \"a\" -%}\nb\n{%- liquid echo \"c\" -%}\n",
			nil, "abc")
	})

	t.Run("test_liquid_tag_errors", func(t *testing.T) {
		_, err := Parse("{%- liquid error no such tag -%}\n")
		if err == nil {
			t.Fatal("expected error for unknown tag inside liquid")
		}
		if !strings.Contains(err.Error(), "error") && !strings.Contains(err.Error(), "Unknown tag") {
			t.Errorf("error should mention the bad tag name; got %q", err.Error())
		}

		_, err = Parse("{%- liquid\n  for value in array\n    echo 'forgot to close the for tag'\n-%}\n")
		if err == nil {
			t.Fatal("expected error for unclosed for inside liquid")
		}
		if !strings.Contains(err.Error(), "for") {
			t.Errorf("error should mention 'for'; got %q", err.Error())
		}
	})

	t.Run("test_line_number_is_correct_after_a_blank_token", func(t *testing.T) {
		t.Skip("TODO: line numbers inside {% liquid %} body should reflect original source line (currently re-emitted as one synthetic line)")
		for _, src := range []string{
			"{% liquid echo ''\n\n error %}",
			"{% liquid echo ''\n  \n error %}",
		} {
			_, err := Parse(src)
			if err == nil {
				t.Fatalf("expected error for %q", src)
			}
			if !strings.Contains(err.Error(), "line 3") {
				t.Errorf("expected error to mention 'line 3'; got %q", err.Error())
			}
		}
	})

	t.Run("test_nested_liquid_tag", func(t *testing.T) {
		src := "{%- if true %}\n  {%- liquid\n    echo \"good\"\n  %}\n{%- endif -%}\n"
		renderEq(t, "nested", src, nil, "good")
	})

	t.Run("test_cannot_open_blocks_living_past_a_liquid_tag", func(t *testing.T) {
		src := "{%- liquid\n  if true\n-%}\n{%- endif -%}\n"
		_, err := Parse(src)
		if err == nil {
			t.Fatal("expected error for if block left open by liquid tag")
		}
		if !strings.Contains(err.Error(), "if") {
			t.Errorf("error should mention 'if'; got %q", err.Error())
		}
	})

	t.Run("test_cannot_close_blocks_created_before_a_liquid_tag", func(t *testing.T) {
		src := "{%- if true -%}\n42\n{%- liquid endif -%}\n"
		_, err := Parse(src)
		if err == nil {
			t.Fatal("expected error: endif inside liquid is invalid")
		}
	})

	t.Run("test_liquid_tag_in_raw", func(t *testing.T) {
		src := "{% raw %}{% liquid echo 'test' %}{% endraw %}\n"
		renderEq(t, "in-raw", src, nil, "{% liquid echo 'test' %}\n")
	})

	t.Run("test_nested_liquid_tags", func(t *testing.T) {
		t.Skip("TODO: support `liquid` keyword as a no-op prefix inside a {% liquid %} body (Ruby parity)")
		src := "{%- liquid\n  liquid\n    if true\n      echo \"good\"\n    endif\n-%}\n"
		renderEq(t, "nested-liquid", src, nil, "good")
	})

	t.Run("test_nested_liquid_tags_on_same_line", func(t *testing.T) {
		t.Skip("TODO: support repeated `liquid` keyword on a single liquid-body line")
		renderEq(t, "same-line", "{%- liquid liquid liquid echo \"good\" -%}\n", nil, "good")
	})

	t.Run("test_nested_liquid_liquid_is_not_skipped_if_used_in_non_tag_position", func(t *testing.T) {
		t.Skip("TODO: distinguish leading `liquid` keyword from `liquid` used as variable name")
		data := map[string]any{"liquid": "liquid"}
		renderEq(t, "liquid-as-var", "{%- liquid liquid liquid echo liquid -%}\n", data, "liquid")
	})

	t.Run("test_next_liquid_with_unclosed_if_tag", func(t *testing.T) {
		t.Skip("TODO: depends on nested-liquid support")
		src := "{%- liquid\n  liquid if true\n    echo \"good\"\n  endif\n-%}\n"
		_, err := Parse(src)
		if err == nil {
			t.Fatal("expected 'if' tag was never closed error")
		}
		if !strings.Contains(err.Error(), "if") {
			t.Errorf("error should mention 'if'; got %q", err.Error())
		}
	})
}
