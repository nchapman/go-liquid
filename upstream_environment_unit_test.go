package liquid

import (
	"io"
	"testing"
)

// Upstream parity: unit/environment_test.rb

type envCustomTag struct{}

func (envCustomTag) Render(w io.Writer, ctx TagContext) error {
	_, err := io.WriteString(w, "custom-out")
	return err
}

func TestUpstream_EnvironmentUnit_CustomTag(t *testing.T) {
	env := NewEnvironment()
	env.RegisterTag("custom", func(markup string) (TagRenderer, error) {
		return envCustomTag{}, nil
	})
	tmpl, err := env.Parse("{% custom %}")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	got, err := tmpl.Render(nil)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if got != "custom-out" {
		t.Fatalf("got %q", got)
	}
}
