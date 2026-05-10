package liquid

// Ported from upstream Ruby Liquid test/integration/tags/statements_test.rb.

import "testing"

func TestUpstreamStatements(t *testing.T) {
	cases := []struct {
		name string
		src  string
		data map[string]any
		want string
	}{
		{"test_true_eql_true", " {% if true == true %} true {% else %} false {% endif %} ", nil, "  true  "},
		{"test_true_not_eql_true", " {% if true != true %} true {% else %} false {% endif %} ", nil, "  false  "},
		{"test_true_lq_true", " {% if 0 > 0 %} true {% else %} false {% endif %} ", nil, "  false  "},
		{"test_one_lq_zero", " {% if 1 > 0 %} true {% else %} false {% endif %} ", nil, "  true  "},
		{"test_zero_lq_one", " {% if 0 < 1 %} true {% else %} false {% endif %} ", nil, "  true  "},
		{"test_zero_lq_or_equal_one", " {% if 0 <= 0 %} true {% else %} false {% endif %} ", nil, "  true  "},
		{"test_zero_lq_or_equal_one_involving_nil/lhs", " {% if null <= 0 %} true {% else %} false {% endif %} ", nil, "  false  "},
		{"test_zero_lq_or_equal_one_involving_nil/rhs", " {% if 0 <= null %} true {% else %} false {% endif %} ", nil, "  false  "},
		{"test_zero_lqq_or_equal_one", " {% if 0 >= 0 %} true {% else %} false {% endif %} ", nil, "  true  "},
		{"test_strings", " {% if 'test' == 'test' %} true {% else %} false {% endif %} ", nil, "  true  "},
		{"test_strings_not_equal", " {% if 'test' != 'test' %} true {% else %} false {% endif %} ", nil, "  false  "},
		{"test_var_strings_equal", ` {% if var == "hello there!" %} true {% else %} false {% endif %} `, map[string]any{"var": "hello there!"}, "  true  "},
		{"test_var_strings_are_not_equal", ` {% if "hello there!" == var %} true {% else %} false {% endif %} `, map[string]any{"var": "hello there!"}, "  true  "},
		{"test_var_and_long_string_are_equal", " {% if var == 'hello there!' %} true {% else %} false {% endif %} ", map[string]any{"var": "hello there!"}, "  true  "},
		{"test_var_and_long_string_are_equal_backwards", " {% if 'hello there!' == var %} true {% else %} false {% endif %} ", map[string]any{"var": "hello there!"}, "  true  "},
		{"test_is_collection_empty", " {% if array == empty %} true {% else %} false {% endif %} ", map[string]any{"array": []any{}}, "  true  "},
		{"test_is_not_collection_empty", " {% if array == empty %} true {% else %} false {% endif %} ", map[string]any{"array": []any{1, 2, 3}}, "  false  "},
		{"test_nil/nil", " {% if var == nil %} true {% else %} false {% endif %} ", map[string]any{"var": nil}, "  true  "},
		{"test_nil/null", " {% if var == null %} true {% else %} false {% endif %} ", map[string]any{"var": nil}, "  true  "},
		{"test_not_nil/nil", " {% if var != nil %} true {% else %} false {% endif %} ", map[string]any{"var": 1}, "  true  "},
		{"test_not_nil/null", " {% if var != null %} true {% else %} false {% endif %} ", map[string]any{"var": 1}, "  true  "},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			renderEq(t, tc.name, tc.src, tc.data, tc.want)
		})
	}
}
