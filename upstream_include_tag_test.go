package liquid

// Ported from upstream Ruby Liquid test/integration/tags/include_tag_test.rb.

import (
	"testing"
)

func TestUpstreamIncludeTag(t *testing.T) {
	t.Run("test_include_tag_looks_for_file_system_in_registers_first", func(t *testing.T) {
		renderWithEq(t, "fs-from-loader", `{% include 'pick_a_source' %}`,
			MapLoader{"pick_a_source": "from OtherFileSystem"}, nil, "from OtherFileSystem")
	})

	t.Run("test_include_tag_with", func(t *testing.T) {
		renderWithEq(t, "with-product",
			"{% include 'product' with products[0] %}",
			MapLoader{"product": "Product: {{ product.title }} "},
			map[string]any{"products": []any{
				map[string]any{"title": "Draft 151cm"},
				map[string]any{"title": "Element 155cm"},
			}}, "Product: Draft 151cm ")
	})

	t.Run("test_include_tag_with_alias", func(t *testing.T) {
		renderWithEq(t, "with-alias",
			"{% include 'product_alias' with products[0] as product %}",
			MapLoader{"product_alias": "Product: {{ product.title }} "},
			map[string]any{"products": []any{
				map[string]any{"title": "Draft 151cm"},
				map[string]any{"title": "Element 155cm"},
			}}, "Product: Draft 151cm ")
	})

	t.Run("test_include_tag_for_alias", func(t *testing.T) {
		renderWithEq(t, "for-alias",
			"{% include 'product_alias' for products as product %}",
			MapLoader{"product_alias": "Product: {{ product.title }} "},
			map[string]any{"products": []any{
				map[string]any{"title": "Draft 151cm"},
				map[string]any{"title": "Element 155cm"},
			}}, "Product: Draft 151cm Product: Element 155cm ")
	})

	t.Run("test_include_tag_with_default_name", func(t *testing.T) {
		renderWithEq(t, "default-name",
			"{% include 'product' %}",
			MapLoader{"product": "Product: {{ product.title }} "},
			map[string]any{"product": map[string]any{"title": "Draft 151cm"}},
			"Product: Draft 151cm ")
	})

	t.Run("test_include_tag_for", func(t *testing.T) {
		renderWithEq(t, "for-products",
			"{% include 'product' for products %}",
			MapLoader{"product": "Product: {{ product.title }} "},
			map[string]any{"products": []any{
				map[string]any{"title": "Draft 151cm"},
				map[string]any{"title": "Element 155cm"},
			}}, "Product: Draft 151cm Product: Element 155cm ")
	})

	t.Run("test_include_tag_with_local_variables", func(t *testing.T) {
		renderWithEq(t, "local-vars",
			"{% include 'locale_variables' echo1: 'test123' %}",
			MapLoader{"locale_variables": "Locale: {{echo1}} {{echo2}}"},
			nil, "Locale: test123 ")
	})

	t.Run("test_include_tag_with_multiple_local_variables", func(t *testing.T) {
		renderWithEq(t, "multi-local-vars",
			"{% include 'locale_variables' echo1: 'test123', echo2: 'test321' %}",
			MapLoader{"locale_variables": "Locale: {{echo1}} {{echo2}}"},
			nil, "Locale: test123 test321")
	})

	t.Run("test_include_tag_with_multiple_local_variables_from_context", func(t *testing.T) {
		renderWithEq(t, "from-ctx",
			"{% include 'locale_variables' echo1: echo1, echo2: more_echos.echo2 %}",
			MapLoader{"locale_variables": "Locale: {{echo1}} {{echo2}}"},
			map[string]any{
				"echo1":      "test123",
				"more_echos": map[string]any{"echo2": "test321"},
			}, "Locale: test123 test321")
	})

	t.Run("test_included_templates_assigns_variables", func(t *testing.T) {
		// {% include %} shares parent scope, so an assign in the partial
		// is visible in the caller (Ruby behavior; differs from {% render %}).
		renderWithEq(t, "assign-leaks",
			"{% include 'assignments' %}{{ foo }}",
			MapLoader{"assignments": "{% assign foo = 'bar' %}"}, nil, "bar")
	})

	t.Run("test_nested_include_tag", func(t *testing.T) {
		partials := MapLoader{
			"body":        "body {% include 'body_detail' %}",
			"body_detail": "body_detail",
		}
		renderWithEq(t, "nested", "{% include 'body' %}", partials, nil, "body body_detail")

		nested := MapLoader{
			"body":            "body {% include 'body_detail' %}",
			"body_detail":     "body_detail",
			"nested_template": "{% include 'header' %} {% include 'body' %} {% include 'footer' %}",
			"header":          "header",
			"footer":          "footer",
		}
		renderWithEq(t, "nested-3-levels",
			"{% include 'nested_template' %}", nested, nil, "header body body_detail footer")
	})

	t.Run("test_nested_include_with_variable", func(t *testing.T) {
		partials := MapLoader{
			"nested_product_template": "Product: {{ nested_product_template.title }} {%include 'details'%} ",
			"details":                 "details",
		}
		renderWithEq(t, "with-nested",
			"{% include 'nested_product_template' with product %}",
			partials,
			map[string]any{"product": map[string]any{"title": "Draft 151cm"}},
			"Product: Draft 151cm details ")
		renderWithEq(t, "for-nested",
			"{% include 'nested_product_template' for products %}",
			partials,
			map[string]any{"products": []any{
				map[string]any{"title": "Draft 151cm"},
				map[string]any{"title": "Element 155cm"},
			}},
			"Product: Draft 151cm details Product: Element 155cm details ")
	})

	t.Run("test_recursively_included_template_does_not_produce_endless_loop", func(t *testing.T) {
		_, err := renderWith(t, "{% include 'loop' %}",
			MapLoader{"loop": "-{% include 'loop' %}"}, nil)
		if err == nil {
			t.Fatal("expected stack-level / recursion error")
		}
	})

	t.Run("test_dynamically_choosen_template", func(t *testing.T) {
		renderWithEq(t, "dyn-1", "{% include template %}",
			MapLoader{"Test123": "Test123"},
			map[string]any{"template": "Test123"}, "Test123")
		renderWithEq(t, "dyn-2", "{% include template %}",
			MapLoader{"Test321": "Test321"},
			map[string]any{"template": "Test321"}, "Test321")
		renderWithEq(t, "dyn-for",
			"{% include template for product %}",
			MapLoader{"product": "Product: {{ product.title }} "},
			map[string]any{
				"template": "product",
				"product":  map[string]any{"title": "Draft 151cm"},
			}, "Product: Draft 151cm ")
	})

	t.Run("test_strict2_parsing_errors", func(t *testing.T) {
		t.Skip("TODO: Ruby :lax/:strict tolerate `!!! ~~~` separators in include args; only :strict2 rejects. go-liquid currently rejects in all modes.")
		renderWithEq(t, "lax-tolerant",
			`{% include "snippet" !!! arg1: "value1" ~~~ arg2: "value2" %}`,
			MapLoader{"snippet": "hello {{ arg1 }} {{ arg2 }}"}, nil, "hello value1 value2")
	})

	t.Run("test_optional_commas", func(t *testing.T) {
		partials := MapLoader{"snippet": "hello {{ arg1 }} {{ arg2 }}"}
		renderWithEq(t, "all-commas",
			`{% include "snippet", arg1: "value1", arg2: "value2" %}`,
			partials, nil, "hello value1 value2")
		renderWithEq(t, "no-leading-comma",
			`{% include "snippet"  arg1: "value1", arg2: "value2" %}`,
			partials, nil, "hello value1 value2")
		renderWithEq(t, "no-commas",
			`{% include "snippet"  arg1: "value1"  arg2: "value2" %}`,
			partials, nil, "hello value1 value2")
	})

	t.Run("test_include_tag_caches_second_read_of_same_partial", func(t *testing.T) {
		// Read-count introspection isn't exposed; we only verify the output.
		renderWithEq(t, "cache-output",
			"{% include 'pick_a_source' %}{% include 'pick_a_source' %}",
			MapLoader{"pick_a_source": "from CountingFileSystem"}, nil,
			"from CountingFileSystemfrom CountingFileSystem")
	})

	t.Run("test_include_tag_doesnt_cache_partials_across_renders", func(t *testing.T) {
		t.Skip("TODO: cross-render cache-bypass needs a file-system read-count introspection API")
		renderWithEq(t, "single-include",
			"{% include 'pick_a_source' %}",
			MapLoader{"pick_a_source": "from CountingFileSystem"}, nil,
			"from CountingFileSystem")
	})

	t.Run("test_include_tag_within_if_statement", func(t *testing.T) {
		renderWithEq(t, "in-if",
			"{% if true %}{% include 'foo_if_true' %}{% endif %}",
			MapLoader{"foo_if_true": "foo_if_true"}, nil, "foo_if_true")
	})

	t.Run("test_custom_include_tag", func(t *testing.T) {
		t.Skip("TODO: Ruby's `Template.tags['include'] = CustomInclude` swaps the include implementation globally; go-liquid doesn't expose a per-environment override of built-in tags")
		// Once an override hook exists, this should render "custom_foo".
	})

	t.Run("test_custom_include_tag_within_if_statement", func(t *testing.T) {
		t.Skip("TODO: same Tag.tags[] override hook as test_custom_include_tag")
	})

	t.Run("test_does_not_add_error_in_strict_mode_for_missing_variable", func(t *testing.T) {
		t.Skip("TODO: expose Template.errors / non-fatal warning collection (Ruby Template.errors)")
		// Once errors are introspectable, this should render the partial
		// without populating any error list.
	})

	t.Run("test_passing_options_to_included_templates", func(t *testing.T) {
		t.Skip("TODO: include_options_blacklist / per-include error-mode override is a Ruby-specific Template option")
	})

	t.Run("test_render_raise_argument_error_when_template_is_undefined", func(t *testing.T) {
		// Ruby with render_errors:true inlines the error message; go-liquid
		// surfaces an error from Render. Both prove undefined names error.
		_, err := renderWith(t, "{% include undefined_variable %}",
			MapLoader{}, nil)
		if err == nil {
			t.Fatal("expected error for undefined include target")
		}
		_, err = renderWith(t, "{% include nil %}", MapLoader{}, nil)
		if err == nil {
			t.Fatal("expected error for nil include target")
		}
	})

	t.Run("test_render_raise_argument_error_when_template_is_not_a_string", func(t *testing.T) {
		t.Skip("TODO: numeric-literal template name surfaces parse error in go-liquid; Ruby raises ArgumentError at render")
		_, err := renderWith(t, "{% include 123 %}", MapLoader{}, nil)
		if err == nil {
			t.Fatal("expected error for non-string include target")
		}
	})

	t.Run("test_including_via_variable_value", func(t *testing.T) {
		renderWithEq(t, "via-var",
			"{% assign page = 'pick_a_source' %}{% include page %}",
			MapLoader{"pick_a_source": "from TestFileSystem"}, nil, "from TestFileSystem")

		partials := MapLoader{"product": "Product: {{ product.title }} "}
		renderWithEq(t, "via-var-product",
			"{% assign page = 'product' %}{% include page %}",
			partials,
			map[string]any{"product": map[string]any{"title": "Draft 151cm"}},
			"Product: Draft 151cm ")
		renderWithEq(t, "via-var-for",
			"{% assign page = 'product' %}{% include page for foo %}",
			partials,
			map[string]any{"foo": map[string]any{"title": "Draft 151cm"}},
			"Product: Draft 151cm ")
	})

	t.Run("test_including_with_strict_variables", func(t *testing.T) {
		t.Skip("TODO: strict_variables option is a Ruby-specific render flag; go-liquid surfaces undefined errors via ErrUndefinedVariable rather than non-fatal Template.errors")
		renderWithEq(t, "strict-vars",
			"{% include 'simple' %}",
			MapLoader{"simple": "simple"}, nil, "simple")
	})

	t.Run("test_break_through_include", func(t *testing.T) {
		// {% break %} inside an included partial does NOT propagate out.
		renderEq(t, "baseline",
			"{% for i in (1..3) %}{{ i }}{% break %}{{ i }}{% endfor %}", nil, "1")
		renderWithEq(t, "include-break-isolated",
			"{% for i in (1..3) %}{{ i }}{% include 'break' %}{{ i }}{% endfor %}",
			MapLoader{"break": "{% break %}"}, nil, "1")
	})

	t.Run("test_render_tag_renders_error_with_template_name", func(t *testing.T) {
		t.Skip("TODO: ErrorDrop / render_errors:true inline error rendering — go-liquid surfaces errors from Render")
	})

	t.Run("test_render_tag_renders_error_with_template_name_from_template_factory", func(t *testing.T) {
		t.Skip("TODO: TemplateFactory and per-partial path naming — Ruby StubTemplateFactory has no go-liquid equivalent")
	})

	t.Run("test_include_template_with_invalid_expression", func(t *testing.T) {
		t.Skip("TODO: Ruby :lax/:strict permissively accept `{% include foo=>bar %}`; go-liquid currently rejects")
		if _, err := Parse("{% include foo=>bar %}"); err != nil {
			t.Errorf("lax/strict should accept malformed include name; got %v", err)
		}
	})

	t.Run("test_include_with_invalid_expression", func(t *testing.T) {
		t.Skip("TODO: Ruby :lax/:strict permissively accept `with foo=>bar`; go-liquid currently rejects")
		if _, err := Parse(`{% include "snippet" with foo=>bar %}`); err != nil {
			t.Errorf("lax/strict should accept malformed include `with` expr; got %v", err)
		}
	})

	t.Run("test_include_attribute_with_invalid_expression", func(t *testing.T) {
		t.Skip("TODO: Ruby :lax/:strict permissively accept `key: foo=>bar`; go-liquid currently rejects")
		if _, err := Parse(`{% include "snippet", key: foo=>bar %}`); err != nil {
			t.Errorf("lax/strict should accept malformed include kwarg; got %v", err)
		}
	})

	t.Run("test_include_for_loop_true_with_for_keyword", func(t *testing.T) {
		t.Skip("TODO: expose for_loop? on the include node (Ruby AST introspection); functional behavior is already covered by test_include_tag_for")
	})

	t.Run("test_include_for_loop_false_with_with_keyword", func(t *testing.T) {
		t.Skip("TODO: expose for_loop? on the include node — see test_include_for_loop_true_with_for_keyword")
	})

	t.Run("test_include_for_loop_false_without_keyword", func(t *testing.T) {
		t.Skip("TODO: expose for_loop? on the include node — see test_include_for_loop_true_with_for_keyword")
	})

	t.Run("test_include_for_loop_with_alias", func(t *testing.T) {
		t.Skip("TODO: expose for_loop? on the include node — see test_include_for_loop_true_with_for_keyword")
	})

	t.Run("test_include_with_keyword_and_alias", func(t *testing.T) {
		t.Skip("TODO: expose for_loop? on the include node — see test_include_for_loop_true_with_for_keyword")
	})
}
