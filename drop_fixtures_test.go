package liquid

import (
	"fmt"
	"strconv"
)

// Drop fixtures for upstream-ported tests. Each type mirrors a small
// Ruby Drop helper from Ruby Liquid's test_helper.rb so the un-skipped
// tests assert the same observable behavior.

// thingWithToLiquid is the Go equivalent of Ruby's `ThingWithToLiquid`:
// a non-Drop value that opts into a Liquid representation via the
// to_liquid hook.
type thingWithToLiquid struct{}

func (thingWithToLiquid) ToLiquid() any { return "foobar" }

// integerDrop is the Go equivalent of Ruby's `IntegerDrop`. The
// underlying value is an int; ToLiquidValue returns it for arithmetic
// and comparison, while String() renders the integer's decimal form for
// `{{ }}` output.
type integerDrop struct{ v int }

func (d *integerDrop) LiquidLookup(key string) (any, bool) { return nil, false }
func (d *integerDrop) ToLiquidValue() any                  { return d.v }
func (d *integerDrop) String() string                      { return strconv.Itoa(d.v) }

// booleanDrop is the Go equivalent of Ruby's `BooleanDrop`. ToLiquidValue
// returns the bool so `{% if foo == true %}` resolves; String renders
// "Yay"/"Nay" so `{{ foo }}` produces those words.
type booleanDrop struct{ v bool }

func (d *booleanDrop) LiquidLookup(key string) (any, bool) { return nil, false }
func (d *booleanDrop) ToLiquidValue() any                  { return d.v }
func (d *booleanDrop) String() string {
	if d.v {
		return "Yay"
	}
	return "Nay"
}

// settingsDrop mirrors Ruby's `SettingsDrop` — a Drop backed by an
// arbitrary key/value map that exposes the map's entries as properties.
type settingsDrop struct{ m map[string]any }

func (d *settingsDrop) LiquidLookup(key string) (any, bool) {
	v, ok := d.m[key]
	return v, ok
}

// customToLiquidDrop mirrors Ruby's `CustomToLiquidDrop`: a Drop that
// hands off its representation via to_liquid, returning the wrapped
// value as the Liquid surface.
type customToLiquidDrop struct{ v any }

func (d *customToLiquidDrop) LiquidLookup(key string) (any, bool) { return nil, false }
func (d *customToLiquidDrop) ToLiquid() any                       { return d.v }

// testEnumerable mirrors Ruby's `TestEnumerable` — a Drop that becomes
// a fixed slice of hashes when iterated.
type testEnumerable struct{}

func (testEnumerable) LiquidLookup(key string) (any, bool) { return nil, false }
func (testEnumerable) LiquidEach() []any {
	return []any{
		map[string]any{"foo": 1, "bar": 2},
		map[string]any{"foo": 2, "bar": 1},
		map[string]any{"foo": 3, "bar": 3},
	}
}

// hundredCentes mirrors Ruby's `HundredCentes`: a plain value (not a
// Drop) that opts into a Liquid representation of literal 100.
type hundredCentes struct{}

func (hundredCentes) ToLiquid() any { return 100 }

// centsDrop mirrors Ruby's `CentsDrop`: a Drop with an `amount`
// property that returns a hundredCentes value (which itself runs
// through to_liquid on access).
type centsDrop struct{}

func (centsDrop) LiquidLookup(key string) (any, bool) {
	if key == "amount" {
		return hundredCentes{}, true
	}
	return nil, false
}

// contextSensitiveDrop mirrors Ruby's `ContextSensitiveDrop`: its
// `test` property reads a sibling variable from the active render
// context.
type contextSensitiveDrop struct{ ctx RenderContext }

func (d *contextSensitiveDrop) SetRenderContext(ctx RenderContext) { d.ctx = ctx }
func (d *contextSensitiveDrop) LiquidLookup(key string) (any, bool) {
	if key == "test" && d.ctx != nil {
		return d.ctx.Get("test")
	}
	return nil, false
}

// errorDrop mirrors Ruby's `ErrorDrop`: a Drop whose accessors return
// tagged inline errors. Renders as `Liquid error: ...` /
// `Liquid syntax error: ...` in the output without aborting the render.
type errorDrop struct{}

func (errorDrop) LiquidLookup(key string) (any, bool) {
	switch key {
	case "standard_error":
		return NewStandardError("standard error"), true
	case "argument_error":
		return NewArgumentError("argument error"), true
	case "syntax_error":
		return NewSyntaxError("syntax error"), true
	}
	return nil, false
}

// testThing mirrors Ruby's `TestThing` from standard_filter_test.rb:
// not a Drop, but opts into a Liquid representation that increments an
// internal counter on each access and renders as "woot: N". Used to
// verify that filter pipelines (map, sort, first, last, truncate) run
// to_liquid on every element they touch.
type testThing struct{ foo int }

func (t *testThing) ToLiquid() any {
	t.foo++
	return t
}
func (t *testThing) LiquidLookup(_ string) (any, bool) { return t.String(), true }
func (t *testThing) String() string                    { return fmt.Sprintf("woot: %d", t.foo) }
