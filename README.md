# go-liquid

A pure-Go implementation of the [Liquid](https://shopify.github.io/liquid/) template language.

Docs: <https://pkg.go.dev/github.com/nchapman/go-liquid>

- **Liquid-compatible** — output (`{{ }}`), tags (`{% %}`), filters, control flow, loops, whitespace control, partials.
- **Fast** — single-pass lexer, no regexes, real `io.Writer` streaming, no reflection on the hot path for `map[string]any` data.
- **Idiomatic Go** — `Parse` once, `Render` many times. Concurrency-safe templates. Structs and `liquid:""`-tagged fields work out of the box. Drops for custom property resolution.
- **Sandbox-friendly** — per-render `ResourceLimits` (mirrors Ruby's `Liquid::ResourceLimits`) and a configurable `ErrorMode` for hosting untrusted templates.
- **Zero dependencies** — Go standard library only. (`gopkg.in/yaml.v3` is a test-only dependency.)

Requires Go 1.24+.

## Install

```sh
go get github.com/nchapman/go-liquid
```

## Usage

One-shot:

```go
out, err := liquid.Render("Hello, {{ name }}!", map[string]any{"name": "World"})
```

Parse once, render many times (templates are safe for concurrent rendering):

```go
tmpl, err := liquid.Parse(source)
out, err := tmpl.Render(data)
```

Stream directly to an `io.Writer`:

```go
err := tmpl.RenderTo(w, data)
```

Templates known at startup:

```go
var tmpl = liquid.MustParse(`{{ greeting }}, {{ name }}!`)
```

### Data

`Render` accepts:

- `map[string]any` (fastest path)
- Any other map keyed by a stringable type
- A struct (or pointer to one) — exported fields are exposed by name; use the `liquid:"alias"` tag to rename:

  ```go
  type User struct {
      Name  string
      Email string `liquid:"email_address"`
  }
  liquid.Render("{{ Name }} <{{ email_address }}>", User{...})
  ```

Inside templates, dot access works on maps, struct fields, and zero-arg methods.

### Drops

Implement `Drop` to take full control of property lookup on a value (Shopify Liquid's drop model). Methods on the type are not auto-dispatched; `LiquidLookup` is the sole source of truth.

```go
type Product struct{ id int }

func (p Product) LiquidLookup(key string) (any, bool) {
    switch key {
    case "id":   return p.id, true
    case "url":  return fmt.Sprintf("/p/%d", p.id), true
    }
    return nil, false
}
```

Implement `ContextAwareDrop` to receive the active `RenderContext` for scope and resource-limit access.

### Environment

`Environment` holds a private set of filters, tags, a loader, an error mode, and resource limits. Use it instead of the package-level `RegisterFilter` / `RegisterTag` when you want isolated configuration or thread-safe registration.

```go
env := liquid.NewEnvironment().
    WithLoader(liquid.NewFileSystemLoader("./templates", ".liquid")).
    WithErrorMode(liquid.ErrorModeWarn)

env.RegisterFilter("shout", func(in any, _ ...any) any {
    return strings.ToUpper(fmt.Sprint(in)) + "!"
})

tmpl, err := env.Parse(source)
```

The package-level `Render`, `Parse`, `RegisterFilter`, etc. operate on a shared default environment.

### Custom filters

```go
liquid.RegisterFilter("shout", func(input any, _ ...any) any {
    return strings.ToUpper(fmt.Sprint(input)) + "!"
})
```

Filter signatures: `FilterFunc` (`func(any, ...any) any`), `FilterFuncE` (returns `error`), and `KwargFilterFunc` for keyword arguments.

### Custom tags

Both inline (`{% mytag … %}`) and block (`{% mytag … %}body{% endmytag %}`) tags are supported. The parser hands you the raw markup; you return a renderer that runs at template evaluation time.

```go
type uppercaseBlock struct{}

func (uppercaseBlock) Render(w io.Writer, ctx liquid.TagContext) error {
    var buf strings.Builder
    if err := ctx.RenderBody(&buf); err != nil {
        return err
    }
    _, err := io.WriteString(w, strings.ToUpper(buf.String()))
    return err
}

liquid.RegisterBlock("uppercase", func(markup string) (liquid.TagRenderer, error) {
    return uppercaseBlock{}, nil
})

// {% uppercase %}hello {{ name }}{% enduppercase %}  →  HELLO WORLD
```

`TagContext` exposes `Get` / `Assign` / `PushScope(fn)` for working with the scope chain (use `PushScope` to make assigns local to the block, like Ruby Liquid's `context.stack`). For tags that need to evaluate Liquid expressions in their markup, use `Environment.ParseExpression` and `TagContext.Eval`.

### Sandboxing untrusted templates

`ResourceLimits` caps the work and output a single render is allowed to perform. Each limit is opt-in (zero means unlimited):

```go
limits := &liquid.ResourceLimits{
    RenderLengthLimit: 1 << 20, // 1 MiB output
    RenderScoreLimit:  100_000,
    AssignScoreLimit:  100_000,
}
out, err := tmpl.Render(data, liquid.WithLimits(limits))
if errors.Is(err, liquid.ErrResourceLimit) {
    // template tripped the cap
}
```

A single `*ResourceLimits` is not safe for concurrent renders — use one per render, or serialize.

`ErrorMode` controls how the parser handles unknown tags: `ErrorModeStrict` (default, fail-fast), `ErrorModeWarn` (collect `Template.Warnings()` and render the offending span as literal text), or `ErrorModeLax` (silently degrade).

## Supported syntax

**Output and filters**

```liquid
{{ user.name | upcase | append: "!" }}
{{ price | round: 2 | default: "n/a" }}
```

**Control flow**

```liquid
{% if user.admin %}…{% elsif user.active %}…{% else %}…{% endif %}
{% unless hidden %}…{% endunless %}
{% case status %}{% when "ok" %}…{% else %}…{% endcase %}
```

**Loops**

```liquid
{% for item in items limit: 10 offset: 5 reversed %}
  {{ forloop.index }}. {{ item }}
{% else %}
  No items.
{% endfor %}

{% for i in (1..5) %}{{ i }}{% endfor %}

{% tablerow item in items cols: 3 %}{{ item }}{% endtablerow %}
{% ifchanged %}{{ product.category }}{% endifchanged %}
```

**Variables**

```liquid
{% assign total = price | plus: tax %}
{% capture greeting %}Hello, {{ name }}{% endcapture %}
{% increment counter %}{% decrement counter %}
{% cycle "a", "b", "c" %}
{% cycle "rows": "odd", "even" %}
```

**Partials**

```liquid
{% include "header" %}
{% render "card" with product %}
{% render "card" for products as product %}
```

A `Loader` resolves partial names to source. The standard library ships `liquid.NewFileSystemLoader(dir, ext)`; implement `liquid.Loader` for other sources.

**Other tags** — `{% comment %}`, `{% raw %}`, `{% liquid %}` (multi-line tag block), `{% echo expr %}`, `{% doc %}`, `{% # inline comment %}`.

**Whitespace control** — `{{- -}}` and `{%- -%}` strip surrounding whitespace.

**Filters** — the standard Liquid string, array, math, and date filters (`upcase`, `replace`, `where`, `map`, `sort`, `plus`, `round`, `date`, etc.). See the [godoc](https://pkg.go.dev/github.com/nchapman/go-liquid) for the full list.

## Status

Tests cover the standard Liquid surface used by Jekyll/Hugo-style templates, including the Shopify "vision" theme benchmark suite. Compared to [`osteele/liquid`](https://github.com/osteele/liquid), go-liquid is single-package, has no runtime dependencies, supports `io.Writer` streaming, and is roughly 5–9× faster on parse+render with ~12–25× less memory (see Performance).

## Performance

Microbenchmarks on an Apple M4 Max (`go test -bench=. -benchmem`):

```
BenchmarkParse-16           1200 ns/op     1624 B/op    33 allocs/op
BenchmarkRenderParsed-16   11810 ns/op     7880 B/op   128 allocs/op
BenchmarkRenderTo-16       10872 ns/op     2480 B/op   117 allocs/op
```

Render benchmark uses a 100-element loop with conditionals, filters, and property access.

### vs `osteele/liquid`

Same template, same data, identical output, Apple M4 Max:

| Phase          | go-liquid | osteele/liquid | Ratio |
|----------------|-----------|----------------|-------|
| Parse          |  1.6 µs   | 14.3 µs        | 8.9×  |
| Render         | 17.5 µs   | 95.5 µs        | 5.4×  |
| Parse + Render | 19.5 µs   | 109.8 µs       | 5.6×  |

Allocation deltas are larger still: ~25× less memory at parse, ~12× at render.

### Shopify "vision" theme benchmark

`go test -bench=Shopify` runs a port of Shopify/liquid's [`performance/benchmark.rb`](https://github.com/Shopify/liquid/blob/main/performance/benchmark.rb) — 30 page templates across 4 real Shopify themes, wrapped in their `theme.liquid` layouts, against the same `vision.database.yml` fixture. `{% paginate %}` and `{% form %}` are registered as block tags via the public `RegisterBlock` API.

Wall time per full pass over the 30-template set, on the same Apple M4 Max (`go test -bench=Shopify -benchtime=5s` and the Ruby gem's bench at `PHASE=… bundle exec ruby performance/benchmark.rb`):

| Phase            | go-liquid | Shopify/liquid (Ruby 3.4 + YJIT) | Ratio |
|------------------|-----------|----------------------------------|-------|
| Tokenize         | 0.18 ms   | 0.26 ms                          | 1.4×  |
| Parse            | 0.50 ms   | 5.96 ms                          | 11.9× |
| Render           | 0.54 ms   | 1.32 ms                          | 2.4×  |
| Parse + Render   | 1.15 ms   | 7.96 ms                          | 6.9×  |

Reproduce the Ruby side:

```sh
cd path/to/Shopify/liquid
bundle install
PHASE=render bundle exec ruby performance/benchmark.rb
PHASE=parse  bundle exec ruby performance/benchmark.rb
```

Caveat: filter stubs (`money`, `asset_url`, `link_to_*`, etc.) are minimal ports of `performance/shopify/*.rb` and don't reproduce every edge case — fine for throughput comparison, not for output diffing.

## Other implementations

- [osteele/liquid](https://github.com/osteele/liquid) (Go)
- [Shopify/liquid](https://github.com/Shopify/liquid) (Ruby — the reference implementation), and Shopify's list of [ports to other environments](https://github.com/Shopify/liquid/wiki/Ports-of-Liquid-to-other-environments).

## License

MIT
