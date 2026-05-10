package liquid

import "testing"

// Upstream parity: integration/hash_rendering_test.rb
//
// Every upstream test relies on Ruby's Hash#to_s / Hash#inspect format
// (e.g. {"key"=>"value"}). go-liquid renders maps differently (or not at
// all — typically empty), so the entire file is skipped with bodies
// retained for the day a Ruby-compatible map renderer is added.

func TestUpstream_HashRendering_Empty(t *testing.T) {
	t.Skip("Ruby Hash#to_s renders '{}'; go-liquid renders empty for maps")
	renderEq(t, "empty", "{{ my_hash }}", map[string]any{"my_hash": map[string]any{}}, "{}")
}

func TestUpstream_HashRendering_StringKeysAndValues(t *testing.T) {
	t.Skip("Ruby Hash#to_s rendering not implemented in go-liquid")
	renderEq(t, "h", "{{ my_hash }}",
		map[string]any{"my_hash": map[string]any{"key1": "value1", "key2": "value2"}},
		`{"key1"=>"value1", "key2"=>"value2"}`)
}

func TestUpstream_HashRendering_SymbolKeysAndIntegerValues(t *testing.T) {
	t.Skip("Ruby symbol-keyed Hash#to_s not modeled in go-liquid")
}

func TestUpstream_HashRendering_NestedHash(t *testing.T) {
	t.Skip("Ruby Hash#to_s rendering not implemented")
}

func TestUpstream_HashRendering_ArrayValues(t *testing.T) {
	t.Skip("Ruby Hash#to_s rendering not implemented")
}

func TestUpstream_HashRendering_RecursiveHash(t *testing.T) {
	t.Skip("Ruby Hash#to_s cycle detection ({...}) not implemented")
}

func TestUpstream_HashRendering_DowncaseFilter(t *testing.T) {
	t.Skip("Filter applied to hash relies on Ruby Hash#to_s first")
}

func TestUpstream_HashRendering_UpcaseFilter(t *testing.T) {
	t.Skip("Filter applied to hash relies on Ruby Hash#to_s first")
}

func TestUpstream_HashRendering_StripFilter(t *testing.T) {
	t.Skip("Filter applied to hash relies on Ruby Hash#to_s first")
}

func TestUpstream_HashRendering_EscapeFilter(t *testing.T) {
	t.Skip("Filter applied to hash relies on Ruby Hash#to_s first")
}

func TestUpstream_HashRendering_UrlEncodeFilter(t *testing.T) {
	t.Skip("Filter applied to hash relies on Ruby Hash#to_s first")
}

func TestUpstream_HashRendering_StripHtmlFilter(t *testing.T) {
	t.Skip("Filter applied to hash relies on Ruby Hash#to_s first")
}

func TestUpstream_HashRendering_TruncateFilter(t *testing.T) {
	t.Skip("Filter applied to hash relies on Ruby Hash#to_s first")
}

func TestUpstream_HashRendering_ReplaceFilter(t *testing.T) {
	t.Skip("Filter applied to hash relies on Ruby Hash#to_s first")
}

func TestUpstream_HashRendering_AppendFilter(t *testing.T) {
	t.Skip("Filter applied to hash relies on Ruby Hash#to_s first")
}

func TestUpstream_HashRendering_PrependFilter(t *testing.T) {
	t.Skip("Filter applied to hash relies on Ruby Hash#to_s first")
}

func TestUpstream_HashRendering_ArrayValuesEmpty(t *testing.T) {
	t.Skip("Ruby Hash#to_s rendering not implemented")
}

func TestUpstream_HashRendering_ArrayValuesHash(t *testing.T) {
	t.Skip("Ruby Hash#to_s rendering not implemented")
}

func TestUpstream_HashRendering_JoinFilterWithHash(t *testing.T) {
	t.Skip("Ruby Hash#to_s rendering needed for hash-glued joins")
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
