# go-liquid

A pure-Go implementation of the [Liquid](https://shopify.github.io/liquid/) template language.

- **Liquid-compatible** — output (`{{ }}`), tags (`{% %}`), filters, control flow, loops, whitespace control.
- **Fast** — single-pass lexer, no regexes, no reflection on the hot path for `map[string]any` data.
- **Idiomatic Go** — `Parse` once, `Render` many times. Streaming via `io.Writer`. Structs and tagged fields work out of the box.
- **No runtime dependencies** — only the Go standard library is linked into your binary. (`gopkg.in/yaml.v3` is a test-only dep used by the Shopify benchmark.)

## Install

```sh
go get github.com/nchapman/go-liquid
```

## Usage

```go
import "github.com/nchapman/go-liquid"

// One-shot
out, err := liquid.Render("Hello, {{ name }}!", map[string]any{"name": "World"})

// Parse once, render many times (concurrency-safe)
tmpl, err := liquid.Parse(source)
out, err := tmpl.Render(data)

// Stream to an io.Writer
err := tmpl.RenderTo(w, data)

// Templates known at startup
var tmpl = liquid.MustParse(`{{ greeting }}, {{ name }}!`)
```

### Data shapes

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

### Custom filters

```go
liquid.RegisterFilter("shout", func(input any, _ ...any) any {
    return strings.ToUpper(fmt.Sprint(input)) + "!"
})
```

### Custom tags

Both inline (`{% mytag … %}`) and block (`{% mytag … %}body{% endmytag %}`)
tags are supported. The parser hands you the raw markup; you return a
renderer that runs at template evaluation time.

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

`TagContext` exposes `Get` / `Assign` / `PushScope(fn)` for working with the
scope chain (use `PushScope` to make assigns local to the block, like Ruby
Liquid's `context.stack`). Registration is global and not safe to call
concurrently with rendering — register at startup.

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

A `Loader` resolves partial names to source. The standard library ships
`liquid.NewFileSystemLoader(dir)`; implement `liquid.Loader` for other sources.

**Other tags** — `{% comment %}…{% endcomment %}`, `{% raw %}…{% endraw %}`,
`{% liquid %}` (multi-line tag block), `{% echo expr %}`, `{% doc %}…{% enddoc %}`,
`{% # inline comment %}`.

**Whitespace control** — `{{- -}}` and `{%- -%}` strip surrounding whitespace.

**Built-in filters** — `upcase`, `downcase`, `capitalize`, `strip`, `lstrip`, `rstrip`, `escape` (alias `h`), `newline_to_br`, `split`, `append`, `prepend`, `replace`, `replace_first`, `remove`, `remove_first`, `truncate`, `truncatewords`, `slice`, `first`, `last`, `size`, `join`, `reverse`, `sort`, `sort_natural`, `map`, `where`, `find`, `uniq`, `compact`, `concat`, `flatten`, `sum`, `default`, `plus`, `minus`, `times`, `divided_by`, `modulo`, `abs`, `round`, `ceil`, `floor`, `at_least`, `at_most`, `date`.

## Performance

Microbenchmarks on an Apple M4 Max (`go test -bench=. -benchmem`):

```
BenchmarkParse-16           1438 ns/op     1560 B/op    33 allocs/op
BenchmarkRenderParsed-16   26734 ns/op    75760 B/op   626 allocs/op
BenchmarkRenderTo-16       25321 ns/op    70352 B/op   615 allocs/op
```

Render benchmark uses a 100-element loop with conditionals, filters, and property access.

### Shopify "vision" theme benchmark

`go test -bench=Shopify` runs a port of Shopify/liquid's
[`performance/benchmark.rb`](https://github.com/Shopify/liquid/blob/main/performance/benchmark.rb)
— 30 page templates across 4 real Shopify themes, wrapped in their
`theme.liquid` layouts, against the same `vision.database.yml` fixture.
`{% paginate %}` and `{% form %}` are registered as block tags via the
public `RegisterBlock` API (see _Custom tags_ above).

Wall time per full pass over the 30-template set, on the same Apple M4 Max
(`go test -bench=Shopify -benchtime=5s` and the Ruby gem's bench at
`PHASE=… bundle exec ruby performance/benchmark.rb`):

| Phase            | go-liquid | Shopify/liquid (Ruby 3.4 + YJIT) | Ratio |
|------------------|-----------|----------------------------------|-------|
| Tokenize         | 0.40 ms   | 0.26 ms                          | 0.7×  |
| Parse            | 0.69 ms   | 5.99 ms                          | 8.7×  |
| Render           | 0.60 ms   | 1.31 ms                          | 2.2×  |
| Parse + Render   | 1.36 ms   | 7.89 ms                          | 5.8×  |

Ruby wins on raw tokenization — its `StringScanner` is very tight C — but
go-liquid pulls ahead the moment any AST or render work is involved.

Reproduce the Ruby side:

```sh
cd path/to/Shopify/liquid
bundle install
PHASE=render bundle exec ruby performance/benchmark.rb
PHASE=parse  bundle exec ruby performance/benchmark.rb
```

Caveat: filter stubs (`money`, `asset_url`, `link_to_*`, etc.) are minimal
ports of `performance/shopify/*.rb` and don't reproduce every edge case —
fine for throughput comparison, not for output diffing.

## Status

Tests cover the standard Liquid surface used by Jekyll/Hugo-style templates,
including `{% include %}` and `{% render %}` partials via a pluggable
`Loader` and custom tags via `RegisterTag` / `RegisterBlock`. Drops are not
implemented yet.

## License

MIT
