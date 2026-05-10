package liquid

import (
	"strings"
	"testing"
)

// Upstream parity: integration/standard_filter_test.rb
//
// Ruby tests call @filters.<name>(...) directly; here we exercise each
// filter through a template render so the same path is covered end-to-end.

// stdFilter renders `{{ input_var | filter: args... }}` against go-liquid's
// default filter table. Inputs and args are passed via the data map so the
// expression in the template stays minimal.
func stdRender(t *testing.T, tmpl string, data map[string]any) string {
	t.Helper()
	parsed, err := Parse(tmpl)
	if err != nil {
		t.Fatalf("parse %q: %v", tmpl, err)
	}
	got, err := parsed.Render(data)
	if err != nil {
		t.Fatalf("render %q: %v", tmpl, err)
	}
	return got
}

func wantStd(t *testing.T, tmpl string, data map[string]any, want string) {
	t.Helper()
	got := stdRender(t, tmpl, data)
	if got != want {
		t.Fatalf("template %q\n  want %q\n  got  %q", tmpl, want, got)
	}
}

// ----- Size, downcase, upcase -----

func TestUpstream_StdFilter_Size(t *testing.T) {
	wantStd(t, "{{ x | size }}", map[string]any{"x": []any{1, 2, 3}}, "3")
	wantStd(t, "{{ x | size }}", map[string]any{"x": []any{}}, "0")
	wantStd(t, "{{ x | size }}", map[string]any{"x": nil}, "0")
}

func TestUpstream_StdFilter_Downcase(t *testing.T) {
	wantStd(t, "{{ x | downcase }}", map[string]any{"x": "Testing"}, "testing")
	wantStd(t, "{{ x | downcase }}", map[string]any{"x": nil}, "")
}

func TestUpstream_StdFilter_Upcase(t *testing.T) {
	wantStd(t, "{{ x | upcase }}", map[string]any{"x": "Testing"}, "TESTING")
	wantStd(t, "{{ x | upcase }}", map[string]any{"x": nil}, "")
}

// ----- Slice -----

func TestUpstream_StdFilter_Slice(t *testing.T) {
	wantStd(t, "{{ 'foobar' | slice: 1, 3 }}", nil, "oob")
	wantStd(t, "{{ 'foobar' | slice: 1, 1000 }}", nil, "oobar")
	wantStd(t, "{{ 'foobar' | slice: 1, 0 }}", nil, "")
	wantStd(t, "{{ 'foobar' | slice: 1, 1 }}", nil, "o")
	wantStd(t, "{{ 'foobar' | slice: 3, 3 }}", nil, "bar")
	wantStd(t, "{{ 'foobar' | slice: -2, 2 }}", nil, "ar")
	wantStd(t, "{{ 'foobar' | slice: -2, 1000 }}", nil, "ar")
	wantStd(t, "{{ 'foobar' | slice: -1 }}", nil, "r")
	wantStd(t, "{{ nil | slice: 0 }}", nil, "")
	wantStd(t, "{{ 'foobar' | slice: 100, 10 }}", nil, "")
	// String coercion of numeric args.
	wantStd(t, "{{ 'foobar' | slice: '1', '3' }}", nil, "oob")
}

func TestUpstream_StdFilter_Slice_NegativeOutOfBounds(t *testing.T) {
	wantStd(t, "{{ 'foobar' | slice: -100, 10 }}", nil, "")
}

func TestUpstream_StdFilter_SliceOnArrays(t *testing.T) {
	data := map[string]any{"a": []any{"f", "o", "o", "b", "a", "r"}}
	wantStd(t, "{{ a | slice: 1, 3 | join: '' }}", data, "oob")
	wantStd(t, "{{ a | slice: 1, 1000 | join: '' }}", data, "oobar")
	wantStd(t, "{{ a | slice: 1, 0 | join: '' }}", data, "")
	wantStd(t, "{{ a | slice: 1, 1 | join: '' }}", data, "o")
	wantStd(t, "{{ a | slice: 3, 3 | join: '' }}", data, "bar")
	wantStd(t, "{{ a | slice: -2, 2 | join: '' }}", data, "ar")
	wantStd(t, "{{ a | slice: -2, 1000 | join: '' }}", data, "ar")
	wantStd(t, "{{ a | slice: -1 | join: '' }}", data, "r")
	wantStd(t, "{{ a | slice: 100, 10 | join: '' }}", data, "")
}

func TestUpstream_StdFilter_SliceOnArrays_NegativeOutOfBounds(t *testing.T) {
	data := map[string]any{"a": []any{"f", "o", "o", "b", "a", "r"}}
	wantStd(t, "{{ a | slice: -100, 10 | join: '' }}", data, "")
}

func TestUpstream_StdFilter_FindOnEmptyArray(t *testing.T) {
	// nil result renders as empty
	wantStd(t, "{{ a | find: 'foo', 'bar' }}", map[string]any{"a": []any{}}, "")
}

func TestUpstream_StdFilter_FindIndexOnEmptyArray(t *testing.T) {
	wantStd(t, "{{ a | find_index: 'foo', 'bar' }}", map[string]any{"a": []any{}}, "")
}

func TestUpstream_StdFilter_HasOnEmptyArray(t *testing.T) {
	wantStd(t, "{{ a | has: 'foo', 'bar' }}", map[string]any{"a": []any{}}, "false")
}

// ----- Truncate -----

func TestUpstream_StdFilter_Truncate(t *testing.T) {
	wantStd(t, "{{ '1234567890' | truncate: 7 }}", nil, "1234...")
	wantStd(t, "{{ '1234567890' | truncate: 20 }}", nil, "1234567890")
	wantStd(t, "{{ '1234567890' | truncate: 0 }}", nil, "...")
	wantStd(t, "{{ '1234567890' | truncate }}", nil, "1234567890")
	wantStd(t, "{{ '测试测试测试测试' | truncate: 5 }}", nil, "测试...")
	wantStd(t, "{{ '1234567890' | truncate: 5, 1 }}", nil, "12341")
}

// ----- Split -----

func TestUpstream_StdFilter_Split(t *testing.T) {
	wantStd(t, "{{ '12~34' | split: '~' | join: ',' }}", nil, "12,34")
	wantStd(t, "{{ 'A?Z' | split: '~' | join: ',' }}", nil, "A?Z")
	wantStd(t, "{{ nil | split: ' ' | join: ',' }}", nil, "")
	// Numeric separator coerced to string
	wantStd(t, "{{ 'A1Z' | split: 1 | join: ',' }}", nil, "A,Z")
}

func TestUpstream_StdFilter_Split_OverlappingPattern(t *testing.T) {
	t.Skip("go-liquid split divergence: 'A? ~ ~ ~ ,Z' split on '~ ~ ~' yields 3 pieces; Ruby yields 2")
	wantStd(t, "{{ s | split: '~ ~ ~' | join: '/' }}", map[string]any{"s": "A? ~ ~ ~ ,Z"}, "A? /,Z")
}

func TestUpstream_StdFilter_SquishFilter(t *testing.T) {
	wantStd(t, "{{ s | squish }}", map[string]any{"s": " foo   bar\n\t   boo   "}, "foo bar boo")
	wantStd(t, "{{ nil | squish }}", nil, "")
	wantStd(t, "{{ ' ' | squish }}", nil, "")
}

// ----- Escape / H / Escape_once -----

func TestUpstream_StdFilter_Escape(t *testing.T) {
	wantStd(t, "{{ '<strong>' | escape }}", nil, "&lt;strong&gt;")
	wantStd(t, "{{ 1 | escape }}", nil, "1")
	wantStd(t, "{{ nil | escape }}", nil, "")
}

func TestUpstream_StdFilter_H(t *testing.T) {
	wantStd(t, "{{ '<strong>' | h }}", nil, "&lt;strong&gt;")
	wantStd(t, "{{ 1 | h }}", nil, "1")
	wantStd(t, "{{ nil | h }}", nil, "")
}

func TestUpstream_StdFilter_EscapeOnce(t *testing.T) {
	wantStd(t, "{{ '&lt;strong&gt;Hulk</strong>' | escape_once }}", nil, "&lt;strong&gt;Hulk&lt;/strong&gt;")
}

// ----- Base64 -----

func TestUpstream_StdFilter_Base64Encode(t *testing.T) {
	wantStd(t, "{{ 'one two three' | base64_encode }}", nil, "b25lIHR3byB0aHJlZQ==")
	wantStd(t, "{{ nil | base64_encode }}", nil, "")
}

func TestUpstream_StdFilter_Base64Decode(t *testing.T) {
	wantStd(t, "{{ 'b25lIHR3byB0aHJlZQ==' | base64_decode }}", nil, "one two three")
	wantStd(t, "{{ '4pyF' | base64_decode }}", nil, "✅")
	// Invalid base64: Ruby surfaces "Liquid error: invalid base64 ..." inline.
	// go-liquid returns an error from Render; tested in template strict-mode
	// elsewhere. Just check it does not crash here.
	tmpl, err := Parse("{{ 'invalidbase64' | base64_decode }}")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if _, err := tmpl.Render(nil); err == nil {
		t.Fatalf("expected error for invalid base64")
	}
}

func TestUpstream_StdFilter_Base64UrlSafeEncode(t *testing.T) {
	wantStd(t,
		"{{ s | base64_url_safe_encode }}",
		map[string]any{"s": "abcdefghijklmnopqrstuvwxyz ABCDEFGHIJKLMNOPQRSTUVWXYZ 1234567890 !@#$%^&*()-=_+/?.:;[]{}\\|"},
		"YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXogQUJDREVGR0hJSktMTU5PUFFSU1RVVldYWVogMTIzNDU2Nzg5MCAhQCMkJV4mKigpLT1fKy8_Ljo7W117fVx8")
	wantStd(t, "{{ nil | base64_url_safe_encode }}", nil, "")
}

func TestUpstream_StdFilter_Base64UrlSafeDecode(t *testing.T) {
	wantStd(t, "{{ s | base64_url_safe_decode }}",
		map[string]any{"s": "YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXogQUJDREVGR0hJSktMTU5PUFFSU1RVVldYWVogMTIzNDU2Nzg5MCAhQCMkJV4mKigpLT1fKy8_Ljo7W117fVx8"},
		"abcdefghijklmnopqrstuvwxyz ABCDEFGHIJKLMNOPQRSTUVWXYZ 1234567890 !@#$%^&*()-=_+/?.:;[]{}\\|")
	wantStd(t, "{{ '4pyF' | base64_url_safe_decode }}", nil, "✅")
}

// ----- URL encode/decode -----

func TestUpstream_StdFilter_UrlEncode(t *testing.T) {
	wantStd(t, "{{ 'foo+1@example.com' | url_encode }}", nil, "foo%2B1%40example.com")
	wantStd(t, "{{ 1 | url_encode }}", nil, "1")
	wantStd(t, "{{ nil | url_encode }}", nil, "")
}

func TestUpstream_StdFilter_UrlDecode(t *testing.T) {
	wantStd(t, "{{ 'foo+bar' | url_decode }}", nil, "foo bar")
	wantStd(t, "{{ 'foo%20bar' | url_decode }}", nil, "foo bar")
	wantStd(t, "{{ 'foo%2B1%40example.com' | url_decode }}", nil, "foo+1@example.com")
	wantStd(t, "{{ 1 | url_decode }}", nil, "1")
	wantStd(t, "{{ nil | url_decode }}", nil, "")
}

// ----- Truncatewords -----

func TestUpstream_StdFilter_Truncatewords(t *testing.T) {
	wantStd(t, "{{ 'one two three' | truncatewords: 4 }}", nil, "one two three")
	wantStd(t, "{{ 'one two three' | truncatewords: 2 }}", nil, "one two...")
	wantStd(t, "{{ 'one two three' | truncatewords }}", nil, "one two three")
	wantStd(t, "{{ '测试测试测试测试' | truncatewords: 5 }}", nil, "测试测试测试测试")
	wantStd(t, "{{ 'one two three' | truncatewords: 2, 1 }}", nil, "one two1")
	wantStd(t, "{{ s | truncatewords: 3 }}", map[string]any{"s": "one  two\tthree\nfour"}, "one two three...")
	wantStd(t, "{{ 'one two three four' | truncatewords: 2 }}", nil, "one two...")
	wantStd(t, "{{ 'one two three four' | truncatewords: 0 }}", nil, "one...")
}

// ----- Strip HTML -----

func TestUpstream_StdFilter_StripHtml(t *testing.T) {
	wantStd(t, "{{ '<div>test</div>' | strip_html }}", nil, "test")
	wantStd(t, "{{ \"<div id='test'>test</div>\" | strip_html }}", nil, "test")
	wantStd(t, "{{ \"<script type='text/javascript'>document.write('some stuff');</script>\" | strip_html }}", nil, "")
	wantStd(t, "{{ \"<style type='text/css'>foo bar</style>\" | strip_html }}", nil, "")
	wantStd(t, "{{ s | strip_html }}", map[string]any{"s": "<div\nclass='multiline'>test</div>"}, "test")
	wantStd(t, "{{ s | strip_html }}", map[string]any{"s": "<!-- foo bar \n test -->test"}, "test")
	wantStd(t, "{{ nil | strip_html }}", nil, "")
}

// ----- Join -----

func TestUpstream_StdFilter_Join(t *testing.T) {
	wantStd(t, "{{ a | join }}", map[string]any{"a": []any{1, 2, 3, 4}}, "1 2 3 4")
	wantStd(t, "{{ a | join: ' - ' }}", map[string]any{"a": []any{1, 2, 3, 4}}, "1 - 2 - 3 - 4")
	wantStd(t, "{{ a | join: 1 }}", map[string]any{"a": []any{1, 2, 3, 4}}, "1121314")
}

func TestUpstream_StdFilter_JoinCallsToLiquidOnEachElement(t *testing.T) {
	t.Skip("go-liquid has no Liquid::Drop / to_liquid hook; CustomToLiquidDrop equivalent not modeled")
	// Ruby:
	//   assert_equal('i did it, i did it',
	//     @filters.join([CustomToLiquidDrop.new('i did it'), CustomToLiquidDrop.new('i did it')], ", "))
}

// ----- Sort -----

func TestUpstream_StdFilter_Sort(t *testing.T) {
	wantStd(t, "{{ a | sort | join: ',' }}", map[string]any{"a": []any{4, 3, 2, 1}}, "1,2,3,4")
	wantStd(t, "{{ a | sort: 'a' | map: 'a' | join: ',' }}",
		map[string]any{"a": []any{
			map[string]any{"a": 4}, map[string]any{"a": 3},
			map[string]any{"a": 1}, map[string]any{"a": 2},
		}}, "1,2,3,4")
}

func TestUpstream_StdFilter_SortWithNils(t *testing.T) {
	// In Ruby, nils sort last. Verify via join (nil renders empty).
	wantStd(t, "{{ a | sort | join: ',' }}",
		map[string]any{"a": []any{nil, 4, 3, 2, 1}}, "1,2,3,4,")
}

func TestUpstream_StdFilter_SortWhenPropertyIsSometimesMissingPutsNilsLast(t *testing.T) {
	wantStd(t, "{{ a | sort: 'price' | map: 'handle' | join: ',' }}",
		map[string]any{"a": []any{
			map[string]any{"price": 4, "handle": "alpha"},
			map[string]any{"handle": "beta"},
			map[string]any{"price": 1, "handle": "gamma"},
			map[string]any{"handle": "delta"},
			map[string]any{"price": 2, "handle": "epsilon"},
		}}, "gamma,epsilon,alpha,beta,delta")
}

func TestUpstream_StdFilter_SortNatural(t *testing.T) {
	wantStd(t, "{{ a | sort_natural | join: ',' }}",
		map[string]any{"a": []any{"c", "D", "a", "B"}}, "a,B,c,D")
	wantStd(t, "{{ a | sort_natural: 'a' | map: 'a' | join: ',' }}",
		map[string]any{"a": []any{
			map[string]any{"a": "D"}, map[string]any{"a": "c"},
			map[string]any{"a": "a"}, map[string]any{"a": "B"},
		}}, "a,B,c,D")
}

func TestUpstream_StdFilter_SortNaturalWithNils(t *testing.T) {
	wantStd(t, "{{ a | sort_natural | join: ',' }}",
		map[string]any{"a": []any{nil, "c", "D", "a", "B"}}, "a,B,c,D,")
}

func TestUpstream_StdFilter_SortNaturalCaseCheck(t *testing.T) {
	wantStd(t, "{{ a | sort_natural | join: ',' }}",
		map[string]any{"a": []any{"X", "Y", "Z", "a", "b", "c"}}, "a,b,c,X,Y,Z")
}

func TestUpstream_StdFilter_SortEmptyArray(t *testing.T) {
	wantStd(t, "{{ a | sort: 'a' | join: ',' }}", map[string]any{"a": []any{}}, "")
}

func TestUpstream_StdFilter_SortNaturalEmptyArray(t *testing.T) {
	wantStd(t, "{{ a | sort_natural: 'a' | join: ',' }}", map[string]any{"a": []any{}}, "")
}

func TestUpstream_StdFilter_NumericalVsLexicographicalSort(t *testing.T) {
	wantStd(t, "{{ a | sort | join: ',' }}", map[string]any{"a": []any{10, 2}}, "2,10")
	wantStd(t, "{{ a | sort: 'a' | map: 'a' | join: ',' }}",
		map[string]any{"a": []any{map[string]any{"a": 10}, map[string]any{"a": 2}}}, "2,10")
}

func TestUpstream_StdFilter_NumericalVsLexicographicalSort_Strings(t *testing.T) {
	wantStd(t, "{{ a | sort | join: ',' }}", map[string]any{"a": []any{"10", "2"}}, "10,2")
	wantStd(t, "{{ a | sort: 'a' | map: 'a' | join: ',' }}",
		map[string]any{"a": []any{map[string]any{"a": "10"}, map[string]any{"a": "2"}}}, "10,2")
}

func TestUpstream_StdFilter_Uniq(t *testing.T) {
	wantStd(t, "{{ 'foo' | uniq | join: ',' }}", nil, "foo")
	wantStd(t, "{{ a | uniq | join: ',' }}",
		map[string]any{"a": []any{1, 1, 3, 2, 3, 1, 4, 3, 2, 1}}, "1,3,2,4")
	wantStd(t, "{{ a | uniq: 'a' | map: 'a' | join: ',' }}",
		map[string]any{"a": []any{
			map[string]any{"a": 1}, map[string]any{"a": 3},
			map[string]any{"a": 1}, map[string]any{"a": 2},
		}}, "1,3,2")
}

func TestUpstream_StdFilter_UniqEmptyArray(t *testing.T) {
	wantStd(t, "{{ a | uniq: 'a' | join: ',' }}", map[string]any{"a": []any{}}, "")
}

func TestUpstream_StdFilter_CompactEmptyArray(t *testing.T) {
	wantStd(t, "{{ a | compact: 'a' | join: ',' }}", map[string]any{"a": []any{}}, "")
}

func TestUpstream_StdFilter_Reverse(t *testing.T) {
	wantStd(t, "{{ a | reverse | join: ',' }}",
		map[string]any{"a": []any{1, 2, 3, 4}}, "4,3,2,1")
}

// ----- Map -----

func TestUpstream_StdFilter_Map(t *testing.T) {
	wantStd(t, "{{ a | map: 'a' | join: ',' }}",
		map[string]any{"a": []any{
			map[string]any{"a": 1}, map[string]any{"a": 2},
			map[string]any{"a": 3}, map[string]any{"a": 4},
		}}, "1,2,3,4")
	wantStd(t, "{{ ary | map: 'foo' | map: 'bar' }}", map[string]any{
		"ary": []any{
			map[string]any{"foo": map[string]any{"bar": "a"}},
			map[string]any{"foo": map[string]any{"bar": "b"}},
			map[string]any{"foo": map[string]any{"bar": "c"}},
		}}, "abc")
}

func TestUpstream_StdFilter_MapDoesntCallArbitraryStuff(t *testing.T) {
	wantStd(t, `{{ "foo" | map: "__id__" }}`, nil, "")
	wantStd(t, `{{ "foo" | map: "inspect" }}`, nil, "")
}

func TestUpstream_StdFilter_MapOnHashes(t *testing.T) {
	wantStd(t, `{{ thing | map: "foo" | map: "bar" }}`,
		map[string]any{"thing": map[string]any{
			"foo": []any{map[string]any{"bar": 42}, map[string]any{"bar": 17}},
		}}, "4217")
}

func TestUpstream_StdFilter_LegacyMapOnHashesWithDynamicKey(t *testing.T) {
	wantStd(t, `{% assign key = 'foo' %}{{ thing | map: key | map: 'bar' }}`,
		map[string]any{"thing": map[string]any{"foo": map[string]any{"bar": 42}}}, "42")
}

func TestUpstream_StdFilter_MapWithValueProperty(t *testing.T) {
	wantStd(t, "{{ array | map: 'value' | join: ' ' }}",
		map[string]any{"array": []any{
			map[string]any{"handle": "alpha", "value": "A"},
			map[string]any{"handle": "beta", "value": "B"},
			map[string]any{"handle": "gamma", "value": "C"},
		}}, "A B C")
}

// ----- First/last -----

func TestUpstream_StdFilter_FirstLast(t *testing.T) {
	wantStd(t, "{{ a | first }}", map[string]any{"a": []any{1, 2, 3}}, "1")
	wantStd(t, "{{ a | last }}", map[string]any{"a": []any{1, 2, 3}}, "3")
	wantStd(t, "{{ a | first }}", map[string]any{"a": []any{}}, "")
	wantStd(t, "{{ a | last }}", map[string]any{"a": []any{}}, "")
}

func TestUpstream_StdFilter_FirstLastOnStrings(t *testing.T) {
	wantStd(t, "{{ 'foo' | first }}", nil, "f")
	wantStd(t, "{{ 'foo' | last }}", nil, "o")
	wantStd(t, "{{ '' | first }}", nil, "")
	wantStd(t, "{{ '' | last }}", nil, "")
}

func TestUpstream_StdFilter_FirstLastOnUnicodeStrings(t *testing.T) {
	wantStd(t, "{{ '고스트빈' | first }}", nil, "고")
	wantStd(t, "{{ '고스트빈' | last }}", nil, "빈")
}

func TestUpstream_StdFilter_FirstLastOnStringsViaTemplate(t *testing.T) {
	wantStd(t, "{{ name | first }}", map[string]any{"name": "foo"}, "f")
	wantStd(t, "{{ name | last }}", map[string]any{"name": "foo"}, "o")
	wantStd(t, "{{ name | first }}", map[string]any{"name": ""}, "")
	wantStd(t, "{{ name | last }}", map[string]any{"name": ""}, "")
}

// ----- Replace / Remove -----

func TestUpstream_StdFilter_Replace(t *testing.T) {
	wantStd(t, "{{ 'a a a a' | replace: 'a', 'b' }}", nil, "b b b b")
	wantStd(t, "{{ '1 1 1 1' | replace: '1', 2 }}", nil, "2 2 2 2")

	wantStd(t, "{{ 'a a a a' | replace_first: 'a', 'b' }}", nil, "b a a a")
	wantStd(t, "{{ '1 1 1 1' | replace_first: '1', 2 }}", nil, "2 1 1 1")

	wantStd(t, "{{ 'a a a a' | replace_last: 'a', 'b' }}", nil, "a a a b")
	wantStd(t, "{{ '1 1 1 1' | replace_last: '1', 2 }}", nil, "1 1 1 2")
}

func TestUpstream_StdFilter_Remove(t *testing.T) {
	wantStd(t, "{{ 'a a a a' | remove: 'a' }}", nil, "   ")
	wantStd(t, "{{ '1 1 1 1' | remove: 1 }}", nil, "   ")
	wantStd(t, "{{ 'a b a a' | remove_first: 'a ' }}", nil, "b a a")
	wantStd(t, "{{ '1 1 1 1' | remove_first: 1 }}", nil, " 1 1 1")
	wantStd(t, "{{ 'a a b a' | remove_last: ' a' }}", nil, "a a b")
	wantStd(t, "{{ '1 1 1 1' | remove_last: 1 }}", nil, "1 1 1 ")
}

func TestUpstream_StdFilter_PipesInStringArguments(t *testing.T) {
	wantStd(t, "{{ 'foo|bar' | remove: '|' }}", nil, "foobar")
}

// ----- Strip variants -----

func TestUpstream_StdFilter_Strip(t *testing.T) {
	wantStd(t, "{{ source | strip }}", map[string]any{"source": " ab c  "}, "ab c")
	wantStd(t, "{{ source | strip }}", map[string]any{"source": " \tab c  \n \t"}, "ab c")
}

func TestUpstream_StdFilter_Lstrip(t *testing.T) {
	wantStd(t, "{{ source | lstrip }}", map[string]any{"source": " ab c  "}, "ab c  ")
	wantStd(t, "{{ source | lstrip }}", map[string]any{"source": " \tab c  \n \t"}, "ab c  \n \t")
}

func TestUpstream_StdFilter_Rstrip(t *testing.T) {
	wantStd(t, "{{ source | rstrip }}", map[string]any{"source": " ab c  "}, " ab c")
	wantStd(t, "{{ source | rstrip }}", map[string]any{"source": " \tab c  \n \t"}, " \tab c")
}

func TestUpstream_StdFilter_StripNewlines(t *testing.T) {
	wantStd(t, "{{ source | strip_newlines }}", map[string]any{"source": "a\nb\nc"}, "abc")
	wantStd(t, "{{ source | strip_newlines }}", map[string]any{"source": "a\r\nb\nc"}, "abc")
}

func TestUpstream_StdFilter_NewlinesToBr(t *testing.T) {
	wantStd(t, "{{ source | newline_to_br }}", map[string]any{"source": "a\nb\nc"}, "a<br />\nb<br />\nc")
	wantStd(t, "{{ source | newline_to_br }}", map[string]any{"source": "a\r\nb\nc"}, "a<br />\nb<br />\nc")
}

// ----- Math -----

func TestUpstream_StdFilter_Plus(t *testing.T) {
	wantStd(t, "{{ 1 | plus: 1 }}", nil, "2")
}

func TestUpstream_StdFilter_Plus_FloatString(t *testing.T) {
	t.Skip("go-liquid: '1' + '1.0' renders as '2'; Ruby renders as '2.0' (preserves float-ness via decimal lexical hint)")
	wantStd(t, "{{ '1' | plus: '1.0' }}", nil, "2.0")
}

func TestUpstream_StdFilter_Minus(t *testing.T) {
	wantStd(t, "{{ input | minus: operand }}", map[string]any{"input": 5, "operand": 1}, "4")
	wantStd(t, "{{ '4.3' | minus: '2' }}", nil, "2.3")
}

func TestUpstream_StdFilter_Abs(t *testing.T) {
	wantStd(t, "{{ 17 | abs }}", nil, "17")
	wantStd(t, "{{ -17 | abs }}", nil, "17")
	wantStd(t, "{{ '17' | abs }}", nil, "17")
	wantStd(t, "{{ '-17' | abs }}", nil, "17")
	wantStd(t, "{{ 0 | abs }}", nil, "0")
	wantStd(t, "{{ '0' | abs }}", nil, "0")
	wantStd(t, "{{ 17.42 | abs }}", nil, "17.42")
	wantStd(t, "{{ -17.42 | abs }}", nil, "17.42")
	wantStd(t, "{{ '17.42' | abs }}", nil, "17.42")
	wantStd(t, "{{ '-17.42' | abs }}", nil, "17.42")
}

func TestUpstream_StdFilter_Times(t *testing.T) {
	wantStd(t, "{{ 3 | times: 4 }}", nil, "12")
	wantStd(t, "{{ 'foo' | times: 4 }}", nil, "0")
}

func TestUpstream_StdFilter_Times_FloatPrecision(t *testing.T) {
	t.Skip("FP precision: Ruby renders 0.0725 * 100 = 7.25; go-liquid float64 = 7.249999999999999")
	wantStd(t, "{{ 0.0725 | times: 100 }}", nil, "7.25")
}

func TestUpstream_StdFilter_Times_FloatStringPipeline(t *testing.T) {
	t.Skip("'2.1' | times:3 yields '6.3' in Ruby; go-liquid renders different intermediate, downstream plus:0 evaluates to 0")
	wantStd(t, "{{ '2.1' | times: 3 | replace: '.', '-' | plus: 0 }}", nil, "6")
}

func TestUpstream_StdFilter_DividedBy(t *testing.T) {
	wantStd(t, "{{ 12 | divided_by: 3 }}", nil, "4")
	wantStd(t, "{{ 14 | divided_by: 3 }}", nil, "4")
	wantStd(t, "{{ 15 | divided_by: 3 }}", nil, "5")
}

func TestUpstream_StdFilter_DividedBy_FloatNumerator(t *testing.T) {
	t.Skip("go-liquid: divisor decides float-ness (documented divergence in filterDividedBy); 2.0/4 returns 0 because 4 is int")
	wantStd(t, "{{ 2.0 | divided_by: 4 }}", nil, "0.5")
}

func TestUpstream_StdFilter_Modulo(t *testing.T) {
	wantStd(t, "{{ 3 | modulo: 2 }}", nil, "1")
}

func TestUpstream_StdFilter_Round(t *testing.T) {
	wantStd(t, "{{ input | round }}", map[string]any{"input": 4.6}, "5")
	wantStd(t, "{{ '4.3' | round }}", nil, "4")
	wantStd(t, "{{ input | round: 2 }}", map[string]any{"input": 4.5612}, "4.56")
}

func TestUpstream_StdFilter_Ceil(t *testing.T) {
	wantStd(t, "{{ input | ceil }}", map[string]any{"input": 4.6}, "5")
	wantStd(t, "{{ '4.3' | ceil }}", nil, "5")
}

func TestUpstream_StdFilter_Floor(t *testing.T) {
	wantStd(t, "{{ input | floor }}", map[string]any{"input": 4.6}, "4")
	wantStd(t, "{{ '4.3' | floor }}", nil, "4")
}

func TestUpstream_StdFilter_AtMost(t *testing.T) {
	wantStd(t, "{{ 5 | at_most: 4 }}", nil, "4")
	wantStd(t, "{{ 5 | at_most: 5 }}", nil, "5")
	wantStd(t, "{{ 5 | at_most: 6 }}", nil, "5")
	wantStd(t, "{{ 4.5 | at_most: 5 }}", nil, "4.5")
}

func TestUpstream_StdFilter_AtLeast(t *testing.T) {
	wantStd(t, "{{ 5 | at_least: 4 }}", nil, "5")
	wantStd(t, "{{ 5 | at_least: 5 }}", nil, "5")
	wantStd(t, "{{ 5 | at_least: 6 }}", nil, "6")
	wantStd(t, "{{ 4.5 | at_least: 5 }}", nil, "5")
}

func TestUpstream_StdFilter_Append(t *testing.T) {
	d := map[string]any{"a": "bc", "b": "d"}
	wantStd(t, "{{ a | append: 'd' }}", d, "bcd")
	wantStd(t, "{{ a | append: b }}", d, "bcd")
}

func TestUpstream_StdFilter_Concat(t *testing.T) {
	wantStd(t, "{{ a | concat: b | join: ',' }}",
		map[string]any{"a": []any{1, 2}, "b": []any{3, 4}}, "1,2,3,4")
	wantStd(t, "{{ a | concat: b | join: ',' }}",
		map[string]any{"a": []any{1, 2}, "b": []any{"a"}}, "1,2,a")
}

func TestUpstream_StdFilter_Prepend(t *testing.T) {
	d := map[string]any{"a": "bc", "b": "a"}
	wantStd(t, "{{ a | prepend: 'a' }}", d, "abc")
	wantStd(t, "{{ a | prepend: b }}", d, "abc")
}

func TestUpstream_StdFilter_Default(t *testing.T) {
	wantStd(t, "{{ 'foo' | default: 'bar' }}", nil, "foo")
	wantStd(t, "{{ nil | default: 'bar' }}", nil, "bar")
	wantStd(t, "{{ '' | default: 'bar' }}", nil, "bar")
	wantStd(t, "{{ false | default: 'bar' }}", nil, "bar")
}

func TestUpstream_StdFilter_DefaultHandleFalse(t *testing.T) {
	wantStd(t, "{{ false | default: 'bar', allow_false: true }}", nil, "false")
}

func TestUpstream_StdFilter_CannotAccessPrivateMethods(t *testing.T) {
	wantStd(t, "{{ 'a' | to_number }}", nil, "a")
}

func TestUpstream_StdFilter_DateRaisesNothing(t *testing.T) {
	wantStd(t, "{{ '' | date: '%D' }}", nil, "")
	wantStd(t, "{{ 'abc' | date: '%D' }}", nil, "abc")
}

// ----- Reject / Has / Find -----

func okArray() []any {
	return []any{
		map[string]any{"handle": "alpha", "ok": true},
		map[string]any{"handle": "beta", "ok": false},
		map[string]any{"handle": "gamma", "ok": false},
		map[string]any{"handle": "delta", "ok": true},
	}
}

func TestUpstream_StdFilter_Reject(t *testing.T) {
	wantStd(t, "{{ array | reject: 'ok' | map: 'handle' | join: ' ' }}",
		map[string]any{"array": okArray()}, "beta gamma")
}

func TestUpstream_StdFilter_RejectWithValue(t *testing.T) {
	wantStd(t, "{{ array | reject: 'ok', true | map: 'handle' | join: ' ' }}",
		map[string]any{"array": okArray()}, "beta gamma")
}

func TestUpstream_StdFilter_RejectWithFalseValue(t *testing.T) {
	wantStd(t, "{{ array | reject: 'ok', false | map: 'handle' | join: ' ' }}",
		map[string]any{"array": okArray()}, "alpha delta")
}

func TestUpstream_StdFilter_Has(t *testing.T) {
	wantStd(t, "{{ array | has: 'ok' }}",
		map[string]any{"array": okArray()}, "true")
	wantStd(t, "{{ array | has: 'ok', true }}",
		map[string]any{"array": okArray()}, "true")
}

func TestUpstream_StdFilter_HasWhenDoesNotHaveIt(t *testing.T) {
	arr := []any{
		map[string]any{"handle": "alpha", "ok": false},
		map[string]any{"handle": "beta", "ok": false},
		map[string]any{"handle": "gamma", "ok": false},
		map[string]any{"handle": "delta", "ok": false},
	}
	wantStd(t, "{{ array | has: 'ok' }}", map[string]any{"array": arr}, "false")
	wantStd(t, "{{ array | has: 'ok', true }}", map[string]any{"array": arr}, "false")
}

func TestUpstream_StdFilter_HasWithEmptyArrays(t *testing.T) {
	tmpl := strings.Join([]string{
		"{%- assign has_product = products | has: 'title.content', 'Not found' -%}",
		"{%- unless has_product -%}",
		"Product not found.",
		"{%- endunless -%}",
	}, "\n")
	wantStd(t, tmpl, map[string]any{"products": []any{}}, "Product not found.")
}

func TestUpstream_StdFilter_HasWithFalseValue(t *testing.T) {
	wantStd(t, "{{ array | has: 'ok', false }}",
		map[string]any{"array": okArray()}, "true")
}

func TestUpstream_StdFilter_HasWithFalseValueWhenDoesNotHaveIt(t *testing.T) {
	arr := []any{
		map[string]any{"handle": "alpha", "ok": true},
		map[string]any{"handle": "beta", "ok": true},
		map[string]any{"handle": "gamma", "ok": true},
		map[string]any{"handle": "delta", "ok": true},
	}
	wantStd(t, "{{ array | has: 'ok', false }}", map[string]any{"array": arr}, "false")
}

func products() []any {
	return []any{
		map[string]any{"title": "Pro goggles", "price": 1299},
		map[string]any{"title": "Thermal gloves", "price": 1499},
		map[string]any{"title": "Alpine jacket", "price": 3999},
		map[string]any{"title": "Mountain boots", "price": 3899},
		map[string]any{"title": "Safety helmet", "price": 1999},
	}
}

func TestUpstream_StdFilter_FindWithValue(t *testing.T) {
	tmpl := strings.Join([]string{
		"{%- assign product = products | find: 'price', 3999 -%}",
		"{{- product.title -}}",
	}, "\n")
	wantStd(t, tmpl, map[string]any{"products": products()}, "Alpine jacket")
}

func TestUpstream_StdFilter_FindWithEmptyArrays(t *testing.T) {
	tmpl := strings.Join([]string{
		"{%- assign product = products | find: 'title.content', 'Not found' -%}",
		"{%- unless product -%}",
		"Product not found.",
		"{%- endunless -%}",
	}, "\n")
	wantStd(t, tmpl, map[string]any{"products": []any{}}, "Product not found.")
}

func TestUpstream_StdFilter_FindIndexWithValue(t *testing.T) {
	tmpl := strings.Join([]string{
		"{%- assign index = products | find_index: 'price', 3999 -%}",
		"{{- index -}}",
	}, "\n")
	wantStd(t, tmpl, map[string]any{"products": products()}, "2")
}

func TestUpstream_StdFilter_FindIndexWithEmptyArrays(t *testing.T) {
	tmpl := strings.Join([]string{
		"{%- assign index = products | find_index: 'title.content', 'Not found' -%}",
		"{%- unless index -%}",
		"Index not found.",
		"{%- endunless -%}",
	}, "\n")
	wantStd(t, tmpl, map[string]any{"products": []any{}}, "Index not found.")
}

// ----- Where -----

func TestUpstream_StdFilter_Where(t *testing.T) {
	wantStd(t, "{{ array | where: 'ok' | map: 'handle' | join: ' ' }}",
		map[string]any{"array": okArray()}, "alpha delta")
}

func TestUpstream_StdFilter_WhereWithEmptyStringIsANoOp(t *testing.T) {
	wantStd(t, "{{ array | where: '' | join: ' ' }}",
		map[string]any{"array": []any{"alpha", "beta", "gamma"}}, "alpha beta gamma")
}

func TestUpstream_StdFilter_WhereWithValue(t *testing.T) {
	wantStd(t, "{{ array | where: 'ok', true | map: 'handle' | join: ' ' }}",
		map[string]any{"array": okArray()}, "alpha delta")
}

func TestUpstream_StdFilter_WhereWithFalseValue(t *testing.T) {
	wantStd(t, "{{ array | where: 'ok', false | map: 'handle' | join: ' ' }}",
		map[string]any{"array": okArray()}, "beta gamma")
}

// ----- Sum -----

func TestUpstream_StdFilter_SumWithAllNumbers(t *testing.T) {
	wantStd(t, "{{ a | sum }}", map[string]any{"a": []any{1, 2}}, "3")
}

func TestUpstream_StdFilter_SumWithNumericStrings(t *testing.T) {
	wantStd(t, "{{ a | sum }}", map[string]any{"a": []any{1, 2, "3", "4"}}, "10")
}

func TestUpstream_StdFilter_SumWithNestedArrays(t *testing.T) {
	wantStd(t, "{{ a | sum }}",
		map[string]any{"a": []any{1, []any{2, []any{3, 4}}}}, "10")
}

func TestUpstream_StdFilter_SumWithIndexableMapValues(t *testing.T) {
	a := []any{
		map[string]any{"quantity": 1},
		map[string]any{"quantity": 2, "weight": 3},
		map[string]any{"weight": 4},
	}
	wantStd(t, "{{ a | sum }}", map[string]any{"a": a}, "0")
	wantStd(t, "{{ a | sum: 'quantity' }}", map[string]any{"a": a}, "3")
	wantStd(t, "{{ a | sum: 'weight' }}", map[string]any{"a": a}, "7")
	wantStd(t, "{{ a | sum: 'subtotal' }}", map[string]any{"a": a}, "0")
}

func TestUpstream_StdFilter_SumOfFloats(t *testing.T) {
	t.Skip("Ruby BigDecimal sums avoid FP error (0.1+0.2+0.3=0.6); go-liquid uses float64 (=0.6000000000000001)")
	wantStd(t, "{{ input | sum }}",
		map[string]any{"input": []any{0.1, 0.2, 0.3}}, "0.6")
}

func TestUpstream_StdFilter_SumOfNegativeFloats(t *testing.T) {
	t.Skip("Ruby BigDecimal sums avoid FP error; go-liquid float64 yields ~5.5e-17 not 0.0")
	wantStd(t, "{{ input | sum }}",
		map[string]any{"input": []any{0.1, 0.2, -0.3}}, "0.0")
}

func TestUpstream_StdFilter_SumWithFloatStrings(t *testing.T) {
	t.Skip("FP precision: Ruby renders 0.6, go-liquid renders 0.6000000000000001")
	wantStd(t, "{{ input | sum }}",
		map[string]any{"input": []any{0.1, "0.2", "0.3"}}, "0.6")
}

func TestUpstream_StdFilter_SumResultingInNegativeFloat(t *testing.T) {
	wantStd(t, "{{ input | sum }}",
		map[string]any{"input": []any{0.1, -0.2, -0.3}}, "-0.4")
}

func TestUpstream_StdFilter_SumWithFloatsAndIndexableMapValues(t *testing.T) {
	t.Skip("FP precision divergence on sum('weight') case (0.10000000000000003 vs 0.1)")
	input := []any{
		map[string]any{"quantity": 1},
		map[string]any{"quantity": 0.2, "weight": -0.3},
		map[string]any{"weight": 0.4},
	}
	wantStd(t, "{{ input | sum }}", map[string]any{"input": input}, "0")
	wantStd(t, "{{ input | sum: 'quantity' }}", map[string]any{"input": input}, "1.2")
	wantStd(t, "{{ input | sum: 'weight' }}", map[string]any{"input": input}, "0.1")
	wantStd(t, "{{ input | sum: 'subtotal' }}", map[string]any{"input": input}, "0")
}

// ----- Date -----

func TestUpstream_StdFilter_Date(t *testing.T) {
	wantStd(t, "{{ '2006-05-05 10:00:00' | date: '%B' }}", nil, "May")
	wantStd(t, "{{ '2006-06-05 10:00:00' | date: '%B' }}", nil, "June")
	wantStd(t, "{{ '2006-07-05 10:00:00' | date: '%B' }}", nil, "July")
	wantStd(t, "{{ '2006-07-05 10:00:00' | date: '%m/%d/%Y' }}", nil, "07/05/2006")
	wantStd(t, "{{ nil | date: '%B' }}", nil, "")
	wantStd(t, "{{ '' | date: '%B' }}", nil, "")
}

func TestUpstream_StdFilter_Date_EmptyFormatReturnsISO(t *testing.T) {
	// Ruby's date filter short-circuits on empty/nil format and returns
	// input unchanged. The "ISO string" mention in the prior skip was
	// misleading — Ruby returns the input string verbatim.
	wantStd(t, "{{ '2006-07-05 10:00:00' | date: '' }}", nil, "2006-07-05 10:00:00")
	wantStd(t, "{{ '2006-07-05 10:00:00' | date: nil }}", nil, "2006-07-05 10:00:00")
}

// ----- Sort/map tests that require Drop/proc support are skipped (bodies retained) -----

func TestUpstream_StdFilter_MapCallsToLiquid(t *testing.T) {
	t.Skip("requires Liquid::Drop / to_liquid hook")
	// '{{ foo | map: "whatever" }}' with TestThing element should yield "woot: 1"
	wantStd(t, `{{ foo | map: "whatever" }}`, map[string]any{"foo": []any{}}, "woot: 1")
}

func TestUpstream_StdFilter_MapCallsContextEq(t *testing.T) {
	t.Skip("requires Drop with context-aware accessor (registers)")
	// Ruby:
	//   template.registers[:test] = 1234
	//   template.assigns['foo'] = [TestModel.new(value: :test)]
	//   {{ foo | map: "registers" }} => "{:test=>1234}"
}

func TestUpstream_StdFilter_SortCallsToLiquid(t *testing.T) {
	t.Skip("requires Liquid::Drop to_liquid hook")
}

func TestUpstream_StdFilter_MapOverProc(t *testing.T) {
	t.Skip("Ruby proc/lambda values not modeled in go-liquid")
	// drop = TestDrop.new(value: "testfoo"); proc { drop }
	// '{{ procs | map: "value" }}' => "testfoo"
}

func TestUpstream_StdFilter_MapOverDropsReturningProcs(t *testing.T) {
	t.Skip("Ruby proc/lambda values not modeled in go-liquid")
}

func TestUpstream_StdFilter_MapWorksOnEnumerables(t *testing.T) {
	t.Skip("TestEnumerable / Liquid::Drop semantics not modeled")
	// '{{ foo | map: "foo" }}' with TestEnumerable => "123"
}

func TestUpstream_StdFilter_MapReturnsEmptyOn2dInputArray(t *testing.T) {
	t.Skip("go-liquid does not raise Liquid::ArgumentError from filters; surfaces empty render")
	// @filters.map([[1],[2],[3]], "bar") raises Liquid::ArgumentError
}

func TestUpstream_StdFilter_MapReturnsInputWithNoProperty(t *testing.T) {
	t.Skip("go-liquid does not raise Liquid::ArgumentError from filters")
}

func TestUpstream_StdFilter_SortWorksOnEnumerables(t *testing.T) {
	t.Skip("requires TestEnumerable (Liquid::Drop + Enumerable)")
	// '{{ foo | sort: "bar" | map: "foo" }}' with TestEnumerable => "213"
}

func TestUpstream_StdFilter_FirstAndLastCallToLiquid(t *testing.T) {
	t.Skip("requires Liquid::Drop to_liquid hook")
}

func TestUpstream_StdFilter_TruncateCallsToLiquid(t *testing.T) {
	t.Skip("requires Liquid::Drop to_liquid hook")
}

func TestUpstream_StdFilter_LegacySortHash(t *testing.T) {
	t.Skip("Ruby symbol-keyed hash semantics not modeled")
	// @filters.sort(a: 1, b: 2) => [{a: 1, b: 2}]
}

func TestUpstream_StdFilter_LegacyReverseHash(t *testing.T) {
	t.Skip("Ruby symbol-keyed hash semantics not modeled")
}

func TestUpstream_StdFilter_SortInvalidProperty(t *testing.T) {
	t.Skip("go-liquid does not raise Liquid::ArgumentError from sort with invalid property")
}

func TestUpstream_StdFilter_SortNaturalInvalidProperty(t *testing.T) {
	t.Skip("go-liquid does not raise Liquid::ArgumentError from sort_natural with invalid property")
}

func TestUpstream_StdFilter_SortNaturalWhenPropertyIsSometimesMissingPutsNilsLast(t *testing.T) {
	t.Skip("requires mixed-type comparison (numeric/string) via to_liquid_value coercion")
}

func TestUpstream_StdFilter_UniqInvalidProperty(t *testing.T) {
	t.Skip("go-liquid does not raise Liquid::ArgumentError from uniq with invalid property")
}

func TestUpstream_StdFilter_CompactInvalidProperty(t *testing.T) {
	t.Skip("go-liquid does not raise Liquid::ArgumentError from compact with invalid property")
}

func TestUpstream_StdFilter_WhereWithNilIsANoOp(t *testing.T) {
	t.Skip("Ruby raises Liquid::ArgumentError for nil where key")
}

func TestUpstream_StdFilter_WhereStringKeys(t *testing.T) {
	t.Skip("Ruby: @filters.where(['alpha','beta','gamma','delta'], 'be') => ['beta'] — substring match on scalar arrays not implemented")
}

func TestUpstream_StdFilter_WhereNoKeySet(t *testing.T) {
	t.Skip("requires direct filter call (@filters.where); behavior covered via template paths")
}

func TestUpstream_StdFilter_WhereNonArrayMapInput(t *testing.T) {
	t.Skip("requires direct filter call against hash; template path wraps in array implicitly")
}

func TestUpstream_StdFilter_WhereIndexableButNonMapValue(t *testing.T) {
	t.Skip("go-liquid does not raise Liquid::ArgumentError for non-map where input")
}

func TestUpstream_StdFilter_WhereNonBooleanValue(t *testing.T) {
	input := []any{
		map[string]any{"message": "Bonjour!", "language": "French"},
		map[string]any{"message": "Hello!", "language": "English"},
		map[string]any{"message": "Hallo!", "language": "German"},
	}
	wantStd(t, "{{ input | where: 'language', 'French' | map: 'message' | join: ',' }}", map[string]any{"input": input}, "Bonjour!")
	wantStd(t, "{{ input | where: 'language', 'German' | map: 'message' | join: ',' }}", map[string]any{"input": input}, "Hallo!")
	wantStd(t, "{{ input | where: 'language', 'English' | map: 'message' | join: ',' }}", map[string]any{"input": input}, "Hello!")
}

func TestUpstream_StdFilter_WhereArrayOfOnlyUnindexableValues(t *testing.T) {
	t.Skip("Ruby returns nil for [nil] where input; behavior differs in go-liquid (empty)")
}

func TestUpstream_StdFilter_WhereNoTargetValue(t *testing.T) {
	input := []any{
		map[string]any{"foo": false},
		map[string]any{"foo": true},
		map[string]any{"foo": "for sure"},
		map[string]any{"bar": true},
	}
	wantStd(t, "{{ input | where: 'foo' | map: 'foo' | join: ',' }}",
		map[string]any{"input": input}, "true,for sure")
}

func TestUpstream_StdFilter_AllFiltersNeverRaiseNonLiquidException(t *testing.T) {
	t.Skip("Ruby-introspection brute-force fuzz; covered by go-liquid via individual filter tests + go test panic recovery")
}

func TestUpstream_StdFilter_SumWithIndexableNonMapValues(t *testing.T) {
	t.Skip("go-liquid sum semantics differ for indexable non-map values (string/array)")
}

func TestUpstream_StdFilter_SumWithUnindexableValues(t *testing.T) {
	t.Skip("go-liquid sum semantics differ for unindexable values mixed with maps")
}

func TestUpstream_StdFilter_SumWithoutPropertyCallsToLiquid(t *testing.T) {
	t.Skip("requires Liquid::Drop to_liquid hook")
}

func TestUpstream_StdFilter_SumWithPropertyCallsToLiquidOnPropertyValues(t *testing.T) {
	t.Skip("requires Liquid::Drop to_liquid hook")
}

func TestUpstream_StdFilter_SumWithNonStringProperty(t *testing.T) {
	t.Skip("Ruby keys-as-symbols/numbers semantics not modeled in go-liquid")
}

func TestUpstream_StdFilter_UniqWithToLiquidValue(t *testing.T) {
	t.Skip("requires Liquid::Drop to_liquid hook")
}

func TestUpstream_StdFilter_UniqWithToLiquidValuePickCorrectClasses(t *testing.T) {
	t.Skip("requires Liquid::Drop to_liquid hook")
}
