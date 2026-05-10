package liquid

// Ported from upstream Ruby Liquid test/integration/tags/table_row_test.rb.

import "testing"

func TestUpstreamTableRowTag(t *testing.T) {
	t.Run("test_table_row", func(t *testing.T) {
		renderEq(t, "6/3",
			"{% tablerow n in numbers cols:3%} {{n}} {% endtablerow %}",
			map[string]any{"numbers": []any{1, 2, 3, 4, 5, 6}},
			"<tr class=\"row1\">\n<td class=\"col1\"> 1 </td><td class=\"col2\"> 2 </td><td class=\"col3\"> 3 </td></tr>\n<tr class=\"row2\"><td class=\"col1\"> 4 </td><td class=\"col2\"> 5 </td><td class=\"col3\"> 6 </td></tr>\n")

		renderEq(t, "empty",
			"{% tablerow n in numbers cols:3%} {{n}} {% endtablerow %}",
			map[string]any{"numbers": []any{}},
			"<tr class=\"row1\">\n</tr>\n")
	})

	t.Run("test_table_row_with_different_cols", func(t *testing.T) {
		renderEq(t, "cols:5",
			"{% tablerow n in numbers cols:5%} {{n}} {% endtablerow %}",
			map[string]any{"numbers": []any{1, 2, 3, 4, 5, 6}},
			"<tr class=\"row1\">\n<td class=\"col1\"> 1 </td><td class=\"col2\"> 2 </td><td class=\"col3\"> 3 </td><td class=\"col4\"> 4 </td><td class=\"col5\"> 5 </td></tr>\n<tr class=\"row2\"><td class=\"col1\"> 6 </td></tr>\n")
	})

	t.Run("test_table_col_counter", func(t *testing.T) {
		renderEq(t, "col-counter",
			"{% tablerow n in numbers cols:2%}{{tablerowloop.col}}{% endtablerow %}",
			map[string]any{"numbers": []any{1, 2, 3, 4, 5, 6}},
			"<tr class=\"row1\">\n<td class=\"col1\">1</td><td class=\"col2\">2</td></tr>\n<tr class=\"row2\"><td class=\"col1\">1</td><td class=\"col2\">2</td></tr>\n<tr class=\"row3\"><td class=\"col1\">1</td><td class=\"col2\">2</td></tr>\n")
	})

	t.Run("test_quoted_fragment", func(t *testing.T) {
		want := "<tr class=\"row1\">\n<td class=\"col1\"> 1 </td><td class=\"col2\"> 2 </td><td class=\"col3\"> 3 </td></tr>\n<tr class=\"row2\"><td class=\"col1\"> 4 </td><td class=\"col2\"> 5 </td><td class=\"col3\"> 6 </td></tr>\n"
		data := map[string]any{"collections": map[string]any{"frontpage": []any{1, 2, 3, 4, 5, 6}}}
		renderEq(t, "dotted",
			"{% tablerow n in collections.frontpage cols:3%} {{n}} {% endtablerow %}",
			data, want)
		renderEq(t, "bracketed",
			"{% tablerow n in collections['frontpage'] cols:3%} {{n}} {% endtablerow %}",
			data, want)
	})

	t.Run("test_enumerable_drop", func(t *testing.T) {
		t.Skip("TODO: Liquid::Drop with Enumerable iteration — Ruby Drop API not exposed in go-liquid")
		renderEq(t, "drop",
			"{% tablerow n in numbers cols:3%} {{n}} {% endtablerow %}", nil,
			"<tr class=\"row1\">\n<td class=\"col1\"> 1 </td><td class=\"col2\"> 2 </td><td class=\"col3\"> 3 </td></tr>\n<tr class=\"row2\"><td class=\"col1\"> 4 </td><td class=\"col2\"> 5 </td><td class=\"col3\"> 6 </td></tr>\n")
	})

	t.Run("test_offset_and_limit", func(t *testing.T) {
		renderEq(t, "offset/limit",
			"{% tablerow n in numbers cols:3 offset:1 limit:6%} {{n}} {% endtablerow %}",
			map[string]any{"numbers": []any{0, 1, 2, 3, 4, 5, 6, 7}},
			"<tr class=\"row1\">\n<td class=\"col1\"> 1 </td><td class=\"col2\"> 2 </td><td class=\"col3\"> 3 </td></tr>\n<tr class=\"row2\"><td class=\"col1\"> 4 </td><td class=\"col2\"> 5 </td><td class=\"col3\"> 6 </td></tr>\n")
	})

	t.Run("test_blank_string_not_iterable", func(t *testing.T) {
		renderEq(t, "blank-string",
			"{% tablerow char in characters cols:3 %}I WILL NOT BE OUTPUT{% endtablerow %}",
			map[string]any{"characters": ""},
			"<tr class=\"row1\">\n</tr>\n")
	})

	t.Run("test_cols_nil_constant_same_as_evaluated_nil_expression", func(t *testing.T) {
		expect := "<tr class=\"row1\">\n<td class=\"col1\">false</td><td class=\"col2\">false</td></tr>\n"
		renderEq(t, "cols:nil-literal",
			"{% tablerow i in (1..2) cols:nil %}{{ tablerowloop.col_last }}{% endtablerow %}",
			nil, expect)
		renderEq(t, "cols:nil-var",
			"{% tablerow i in (1..2) cols:var %}{{ tablerowloop.col_last }}{% endtablerow %}",
			map[string]any{"var": nil}, expect)
	})

	t.Run("test_nil_limit_is_treated_as_zero", func(t *testing.T) {
		expect := "<tr class=\"row1\">\n</tr>\n"
		renderEq(t, "limit:nil-literal",
			"{% tablerow i in (1..2) limit:nil %}{{ i }}{% endtablerow %}",
			nil, expect)
		renderEq(t, "limit:nil-var",
			"{% tablerow i in (1..2) limit:var %}{{ i }}{% endtablerow %}",
			map[string]any{"var": nil}, expect)
	})

	t.Run("test_nil_offset_is_treated_as_zero", func(t *testing.T) {
		expect := "<tr class=\"row1\">\n<td class=\"col1\">1:false</td><td class=\"col2\">2:true</td></tr>\n"
		renderEq(t, "offset:nil-literal",
			"{% tablerow i in (1..2) offset:nil %}{{ i }}:{{ tablerowloop.col_last }}{% endtablerow %}",
			nil, expect)
		renderEq(t, "offset:nil-var",
			"{% tablerow i in (1..2) offset:var %}{{ i }}:{{ tablerowloop.col_last }}{% endtablerow %}",
			map[string]any{"var": nil}, expect)
	})

	t.Run("test_tablerow_loop_drop_attributes", func(t *testing.T) {
		tmpl := "{% tablerow i in (1..2) %}\ncol: {{ tablerowloop.col }}\ncol0: {{ tablerowloop.col0 }}\ncol_first: {{ tablerowloop.col_first }}\ncol_last: {{ tablerowloop.col_last }}\nfirst: {{ tablerowloop.first }}\nindex: {{ tablerowloop.index }}\nindex0: {{ tablerowloop.index0 }}\nlast: {{ tablerowloop.last }}\nlength: {{ tablerowloop.length }}\nrindex: {{ tablerowloop.rindex }}\nrindex0: {{ tablerowloop.rindex0 }}\nrow: {{ tablerowloop.row }}\n{% endtablerow %}"
		expected := "<tr class=\"row1\">\n<td class=\"col1\">\ncol: 1\ncol0: 0\ncol_first: true\ncol_last: false\nfirst: true\nindex: 1\nindex0: 0\nlast: false\nlength: 2\nrindex: 2\nrindex0: 1\nrow: 1\n</td><td class=\"col2\">\ncol: 2\ncol0: 1\ncol_first: false\ncol_last: true\nfirst: false\nindex: 2\nindex0: 1\nlast: true\nlength: 2\nrindex: 1\nrindex0: 0\nrow: 1\n</td></tr>\n"
		renderEq(t, "drop-attrs", tmpl, nil, expected)
	})

	t.Run("test_table_row_renders_correct_error_message_for_invalid_parameters", func(t *testing.T) {
		t.Skip("TODO: emit Liquid::ArgumentError 'invalid integer' for non-integer cols/limit/offset and inline it under render_errors:true (Ruby parity)")
		_, err := Render(`{% tablerow n in (1..10) limit:true %} {{n}} {% endtablerow %}`, nil)
		if err == nil {
			t.Fatal("expected invalid integer error")
		}
		_, err = Render(`{% tablerow n in (1..10) offset:true %} {{n}} {% endtablerow %}`, nil)
		if err == nil {
			t.Fatal("expected invalid integer error")
		}
		_, err = Render(`{% tablerow n in (1..10) cols:true %} {{n}} {% endtablerow %}`, nil)
		if err == nil {
			t.Fatal("expected invalid integer error")
		}
	})

	t.Run("test_table_row_handles_interrupts", func(t *testing.T) {
		renderEq(t, "break",
			"{% tablerow n in (1..3) cols:2 %} {{n}} {% break %} {{n}} {% endtablerow %}",
			nil, "<tr class=\"row1\">\n<td class=\"col1\"> 1 </td></tr>\n")
		renderEq(t, "continue",
			"{% tablerow n in (1..3) cols:2 %} {{n}} {% continue %} {{n}} {% endtablerow %}",
			nil, "<tr class=\"row1\">\n<td class=\"col1\"> 1 </td><td class=\"col2\"> 2 </td></tr>\n<tr class=\"row2\"><td class=\"col1\"> 3 </td></tr>\n")
	})

	t.Run("test_table_row_does_not_leak_interrupts", func(t *testing.T) {
		tmpl := "{% for i in (1..2) -%}\n{% for j in (1..2) -%}\n{% tablerow k in (1..3) %}{% break %}{% endtablerow -%}\nloop j={{ j }}\n{% endfor -%}\nloop i={{ i }}\n{% endfor -%}\nafter loop\n"
		expected := "<tr class=\"row1\">\n<td class=\"col1\"></td></tr>\nloop j=1\n<tr class=\"row1\">\n<td class=\"col1\"></td></tr>\nloop j=2\nloop i=1\n<tr class=\"row1\">\n<td class=\"col1\"></td></tr>\nloop j=1\n<tr class=\"row1\">\n<td class=\"col1\"></td></tr>\nloop j=2\nloop i=2\nafter loop\n"
		renderEq(t, "no-leak", tmpl, nil, expected)
	})

	// strict2-only tests below: Ruby uses strict2 mode which we don't implement.
	// They still exercise valid behavior (the strict2 form is the same syntax),
	// so we run them in our default mode.

	t.Run("test_tablerow_with_cols_attribute", func(t *testing.T) {
		renderEq(t, "cols:3",
			"{% tablerow i in (1..6) cols: 3 %}{{ i }}{% endtablerow %}",
			nil,
			"<tr class=\"row1\">\n<td class=\"col1\">1</td><td class=\"col2\">2</td><td class=\"col3\">3</td></tr>\n<tr class=\"row2\"><td class=\"col1\">4</td><td class=\"col2\">5</td><td class=\"col3\">6</td></tr>\n")
	})

	t.Run("test_tablerow_with_limit_attribute", func(t *testing.T) {
		renderEq(t, "limit:3",
			"{% tablerow i in (1..10) limit: 3 %}{{ i }}{% endtablerow %}",
			nil, "<tr class=\"row1\">\n<td class=\"col1\">1</td><td class=\"col2\">2</td><td class=\"col3\">3</td></tr>\n")
	})

	t.Run("test_tablerow_with_offset_attribute", func(t *testing.T) {
		renderEq(t, "offset:2",
			"{% tablerow i in (1..5) offset: 2 %}{{ i }}{% endtablerow %}",
			nil, "<tr class=\"row1\">\n<td class=\"col1\">3</td><td class=\"col2\">4</td><td class=\"col3\">5</td></tr>\n")
	})

	t.Run("test_tablerow_with_range_attribute_in_strict2_mode", func(t *testing.T) {
		t.Skip("TODO: tablerow `range:` attribute is a strict2-only knob (overrides the iteration source); not implemented")
		renderEq(t, "range",
			"{% tablerow i in (1..3) range: (1..10) %}{{ i }}{% endtablerow %}",
			nil, "<tr class=\"row1\">\n<td class=\"col1\">1</td><td class=\"col2\">2</td><td class=\"col3\">3</td></tr>\n")
	})

	t.Run("test_tablerow_with_multiple_attributes", func(t *testing.T) {
		renderEq(t, "multi",
			"{% tablerow i in (1..10) cols: 2, limit: 4, offset: 1 %}{{ i }}{% endtablerow %}",
			nil, "<tr class=\"row1\">\n<td class=\"col1\">2</td><td class=\"col2\">3</td></tr>\n<tr class=\"row2\"><td class=\"col1\">4</td><td class=\"col2\">5</td></tr>\n")
	})

	t.Run("test_tablerow_with_variable_collection", func(t *testing.T) {
		renderEq(t, "var-coll",
			"{% tablerow n in numbers cols: 2 %}{{ n }}{% endtablerow %}",
			map[string]any{"numbers": []any{1, 2, 3, 4}},
			"<tr class=\"row1\">\n<td class=\"col1\">1</td><td class=\"col2\">2</td></tr>\n<tr class=\"row2\"><td class=\"col1\">3</td><td class=\"col2\">4</td></tr>\n")
	})

	t.Run("test_tablerow_with_dotted_access", func(t *testing.T) {
		renderEq(t, "dotted",
			"{% tablerow n in obj.numbers cols: 2 %}{{ n }}{% endtablerow %}",
			map[string]any{"obj": map[string]any{"numbers": []any{1, 2, 3, 4}}},
			"<tr class=\"row1\">\n<td class=\"col1\">1</td><td class=\"col2\">2</td></tr>\n<tr class=\"row2\"><td class=\"col1\">3</td><td class=\"col2\">4</td></tr>\n")
	})

	t.Run("test_tablerow_with_bracketed_access", func(t *testing.T) {
		renderEq(t, "bracketed",
			`{% tablerow n in obj["numbers"] cols: 2 %}{{ n }}{% endtablerow %}`,
			map[string]any{"obj": map[string]any{"numbers": []any{10, 20}}},
			"<tr class=\"row1\">\n<td class=\"col1\">10</td><td class=\"col2\">20</td></tr>\n")
	})

	t.Run("test_tablerow_without_attributes", func(t *testing.T) {
		renderEq(t, "no-attrs",
			"{% tablerow i in (1..3) %}{{ i }}{% endtablerow %}",
			nil, "<tr class=\"row1\">\n<td class=\"col1\">1</td><td class=\"col2\">2</td><td class=\"col3\">3</td></tr>\n")
	})

	t.Run("test_tablerow_without_in_keyword", func(t *testing.T) {
		if _, err := Parse("{% tablerow i (1..10) %}{{ i }}{% endtablerow %}"); err == nil {
			t.Fatal("expected parse error: tablerow without 'in'")
		}
	})

	t.Run("test_tablerow_with_multiple_invalid_attributes_reports_first_in_strict2_mode", func(t *testing.T) {
		t.Skip("TODO: strict2-only — reject unknown tablerow attributes with 'Invalid attribute' error message; lax/strict ignore them")
		_, err := Parse("{% tablerow i in (1..10) invalid1: 5, invalid2: 10 %}{{ i }}{% endtablerow %}")
		if err == nil {
			t.Fatal("expected error for invalid attributes")
		}
	})

	t.Run("test_tablerow_with_empty_collection", func(t *testing.T) {
		renderEq(t, "empty-coll",
			"{% tablerow i in empty_array cols: 2 %}{{ i }}{% endtablerow %}",
			map[string]any{"empty_array": []any{}},
			"<tr class=\"row1\">\n</tr>\n")
	})

	t.Run("test_tablerow_with_invalid_attribute_strict_vs_strict2", func(t *testing.T) {
		t.Skip("TODO: Ruby :lax/:strict silently ignore unknown tablerow attributes; go-liquid currently surfaces a parse error")
		// In :lax/:strict the bad attribute is silently ignored; the
		// iteration runs with no limit/offset.
		renderEq(t, "lax-ignores-bad-attr",
			"{% tablerow i in (1..5) invalid_attr: 10 %}{{ i }}{% endtablerow %}",
			nil, "<tr class=\"row1\">\n<td class=\"col1\">1</td><td class=\"col2\">2</td><td class=\"col3\">3</td><td class=\"col4\">4</td><td class=\"col5\">5</td></tr>\n")
	})

	t.Run("test_tablerow_with_invalid_expression_strict_vs_strict2", func(t *testing.T) {
		t.Skip("TODO: Ruby :lax/:strict tolerate `limit: foo=>bar` and render an empty row; go-liquid currently rejects")
		renderEq(t, "lax-ignores-bad-expr",
			"{% tablerow i in (1..5) limit: foo=>bar %}{{ i }}{% endtablerow %}",
			nil, "<tr class=\"row1\">\n</tr>\n")
	})
}
