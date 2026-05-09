package liquid

import (
	"errors"
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
		for i := 0; i < 200; i++ {
			tmpl.WithLoader(MapLoader{"p": "A"})
		}
		close(done)
	}()
	for i := 0; i < 200; i++ {
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
