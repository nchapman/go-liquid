package liquid

import (
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"
)

func TestMapLoaderMissing(t *testing.T) {
	tmpl := MustParse(`{% render "missing" %}`).WithLoader(MapLoader{})
	_, err := tmpl.Render(nil)
	if err == nil || !strings.Contains(err.Error(), "missing") {
		t.Fatalf("expected missing-partial error, got %v", err)
	}
}

func TestRenderWithoutLoader(t *testing.T) {
	tmpl := MustParse(`{% render "anything" %}`)
	_, err := tmpl.Render(nil)
	if err == nil || !strings.Contains(err.Error(), "loader") {
		t.Fatalf("expected no-loader error, got %v", err)
	}
}

func TestFileSystemLoaderEscape(t *testing.T) {
	fsys := fstest.MapFS{
		"a.liquid": &fstest.MapFile{Data: []byte("ok")},
	}
	l := &FileSystemLoader{FS: fsys, Ext: ".liquid"}
	if _, err := l.Load("a"); err != nil {
		t.Fatalf("base case: %v", err)
	}
	if _, err := l.Load("../etc/passwd"); err == nil {
		t.Fatal("expected escape rejection")
	}
	if _, err := l.Load("/abs"); err == nil {
		t.Fatal("expected absolute-path rejection")
	}
}

func TestFileSystemLoaderRender(t *testing.T) {
	fsys := fstest.MapFS{
		"greet.liquid": &fstest.MapFile{Data: []byte("hi {{ name }}")},
	}
	tmpl := MustParse(`{% render "greet", name: "Ada" %}`).WithLoader(
		&FileSystemLoader{FS: fsys, Ext: ".liquid"},
	)
	out, err := tmpl.Render(nil)
	if err != nil {
		t.Fatal(err)
	}
	if out != "hi Ada" {
		t.Fatalf("got %q", out)
	}
}

func TestPartialCacheReused(t *testing.T) {
	calls := 0
	loader := callCountLoader{
		f: func(name string) (string, error) {
			calls++
			return "x", nil
		},
	}
	tmpl := MustParse(`{% render "p" %}{% render "p" %}{% render "p" %}`).WithLoader(loader)
	if _, err := tmpl.Render(nil); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("expected 1 loader call (cached), got %d", calls)
	}
}

type callCountLoader struct {
	f func(string) (string, error)
}

func (c callCountLoader) Load(name string) (string, error) { return c.f(name) }

func TestWithAndAsUsableAsVariables(t *testing.T) {
	// Regression: with/as must not be reserved globally.
	out, err := Render(
		"{{ with }}-{{ as }}-{% assign with = 1 %}{% assign as = 2 %}{{ with | plus: as }}",
		map[string]any{"with": "W", "as": "A"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if out != "W-A-3" {
		t.Fatalf("got %q", out)
	}
}

func TestRecursivePartialReturnsError(t *testing.T) {
	tmpl := MustParse(`{% render "a" %}`).WithLoader(MapLoader{
		"a": `before {% render "a" %} after`,
	})
	_, err := tmpl.Render(nil)
	if err == nil || !strings.Contains(err.Error(), "depth") {
		t.Fatalf("expected depth-exceeded error, got %v", err)
	}
}

func TestMutualRecursionReturnsError(t *testing.T) {
	tmpl := MustParse(`{% render "a" %}`).WithLoader(MapLoader{
		"a": `{% render "b" %}`,
		"b": `{% render "a" %}`,
	})
	_, err := tmpl.Render(nil)
	if err == nil || !strings.Contains(err.Error(), "depth") {
		t.Fatalf("expected depth-exceeded error, got %v", err)
	}
}

func TestIncludeRecursionReturnsError(t *testing.T) {
	tmpl := MustParse(`{% include "a" %}`).WithLoader(MapLoader{
		"a": `{% include "a" %}`,
	})
	_, err := tmpl.Render(nil)
	if err == nil || !strings.Contains(err.Error(), "depth") {
		t.Fatalf("expected depth-exceeded error, got %v", err)
	}
}

func TestRenderWithThenNamedArgsLatestWins(t *testing.T) {
	// Named arg with same key as `with` alias should override `with`.
	out, err := MustParse(`{% render "p" with first as x, x: "named" %}`).WithLoader(MapLoader{
		"p": `{{ x }}`,
	}).Render(map[string]any{"first": "with"})
	if err != nil {
		t.Fatal(err)
	}
	if out != "named" {
		t.Fatalf("got %q", out)
	}
}

func TestRenderDefaultAliasIsBasename(t *testing.T) {
	// {% render "shared/card" with prod %} should expose `card`, not `shared/card`.
	out, err := MustParse(`{% render "shared/card" with prod %}`).WithLoader(MapLoader{
		"shared/card": `name={{ card.name }}`,
	}).Render(map[string]any{"prod": map[string]any{"name": "Shoe"}})
	if err != nil {
		t.Fatal(err)
	}
	if out != "name=Shoe" {
		t.Fatalf("got %q", out)
	}
}

func TestIncludeForExposesForloop(t *testing.T) {
	out, err := MustParse(`{% include "row" for items as item %}`).WithLoader(MapLoader{
		"row": `[{{ forloop.index }}/{{ forloop.length }}:{{ item }}]`,
	}).Render(map[string]any{"items": []any{"a", "b"}})
	if err != nil {
		t.Fatal(err)
	}
	if out != "[1/2:a][2/2:b]" {
		t.Fatalf("got %q", out)
	}
}

func TestForRangeCapped(t *testing.T) {
	_, err := Render(`{% for i in (1..n) %}x{% endfor %}`, map[string]any{"n": 5_000_000})
	if err == nil || !strings.Contains(err.Error(), "range size") {
		t.Fatalf("expected range-size error, got %v", err)
	}
}

func TestConcurrentRenderAndWithLoader(t *testing.T) {
	tmpl := MustParse(`{% render "p" %}`).WithLoader(MapLoader{"p": "A"})
	done := make(chan struct{})
	go func() {
		for range 200 {
			tmpl.WithLoader(MapLoader{"p": "A"})
		}
		close(done)
	}()
	for range 200 {
		if _, err := tmpl.Render(nil); err != nil {
			t.Fatalf("render: %v", err)
		}
	}
	<-done
}

func TestFileSystemLoaderErrorDoesNotLeakPath(t *testing.T) {
	tmpdir := t.TempDir()
	l := NewFileSystemLoader(tmpdir, ".liquid")
	_, err := l.Load("missing")
	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Contains(err.Error(), tmpdir) {
		t.Fatalf("error leaks tmp path %q: %v", tmpdir, err)
	}
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("expected errors.Is(err, fs.ErrNotExist), got %v", err)
	}
}

func TestFormerKeywordsUsableAsVariables(t *testing.T) {
	// Every word that used to be a globally-reserved token type should now be
	// a usable identifier in expression position. Tag names are still reserved
	// at the structural position (immediately after `{%`).
	cases := []string{
		"raw", "comment", "cycle", "render", "include", "capture",
		"if", "unless", "case", "when", "for", "break", "continue", "assign",
		"increment", "decrement", "elsif", "else", "endif", "endfor",
		"limit", "offset", "reversed", "with", "as", "in",
	}
	for _, w := range cases {
		t.Run(w, func(t *testing.T) {
			out, err := Render("{{ "+w+" }}", map[string]any{w: "X"})
			if err != nil {
				t.Fatalf("Render({{ %s }}): %v", w, err)
			}
			if out != "X" {
				t.Fatalf("{{ %s }} got %q, want X", w, out)
			}
		})
	}
}

func TestFormerKeywordsAsForLoopVariable(t *testing.T) {
	out, err := Render(`{% for if in items %}{{ if }}{% endfor %}`,
		map[string]any{"items": []any{1, 2, 3}})
	if err != nil {
		t.Fatal(err)
	}
	if out != "123" {
		t.Fatalf("got %q", out)
	}
}

func TestFormerKeywordsAsAssignTarget(t *testing.T) {
	out, err := Render(`{% assign comment = 5 %}{% assign cycle = 7 %}{{ comment | plus: cycle }}`, nil)
	if err != nil {
		t.Fatal(err)
	}
	if out != "12" {
		t.Fatalf("got %q", out)
	}
}

func TestExpressionKeywordsStillReserved(t *testing.T) {
	// true/false/nil/empty/blank/and/or/contains keep their meanings; trying
	// to assign to them or use them as variable values must not silently
	// shadow the literal.
	out, err := Render(`{% if true %}T{% endif %}-{% if false %}F{% else %}else{% endif %}`, nil)
	if err != nil {
		t.Fatal(err)
	}
	if out != "T-else" {
		t.Fatalf("got %q", out)
	}
}

func TestLiquidTruthiness(t *testing.T) {
	// Only nil and false are falsy. 0, "", [], {} are all truthy.
	cases := []struct {
		template string
		data     map[string]any
		want     string
	}{
		{`{% if x %}T{% else %}F{% endif %}`, map[string]any{"x": ""}, "T"},
		{`{% if x %}T{% else %}F{% endif %}`, map[string]any{"x": 0}, "T"},
		{`{% if x %}T{% else %}F{% endif %}`, map[string]any{"x": []any{}}, "T"},
		{`{% if x %}T{% else %}F{% endif %}`, map[string]any{"x": false}, "F"},
		{`{% if x %}T{% else %}F{% endif %}`, map[string]any{"x": nil}, "F"},
		{`{% if x %}T{% else %}F{% endif %}`, map[string]any{}, "F"}, // undefined → nil → falsy
		// `default` follows broader "blank-or-falsy" semantics
		{`{{ x | default: "fallback" }}`, map[string]any{"x": ""}, "fallback"},
		{`{{ x | default: "fallback" }}`, map[string]any{"x": 0}, "0"},
		{`{{ x | default: "fallback" }}`, map[string]any{"x": []any{}}, "fallback"},
	}
	for _, tc := range cases {
		got, err := Render(tc.template, tc.data)
		if err != nil {
			t.Errorf("%s: %v", tc.template, err)
			continue
		}
		if got != tc.want {
			t.Errorf("%s with %v: got %q, want %q", tc.template, tc.data, got, tc.want)
		}
	}
}

func TestForOverStringIsSingleItem(t *testing.T) {
	out, err := Render(`{% for x in s %}[{{ x }}]{% endfor %}`,
		map[string]any{"s": "hello"})
	if err != nil {
		t.Fatal(err)
	}
	if out != "[hello]" {
		t.Fatalf("got %q, want [hello]", out)
	}
}

func TestStringFiltersStillCharSplit(t *testing.T) {
	// first/size/join on a string should still treat it as characters.
	out, err := Render(`{{ "hello" | first }}-{{ "hello" | size }}`, nil)
	if err != nil {
		t.Fatal(err)
	}
	if out != "h-5" {
		t.Fatalf("got %q", out)
	}
}

func TestUniqDeepEquality(t *testing.T) {
	// Distinct objects with the same string form should NOT collapse.
	data := map[string]any{
		"items": []any{
			map[string]any{"id": 1, "name": "a"},
			map[string]any{"id": 2, "name": "b"},
			map[string]any{"id": 1, "name": "a"}, // exact dup of first
			map[string]any{"id": 3, "name": "c"},
		},
	}
	out, err := Render(`{{ items | uniq | size }}`, data)
	if err != nil {
		t.Fatal(err)
	}
	if out != "3" {
		t.Fatalf("got %q, want 3", out)
	}
}

func TestUniqByProperty(t *testing.T) {
	data := map[string]any{
		"items": []any{
			map[string]any{"id": 1, "name": "a"},
			map[string]any{"id": 2, "name": "b"},
			map[string]any{"id": 1, "name": "different"},
		},
	}
	out, err := Render(`{{ items | uniq: "id" | size }}`, data)
	if err != nil {
		t.Fatal(err)
	}
	if out != "2" {
		t.Fatalf("got %q, want 2", out)
	}
}

func TestForloopName(t *testing.T) {
	out, err := Render(`{% for item in products %}{{ forloop.name }}|{% endfor %}`,
		map[string]any{"products": []any{1, 2}})
	if err != nil {
		t.Fatal(err)
	}
	if out != "item-products|item-products|" {
		t.Fatalf("got %q", out)
	}
}

func TestForloopParentloop(t *testing.T) {
	out, err := Render(
		`{% for row in rows %}{% for cell in row %}{{ forloop.parentloop.index }}.{{ forloop.index }} {% endfor %}{% endfor %}`,
		map[string]any{"rows": []any{
			[]any{"a", "b"},
			[]any{"c", "d", "e"},
		}})
	if err != nil {
		t.Fatal(err)
	}
	if out != "1.1 1.2 2.1 2.2 2.3 " {
		t.Fatalf("got %q", out)
	}
}

func TestForloopParentloopNilAtTopLevel(t *testing.T) {
	out, err := Render(`{% for x in xs %}{{ forloop.parentloop | default: "none" }}{% endfor %}`,
		map[string]any{"xs": []any{1}})
	if err != nil {
		t.Fatal(err)
	}
	if out != "none" {
		t.Fatalf("got %q", out)
	}
}

func TestWhereDefaultsToTruthy(t *testing.T) {
	// When target_value is omitted, items whose property is any truthy value
	// (not nil, not false) should match — including string "yes", number 1, etc.
	data := map[string]any{
		"items": []any{
			map[string]any{"flag": true, "name": "a"},
			map[string]any{"flag": false, "name": "b"},
			map[string]any{"flag": "yes", "name": "c"},
			map[string]any{"flag": nil, "name": "d"},
			map[string]any{"flag": 1, "name": "e"},
		},
	}
	out, err := Render(`{{ items | where: "flag" | map: "name" | join: "," }}`, data)
	if err != nil {
		t.Fatal(err)
	}
	if out != "a,c,e" {
		t.Fatalf("where default: got %q, want a,c,e", out)
	}
}

func TestRejectDefaultsToTruthy(t *testing.T) {
	data := map[string]any{
		"items": []any{
			map[string]any{"flag": true, "name": "a"},
			map[string]any{"flag": false, "name": "b"},
			map[string]any{"flag": nil, "name": "c"},
		},
	}
	out, err := Render(`{{ items | reject: "flag" | map: "name" | join: "," }}`, data)
	if err != nil {
		t.Fatal(err)
	}
	if out != "b,c" {
		t.Fatalf("reject default: got %q, want b,c", out)
	}
}

func TestFindIndexDefaultsToTruthy(t *testing.T) {
	data := map[string]any{
		"items": []any{
			map[string]any{"flag": false},
			map[string]any{"flag": "x"}, // truthy
			map[string]any{"flag": true},
		},
	}
	out, err := Render(`{{ items | find_index: "flag" }}`, data)
	if err != nil {
		t.Fatal(err)
	}
	if out != "1" {
		t.Fatalf("find_index default: got %q, want 1", out)
	}
}

func TestHasDefaultsToTruthy(t *testing.T) {
	data := map[string]any{
		"items": []any{
			map[string]any{"flag": false},
			map[string]any{"flag": "yes"},
		},
	}
	out, err := Render(`{{ items | has: "flag" }}`, data)
	if err != nil {
		t.Fatal(err)
	}
	if out != "true" {
		t.Fatalf("has default: got %q, want true", out)
	}
}

func TestLogicalOperatorPrecedence(t *testing.T) {
	// Liquid's and/or are evaluated right-to-left as a single chain (no
	// precedence between them). This matches Shopify; classical C-style
	// `and-binds-tighter-than-or` would give different answers for cases
	// like "F and T or T".
	cases := []struct {
		template string
		want     string
	}{
		// `F and T or T` — Shopify: F (short-circuit on F at the start of the chain)
		{`{% if false and true or true %}T{% else %}F{% endif %}`, "F"},
		// `T or F and F` — Shopify: T (short-circuit on T)
		{`{% if true or false and false %}T{% else %}F{% endif %}`, "T"},
		// `T and F or T` — Shopify: a=T not falsy → continue. b=F not truthy (or-rel)?
		// Actually: T relation=and not falsy → continue. F relation=or truthy(F)=false → continue. T return T → truthy.
		{`{% if true and false or true %}T{% else %}F{% endif %}`, "T"},
		// `F or T and F` — Shopify: F (or-rel, not truthy → continue). T (and-rel, truthy → continue). F → return F → falsy.
		{`{% if false or true and false %}T{% else %}F{% endif %}`, "F"},
		// Longer chain: `F or T or F` → eventually return F (last), but T short-circuits or → T
		{`{% if false or true or false %}T{% else %}F{% endif %}`, "T"},
		// All-and: `T and T and F` → F (last)
		{`{% if true and true and false %}T{% else %}F{% endif %}`, "F"},
	}
	for _, tc := range cases {
		got, err := Render(tc.template, nil)
		if err != nil {
			t.Errorf("%s: %v", tc.template, err)
			continue
		}
		if got != tc.want {
			t.Errorf("%s: got %q, want %q", tc.template, got, tc.want)
		}
	}
}

func TestDivisionByZeroErrors(t *testing.T) {
	cases := []string{
		`{{ 10 | divided_by: 0 }}`,
		`{{ 10 | modulo: 0 }}`,
	}
	for _, tmpl := range cases {
		_, err := Render(tmpl, nil)
		if err == nil {
			t.Errorf("%s: expected error, got nil", tmpl)
			continue
		}
		if !strings.Contains(err.Error(), "zero") {
			t.Errorf("%s: error %q does not mention zero", tmpl, err)
		}
	}
}

func TestBase64DecodeInvalidErrors(t *testing.T) {
	_, err := Render(`{{ "not!valid!base64" | base64_decode }}`, nil)
	if err == nil {
		t.Fatal("expected error for invalid base64 input")
	}
	if !strings.Contains(err.Error(), "base64") {
		t.Fatalf("error does not mention base64: %v", err)
	}
}

func TestSliceReturnsEmptyArrayNotNil(t *testing.T) {
	// slice on an empty/over-shot range should return [], not nil, so that
	// chaining with size or default works as expected.
	out, err := Render(`{{ items | slice: 10, 5 | size }}-{{ items | slice: 10, 5 | default: "fallback" }}`,
		map[string]any{"items": []any{1, 2, 3}})
	if err != nil {
		t.Fatal(err)
	}
	if out != "0-fallback" {
		t.Fatalf("got %q", out)
	}
}

func TestConcatNonArrayErrors(t *testing.T) {
	_, err := Render(`{{ items | concat: "not an array" }}`,
		map[string]any{"items": []any{1, 2}})
	if err == nil {
		t.Fatal("expected error for concat with non-array")
	}
	if !strings.Contains(err.Error(), "concat") {
		t.Fatalf("error does not mention concat: %v", err)
	}
}

func TestTruncateShortLength(t *testing.T) {
	// Matches Shopify: keep = max(0, length - len(ellipsis)) then append the
	// full ellipsis. Output can exceed `length` when length is smaller than
	// the ellipsis itself — that's the canonical behavior.
	cases := []struct {
		in, want string
	}{
		{`{{ "hello world" | truncate: 5 }}`, "he..."},
		{`{{ "hello world" | truncate: 3 }}`, "..."},
		{`{{ "hello world" | truncate: 2 }}`, "..."},
		{`{{ "hello world" | truncate: 0 }}`, "..."},
	}
	for _, tc := range cases {
		got, err := Render(tc.in, nil)
		if err != nil {
			t.Errorf("%s: %v", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("%s: got %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestStripIsAsciiOnly(t *testing.T) {
	// Ruby's String#strip and Shopify's `strip` filter remove only ASCII
	// whitespace plus null. Unicode whitespace such as U+00A0 (NBSP) is
	// preserved so go-liquid matches Shopify byte-for-byte. We build the
	// input through data binding so the NBSP travels intact.
	const nbsp = " "
	in := " \thi\t " + nbsp + "x" + nbsp
	out, err := Render(`{{ s | strip }}`, map[string]any{"s": in})
	if err != nil {
		t.Fatal(err)
	}
	want := "hi\t " + nbsp + "x" + nbsp
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestRenderForLoopNameIsTemplateName(t *testing.T) {
	out, err := MustParse(`{% render "card" for items as item %}`).WithLoader(MapLoader{
		"card": `{{ forloop.name }}|`,
	}).Render(map[string]any{"items": []any{1, 2, 3}})
	if err != nil {
		t.Fatal(err)
	}
	if out != "card|card|card|" {
		t.Fatalf("got %q", out)
	}
}

func TestAndOrShortCircuit(t *testing.T) {
	// `and` should not evaluate the right side when the left is falsy;
	// `or` should not evaluate the right side when the left is truthy.
	// {% cycle %} mutates state visibly between renders, so we use it to
	// detect whether the right side was evaluated.
	cases := []struct {
		template string
		want     string
	}{
		// false and (cycle) — cycle must NOT advance
		{`{% if false and 1 %}{% endif %}{% cycle "a","b" %}-{% cycle "a","b" %}`, "a-b"},
		// true or (cycle) — cycle must NOT advance
		{`{% if true or 1 %}{% endif %}{% cycle "x","y" %}-{% cycle "x","y" %}`, "x-y"},
		// true and (cycle) — cycle WILL advance, baseline check
		{`{% if true and 1 %}{% endif %}{% cycle "p","q" %}-{% cycle "p","q" %}`, "p-q"},
	}
	for _, tc := range cases {
		got, err := Render(tc.template, nil)
		if err != nil {
			t.Errorf("%s: %v", tc.template, err)
			continue
		}
		if got != tc.want {
			t.Errorf("%s: got %q, want %q", tc.template, got, tc.want)
		}
	}
}

func TestInlineCommentBasic(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{`A{% # comment %}B`, "AB"},
		{`A{%- # trim -%}B`, "AB"}, // trim markers also work
		// Apostrophes in the comment body must NOT start a string scan.
		{`A{% # don't worry, it's fine %}B`, "AB"},
		// `%}` inside a single-quoted Liquid string would terminate; comment
		// body has no string semantics so it just outputs nothing.
		{`{% # quoted "x" 'y' done %}ok`, "ok"},
	}
	for _, tc := range cases {
		got, err := Render(tc.in, nil)
		if err != nil {
			t.Errorf("%s: %v", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("%s: got %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestLiquidBlockBasic(t *testing.T) {
	out, err := Render(
		"{% liquid\n  assign x = 5\n  assign y = x | times: 2\n%}{{ x }}+{{ y }}",
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if out != "5+10" {
		t.Fatalf("got %q", out)
	}
}

func TestLiquidBlockWithControlFlow(t *testing.T) {
	out, err := Render(
		`{% liquid
  assign greeting = "hi"
  if name
    assign greeting = greeting | append: ", "
    assign greeting = greeting | append: name
  endif
%}{{ greeting }}`,
		map[string]any{"name": "Ada"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if out != "hi, Ada" {
		t.Fatalf("got %q", out)
	}
}

func TestLiquidBlockPreservesQuotedPercent(t *testing.T) {
	// `%}` inside a string literal must not terminate the block early.
	out, err := Render(
		`{% liquid
  assign s = "a %} b"
%}{{ s }}`,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if out != "a %} b" {
		t.Fatalf("got %q", out)
	}
}

func TestNamedFilterArgs(t *testing.T) {
	cases := []struct {
		template string
		data     map[string]any
		want     string
	}{
		// Positional usage still works (default lives in kwargFilters but
		// kwargs may be empty).
		{`{{ x | default: "fallback" }}`, map[string]any{"x": ""}, "fallback"},
		// allow_false:true preserves a literal false; default no longer fires.
		{`{{ x | default: "fallback", allow_false: true }}`, map[string]any{"x": false}, "false"},
		// allow_false:false (the default) treats false as missing.
		{`{{ x | default: "fallback" }}`, map[string]any{"x": false}, "fallback"},
		// Empty array still falls back even with allow_false.
		{`{{ x | default: "fallback", allow_false: true }}`, map[string]any{"x": []any{}}, "fallback"},
	}
	for _, tc := range cases {
		got, err := Render(tc.template, tc.data)
		if err != nil {
			t.Errorf("%s: %v", tc.template, err)
			continue
		}
		if got != tc.want {
			t.Errorf("%s with %v: got %q, want %q", tc.template, tc.data, got, tc.want)
		}
	}
}

func TestPositionalCannotFollowNamed(t *testing.T) {
	// `key: value, positional` is a parse error.
	_, err := Parse(`{{ x | default: allow_false: true, "fallback" }}`)
	if err == nil {
		t.Fatal("expected parse error for positional arg after named arg")
	}
	if !strings.Contains(err.Error(), "named") {
		t.Fatalf("error doesn't mention named arg: %v", err)
	}
}

func TestUnknownKwargsIgnoredOnPlainFilter(t *testing.T) {
	// upcase doesn't accept kwargs; unknown kwargs are silently dropped
	// (matching Shopify's "extra hash arg is no-op" behavior).
	out, err := Render(`{{ "hi" | upcase: extra: 1 }}`, nil)
	if err != nil {
		t.Fatal(err)
	}
	if out != "HI" {
		t.Fatalf("got %q", out)
	}
}

func TestStrictVariablesErrorsOnUndefined(t *testing.T) {
	tmpl := MustParse("Hello, {{ name }}!\n{{ missing }}")
	_, err := tmpl.Render(map[string]any{"name": "Ada"}, StrictVariables())
	if err == nil {
		t.Fatal("expected error for undefined variable in strict mode")
	}
	var re *RenderError
	if !errors.As(err, &re) {
		t.Fatalf("expected *RenderError, got %T: %v", err, err)
	}
	if re.Line != 2 {
		t.Errorf("expected line 2, got %d", re.Line)
	}
	if !strings.Contains(re.Inner.Error(), "missing") {
		t.Errorf("error doesn't mention missing variable: %v", re.Inner)
	}
}

func TestStrictVariablesAllowsExplicitNil(t *testing.T) {
	// A variable explicitly set to nil is still "defined" — strict mode
	// distinguishes undefined from explicitly-nil.
	out, err := Render(`{{ x }}!`, map[string]any{"x": nil}, StrictVariables())
	if err != nil {
		t.Fatal(err)
	}
	if out != "!" {
		t.Fatalf("got %q", out)
	}
}

func TestStrictFiltersErrorsOnUnknown(t *testing.T) {
	_, err := Render(`{{ "x" | nonsense }}`, nil, StrictFilters())
	if err == nil {
		t.Fatal("expected error for unknown filter in strict mode")
	}
	var re *RenderError
	if !errors.As(err, &re) {
		t.Fatalf("expected *RenderError, got %T: %v", err, err)
	}
	if !strings.Contains(re.Inner.Error(), "nonsense") {
		t.Errorf("error doesn't mention filter name: %v", re.Inner)
	}
}

func TestRenderErrorCarriesPositionForFilterFailure(t *testing.T) {
	// divided_by 0 raises via filterError; the wrapper at FilterExpr's
	// position attaches line/col so callers can pinpoint the offending
	// filter even on a multi-line template.
	_, err := Render("ok\nthen {{ 10 | divided_by: 0 }} done\n", nil)
	var re *RenderError
	if !errors.As(err, &re) {
		t.Fatalf("expected *RenderError, got %T: %v", err, err)
	}
	if re.Line != 2 {
		t.Errorf("expected line 2, got %d (err=%v)", re.Line, err)
	}
}

func TestLaxModeStillSilentOnUndefined(t *testing.T) {
	out, err := Render(`Hello, {{ missing | default: "stranger" }}`, nil)
	if err != nil {
		t.Fatal(err)
	}
	if out != "Hello, stranger" {
		t.Fatalf("got %q", out)
	}
}

func TestEchoTag(t *testing.T) {
	cases := []struct {
		template string
		data     map[string]any
		want     string
	}{
		{`{% echo "hi" %}`, nil, "hi"},
		{`{% echo name | upcase %}`, map[string]any{"name": "ada"}, "ADA"},
		{`{% liquid
  assign x = 5
  echo x | times: 3
%}`, nil, "15"},
		{`{% liquid
  for i in (1..3)
    echo i
    echo "-"
  endfor
%}`, nil, "1-2-3-"},
	}
	for _, tc := range cases {
		got, err := Render(tc.template, tc.data)
		if err != nil {
			t.Errorf("%s: %v", tc.template, err)
			continue
		}
		if got != tc.want {
			t.Errorf("%s: got %q, want %q", tc.template, got, tc.want)
		}
	}
}

func TestLiquidBlockRejectsBlockTags(t *testing.T) {
	cases := []string{
		"{% liquid\n  raw\n  hello\n  endraw\n%}",
		"{% liquid\n  comment\n  hello\n  endcomment\n%}",
	}
	for _, src := range cases {
		_, err := Parse(src)
		if err == nil {
			t.Errorf("expected parse error for: %q", src)
			continue
		}
		if !strings.Contains(err.Error(), "not allowed inside") {
			t.Errorf("error doesn't mention disallowed: %v", err)
		}
	}
}

func TestLiquidBlockBareLiquidLineIsNoOp(t *testing.T) {
	// A bare `liquid` line inside a {% liquid %} body is a Ruby-parity
	// no-op: after stripping the leading keyword the line is empty, so it
	// must be silently dropped (not raised, not rendered).
	got, err := Render("{% liquid\n  liquid\n%}", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestStrictVariablesProperty(t *testing.T) {
	tmpl := MustParse("{{ user.name }}\n{{ user.missing }}")
	_, err := tmpl.Render(map[string]any{
		"user": map[string]any{"name": "Ada"},
	}, StrictVariables())
	if err == nil {
		t.Fatal("expected error for undefined property in strict mode")
	}
	var re *RenderError
	if !errors.As(err, &re) {
		t.Fatalf("expected *RenderError, got %T: %v", err, err)
	}
	if re.Line != 2 {
		t.Errorf("expected line 2, got %d", re.Line)
	}
	if !strings.Contains(re.Inner.Error(), "missing") {
		t.Errorf("error doesn't mention missing property: %v", re.Inner)
	}
}

func TestStrictVariablesIndex(t *testing.T) {
	tmpl := MustParse(`{{ items[10] }}`)
	_, err := tmpl.Render(map[string]any{"items": []any{1, 2, 3}}, StrictVariables())
	if err == nil {
		t.Fatal("expected error for out-of-range index in strict mode")
	}
	var re *RenderError
	if !errors.As(err, &re) || !strings.Contains(re.Inner.Error(), "index") {
		t.Fatalf("expected RenderError mentioning index: %v", err)
	}
}

func TestStrictVariablesAllowsMapWithExplicitNil(t *testing.T) {
	// Property explicitly set to nil is "found" — strict mode shouldn't fire.
	out, err := Render(`[{{ user.name }}]`,
		map[string]any{"user": map[string]any{"name": nil}}, StrictVariables())
	if err != nil {
		t.Fatal(err)
	}
	if out != "[]" {
		t.Fatalf("got %q", out)
	}
}

func TestUnclosedLiquidBlockErrors(t *testing.T) {
	_, err := Parse("{% liquid\n  assign x = 1\n  echo x")
	if err == nil {
		t.Fatal("expected parse error for unclosed {% liquid %} block")
	}
	if !strings.Contains(err.Error(), "unterminated") {
		t.Fatalf("error doesn't mention unterminated: %v", err)
	}
}

func TestUnclosedInlineCommentErrors(t *testing.T) {
	_, err := Parse(`{% # never closes`)
	if err == nil {
		t.Fatal("expected parse error for unclosed inline comment")
	}
	if !strings.Contains(err.Error(), "unterminated") {
		t.Fatalf("error doesn't mention unterminated: %v", err)
	}
}

func TestParseErrorEscapesNewlinesInTokenLiteral(t *testing.T) {
	// The string literal includes a newline (\n). The error message must
	// quote the literal (%q) so log lines aren't injected by the offending
	// template.
	_, err := Parse(`{{ "fake\nlevel=INFO" badnesss `) // missing close
	if err == nil {
		t.Fatal("expected parse error")
	}
	if strings.Contains(err.Error(), "\nlevel=INFO") {
		t.Fatalf("error contains raw newline (log injection): %q", err.Error())
	}
}

// hardenable types for method-dispatch tests
type personMethods struct{ first, last string }

func (p personMethods) FullName() string { return p.first + " " + p.last }
func (p personMethods) Initials() (string, error) {
	return string(p.first[0]) + string(p.last[0]), nil
}
func (p *personMethods) Close() error     { panic("Close should not be invoked from a template") }
func (p personMethods) Save() error       { panic("Save should not be invoked from a template") }
func (p personMethods) Reset()            { panic("Reset should not be invoked from a template") }
func (p personMethods) Stats() (int, int) { return 1, 2 } // multi-value, not (T, error)

func TestMethodDispatchAllowsDataAccessors(t *testing.T) {
	p := personMethods{first: "Ada", last: "Lovelace"}
	out, err := Render(`{{ p.FullName }}|{{ p.Initials }}`, map[string]any{"p": p})
	if err != nil {
		t.Fatal(err)
	}
	if out != "Ada Lovelace|AL" {
		t.Fatalf("got %q", out)
	}
}

func TestMethodDispatchSkipsSideEffecting(t *testing.T) {
	p := &personMethods{first: "Ada", last: "Lovelace"}
	// Close, Save, Reset, Stats must NOT be invoked. Templates resolve them
	// to nil (or empty string when output) without panicking.
	cases := []string{
		`{{ p.Close }}`, `{{ p.Save }}`, `{{ p.Reset }}`, `{{ p.Stats }}`,
	}
	for _, src := range cases {
		out, err := Render(src, map[string]any{"p": p})
		if err != nil {
			t.Errorf("%s: unexpected error: %v", src, err)
		}
		if out != "" {
			t.Errorf("%s: expected empty output, got %q", src, out)
		}
	}
}

// dropImpl shows the opt-in route for full property control.
type dropImpl struct{ data map[string]any }

func (d dropImpl) LiquidLookup(k string) (any, bool) {
	v, ok := d.data[k]
	return v, ok
}

func TestDropInterfaceTakesPrecedenceOverFields(t *testing.T) {
	// A Drop wraps any backing data and templates only see what LiquidLookup
	// returns, not raw struct methods/fields.
	d := dropImpl{data: map[string]any{"shown": "hello"}}
	out, err := Render(`{{ d.shown }}|{{ d.X | default: "absent" }}`,
		map[string]any{"d": d})
	if err != nil {
		t.Fatal(err)
	}
	if out != "hello|absent" {
		t.Fatalf("got %q", out)
	}
}

func TestDropStrictVariablesUsesLookupOK(t *testing.T) {
	d := dropImpl{data: map[string]any{"a": 1}}
	_, err := Render(`{{ d.b }}`, map[string]any{"d": d}, StrictVariables())
	if err == nil {
		t.Fatal("expected error in strict mode for absent Drop key")
	}
	if !errors.Is(err, ErrUndefinedVariable) {
		t.Fatalf("expected ErrUndefinedVariable, got %v", err)
	}
}

func TestRegisterKwargFilter(t *testing.T) {
	RegisterKwargFilter("greet", func(input any, args []any, kwargs map[string]any) any {
		greeting, _ := kwargs["greeting"].(string)
		if greeting == "" {
			greeting = "Hello"
		}
		return greeting + ", " + toStringForTest(input) + "!"
	})
	defer delete(Default().filters, "greet")

	out, err := Render(`{{ "Ada" | greet }} {{ "Bob" | greet: greeting: "Hi" }}`, nil)
	if err != nil {
		t.Fatal(err)
	}
	if out != "Hello, Ada! Hi, Bob!" {
		t.Fatalf("got %q", out)
	}
}

// toStringForTest mirrors the package's internal toString — exposed via
// fmt to avoid coupling the test to private helpers.
func toStringForTest(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func TestTablerowBasic(t *testing.T) {
	out, err := Render(
		`{% tablerow x in items cols: 2 %}{{ x }}{% endtablerow %}`,
		map[string]any{"items": []any{"a", "b", "c", "d", "e"}},
	)
	if err != nil {
		t.Fatal(err)
	}
	want := "<tr class=\"row1\">\n<td class=\"col1\">a</td><td class=\"col2\">b</td></tr>\n<tr class=\"row2\"><td class=\"col1\">c</td><td class=\"col2\">d</td></tr>\n<tr class=\"row3\"><td class=\"col1\">e</td></tr>\n"
	if out != want {
		t.Fatalf("\nwant: %q\ngot:  %q", want, out)
	}
}

func TestTablerowLoopFields(t *testing.T) {
	out, err := Render(
		`{% tablerow x in items cols: 2 %}{{ tablerowloop.row }}.{{ tablerowloop.col }}/{{ tablerowloop.index }}{% endtablerow %}`,
		map[string]any{"items": []any{"a", "b", "c"}},
	)
	if err != nil {
		t.Fatal(err)
	}
	// row1 col1=a → 1.1/1; col2=b → 1.2/2; new row; row2 col1=c → 2.1/3
	if !strings.Contains(out, "1.1/1") || !strings.Contains(out, "1.2/2") || !strings.Contains(out, "2.1/3") {
		t.Fatalf("missing tablerowloop values in: %q", out)
	}
}

func TestTablerowLimitOffset(t *testing.T) {
	out, err := Render(
		`{% tablerow x in items cols: 2 limit: 3 offset: 1 %}{{ x }}{% endtablerow %}`,
		map[string]any{"items": []any{"a", "b", "c", "d", "e"}},
	)
	if err != nil {
		t.Fatal(err)
	}
	// offset 1 -> [b,c,d,e]; limit 3 -> [b,c,d]; cols 2 -> rows: [b,c] [d]
	if !strings.Contains(out, ">b<") || !strings.Contains(out, ">c<") || !strings.Contains(out, ">d<") {
		t.Fatalf("expected b/c/d, got %q", out)
	}
	if strings.Contains(out, ">a<") || strings.Contains(out, ">e<") {
		t.Fatalf("offset/limit not applied: %q", out)
	}
}

func TestIfchangedSuppressesRepeats(t *testing.T) {
	data := map[string]any{
		"items": []any{
			map[string]any{"category": "A", "name": "a1"},
			map[string]any{"category": "A", "name": "a2"},
			map[string]any{"category": "B", "name": "b1"},
			map[string]any{"category": "B", "name": "b2"},
			map[string]any{"category": "A", "name": "a3"}, // back to A
		},
	}
	out, err := Render(
		`{% for it in items %}{% ifchanged %}[{{ it.category }}]{% endifchanged %}{{ it.name }} {% endfor %}`,
		data,
	)
	if err != nil {
		t.Fatal(err)
	}
	want := "[A]a1 a2 [B]b1 b2 [A]a3 "
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestDocTagDiscardsBody(t *testing.T) {
	out, err := Render(`before {% doc %}@param x A widget{% enddoc %}after`, nil)
	if err != nil {
		t.Fatal(err)
	}
	if out != "before after" {
		t.Fatalf("got %q", out)
	}
}

func TestDocTagDoesNotInterpretBody(t *testing.T) {
	// Liquid syntax inside {% doc %} is not parsed/executed.
	out, err := Render(`{% doc %}{{ undefined.thing | bogus }}{% enddoc %}ok`, nil)
	if err != nil {
		t.Fatal(err)
	}
	if out != "ok" {
		t.Fatalf("got %q", out)
	}
}

func TestUnclosedDocErrors(t *testing.T) {
	_, err := Parse(`{% doc %}forever`)
	if err == nil || !strings.Contains(err.Error(), "unterminated") {
		t.Fatalf("expected unterminated error, got %v", err)
	}
}

func TestRenderErrorCarriesTemplateName(t *testing.T) {
	tmpl := MustParse(`{{ x | divided_by: 0 }}`).WithName("alerts.liquid")
	_, err := tmpl.Render(map[string]any{"x": 10})
	var re *RenderError
	if !errors.As(err, &re) {
		t.Fatalf("expected RenderError, got %T: %v", err, err)
	}
	if re.TemplateName != "alerts.liquid" {
		t.Errorf("expected TemplateName=alerts.liquid, got %q", re.TemplateName)
	}
	if !strings.Contains(re.Error(), "alerts.liquid") {
		t.Errorf("Error() missing template name: %q", re.Error())
	}
}

func TestPartialErrorCarriesPartialName(t *testing.T) {
	tmpl := MustParse(`{% render "broken" %}`).WithLoader(MapLoader{
		"broken": `oh no {{ 10 | divided_by: 0 }}`,
	})
	_, err := tmpl.Render(nil)
	var re *RenderError
	if !errors.As(err, &re) {
		t.Fatalf("expected RenderError, got %T: %v", err, err)
	}
	if re.TemplateName != "broken" {
		t.Errorf("expected TemplateName=broken, got %q", re.TemplateName)
	}
}

func TestParseErrorCarriesPartialNameOnLoaderFailure(t *testing.T) {
	tmpl := MustParse(`{% render "bad" %}`).WithLoader(MapLoader{
		"bad": `{% if missing_endif`,
	})
	_, err := tmpl.Render(nil)
	var pe *ParseError
	if !errors.As(err, &pe) {
		t.Fatalf("expected ParseError, got %T: %v", err, err)
	}
	if pe.TemplateName != "bad" {
		t.Errorf("expected TemplateName=bad, got %q", pe.TemplateName)
	}
}

func TestIfchangedSharesRegisterAcrossBlocks(t *testing.T) {
	// Two distinct {% ifchanged %} blocks share state per Shopify. If A
	// emits "x" and B's body also evaluates to "x", B is suppressed.
	out, err := Render(
		`{% for i in items %}{% ifchanged %}A:{{ i }}{% endifchanged %}{% ifchanged %}B:{{ i }}{% endifchanged %} {% endfor %}`,
		map[string]any{"items": []any{1, 2, 2}},
	)
	if err != nil {
		t.Fatal(err)
	}
	// i=1: A emits "A:1", saved. B evaluates "B:1", different → emits "B:1", saved.
	// i=2: A evaluates "A:2", different → emits. B evaluates "B:2", different → emits.
	// i=2: A evaluates "A:2", same as last (B:2 → A:2 different actually)
	// Trace carefully:
	// last="" → A:"A:1" → emit, last="A:1"
	//   → B:"B:1" → diff, emit, last="B:1"  → " "
	// → A:"A:2" → diff, emit, last="A:2"
	//   → B:"B:2" → diff, emit, last="B:2" → " "
	// → A:"A:2" → diff (last="B:2"), emit, last="A:2"
	//   → B:"B:2" → diff (last="A:2"), emit, last="B:2" → " "
	want := "A:1B:1 A:2B:2 A:2B:2 "
	if out != want {
		t.Fatalf("got %q\nwant %q", out, want)
	}
}

func TestNestedComments(t *testing.T) {
	out, err := Render(
		`A{% comment %}outer {% comment %}inner{% endcomment %} still in outer{% endcomment %}B`,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if out != "AB" {
		t.Fatalf("got %q", out)
	}
}

func TestCommentSkipsNestedRaw(t *testing.T) {
	// `{% endcomment %}` inside a `{% raw %}…{% endraw %}` must NOT close
	// the outer comment.
	out, err := Render(
		`A{% comment %}before {% raw %}{% endcomment %}{% endraw %} after{% endcomment %}B`,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if out != "AB" {
		t.Fatalf("got %q", out)
	}
}

// pointerOnly has its accessor on a pointer receiver — historically
// unreachable when the struct was passed by value via a map.
type pointerOnly struct{ name string }

func (p *pointerOnly) Display() string { return "<<" + p.name + ">>" }

func TestMethodDispatchHandlesPointerReceiverOnValueStruct(t *testing.T) {
	// Pass by value; the data path goes through interface{} and reflect
	// can't normally see *T methods on a T value. callTemplateMethod
	// should promote to a pointer.
	out, err := Render(`{{ p.Display }}`, map[string]any{
		"p": pointerOnly{name: "Ada"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out != "<<Ada>>" {
		t.Fatalf("got %q", out)
	}
}

// failingMethod has a (T, error) accessor that returns an error.
type failingMethod struct{}

func (failingMethod) Risky() (string, error) {
	return "", fmt.Errorf("denied")
}

func TestErrorReturningMethodTreatedAsUndefined(t *testing.T) {
	// In strict mode, a (T, error) method that errored should surface as
	// undefined-property, not silently render empty.
	_, err := Render(`{{ x.Risky }}`,
		map[string]any{"x": failingMethod{}}, StrictVariables())
	if err == nil {
		t.Fatal("expected strict-mode error for failing method")
	}
	if !errors.Is(err, ErrUndefinedVariable) {
		t.Fatalf("expected ErrUndefinedVariable, got %v", err)
	}
}

func TestTablerowNilCollectionReturnsEmpty(t *testing.T) {
	out, err := Render(
		`{% tablerow x in missing %}{{ x }}{% endtablerow %}`, nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if out != "" {
		t.Fatalf("got %q, want empty", out)
	}
}

func TestTablerowAcceptsRangeAttribute(t *testing.T) {
	// Shopify accepts (and silently ignores) a `range:` attribute. Templates
	// that include it must parse without error.
	out, err := Render(
		`{% tablerow x in items cols: 2 range: items %}{{ x }}{% endtablerow %}`,
		map[string]any{"items": []any{"a", "b", "c"}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, ">a<") || !strings.Contains(out, ">b<") || !strings.Contains(out, ">c<") {
		t.Fatalf("got %q", out)
	}
}

func TestTemplateNameQuotedInError(t *testing.T) {
	// Prevent log injection: the template name appears %q-quoted, so
	// embedded newlines are escaped.
	tmpl := MustParse(`{{ 10 | divided_by: 0 }}`).WithName("evil\nlevel=INFO")
	_, err := tmpl.Render(nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Contains(err.Error(), "\nlevel=INFO") {
		t.Fatalf("error contains raw newline: %q", err.Error())
	}
}

func TestLogicalOperatorsPreserveValues(t *testing.T) {
	// Shopify's and/or are short-circuit value operators: they return the
	// actual chosen operand, not a coerced boolean.
	cases := []struct {
		template string
		data     map[string]any
		want     string
	}{
		// `or` returns the first truthy operand.
		{`{% assign x = a or b %}{{ x }}`, map[string]any{"a": "first", "b": "second"}, "first"},
		// When the left is falsy, returns the right verbatim — even when right is "".
		// Use `inspect-style` rendering by appending a marker to make the empty visible.
		{`{% assign x = a or b %}[{{ x }}]`, map[string]any{"a": nil, "b": ""}, "[]"},
		// `and` returns the right when left is truthy.
		{`{% assign x = a and b %}{{ x }}`, map[string]any{"a": "x", "b": "y"}, "y"},
		// `and` returns the left when left is falsy (e.g. false → "false").
		{`{% assign x = a and b %}{{ x }}`, map[string]any{"a": false, "b": "y"}, "false"},
		// Conditional contexts still see the right boolean answer.
		{`{% if a or b %}T{% else %}F{% endif %}`, map[string]any{"a": "", "b": false}, "T"}, // "" is truthy
		{`{% if a and b %}T{% else %}F{% endif %}`, map[string]any{"a": false, "b": "y"}, "F"},
	}
	for _, tc := range cases {
		got, err := Render(tc.template, tc.data)
		if err != nil {
			t.Errorf("%s: %v", tc.template, err)
			continue
		}
		if got != tc.want {
			t.Errorf("%s with %v: got %q, want %q", tc.template, tc.data, got, tc.want)
		}
	}
}

func TestEmptyAndBlankAreTruthyAsBareConditions(t *testing.T) {
	// `empty` and `blank` are sentinel objects; as bare conditions they're
	// non-nil non-false and therefore truthy. They retain their special
	// meaning when used as the right side of == / != against actual values.
	cases := []struct {
		template, want string
	}{
		{`{% if empty %}T{% else %}F{% endif %}`, "T"},
		{`{% if blank %}T{% else %}F{% endif %}`, "T"},
		{`{% if "" == empty %}T{% else %}F{% endif %}`, "T"},
		{`{% if "x" == empty %}T{% else %}F{% endif %}`, "F"},
	}
	for _, tc := range cases {
		got, err := Render(tc.template, nil)
		if err != nil {
			t.Errorf("%s: %v", tc.template, err)
			continue
		}
		if got != tc.want {
			t.Errorf("%s: got %q, want %q", tc.template, got, tc.want)
		}
	}
}
