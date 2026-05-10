package liquid

import (
	"strings"
	"testing"
)

// ctxAwareDrop records the most recently bound RenderContext so tests can
// assert what the engine plumbed through.
type ctxAwareDrop struct {
	value      string
	gotCtx     RenderContext
	setCount   int
	failOnMiss bool
}

func (d *ctxAwareDrop) LiquidLookup(key string) (any, bool) {
	if key == "value" {
		return d.value, true
	}
	if d.failOnMiss && d.gotCtx != nil && d.gotCtx.StrictVariables() {
		return nil, false
	}
	return nil, true
}

func (d *ctxAwareDrop) SetRenderContext(ctx RenderContext) {
	d.gotCtx = ctx
	d.setCount++
}

func TestContextAwareDropReceivesContext(t *testing.T) {
	d := &ctxAwareDrop{value: "hi"}
	out, err := Render(`{{ d.value }}`, map[string]any{"d": d})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if out != "hi" {
		t.Errorf("got %q want %q", out, "hi")
	}
	if d.gotCtx == nil {
		t.Fatal("SetRenderContext was never called")
	}
	if d.setCount < 1 {
		t.Errorf("SetRenderContext called %d times, want >= 1", d.setCount)
	}
}

func TestContextAwareDropSeesStrictVariables(t *testing.T) {
	d := &ctxAwareDrop{value: "x"}
	_, _ = Render(`{{ d.value }}`, map[string]any{"d": d}, StrictVariables())
	if d.gotCtx == nil {
		t.Fatal("no context")
	}
	if !d.gotCtx.StrictVariables() {
		t.Error("StrictVariables() = false, want true")
	}
}

func TestContextAwareDropSeesLimits(t *testing.T) {
	d := &ctxAwareDrop{value: "x"}
	limits := &ResourceLimits{RenderLengthLimit: 100}
	_, _ = Render(`{{ d.value }}`, map[string]any{"d": d}, WithLimits(limits))
	if d.gotCtx == nil {
		t.Fatal("no context")
	}
	if d.gotCtx.ResourceLimits() != limits {
		t.Error("ResourceLimits() did not return the attached limits")
	}
}

func TestContextAwareDropReachableViaTopLevel(t *testing.T) {
	// `{% if drop %}` evaluates drop without any property descent —
	// context must still be set so the drop can adapt its truthiness.
	d := &ctxAwareDrop{value: "x"}
	_, _ = Render(`{% if d %}yes{% endif %}`, map[string]any{"d": d})
	if d.gotCtx == nil {
		t.Fatal("top-level identifier did not bind context")
	}
}

func TestContextAwareDropGetReadsScope(t *testing.T) {
	// A drop should be able to peek at sibling variables via Get.
	type peekDrop struct {
		ctx RenderContext
	}
	// Inline implementation using anonymous types is awkward; use a closure-y
	// implementation via a small helper.
	_ = peekDrop{}

	d := &peekDropImpl{}
	out, err := Render(`{{ d.peer }}`, map[string]any{"d": d, "peer": "neighbor"})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if out != "neighbor" {
		t.Errorf("got %q want %q", out, "neighbor")
	}
}

type peekDropImpl struct {
	ctx RenderContext
}

func (p *peekDropImpl) LiquidLookup(key string) (any, bool) {
	if p.ctx == nil {
		return nil, false
	}
	v, ok := p.ctx.Get(key)
	return v, ok
}

func (p *peekDropImpl) SetRenderContext(ctx RenderContext) { p.ctx = ctx }

func TestContextAwareDropChargesAssignScore(t *testing.T) {
	// A drop that does an "expensive" lookup can read the active limits
	// budget from context. Charging against the budget is observable via
	// Reached() / counter accessors after Render returns.
	d := &chargingDrop{cost: 100}
	limits := &ResourceLimits{AssignScoreLimit: 50}
	out, _ := Render(`{{ d.expensive }}`, map[string]any{"d": d}, WithLimits(limits))
	if !limits.Reached() {
		t.Errorf("expected limits.Reached() = true, got false (out=%q)", out)
	}
	if got := limits.AssignScore(); got != 100 {
		t.Errorf("AssignScore = %d, want 100", got)
	}
}

type chargingDrop struct {
	ctx  RenderContext
	cost int
}

func (d *chargingDrop) LiquidLookup(key string) (any, bool) {
	if d.ctx != nil {
		if l := d.ctx.ResourceLimits(); l != nil {
			_ = l.incrementAssignScore(d.cost)
		}
	}
	return "value", true
}

func (d *chargingDrop) SetRenderContext(ctx RenderContext) { d.ctx = ctx }

func TestPlainDropStillWorks(t *testing.T) {
	// A non-context-aware Drop continues to function with no behavior change.
	d := plainDrop{value: "ok"}
	out, err := Render(`{{ d.value }}`, map[string]any{"d": d})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if out != "ok" {
		t.Errorf("got %q want %q", out, "ok")
	}
}

type plainDrop struct{ value string }

func (d plainDrop) LiquidLookup(key string) (any, bool) {
	if key == "value" {
		return d.value, true
	}
	return nil, false
}

func TestContextAwareDropTemplateName(t *testing.T) {
	d := &ctxAwareDrop{value: "x"}
	tmpl := MustParse(`{{ d.value }}`).WithName("greetings.liquid")
	_, _ = tmpl.Render(map[string]any{"d": d})
	if d.gotCtx == nil {
		t.Fatal("no ctx")
	}
	if !strings.Contains(d.gotCtx.TemplateName(), "greetings") {
		t.Errorf("TemplateName = %q, want contains 'greetings'", d.gotCtx.TemplateName())
	}
}
