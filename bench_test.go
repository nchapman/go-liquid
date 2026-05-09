package liquid

import (
	"io"
	"testing"
)

var benchSource = `
{%- for user in users -%}
  {%- if user.active -%}
    <li>{{ user.name | upcase }} ({{ user.email }})</li>
  {%- endif -%}
{%- endfor -%}
`

func benchData() map[string]any {
	users := make([]any, 100)
	for i := range users {
		users[i] = map[string]any{
			"name":   "User",
			"email":  "user@example.com",
			"active": i%2 == 0,
		}
	}
	return map[string]any{"users": users}
}

func BenchmarkParse(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		if _, err := Parse(benchSource); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRenderParsed(b *testing.B) {
	tmpl := MustParse(benchSource)
	data := benchData()
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		if _, err := tmpl.Render(data); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRenderTo(b *testing.B) {
	tmpl := MustParse(benchSource)
	data := benchData()
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		if err := tmpl.RenderTo(io.Discard, data); err != nil {
			b.Fatal(err)
		}
	}
}
