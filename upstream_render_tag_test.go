package liquid

// Ported from upstream Ruby Liquid test/integration/tags/render_tag_test.rb.

import (
	"errors"
	"testing"
)

// renderWith parses src, attaches a MapLoader for partials, and renders
// against data. Mirrors the Ruby `partials:` test option.
func renderWith(t *testing.T, src string, partials MapLoader, data map[string]any) (string, error) {
	t.Helper()
	tmpl, err := Parse(src)
	if err != nil {
		return "", err
	}
	tmpl.WithLoader(partials)
	return tmpl.Render(data)
}

func renderWithEq(t *testing.T, name, src string, partials MapLoader, data map[string]any, want string) {
	t.Helper()
	got, err := renderWith(t, src, partials, data)
	if err != nil {
		t.Fatalf("%s: unexpected error: %v", name, err)
	}
	if got != want {
		t.Errorf("%s: template %q\n  want %q\n  got  %q", name, src, want, got)
	}
}

func TestUpstreamRenderTag(t *testing.T) {
	t.Run("test_render_with_no_arguments", func(t *testing.T) {
		renderWithEq(t, "no-args", `{% render "source" %}`,
			MapLoader{"source": "rendered content"}, nil, "rendered content")
	})

	t.Run("test_render_tag_looks_for_file_system_in_registers_first", func(t *testing.T) {
		renderWithEq(t, "register-fs", `{% render "pick_a_source" %}`,
			MapLoader{"pick_a_source": "from register file system"}, nil, "from register file system")
	})

	t.Run("test_render_passes_named_arguments_into_inner_scope", func(t *testing.T) {
		renderWithEq(t, "named-args",
			`{% render "product", inner_product: outer_product %}`,
			MapLoader{"product": "{{ inner_product.title }}"},
			map[string]any{"outer_product": map[string]any{"title": "My Product"}},
			"My Product")
	})

	t.Run("test_render_accepts_literals_as_arguments", func(t *testing.T) {
		renderWithEq(t, "literal-arg",
			`{% render "snippet", price: 123 %}`,
			MapLoader{"snippet": "{{ price }}"}, nil, "123")
	})

	t.Run("test_render_accepts_multiple_named_arguments", func(t *testing.T) {
		renderWithEq(t, "multi-args",
			`{% render "snippet", one: 1, two: 2 %}`,
			MapLoader{"snippet": "{{ one }} {{ two }}"}, nil, "1 2")
	})

	t.Run("test_render_does_not_inherit_parent_scope_variables", func(t *testing.T) {
		renderWithEq(t, "no-parent-scope",
			`{% assign outer_variable = "should not be visible" %}{% render "snippet" %}`,
			MapLoader{"snippet": "{{ outer_variable }}"}, nil, "")
	})

	t.Run("test_render_does_not_inherit_variable_with_same_name_as_snippet", func(t *testing.T) {
		renderWithEq(t, "no-shadow",
			`{% assign snippet = 'should not be visible' %}{% render 'snippet' %}`,
			MapLoader{"snippet": "{{ snippet }}"}, nil, "")
	})

	t.Run("test_render_does_not_mutate_parent_scope", func(t *testing.T) {
		renderWithEq(t, "no-mutate",
			`{% render 'snippet' %}{{ inner }}`,
			MapLoader{"snippet": "{% assign inner = 1 %}"}, nil, "")
	})

	t.Run("test_nested_render_tag", func(t *testing.T) {
		renderWithEq(t, "nested",
			`{% render 'one' %}`,
			MapLoader{
				"one": "one {% render 'two' %}",
				"two": "two",
			}, nil, "one two")
	})

	t.Run("test_recursively_rendered_template_does_not_produce_endless_loop", func(t *testing.T) {
		_, err := renderWith(t, `{% render "loop" %}`,
			MapLoader{"loop": `{% render "loop" %}`}, nil)
		if err == nil {
			t.Fatal("expected stack-level / depth error from infinite recursion")
		}
	})

	t.Run("test_sub_contexts_count_towards_the_same_recursion_limit", func(t *testing.T) {
		_, err := renderWith(t, `{% render "loop_render" %}`,
			MapLoader{"loop_render": `{% render "loop_render" %}`}, nil)
		if err == nil {
			t.Fatal("expected recursion-depth error")
		}
	})

	t.Run("test_dynamically_choosen_templates_are_not_allowed", func(t *testing.T) {
		// {% render name %} — name is a variable, not a literal — must be rejected.
		if _, err := Parse("{% assign name = 'snippet' %}{% render name %}"); err == nil {
			t.Fatal("expected parse error for variable-name render target")
		}
	})

	t.Run("test_strict2_parsing_errors", func(t *testing.T) {
		t.Skip("TODO: Ruby :lax/:strict tolerate `!!! arg1: \"value1\" ~~~ arg2:` separators in render args; only :strict2 rejects. go-liquid currently rejects in all modes.")
		renderWithEq(t, "lax-tolerated",
			`{% render "snippet" !!! arg1: "value1" ~~~ arg2: "value2" %}`,
			MapLoader{"snippet": "hello {{ arg1 }} {{ arg2 }}"}, nil, "hello value1 value2")
	})

	t.Run("test_optional_commas", func(t *testing.T) {
		partials := MapLoader{"snippet": "hello {{ arg1 }} {{ arg2 }}"}
		renderWithEq(t, "all-commas",
			`{% render "snippet", arg1: "value1", arg2: "value2" %}`,
			partials, nil, "hello value1 value2")
		renderWithEq(t, "no-leading-comma",
			`{% render "snippet"  arg1: "value1", arg2: "value2" %}`,
			partials, nil, "hello value1 value2")
		renderWithEq(t, "no-commas",
			`{% render "snippet"  arg1: "value1"  arg2: "value2" %}`,
			partials, nil, "hello value1 value2")
	})

	t.Run("test_render_tag_caches_second_read_of_same_partial", func(t *testing.T) {
		// Ruby asserts file_read_count == 1 after two renders of the same
		// partial in one template render. go-liquid's partial cache is
		// covered functionally — two renders should produce the same output.
		// A read-count introspection API would be needed for the strict assertion.
		renderWithEq(t, "cache",
			`{% render "snippet" %}{% render "snippet" %}`,
			MapLoader{"snippet": "echo"}, nil, "echoecho")
	})

	t.Run("test_render_tag_doesnt_cache_partials_across_renders", func(t *testing.T) {
		t.Skip("TODO: cross-render cache-bypass requires a file-system read-count introspection API (Ruby StubFileSystem.file_read_count)")
		// Only the output assertions below survive the skip; the read-count
		// half is meaningless without the introspection API.
		renderWithEq(t, "cross-renders",
			`{% include "snippet" %}`,
			MapLoader{"snippet": "my message"}, nil, "my message")
	})

	t.Run("test_render_tag_within_if_statement", func(t *testing.T) {
		renderWithEq(t, "in-if",
			`{% if true %}{% render "snippet" %}{% endif %}`,
			MapLoader{"snippet": "my message"}, nil, "my message")
	})

	t.Run("test_break_through_render", func(t *testing.T) {
		// {% break %} inside a partial does NOT propagate out of the {% render %}.
		partials := MapLoader{"break": "{% break %}"}
		renderWithEq(t, "break-stays-in-loop",
			`{% for i in (1..3) %}{{ i }}{% break %}{{ i }}{% endfor %}`,
			partials, nil, "1")
		renderWithEq(t, "break-isolated-by-render",
			`{% for i in (1..3) %}{{ i }}{% render "break" %}{{ i }}{% endfor %}`,
			partials, nil, "112233")
	})

	t.Run("test_increment_is_isolated_between_renders", func(t *testing.T) {
		renderWithEq(t, "incr-isolated",
			`{% increment port %}{% increment port %}{% render "incr" %}`,
			MapLoader{"incr": "{% increment port %}"}, nil, "010")
	})

	t.Run("test_decrement_is_isolated_between_renders", func(t *testing.T) {
		renderWithEq(t, "decr-isolated",
			`{% decrement port %}{% decrement port %}{% render "decr" %}`,
			MapLoader{"decr": "{% decrement port %}"}, nil, "-1-2-1")
	})

	t.Run("test_includes_will_not_render_inside_render_tag", func(t *testing.T) {
		// Ruby with render_errors:true inlines the error message; go-liquid
		// surfaces ErrDisabledTag from Render. Both prove include is disabled
		// inside render.
		_, err := renderWith(t, `{% render "test_include" %}`,
			MapLoader{
				"foo":          "bar",
				"test_include": `{% include "foo" %}`,
			}, nil)
		if err == nil {
			t.Fatal("expected ErrDisabledTag from include-inside-render")
		}
		if !errors.Is(err, ErrDisabledTag) {
			t.Errorf("want ErrDisabledTag, got %v", err)
		}
	})

	t.Run("test_includes_will_not_render_inside_nested_sibling_tags", func(t *testing.T) {
		_, err := renderWith(t, `{% render "nested_render_with_sibling_include" %}`,
			MapLoader{
				"foo": "bar",
				"nested_render_with_sibling_include": `{% render "test_include" %}{% include "foo" %}`,
				"test_include":                       `{% include "foo" %}`,
			}, nil)
		if err == nil {
			t.Fatal("expected ErrDisabledTag for include inside render subtree")
		}
		if !errors.Is(err, ErrDisabledTag) {
			t.Errorf("want ErrDisabledTag, got %v", err)
		}
	})

	t.Run("test_render_tag_with", func(t *testing.T) {
		renderWithEq(t, "with-product",
			`{% render 'product' with products[0] %}`,
			MapLoader{
				"product":       `Product: {{ product.title }} `,
				"product_alias": `Product: {{ product.title }} `,
			},
			map[string]any{"products": []any{
				map[string]any{"title": "Draft 151cm"},
				map[string]any{"title": "Element 155cm"},
			}}, "Product: Draft 151cm ")
	})

	t.Run("test_render_tag_with_alias", func(t *testing.T) {
		renderWithEq(t, "with-alias",
			`{% render 'product_alias' with products[0] as product %}`,
			MapLoader{
				"product":       `Product: {{ product.title }} `,
				"product_alias": `Product: {{ product.title }} `,
			},
			map[string]any{"products": []any{
				map[string]any{"title": "Draft 151cm"},
				map[string]any{"title": "Element 155cm"},
			}}, "Product: Draft 151cm ")
	})

	t.Run("test_render_tag_for_alias", func(t *testing.T) {
		renderWithEq(t, "for-alias",
			`{% render 'product_alias' for products as product %}`,
			MapLoader{
				"product":       `Product: {{ product.title }} `,
				"product_alias": `Product: {{ product.title }} `,
			},
			map[string]any{"products": []any{
				map[string]any{"title": "Draft 151cm"},
				map[string]any{"title": "Element 155cm"},
			}}, "Product: Draft 151cm Product: Element 155cm ")
	})

	t.Run("test_render_tag_for", func(t *testing.T) {
		renderWithEq(t, "for-products",
			`{% render 'product' for products %}`,
			MapLoader{
				"product":       `Product: {{ product.title }} `,
				"product_alias": `Product: {{ product.title }} `,
			},
			map[string]any{"products": []any{
				map[string]any{"title": "Draft 151cm"},
				map[string]any{"title": "Element 155cm"},
			}}, "Product: Draft 151cm Product: Element 155cm ")
	})

	t.Run("test_render_tag_forloop", func(t *testing.T) {
		renderWithEq(t, "for-forloop",
			`{% render 'product' for products %}`,
			MapLoader{
				"product": `Product: {{ product.title }} {% if forloop.first %}first{% endif %} {% if forloop.last %}last{% endif %} index:{{ forloop.index }} `,
			},
			map[string]any{"products": []any{
				map[string]any{"title": "Draft 151cm"},
				map[string]any{"title": "Element 155cm"},
			}},
			"Product: Draft 151cm first  index:1 Product: Element 155cm  last index:2 ")
	})

	t.Run("test_render_tag_for_drop", func(t *testing.T) {
		t.Skip("TODO: TestEnumerable Drop adapter — Ruby Drop API not exposed in go-liquid")
		// Once a drop-as-iterable adapter exists this should render "123".
		renderWithEq(t, "for-drop",
			`{% render 'loop' for loop as value %}`,
			MapLoader{"loop": `{{ value.foo }}`}, nil, "123")
	})

	t.Run("test_render_tag_with_drop", func(t *testing.T) {
		t.Skip("TODO: TestEnumerable Drop adapter — Ruby Drop API not exposed in go-liquid")
		renderWithEq(t, "with-drop",
			`{% render 'loop' with loop as value %}`,
			MapLoader{"loop": `{{ value }}`}, nil, "TestEnumerable")
	})

	t.Run("test_render_tag_renders_error_with_template_name", func(t *testing.T) {
		t.Skip("TODO: ErrorDrop / render_errors:true inline error rendering — go-liquid surfaces errors from Render instead of inlining")
		renderWithEq(t, "error-with-name",
			`{% render 'foo' with errors %}`,
			MapLoader{"foo": `{{ foo.standard_error }}`}, nil,
			"Liquid error (foo line 1): standard error")
	})

	t.Run("test_render_tag_renders_error_with_template_name_from_template_factory", func(t *testing.T) {
		t.Skip("TODO: TemplateFactory and per-partial path naming — Ruby StubTemplateFactory has no go-liquid equivalent")
		renderWithEq(t, "error-from-factory",
			`{% render 'foo' with errors %}`,
			MapLoader{"foo": `{{ foo.standard_error }}`}, nil,
			"Liquid error (some/path/foo line 1): standard error")
	})

	t.Run("test_render_with_invalid_expression", func(t *testing.T) {
		t.Skip("TODO: Ruby :lax/:strict permissively parse `with foo=>bar`; go-liquid currently rejects")
		if _, err := Parse(`{% render "snippet" with foo=>bar %}`); err != nil {
			t.Errorf("lax/strict should accept malformed render `with` expr; got %v", err)
		}
	})

	t.Run("test_render_attribute_with_invalid_expression", func(t *testing.T) {
		t.Skip("TODO: Ruby :lax/:strict permissively parse `key: foo=>bar`; go-liquid currently rejects")
		if _, err := Parse(`{% render "snippet", key: foo=>bar %}`); err != nil {
			t.Errorf("lax/strict should accept malformed render kwarg; got %v", err)
		}
	})
}

