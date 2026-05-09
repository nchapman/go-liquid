package liquid

import (
	"testing"
)

func TestRenderSimple(t *testing.T) {
	tests := []struct {
		name     string
		template string
		data     map[string]any
		expected string
	}{
		{
			name:     "plain text",
			template: "Hello, World!",
			data:     nil,
			expected: "Hello, World!",
		},
		{
			name:     "variable output",
			template: "Hello, {{ name }}!",
			data:     map[string]any{"name": "Alice"},
			expected: "Hello, Alice!",
		},
		{
			name:     "undefined variable",
			template: "Hello, {{ name }}!",
			data:     nil,
			expected: "Hello, !",
		},
		{
			name:     "integer variable",
			template: "Count: {{ count }}",
			data:     map[string]any{"count": 42},
			expected: "Count: 42",
		},
		{
			name:     "float variable",
			template: "Price: {{ price }}",
			data:     map[string]any{"price": 19.99},
			expected: "Price: 19.99",
		},
		{
			name:     "boolean true",
			template: "Active: {{ active }}",
			data:     map[string]any{"active": true},
			expected: "Active: true",
		},
		{
			name:     "boolean false",
			template: "Active: {{ active }}",
			data:     map[string]any{"active": false},
			expected: "Active: false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Render(tt.template, tt.data)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestDotAccess(t *testing.T) {
	tests := []struct {
		name     string
		template string
		data     map[string]any
		expected string
	}{
		{
			name:     "nested map access",
			template: "{{ user.name }}",
			data: map[string]any{
				"user": map[string]any{"name": "Bob"},
			},
			expected: "Bob",
		},
		{
			name:     "deep nested access",
			template: "{{ user.address.city }}",
			data: map[string]any{
				"user": map[string]any{
					"address": map[string]any{"city": "NYC"},
				},
			},
			expected: "NYC",
		},
		{
			name:     "array first",
			template: "{{ items.first }}",
			data:     map[string]any{"items": []any{1, 2, 3}},
			expected: "1",
		},
		{
			name:     "array last",
			template: "{{ items.last }}",
			data:     map[string]any{"items": []any{1, 2, 3}},
			expected: "3",
		},
		{
			name:     "array size",
			template: "{{ items.size }}",
			data:     map[string]any{"items": []any{1, 2, 3}},
			expected: "3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Render(tt.template, tt.data)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestIndexAccess(t *testing.T) {
	tests := []struct {
		name     string
		template string
		data     map[string]any
		expected string
	}{
		{
			name:     "array index",
			template: "{{ items[0] }}",
			data:     map[string]any{"items": []any{"a", "b", "c"}},
			expected: "a",
		},
		{
			name:     "map with string key",
			template: `{{ data["key"] }}`,
			data:     map[string]any{"data": map[string]any{"key": "value"}},
			expected: "value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Render(tt.template, tt.data)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestFilters(t *testing.T) {
	tests := []struct {
		name     string
		template string
		data     map[string]any
		expected string
	}{
		{
			name:     "upcase",
			template: "{{ name | upcase }}",
			data:     map[string]any{"name": "alice"},
			expected: "ALICE",
		},
		{
			name:     "downcase",
			template: "{{ name | downcase }}",
			data:     map[string]any{"name": "ALICE"},
			expected: "alice",
		},
		{
			name:     "strip",
			template: "{{ text | strip }}",
			data:     map[string]any{"text": "  hello  "},
			expected: "hello",
		},
		{
			name:     "escape",
			template: "{{ html | escape }}",
			data:     map[string]any{"html": "<b>bold</b>"},
			expected: "&lt;b&gt;bold&lt;/b&gt;",
		},
		{
			name:     "first",
			template: "{{ items | first }}",
			data:     map[string]any{"items": []any{1, 2, 3}},
			expected: "1",
		},
		{
			name:     "last",
			template: "{{ items | last }}",
			data:     map[string]any{"items": []any{1, 2, 3}},
			expected: "3",
		},
		{
			name:     "size",
			template: "{{ items | size }}",
			data:     map[string]any{"items": []any{1, 2, 3}},
			expected: "3",
		},
		{
			name:     "join default",
			template: "{{ items | join }}",
			data:     map[string]any{"items": []any{"a", "b", "c"}},
			expected: "a b c",
		},
		{
			name:     "join with separator",
			template: `{{ items | join: ", " }}`,
			data:     map[string]any{"items": []any{"a", "b", "c"}},
			expected: "a, b, c",
		},
		{
			name:     "reverse",
			template: "{{ items | reverse | join }}",
			data:     map[string]any{"items": []any{"a", "b", "c"}},
			expected: "c b a",
		},
		{
			name:     "default with nil",
			template: `{{ missing | default: "N/A" }}`,
			data:     nil,
			expected: "N/A",
		},
		{
			name:     "default with value",
			template: `{{ name | default: "N/A" }}`,
			data:     map[string]any{"name": "Bob"},
			expected: "Bob",
		},
		{
			name:     "plus",
			template: "{{ num | plus: 5 }}",
			data:     map[string]any{"num": 10},
			expected: "15",
		},
		{
			name:     "minus",
			template: "{{ num | minus: 3 }}",
			data:     map[string]any{"num": 10},
			expected: "7",
		},
		{
			name:     "filter chain",
			template: "{{ name | upcase | strip }}",
			data:     map[string]any{"name": "  alice  "},
			expected: "ALICE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Render(tt.template, tt.data)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestIfTag(t *testing.T) {
	tests := []struct {
		name     string
		template string
		data     map[string]any
		expected string
	}{
		{
			name:     "if true",
			template: "{% if show %}visible{% endif %}",
			data:     map[string]any{"show": true},
			expected: "visible",
		},
		{
			name:     "if false",
			template: "{% if show %}visible{% endif %}",
			data:     map[string]any{"show": false},
			expected: "",
		},
		{
			name:     "if else true",
			template: "{% if show %}yes{% else %}no{% endif %}",
			data:     map[string]any{"show": true},
			expected: "yes",
		},
		{
			name:     "if else false",
			template: "{% if show %}yes{% else %}no{% endif %}",
			data:     map[string]any{"show": false},
			expected: "no",
		},
		{
			name:     "if elsif else",
			template: "{% if x == 1 %}one{% elsif x == 2 %}two{% else %}other{% endif %}",
			data:     map[string]any{"x": 2},
			expected: "two",
		},
		{
			name:     "if with comparison ==",
			template: "{% if x == 5 %}yes{% endif %}",
			data:     map[string]any{"x": 5},
			expected: "yes",
		},
		{
			name:     "if with comparison !=",
			template: "{% if x != 5 %}yes{% endif %}",
			data:     map[string]any{"x": 3},
			expected: "yes",
		},
		{
			name:     "if with comparison <",
			template: "{% if x < 5 %}yes{% endif %}",
			data:     map[string]any{"x": 3},
			expected: "yes",
		},
		{
			name:     "if with comparison >",
			template: "{% if x > 5 %}yes{% endif %}",
			data:     map[string]any{"x": 7},
			expected: "yes",
		},
		{
			name:     "if with and",
			template: "{% if a and b %}yes{% endif %}",
			data:     map[string]any{"a": true, "b": true},
			expected: "yes",
		},
		{
			name:     "if with or",
			template: "{% if a or b %}yes{% endif %}",
			data:     map[string]any{"a": false, "b": true},
			expected: "yes",
		},
		{
			name:     "if with contains string",
			template: `{% if text contains "hello" %}yes{% endif %}`,
			data:     map[string]any{"text": "hello world"},
			expected: "yes",
		},
		{
			name:     "if with contains array",
			template: "{% if items contains 2 %}yes{% endif %}",
			data:     map[string]any{"items": []any{1, 2, 3}},
			expected: "yes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Render(tt.template, tt.data)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestUnlessTag(t *testing.T) {
	tests := []struct {
		name     string
		template string
		data     map[string]any
		expected string
	}{
		{
			name:     "unless false",
			template: "{% unless hidden %}visible{% endunless %}",
			data:     map[string]any{"hidden": false},
			expected: "visible",
		},
		{
			name:     "unless true",
			template: "{% unless hidden %}visible{% endunless %}",
			data:     map[string]any{"hidden": true},
			expected: "",
		},
		{
			name:     "unless with else",
			template: "{% unless show %}hidden{% else %}visible{% endunless %}",
			data:     map[string]any{"show": true},
			expected: "visible",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Render(tt.template, tt.data)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestCaseTag(t *testing.T) {
	tests := []struct {
		name     string
		template string
		data     map[string]any
		expected string
	}{
		{
			name: "case match first",
			template: `{% case value %}
{% when 1 %}one
{% when 2 %}two
{% else %}other
{% endcase %}`,
			data:     map[string]any{"value": 1},
			expected: "one\n",
		},
		{
			name: "case match second",
			template: `{% case value %}
{% when 1 %}one
{% when 2 %}two
{% else %}other
{% endcase %}`,
			data:     map[string]any{"value": 2},
			expected: "two\n",
		},
		{
			name: "case else",
			template: `{% case value %}
{% when 1 %}one
{% when 2 %}two
{% else %}other
{% endcase %}`,
			data:     map[string]any{"value": 3},
			expected: "other\n",
		},
		{
			name:     "case string",
			template: `{% case color %}{% when "red" %}R{% when "green" %}G{% when "blue" %}B{% endcase %}`,
			data:     map[string]any{"color": "green"},
			expected: "G",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Render(tt.template, tt.data)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestForTag(t *testing.T) {
	tests := []struct {
		name     string
		template string
		data     map[string]any
		expected string
	}{
		{
			name:     "simple for",
			template: "{% for item in items %}{{ item }}{% endfor %}",
			data:     map[string]any{"items": []any{1, 2, 3}},
			expected: "123",
		},
		{
			name:     "for with separator",
			template: "{% for item in items %}{% if forloop.first == false %}, {% endif %}{{ item }}{% endfor %}",
			data:     map[string]any{"items": []any{"a", "b", "c"}},
			expected: "a, b, c",
		},
		{
			name:     "forloop.index",
			template: "{% for item in items %}{{ forloop.index }}{% endfor %}",
			data:     map[string]any{"items": []any{"a", "b", "c"}},
			expected: "123",
		},
		{
			name:     "forloop.index0",
			template: "{% for item in items %}{{ forloop.index0 }}{% endfor %}",
			data:     map[string]any{"items": []any{"a", "b", "c"}},
			expected: "012",
		},
		{
			name:     "forloop.first",
			template: "{% for item in items %}{% if forloop.first %}*{% endif %}{{ item }}{% endfor %}",
			data:     map[string]any{"items": []any{"a", "b", "c"}},
			expected: "*abc",
		},
		{
			name:     "forloop.last",
			template: "{% for item in items %}{{ item }}{% if forloop.last %}!{% endif %}{% endfor %}",
			data:     map[string]any{"items": []any{"a", "b", "c"}},
			expected: "abc!",
		},
		{
			name:     "forloop.length",
			template: "{% for item in items %}{{ forloop.length }}{% endfor %}",
			data:     map[string]any{"items": []any{"a", "b", "c"}},
			expected: "333",
		},
		{
			name:     "for range",
			template: "{% for i in (1..3) %}{{ i }}{% endfor %}",
			data:     nil,
			expected: "123",
		},
		{
			name:     "for with limit",
			template: "{% for item in items limit: 2 %}{{ item }}{% endfor %}",
			data:     map[string]any{"items": []any{1, 2, 3, 4, 5}},
			expected: "12",
		},
		{
			name:     "for with offset",
			template: "{% for item in items offset: 2 %}{{ item }}{% endfor %}",
			data:     map[string]any{"items": []any{1, 2, 3, 4, 5}},
			expected: "345",
		},
		{
			name:     "for with limit and offset",
			template: "{% for item in items limit: 2 offset: 1 %}{{ item }}{% endfor %}",
			data:     map[string]any{"items": []any{1, 2, 3, 4, 5}},
			expected: "23",
		},
		{
			name:     "for reversed",
			template: "{% for item in items reversed %}{{ item }}{% endfor %}",
			data:     map[string]any{"items": []any{1, 2, 3}},
			expected: "321",
		},
		{
			name:     "for else empty",
			template: "{% for item in items %}{{ item }}{% else %}empty{% endfor %}",
			data:     map[string]any{"items": []any{}},
			expected: "empty",
		},
		{
			name:     "for else with items",
			template: "{% for item in items %}{{ item }}{% else %}empty{% endfor %}",
			data:     map[string]any{"items": []any{1, 2}},
			expected: "12",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Render(tt.template, tt.data)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestBreakContinue(t *testing.T) {
	tests := []struct {
		name     string
		template string
		data     map[string]any
		expected string
	}{
		{
			name:     "break",
			template: "{% for i in (1..5) %}{% if i == 3 %}{% break %}{% endif %}{{ i }}{% endfor %}",
			data:     nil,
			expected: "12",
		},
		{
			name:     "continue",
			template: "{% for i in (1..5) %}{% if i == 3 %}{% continue %}{% endif %}{{ i }}{% endfor %}",
			data:     nil,
			expected: "1245",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Render(tt.template, tt.data)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestAssignTag(t *testing.T) {
	tests := []struct {
		name     string
		template string
		data     map[string]any
		expected string
	}{
		{
			name:     "assign literal",
			template: `{% assign name = "Alice" %}Hello, {{ name }}!`,
			data:     nil,
			expected: "Hello, Alice!",
		},
		{
			name:     "assign from variable",
			template: "{% assign greeting = message %}{{ greeting }}",
			data:     map[string]any{"message": "Hi!"},
			expected: "Hi!",
		},
		{
			name:     "assign with filter",
			template: `{% assign name = "alice" | upcase %}{{ name }}`,
			data:     nil,
			expected: "ALICE",
		},
		{
			name:     "assign number",
			template: "{% assign x = 42 %}{{ x }}",
			data:     nil,
			expected: "42",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Render(tt.template, tt.data)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestCaptureTag(t *testing.T) {
	tests := []struct {
		name     string
		template string
		data     map[string]any
		expected string
	}{
		{
			name:     "capture text",
			template: "{% capture greeting %}Hello, World!{% endcapture %}{{ greeting }}",
			data:     nil,
			expected: "Hello, World!",
		},
		{
			name:     "capture with variable",
			template: "{% capture msg %}Hello, {{ name }}!{% endcapture %}{{ msg }}",
			data:     map[string]any{"name": "Bob"},
			expected: "Hello, Bob!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Render(tt.template, tt.data)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestCommentTag(t *testing.T) {
	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "comment removed",
			template: "Before{% comment %}This is hidden{% endcomment %}After",
			expected: "BeforeAfter",
		},
		{
			name:     "multiline comment",
			template: "A{% comment %}Line 1\nLine 2{% endcomment %}B",
			expected: "AB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Render(tt.template, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestRawTag(t *testing.T) {
	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "raw preserves liquid syntax",
			template: "{% raw %}{{ name }}{% endraw %}",
			expected: "{{ name }}",
		},
		{
			name:     "raw preserves tags",
			template: "{% raw %}{% if true %}yes{% endif %}{% endraw %}",
			expected: "{% if true %}yes{% endif %}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Render(tt.template, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestEmptyBlank(t *testing.T) {
	tests := []struct {
		name     string
		template string
		data     map[string]any
		expected string
	}{
		{
			name:     "empty string is empty",
			template: `{% if text == empty %}yes{% endif %}`,
			data:     map[string]any{"text": ""},
			expected: "yes",
		},
		{
			name:     "empty array is empty",
			template: `{% if items == empty %}yes{% endif %}`,
			data:     map[string]any{"items": []any{}},
			expected: "yes",
		},
		{
			name:     "non-empty is not empty",
			template: `{% if text == empty %}yes{% else %}no{% endif %}`,
			data:     map[string]any{"text": "hello"},
			expected: "no",
		},
		{
			name:     "whitespace is blank",
			template: `{% if text == blank %}yes{% endif %}`,
			data:     map[string]any{"text": "   "},
			expected: "yes",
		},
		{
			name:     "false is blank",
			template: `{% if flag == blank %}yes{% endif %}`,
			data:     map[string]any{"flag": false},
			expected: "yes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Render(tt.template, tt.data)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestMustParse(t *testing.T) {
	// Valid template
	tmpl := MustParse("Hello, {{ name }}!")
	result, err := tmpl.Render(map[string]any{"name": "World"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "Hello, World!" {
		t.Errorf("expected 'Hello, World!', got %q", result)
	}

	// Invalid template should panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for invalid template")
		}
	}()
	MustParse("{% if %}") // Missing condition
}

func TestMustRender(t *testing.T) {
	result := MustRender("Hello, {{ name }}!", map[string]any{"name": "World"})
	if result != "Hello, World!" {
		t.Errorf("expected 'Hello, World!', got %q", result)
	}
}

func TestParseError(t *testing.T) {
	tests := []struct {
		name     string
		template string
	}{
		{"missing endif", "{% if true %}hello"},
		{"missing endfor", "{% for i in items %}{{ i }}"},
		{"invalid tag", "{% invalid %}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(tt.template)
			if err == nil {
				t.Error("expected parse error")
			}
		})
	}
}

func TestNestedLoops(t *testing.T) {
	template := `{% for i in outer %}{% for j in inner %}{{ i }}-{{ j }} {% endfor %}{% endfor %}`
	data := map[string]any{
		"outer": []any{1, 2},
		"inner": []any{"a", "b"},
	}

	result, err := Render(template, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "1-a 1-b 2-a 2-b "
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestStringLiterals(t *testing.T) {
	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "double quoted",
			template: `{{ "hello" }}`,
			expected: "hello",
		},
		{
			name:     "single quoted",
			template: `{{ 'hello' }}`,
			expected: "hello",
		},
		{
			name:     "escaped characters",
			template: `{{ "hello\nworld" }}`,
			expected: "hello\nworld",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Render(tt.template, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}
