package liquid

// Shopify "vision" theme benchmark — a port of performance/benchmark.rb +
// theme_runner.rb from github.com/Shopify/liquid. The fixtures (templates +
// vision.database.yml) are copied verbatim into testdata/shopify_bench so the
// numbers are directly comparable to a `bundle exec rake benchmark` run on
// the Ruby gem (modulo runtime).
//
// Skipped: the 12 templates that use {% paginate %} or {% form %} — those are
// custom Block tags in the Ruby gem's perf harness, and go-liquid has no tag
// plugin API yet. We still cover all 4 themes.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"

	"gopkg.in/yaml.v3"
)

const shopifyBenchRoot = "testdata/shopify_bench"

// shopifyData is the assigns map shared by every render in the benchmark.
// It mirrors Database.tables in performance/shopify/database.rb: top-level
// tables get re-keyed by handle, products gain a back-reference to their
// collections, and a few singular accessors (collection, product, blog,
// article, cart) are added for templates that expect them.
var (
	shopifyDataOnce sync.Once
	shopifyData     map[string]any
	shopifyFilters  sync.Once
)

func loadShopifyData(tb testing.TB) map[string]any {
	shopifyDataOnce.Do(func() {
		raw, err := os.ReadFile(filepath.Join(shopifyBenchRoot, "database.yml"))
		if err != nil {
			tb.Fatalf("read database.yml: %v", err)
		}
		var db map[string]any
		if err := yaml.Unmarshal(raw, &db); err != nil {
			tb.Fatalf("yaml unmarshal: %v", err)
		}
		shopifyData = reshapeShopifyDB(db)
	})
	return shopifyData
}

// reshapeShopifyDB mirrors Database.tables in database.rb. The YAML loads
// each table as []any of map[string]any; we cross-link products↔collections,
// then re-key each table by 'handle'.
func reshapeShopifyDB(db map[string]any) map[string]any {
	products := asSlice(db["products"])
	collections := asSlice(db["collections"])

	// Back-link: each product gets the collections it belongs to.
	for _, p := range products {
		pm := p.(map[string]any)
		var owned []any
		pid := shopifyIntHelper(pm["id"])
		for _, c := range collections {
			cm := c.(map[string]any)
			for _, cp := range asSlice(cm["products"]) {
				if shopifyIntHelper(cp.(map[string]any)["id"]) == pid {
					owned = append(owned, cm)
					break
				}
			}
		}
		pm["collections"] = owned
	}

	out := map[string]any{}
	for k, v := range db {
		rows := asSlice(v)
		if rows == nil {
			out[k] = v
			continue
		}
		byHandle := map[string]any{}
		for _, row := range rows {
			rm, ok := row.(map[string]any)
			if !ok {
				continue
			}
			if h, ok := rm["handle"].(string); ok {
				byHandle[h] = rm
			}
		}
		out[k] = byHandle
	}

	out["collection"] = firstValue(out["collections"])
	out["product"] = firstValue(out["products"])
	blog, _ := firstValue(out["blogs"]).(map[string]any)
	out["blog"] = blog
	if blog != nil {
		if articles := asSlice(blog["articles"]); len(articles) > 0 {
			out["article"] = articles[0]
		}
	}

	// Cart aggregates from line_items.
	var totalPrice, itemCount int
	var items []any
	if li, ok := out["line_items"].(map[string]any); ok {
		for _, v := range li {
			m := v.(map[string]any)
			items = append(items, m)
			totalPrice += shopifyIntHelper(m["line_price"]) * shopifyIntHelper(m["quantity"])
			itemCount += shopifyIntHelper(m["quantity"])
		}
	}
	out["cart"] = map[string]any{
		"total_price": totalPrice,
		"item_count":  itemCount,
		"items":       items,
	}
	return out
}

func asSlice(v any) []any {
	s, _ := v.([]any)
	return s
}

func shopifyIntHelper(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	default:
		return 0
	}
}

func firstValue(v any) any {
	m, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	// Pick a deterministic "first" by sorted handle so reruns are stable.
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	if len(keys) == 0 {
		return nil
	}
	sort.Strings(keys)
	return m[keys[0]]
}

// registerShopifyFilters installs stubs for the Shopify-specific filters used
// by the templates. Behaviors mirror performance/shopify/*.rb closely enough
// to produce non-empty output; exact byte-for-byte parity isn't required for
// benchmarking.
func registerShopifyFilters(tb testing.TB) {
	shopifyFilters.Do(func() {
		// money — cents → "$ 19.90"
		RegisterFilter("money", func(input any, _ ...any) any {
			n, ok := numAsFloat(input)
			if !ok {
				return ""
			}
			return fmt.Sprintf("$ %.2f", n/100)
		})
		RegisterFilter("money_with_currency", func(input any, _ ...any) any {
			n, ok := numAsFloat(input)
			if !ok {
				return ""
			}
			return fmt.Sprintf("$ %.2f USD", n/100)
		})

		// weight
		RegisterFilter("weight", func(input any, _ ...any) any {
			n, ok := numAsFloat(input)
			if !ok {
				return ""
			}
			return fmt.Sprintf("%.2f", n/1000)
		})
		RegisterFilter("weight_with_unit", func(input any, _ ...any) any {
			n, ok := numAsFloat(input)
			if !ok {
				return ""
			}
			return fmt.Sprintf("%.2f kg", n/1000)
		})

		// asset_url family
		RegisterFilter("asset_url", func(input any, _ ...any) any {
			return "/files/1/[shop_id]/[shop_id]/assets/" + fmt.Sprint(input)
		})
		RegisterFilter("global_asset_url", func(input any, _ ...any) any {
			return "/global/" + fmt.Sprint(input)
		})
		RegisterFilter("shopify_asset_url", func(input any, _ ...any) any {
			return "/shopify/" + fmt.Sprint(input)
		})

		// HTML helpers
		RegisterFilter("script_tag", func(input any, _ ...any) any {
			return fmt.Sprintf(`<script src="%s" type="text/javascript"></script>`, input)
		})
		RegisterFilter("stylesheet_tag", func(input any, args ...any) any {
			media := "all"
			if len(args) > 0 {
				media = fmt.Sprint(args[0])
			}
			return fmt.Sprintf(`<link href="%s" rel="stylesheet" type="text/css"  media="%s"  />`, input, media)
		})
		RegisterFilter("link_to", func(input any, args ...any) any {
			url := ""
			title := ""
			if len(args) > 0 {
				url = fmt.Sprint(args[0])
			}
			if len(args) > 1 {
				title = fmt.Sprint(args[1])
			}
			return fmt.Sprintf(`<a href="%s" title="%s">%s</a>`, url, title, input)
		})
		RegisterFilter("img_tag", func(input any, args ...any) any {
			alt := ""
			if len(args) > 0 {
				alt = fmt.Sprint(args[0])
			}
			return fmt.Sprintf(`<img src="%s" alt="%s" />`, input, alt)
		})

		// Vendor / type linking
		linkVendor := func(label any) string {
			if label == nil || label == "" {
				return "Unknown Vendor"
			}
			s := fmt.Sprint(label)
			return fmt.Sprintf(`<a href="/collections/%s" title="%s">%s</a>`, toHandle(s), s, s)
		}
		RegisterFilter("link_to_vendor", func(input any, _ ...any) any { return linkVendor(input) })
		RegisterFilter("link_to_type", func(input any, _ ...any) any { return linkVendor(input) })
		RegisterFilter("url_for_vendor", func(input any, _ ...any) any {
			return "/collections/" + toHandle(fmt.Sprint(input))
		})
		RegisterFilter("url_for_type", func(input any, _ ...any) any {
			return "/collections/" + toHandle(fmt.Sprint(input))
		})

		// product_img_url
		RegisterFilter("product_img_url", func(input any, args ...any) any {
			style := "small"
			if len(args) > 0 {
				style = fmt.Sprint(args[0])
			}
			s := fmt.Sprint(input)
			m := productImgURLRE.FindStringSubmatch(s)
			if m == nil {
				return ""
			}
			if style == "original" {
				return "/files/shops/random_number/" + s
			}
			return fmt.Sprintf("/files/shops/random_number/products/%s_%s.%s", m[1], style, m[2])
		})

		// pagination helper (used outside paginate blocks too)
		RegisterFilter("default_pagination", func(_ any, _ ...any) any { return "" })

		// pluralize
		RegisterFilter("pluralize", func(input any, args ...any) any {
			n := shopifyIntHelper(input)
			singular, plural := "", ""
			if len(args) > 0 {
				singular = fmt.Sprint(args[0])
			}
			if len(args) > 1 {
				plural = fmt.Sprint(args[1])
			}
			if n == 1 {
				return singular
			}
			return plural
		})

		// json — drop "collections" key to match the Ruby stub (avoids cycles)
		RegisterFilter("json", func(input any, _ ...any) any {
			if m, ok := input.(map[string]any); ok {
				cp := make(map[string]any, len(m))
				for k, v := range m {
					if k == "collections" {
						continue
					}
					cp[k] = v
				}
				return jsonDump(cp)
			}
			return jsonDump(input)
		})

		// tag filters — context vars (handle, current_tags) are not exposed to
		// our filter API, so we render with empty defaults. Faithful enough
		// for benchmarking template throughput.
		RegisterFilter("link_to_tag", func(input any, args ...any) any {
			tag := ""
			if len(args) > 0 {
				tag = fmt.Sprint(args[0])
			}
			return fmt.Sprintf(`<a title="Show tag %s" href="/collections//%s">%s</a>`, tag, tag, input)
		})
		RegisterFilter("highlight_active_tag", func(input any, _ ...any) any { return input })
		RegisterFilter("link_to_add_tag", func(input any, args ...any) any {
			tag := ""
			if len(args) > 0 {
				tag = fmt.Sprint(args[0])
			}
			return fmt.Sprintf(`<a title="Show tag %s" href="/collections//%s">%s</a>`, tag, tag, input)
		})
		RegisterFilter("link_to_remove_tag", func(input any, args ...any) any {
			tag := ""
			if len(args) > 0 {
				tag = fmt.Sprint(args[0])
			}
			return fmt.Sprintf(`<a title="Show tag %s" href="/collections//%s">%s</a>`, tag, tag, input)
		})

		// `within: collection` — returns the input URL unchanged.
		RegisterFilter("within", func(input any, _ ...any) any { return input })
		// `highlight: terms` — wrap occurrences in <strong>; cheap stub.
		RegisterFilter("highlight", func(input any, args ...any) any {
			s := fmt.Sprint(input)
			if len(args) == 0 {
				return s
			}
			term := fmt.Sprint(args[0])
			if term == "" {
				return s
			}
			return strings.ReplaceAll(s, term, "<strong>"+term+"</strong>")
		})
	})
}

// shopifyTagsOnce guards RegisterBlock calls so re-running the bench in the
// same process doesn't double-register (RegisterBlock panics on collision
// only against built-ins, but re-registering would silently mask earlier
// state — sync.Once keeps the harness deterministic).
var shopifyTagsOnce sync.Once

// registerShopifyTags installs the {% paginate %} and {% form %} block tags
// used by the Shopify benchmark. Behavior mirrors performance/shopify/
// {paginate,comment_form}.rb closely enough to render the templates — exact
// pagination math is not needed for benchmarking throughput.
func registerShopifyTags(tb testing.TB) {
	shopifyTagsOnce.Do(func() {
		// {% paginate COLLECTION by N %}…{% endpaginate %}
		// Stub: assigns a `paginate` map with sensible fixed values, then
		// renders the body. Mirrors the Ruby stub which also assigns a
		// largely fixed hash regardless of input.
		paginateRE := regexp.MustCompile(`^\s*([\w.]+)\s+by\s+(\d+)`)
		RegisterBlock("paginate", func(markup string) (TagRenderer, error) {
			m := paginateRE.FindStringSubmatch(markup)
			if m == nil {
				return nil, fmt.Errorf("paginate: expected `COLLECTION by N`, got %q", markup)
			}
			pageSize, _ := strconvAtoi(m[2])
			return &paginateBlock{collectionPath: m[1], pageSize: pageSize}, nil
		})

		// {% form ARTICLE %}…{% endform %} — wraps body in a comment form
		// element. Mirrors performance/shopify/comment_form.rb.
		RegisterBlock("form", func(markup string) (TagRenderer, error) {
			name := strings.TrimSpace(markup)
			if name == "" {
				return nil, fmt.Errorf("form: missing variable name")
			}
			return &formBlock{varName: name}, nil
		})
	})
}

type paginateBlock struct {
	collectionPath string
	pageSize       int
}

func (b *paginateBlock) Render(w io.Writer, ctx TagContext) error {
	collection := lookupPath(ctx, b.collectionPath)
	collectionSize := sizeOf(collection)
	pageCount := 1
	if b.pageSize > 0 {
		pageCount = (collectionSize+b.pageSize-1)/b.pageSize + 1
	}
	currentPage := 1

	parts := make([]any, 0, pageCount)
	for i := 1; i < pageCount; i++ {
		title := fmt.Sprintf("%d", i)
		isCurrent := i == currentPage
		if isCurrent {
			parts = append(parts, map[string]any{"title": title, "is_link": false})
		} else {
			parts = append(parts, map[string]any{
				"title":   title,
				"url":     fmt.Sprintf("/collections/frontpage?page=%d", i),
				"is_link": true,
			})
		}
	}

	paginate := map[string]any{
		"page_size":      b.pageSize,
		"current_page":   currentPage,
		"current_offset": b.pageSize * (currentPage - 1),
		"items":          collectionSize,
		"pages":          pageCount - 1,
		"parts":          parts,
		"previous":       nil,
		"next":           nil,
	}
	if pageCount > currentPage+1 {
		paginate["next"] = map[string]any{
			"title":   "Next &raquo;",
			"url":     fmt.Sprintf("/collections/frontpage?page=%d", currentPage+1),
			"is_link": true,
		}
	}

	return ctx.PushScope(func() error {
		ctx.Assign("paginate", paginate)
		return ctx.RenderBody(w)
	})
}

type formBlock struct {
	varName string
}

func (b *formBlock) Render(w io.Writer, ctx TagContext) error {
	article := lookupPath(ctx, b.varName)
	id := ""
	if m, ok := article.(map[string]any); ok {
		id = fmt.Sprint(m["id"])
	}
	if _, err := fmt.Fprintf(w, `<form id="article-%s-comment-form" class="comment-form" method="post" action="">`+"\n", id); err != nil {
		return err
	}
	err := ctx.PushScope(func() error {
		ctx.Assign("form", map[string]any{
			"posted_successfully?": false,
			"errors":               nil,
			"author":               "",
			"email":                "",
			"body":                 "",
		})
		return ctx.RenderBody(w)
	})
	if err != nil {
		return err
	}
	_, err = io.WriteString(w, "\n</form>")
	return err
}

// lookupPath resolves dotted names like `collection.products` against the
// active scope chain. Custom tags don't get the parser's expression engine,
// so we walk the path manually using whatever the data model exposes.
func lookupPath(ctx TagContext, path string) any {
	parts := strings.Split(path, ".")
	cur := ctx.Get(parts[0])
	for _, p := range parts[1:] {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur = m[p]
	}
	return cur
}

func sizeOf(v any) int {
	switch x := v.(type) {
	case []any:
		return len(x)
	case map[string]any:
		return len(x)
	default:
		return 0
	}
}

// strconvAtoi is the standard library's; aliased here to keep imports tidy.
func strconvAtoi(s string) (int, error) {
	return strconv.Atoi(s)
}

var productImgURLRE = regexp.MustCompile(`^products/([\w\-_]+)\.(\w{2,4})`)

func numAsFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case float64:
		return n, true
	default:
		return 0, false
	}
}

var nonWordRE = regexp.MustCompile(`\W+`)

func toHandle(s string) string {
	s = strings.ToLower(s)
	for _, ch := range []string{"'", `"`, "(", ")", "[", "]"} {
		s = strings.ReplaceAll(s, ch, "")
	}
	s = nonWordRE.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return s
}

// jsonDump is a tiny JSON encoder for the `json` filter stub. We avoid
// encoding/json so the benchmark doesn't pull in another stdlib hot path
// that the Ruby version doesn't (Ruby uses JSON.dump from the json gem).
// Simplest correct thing: defer to encoding/json.
func jsonDump(v any) string {
	out, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(out)
}

// shopifyTemplate is a parsed page + its theme layout, ready to render.
type shopifyTemplate struct {
	theme  string
	name   string
	source string
	layout string // raw source of theme.liquid; empty if missing
	page   *Template
	tmpl   *Template // theme layout, parsed once
}

// loadShopifyTemplates discovers all (theme, page) pairs across all themes.
func loadShopifyTemplates(tb testing.TB) []shopifyTemplate {
	registerShopifyFilters(tb)
	registerShopifyTags(tb)

	themes, err := os.ReadDir(shopifyBenchRoot)
	if err != nil {
		tb.Fatalf("read shopify_bench: %v", err)
	}

	var out []shopifyTemplate
	for _, th := range themes {
		if !th.IsDir() {
			continue
		}
		themeDir := filepath.Join(shopifyBenchRoot, th.Name())
		layoutPath := filepath.Join(themeDir, "theme.liquid")
		layoutSrc, err := os.ReadFile(layoutPath)
		if err != nil {
			continue // theme has no layout — skip whole theme
		}
		layout, err := Parse(string(layoutSrc))
		if err != nil {
			tb.Fatalf("parse %s: %v", layoutPath, err)
		}

		pages, err := filepath.Glob(filepath.Join(themeDir, "*.liquid"))
		if err != nil {
			tb.Fatalf("glob %s: %v", themeDir, err)
		}
		for _, p := range pages {
			if filepath.Base(p) == "theme.liquid" {
				continue
			}
			src, err := os.ReadFile(p)
			if err != nil {
				tb.Fatalf("read %s: %v", p, err)
			}
			page, err := Parse(string(src))
			if err != nil {
				tb.Fatalf("parse %s: %v", p, err)
			}
			out = append(out, shopifyTemplate{
				theme:  th.Name(),
				name:   filepath.Base(p),
				source: string(src),
				layout: string(layoutSrc),
				page:   page,
				tmpl:   layout,
			})
		}
	}
	if len(out) == 0 {
		tb.Fatalf("no Shopify templates loaded from %s", shopifyBenchRoot)
	}
	return out
}

// renderShopifyOnce renders one (page, layout) pair into w, mirroring
// theme_runner.rb#render: page → content_for_layout → layout.
func renderShopifyOnce(tb testing.TB, t shopifyTemplate, w io.Writer) {
	pageBuf := bufPool.Get().(*bytes.Buffer)
	pageBuf.Reset()
	defer bufPool.Put(pageBuf)

	assigns := make(map[string]any, len(shopifyData)+2)
	for k, v := range shopifyData {
		assigns[k] = v
	}
	assigns["page_title"] = "Page title"
	assigns["template"] = strings.TrimSuffix(t.name, ".liquid")

	if err := t.page.RenderTo(pageBuf, assigns); err != nil {
		tb.Fatalf("render page %s/%s: %v", t.theme, t.name, err)
	}
	assigns["content_for_layout"] = pageBuf.String()
	if err := t.tmpl.RenderTo(w, assigns); err != nil {
		tb.Fatalf("render layout %s/%s: %v", t.theme, t.name, err)
	}
}

var bufPool = sync.Pool{New: func() any { return new(bytes.Buffer) }}

// TestShopifyBenchSmoke renders every benchmarked template once and verifies
// the output is non-empty. Runs as a normal test so `go test` catches porting
// regressions before benchmarks ever run.
func TestShopifyBenchSmoke(t *testing.T) {
	loadShopifyData(t)
	tmpls := loadShopifyTemplates(t)
	t.Logf("loaded %d Shopify templates", len(tmpls))
	var buf bytes.Buffer
	for _, tm := range tmpls {
		buf.Reset()
		renderShopifyOnce(t, tm, &buf)
		if buf.Len() == 0 {
			t.Errorf("%s/%s rendered empty", tm.theme, tm.name)
		}
	}
}

// BenchmarkShopifyTokenize mirrors the Ruby benchmark's `tokenize:` phase —
// drive the lexer to EOF for every template+layout source without invoking
// the parser. Useful for isolating regressions in the scanner.
func BenchmarkShopifyTokenize(b *testing.B) {
	loadShopifyData(b)
	tmpls := loadShopifyTemplates(b)
	sources := make([]string, 0, len(tmpls)*2)
	for _, t := range tmpls {
		sources = append(sources, t.source, t.layout)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, src := range sources {
			l := newLexer(src)
			for {
				tok := l.nextToken()
				if tok.typ == tokenEOF {
					break
				}
			}
		}
	}
}

func BenchmarkShopifyParse(b *testing.B) {
	loadShopifyData(b)
	tmpls := loadShopifyTemplates(b)
	sources := make([]string, 0, len(tmpls)*2)
	for _, t := range tmpls {
		sources = append(sources, t.source, t.layout)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, src := range sources {
			if _, err := Parse(src); err != nil {
				b.Fatal(err)
			}
		}
	}
}

func BenchmarkShopifyRender(b *testing.B) {
	loadShopifyData(b)
	tmpls := loadShopifyTemplates(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, t := range tmpls {
			renderShopifyOnce(b, t, io.Discard)
		}
	}
}

func BenchmarkShopifyParseAndRender(b *testing.B) {
	loadShopifyData(b)
	tmpls := loadShopifyTemplates(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, t := range tmpls {
			page, err := Parse(t.source)
			if err != nil {
				b.Fatal(err)
			}
			layout, err := Parse(t.layout)
			if err != nil {
				b.Fatal(err)
			}
			t2 := shopifyTemplate{theme: t.theme, name: t.name, page: page, tmpl: layout}
			renderShopifyOnce(b, t2, io.Discard)
		}
	}
}
