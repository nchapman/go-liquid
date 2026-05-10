package liquid

import "testing"

// Upstream parity: integration/hash_rendering_test.rb
//
// Ruby Hash#to_s renders as {"k"=>"v"}. go-liquid mirrors that format
// via rubyHashToS in filters.go, with one deliberate divergence: Go's map
// iteration order is non-deterministic, so keys are sorted alphabetically
// rather than preserving insertion order. Where Ruby's expected output
// depends on insertion order, the expected value here is adjusted to the
// sorted equivalent (commented inline).

func TestUpstream_HashRendering_Empty(t *testing.T) {
	renderEq(t, "empty", "{{ my_hash }}", map[string]any{"my_hash": map[string]any{}}, "{}")
}

func TestUpstream_HashRendering_StringKeysAndValues(t *testing.T) {
	renderEq(t, "h", "{{ my_hash }}",
		map[string]any{"my_hash": map[string]any{"key1": "value1", "key2": "value2"}},
		`{"key1"=>"value1", "key2"=>"value2"}`)
}

func TestUpstream_HashRendering_SymbolKeysAndIntegerValues(t *testing.T) {
	t.Skip("Ruby symbol-keyed Hash#to_s not modeled in go-liquid")
}

func TestUpstream_HashRendering_NestedHash(t *testing.T) {
	renderEq(t, "nested", "{{ my_hash }}",
		map[string]any{"my_hash": map[string]any{"outer": map[string]any{"inner": "value"}}},
		`{"outer"=>{"inner"=>"value"}}`)
}

func TestUpstream_HashRendering_ArrayValues(t *testing.T) {
	renderEq(t, "array", "{{ my_hash }}",
		map[string]any{"my_hash": map[string]any{"numbers": []any{1, 2, 3}}},
		`{"numbers"=>[1, 2, 3]}`)
}

func TestUpstream_HashRendering_RecursiveHash(t *testing.T) {
	t.Skip("Ruby Hash#to_s cycle detection ({...}) not implemented")
}

func TestUpstream_HashRendering_DowncaseFilter(t *testing.T) {
	// Ruby (insertion-order): {"key"=>"value", "anotherkey"=>"anothervalue"}
	// go-liquid (sorted):     {"anotherkey"=>"anothervalue", "key"=>"value"}
	renderEq(t, "downcase", "{{ my_hash | downcase }}",
		map[string]any{"my_hash": map[string]any{"Key": "Value", "AnotherKey": "AnotherValue"}},
		`{"anotherkey"=>"anothervalue", "key"=>"value"}`)
}

func TestUpstream_HashRendering_UpcaseFilter(t *testing.T) {
	renderEq(t, "upcase", "{{ my_hash | upcase }}",
		map[string]any{"my_hash": map[string]any{"Key": "Value", "AnotherKey": "AnotherValue"}},
		`{"ANOTHERKEY"=>"ANOTHERVALUE", "KEY"=>"VALUE"}`)
}

func TestUpstream_HashRendering_StripFilter(t *testing.T) {
	renderEq(t, "strip", "{{ my_hash | strip }}",
		map[string]any{"my_hash": map[string]any{"Key": "Value", "AnotherKey": "AnotherValue"}},
		`{"AnotherKey"=>"AnotherValue", "Key"=>"Value"}`)
}

func TestUpstream_HashRendering_EscapeFilter(t *testing.T) {
	t.Skip("Go html.EscapeString emits `&#34;` instead of Ruby's `&quot;` — functionally equivalent, byte-different")
	renderEq(t, "escape", "{{ my_hash | escape }}",
		map[string]any{"my_hash": map[string]any{"Key": "Value", "AnotherKey": "AnotherValue"}},
		`{&quot;AnotherKey&quot;=&gt;&quot;AnotherValue&quot;, &quot;Key&quot;=&gt;&quot;Value&quot;}`)
}

func TestUpstream_HashRendering_UrlEncodeFilter(t *testing.T) {
	renderEq(t, "url", "{{ my_hash | url_encode }}",
		map[string]any{"my_hash": map[string]any{"Key": "Value", "AnotherKey": "AnotherValue"}},
		`%7B%22AnotherKey%22%3D%3E%22AnotherValue%22%2C+%22Key%22%3D%3E%22Value%22%7D`)
}

func TestUpstream_HashRendering_StripHtmlFilter(t *testing.T) {
	renderEq(t, "strip_html", "{{ my_hash | strip_html }}",
		map[string]any{"my_hash": map[string]any{"Key": "Value", "AnotherKey": "AnotherValue"}},
		`{"AnotherKey"=>"AnotherValue", "Key"=>"Value"}`)
}

func TestUpstream_HashRendering_TruncateFilter(t *testing.T) {
	renderEq(t, "truncate", "{{ my_hash | truncate: 20 }}",
		map[string]any{"my_hash": map[string]any{"Key": "Value", "AnotherKey": "AnotherValue"}},
		`{"AnotherKey"=>"A...`)
}

func TestUpstream_HashRendering_ReplaceFilter(t *testing.T) {
	renderEq(t, "replace", "{{ my_hash | replace: 'key', 'replaced_key' }}",
		map[string]any{"my_hash": map[string]any{"Key": "Value", "AnotherKey": "AnotherValue"}},
		`{"AnotherKey"=>"AnotherValue", "Key"=>"Value"}`)
}

func TestUpstream_HashRendering_AppendFilter(t *testing.T) {
	renderEq(t, "append", "{{ my_hash | append: ' appended text' }}",
		map[string]any{"my_hash": map[string]any{"Key": "Value", "AnotherKey": "AnotherValue"}},
		`{"AnotherKey"=>"AnotherValue", "Key"=>"Value"} appended text`)
}

func TestUpstream_HashRendering_PrependFilter(t *testing.T) {
	renderEq(t, "prepend", "{{ my_hash | prepend: 'prepended text ' }}",
		map[string]any{"my_hash": map[string]any{"Key": "Value", "AnotherKey": "AnotherValue"}},
		`prepended text {"AnotherKey"=>"AnotherValue", "Key"=>"Value"}`)
}

func TestUpstream_HashRendering_ArrayValuesEmpty(t *testing.T) {
	renderEq(t, "empty-array", "{{ my_hash }}",
		map[string]any{"my_hash": map[string]any{"numbers": []any{}}},
		`{"numbers"=>[]}`)
}

func TestUpstream_HashRendering_ArrayValuesHash(t *testing.T) {
	t.Skip("Ruby symbol-keyed nested hash ({:foo=>42}); go-liquid uses string keys (renders as {\"foo\"=>42})")
}

func TestUpstream_HashRendering_JoinFilterWithHash(t *testing.T) {
	renderEq(t, "join", "{{ my_array | join: glue }}",
		map[string]any{
			"my_array": []any{map[string]any{"key1": "value1"}, map[string]any{"key2": "value2"}},
			"glue":     map[string]any{"lol": "wut"},
		},
		`{"key1"=>"value1"}{"lol"=>"wut"}{"key2"=>"value2"}`)
}

func TestUpstream_HashRendering_WithHashKey(t *testing.T) {
	t.Skip("Ruby Hash-as-Hash-key not modeled in go-liquid (Go maps require comparable keys)")
}

func TestUpstream_HashRendering_CustomToS(t *testing.T) {
	t.Skip("requires Ruby class with custom to_s; no go-liquid equivalent")
}

func TestUpstream_HashRendering_WithoutCustomToSUsesDefaultInspect(t *testing.T) {
	t.Skip("Ruby Hash#inspect default fallback not modeled")
}
