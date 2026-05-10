package liquid

import (
	"errors"
	"fmt"
	"io"
	"testing"
)

// Mirrors Ruby Liquid's `template.render(data, registers: {...})`:
// custom tags reach the user-supplied state map via TagContext.Registers,
// and mutations stick after Render returns.
func TestRegistersAccessibleFromCustomTag(t *testing.T) {
	env := NewEnvironment()
	env.RegisterTag("touch", func(_ string) (TagRenderer, error) {
		return tagRendererFunc(func(w io.Writer, ctx TagContext) error {
			regs := ctx.Registers()
			if regs == nil {
				return errors.New("registers nil")
			}
			n, _ := regs["count"].(int)
			regs["count"] = n + 1
			_, err := fmt.Fprintf(w, "%d", regs["count"])
			return err
		}), nil
	})

	regs := map[string]any{"count": 0}
	out, err := env.Render(`{% touch %}-{% touch %}-{% touch %}`, nil, WithRegisters(regs))
	if err != nil {
		t.Fatal(err)
	}
	if out != "1-2-3" {
		t.Errorf("got %q, want %q", out, "1-2-3")
	}
	if regs["count"] != 3 {
		t.Errorf("caller-side registers not updated: count=%v", regs["count"])
	}
}

// Registers propagate into partials so a custom tag inside a {% render %}d
// partial sees the same map and can mutate it.
func TestRegistersPropagateIntoRenderPartial(t *testing.T) {
	env := NewEnvironment()
	env.RegisterTag("note", func(_ string) (TagRenderer, error) {
		return tagRendererFunc(func(w io.Writer, ctx TagContext) error {
			regs := ctx.Registers()
			regs["seen"] = append(regs["seen"].([]string), "inner")
			_, err := io.WriteString(w, "k")
			return err
		}), nil
	})
	env = env.WithLoader(MapLoader{"p": `{% note %}`})

	regs := map[string]any{"seen": []string{}}
	_, err := env.Render(`{% render "p" %}`, nil, WithRegisters(regs))
	if err != nil {
		t.Fatal(err)
	}
	got := regs["seen"].([]string)
	if len(got) != 1 || got[0] != "inner" {
		t.Errorf("got %v", got)
	}
}

// When no WithRegisters option is supplied, Registers returns nil so
// custom tags must nil-check (matching documented contract).
func TestRegistersNilWhenNotProvided(t *testing.T) {
	env := NewEnvironment()
	var saw map[string]any
	env.RegisterTag("peek", func(_ string) (TagRenderer, error) {
		return tagRendererFunc(func(w io.Writer, ctx TagContext) error {
			saw = ctx.Registers()
			return nil
		}), nil
	})
	if _, err := env.Render(`{% peek %}`, nil); err != nil {
		t.Fatal(err)
	}
	if saw != nil {
		t.Errorf("expected nil registers, got %v", saw)
	}
}

// Drops can reach Registers via RenderContext.
func TestRegistersAccessibleFromContextAwareDrop(t *testing.T) {
	d := &registerProbeDrop{}
	regs := map[string]any{"hello": "world"}
	out, err := Render(`{{ d.value }}`, map[string]any{"d": d}, WithRegisters(regs))
	if err != nil {
		t.Fatal(err)
	}
	if out != "world" {
		t.Errorf("got %q, want %q", out, "world")
	}
}

type registerProbeDrop struct {
	ctx RenderContext
}

func (d *registerProbeDrop) SetRenderContext(ctx RenderContext) { d.ctx = ctx }
func (d *registerProbeDrop) LiquidLookup(key string) (any, bool) {
	if key == "value" {
		regs := d.ctx.Registers()
		return regs["hello"], true
	}
	return nil, false
}
