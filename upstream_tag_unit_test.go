package liquid

import (
	"io"
	"testing"
)

// Upstream parity: unit/tag_unit_test.rb

type rawTextTag struct{ raw string }

func (r rawTextTag) Render(w io.Writer, ctx TagContext) error {
	_, err := io.WriteString(w, r.raw)
	return err
}

func TestUpstream_TagUnit_Tag(t *testing.T) {
	env := NewEnvironment()
	env.RegisterTag("noop", func(markup string) (TagRenderer, error) {
		return rawTextTag{raw: ""}, nil
	})
	if _, err := env.Parse("{% noop %}"); err != nil {
		t.Fatalf("parse: %v", err)
	}
}

func TestUpstream_TagUnit_ReturnRawTextOfTag(t *testing.T) {
	// Custom tag receives raw markup and can echo it.
	env := NewEnvironment()
	env.RegisterTag("raw_markup", func(markup string) (TagRenderer, error) {
		return rawTextTag{raw: markup}, nil
	})
	tmpl, err := env.Parse("{% raw_markup hello world %}")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	got, err := tmpl.Render(nil)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if got != " hello world " {
		t.Fatalf("got %q (markup should include surrounding spaces verbatim)", got)
	}
}

func TestUpstream_TagUnit_TagNameShouldReturnNameOfTheTag(t *testing.T) {
	t.Skip("Ruby Tag#tag_name attribute not exposed in go-liquid TagRenderer interface")
}

func TestUpstream_TagUnit_TagRenderToOutputBufferNilValue(t *testing.T) {
	// Tag that writes nothing renders as empty.
	env := NewEnvironment()
	env.RegisterTag("nilout", func(markup string) (TagRenderer, error) {
		return rawTextTag{raw: ""}, nil
	})
	renderEnvEq(t, env, "{% nilout %}", nil, "")
}

// renderEnvEq is a small helper specific to env-scoped tag tests.
func renderEnvEq(t *testing.T, env *Environment, src string, data map[string]any, want string) {
	t.Helper()
	tmpl, err := env.Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	got, err := tmpl.Render(data)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if got != want {
		t.Fatalf("want %q got %q", want, got)
	}
}
