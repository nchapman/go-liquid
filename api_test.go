package liquid

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
)

// TestRegisterFilter installs a custom positional-argument filter and
// confirms it is callable from a template. Counterpart to the existing
// TestRegisterKwargFilter.
func TestRegisterFilter(t *testing.T) {
	const name = "shout_test_filter"
	RegisterFilter(name, func(input any, args ...any) any {
		s := toString(input)
		suffix := "!"
		if len(args) > 0 {
			suffix = toString(args[0])
		}
		return strings.ToUpper(s) + suffix
	})
	t.Cleanup(func() { delete(filters, name) })

	got, err := Render(`{{ "hi" | `+name+`: "?!" }}`, nil)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if got != "HI?!" {
		t.Errorf("got %q, want %q", got, "HI?!")
	}

	// A custom filter overrides a built-in of the same name.
	RegisterFilter("upcase", func(input any, args ...any) any { return "OVERRIDDEN" })
	t.Cleanup(func() {
		// Restore the built-in so other tests aren't poisoned.
		filters["upcase"] = FilterFunc(filterUpcase)
	})
	got, err = Render(`{{ "x" | upcase }}`, nil)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if got != "OVERRIDDEN" {
		t.Errorf("override: got %q, want OVERRIDDEN", got)
	}
}

// TestRegisterFilterE exercises the new (any, error)-returning filter
// signature: errors flow back as render errors at the filter's source
// position without going through the filterError sentinel.
func TestRegisterFilterE(t *testing.T) {
	const name = "must_be_positive"
	RegisterFilterE(name, func(input any, args []any, kwargs map[string]any) (any, error) {
		n := toInt(toNumber(input))
		if n <= 0 {
			return nil, fmt.Errorf("%s: expected positive int, got %v", name, input)
		}
		return n * 2, nil
	})
	t.Cleanup(func() { delete(filters, name) })

	got, err := Render(`{{ x | `+name+` }}`, map[string]any{"x": 5})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if got != "10" {
		t.Errorf("happy path: got %q want 10", got)
	}

	_, err = Render(`{{ x | `+name+` }}`, map[string]any{"x": -1})
	if err == nil {
		t.Fatal("expected error from FilterE filter")
	}
	var rerr *RenderError
	if !errors.As(err, &rerr) {
		t.Fatalf("expected *RenderError, got %T", err)
	}
	if !strings.Contains(rerr.Error(), "expected positive") {
		t.Errorf("error should propagate the message: %q", rerr.Error())
	}
}

// TestTemplateWithName confirms the name attaches to ParseError /
// RenderError so callers can distinguish which template failed.
func TestTemplateWithName(t *testing.T) {
	tpl, err := Parse(`{{ x | divided_by: 0 }}`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	tpl.WithName("templates/checkout.liquid")

	_, err = tpl.Render(map[string]any{"x": 10})
	if err == nil {
		t.Fatal("expected render error")
	}
	var rerr *RenderError
	if !errors.As(err, &rerr) {
		t.Fatalf("expected *RenderError, got %T", err)
	}
	if rerr.TemplateName != "templates/checkout.liquid" {
		t.Errorf("TemplateName: got %q", rerr.TemplateName)
	}
	if !strings.Contains(rerr.Error(), "templates/checkout.liquid") {
		t.Errorf("error string should mention template name: %q", rerr.Error())
	}
}

// TestRenderToWritesOutput confirms RenderTo streams to an io.Writer and
// surfaces render errors without writing partial output.
func TestRenderToWritesOutput(t *testing.T) {
	tpl := MustParse(`Hello, {{ name }}!`)
	var buf bytes.Buffer
	if err := tpl.RenderTo(&buf, map[string]any{"name": "Ada"}); err != nil {
		t.Fatalf("RenderTo: %v", err)
	}
	if buf.String() != "Hello, Ada!" {
		t.Errorf("got %q", buf.String())
	}

	// An error from rendering propagates and the buffer stays clean.
	bad := MustParse(`{{ x | divided_by: 0 }}`)
	var b2 bytes.Buffer
	err := bad.RenderTo(&b2, map[string]any{"x": 1})
	if err == nil {
		t.Fatal("expected error")
	}
	if b2.Len() != 0 {
		t.Errorf("buffer should be empty on error, got %q", b2.String())
	}
}

// TestRenderErrorUnwrap exercises errors.Unwrap / errors.Is plumbing so
// callers can match the inner error type, not just the wrapped string.
func TestRenderErrorUnwrap(t *testing.T) {
	sentinel := errors.New("boom from custom filter")
	const name = "boom_filter_for_unwrap"
	RegisterFilter(name, func(input any, args ...any) any {
		return filterErrorf("%w", sentinel)
	})
	t.Cleanup(func() { delete(filters, name) })

	_, err := Render(`{{ "x" | `+name+` }}`, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	var rerr *RenderError
	if !errors.As(err, &rerr) {
		t.Fatalf("expected *RenderError, got %T", err)
	}
	if rerr.Unwrap() == nil {
		t.Fatal("RenderError.Unwrap() returned nil")
	}
	if !errors.Is(err, sentinel) {
		t.Errorf("errors.Is should reach sentinel through RenderError")
	}
}

// TestConcurrentRenderSharedTemplate verifies that a single parsed
// Template can be rendered concurrently from many goroutines without data
// races or corrupted output. The existing concurrency test exercises
// WithLoader swaps; this one targets the hotter path of repeated renders.
func TestConcurrentRenderSharedTemplate(t *testing.T) {
	tpl := MustParse(`{% for x in items %}{{ x | times: 2 }}{% if forloop.last == false %},{% endif %}{% endfor %}`)
	const goroutines = 32
	const iterations = 200

	var wg sync.WaitGroup
	errs := make(chan error, goroutines*iterations)
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				want := fmt.Sprintf("%d,%d,%d", g*2, (g+1)*2, (g+2)*2)
				got, err := tpl.Render(map[string]any{"items": []any{g, g + 1, g + 2}})
				if err != nil {
					errs <- fmt.Errorf("g=%d i=%d: %w", g, i, err)
					return
				}
				if got != want {
					errs <- fmt.Errorf("g=%d i=%d: got %q want %q", g, i, got, want)
					return
				}
			}
		}(g)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
}

// TestConcurrentRenderWithPartials renders a template that loads partials
// concurrently — exercises partialCache under contention. Each goroutine
// requests the same partials so the cache reuse path runs hot.
func TestConcurrentRenderWithPartials(t *testing.T) {
	tpl := MustParse(`{% render "row" with item %}{% include "footer" %}`)
	tpl.WithLoader(MapLoader{
		"row":    `[{{ row.id }}:{{ row.name }}]`,
		"footer": `--END--`,
	})

	var wg sync.WaitGroup
	const N = 64
	bad := make(chan string, N)
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			data := map[string]any{"item": map[string]any{"id": i, "name": "x"}}
			out, err := tpl.Render(data)
			if err != nil {
				bad <- err.Error()
				return
			}
			want := fmt.Sprintf("[%d:x]--END--", i)
			if out != want {
				bad <- fmt.Sprintf("got %q want %q", out, want)
			}
		}(i)
	}
	wg.Wait()
	close(bad)
	for msg := range bad {
		t.Error(msg)
	}
}

// TestParseErrorTemplateName: the name set by the loader on a partial
// surfaces through ParseError when the partial is malformed.
func TestParseErrorTemplateName(t *testing.T) {
	tpl := MustParse(`{% include "broken" %}`)
	tpl.WithLoader(MapLoader{
		"broken": `{% if x %}no closer`,
	})
	_, err := tpl.Render(nil)
	if err == nil {
		t.Fatal("expected parse error from partial")
	}
	var perr *ParseError
	if !errors.As(err, &perr) {
		t.Fatalf("expected *ParseError, got %T: %v", err, err)
	}
	if perr.TemplateName != "broken" {
		t.Errorf("TemplateName: got %q want %q", perr.TemplateName, "broken")
	}
}
