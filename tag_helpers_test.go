package liquid

import (
	"fmt"
	"io"
	"strings"
	"testing"
)

// upcaseTag is a custom tag {% upcase EXPR %} that evaluates EXPR as a
// Liquid expression and emits the uppercased string. Used to verify the
// ParseExpression + TagContext.Eval pairing.
type upcaseRenderer struct{ expr Expression }

func (r upcaseRenderer) Render(w io.Writer, ctx TagContext) error {
	v, err := ctx.Eval(r.expr)
	if err != nil {
		return err
	}
	_, err = io.WriteString(w, strings.ToUpper(fmt.Sprint(v)))
	return err
}

func TestParseExpressionAndEval(t *testing.T) {
	env := NewEnvironment()
	env.RegisterTag("upcase", func(markup string) (TagRenderer, error) {
		expr, err := ParseExpression(markup)
		if err != nil {
			return nil, err
		}
		return upcaseRenderer{expr}, nil
	})
	tmpl, err := env.Parse(`{% upcase greeting | append: "!" %}`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	out, err := tmpl.Render(map[string]any{"greeting": "hello"})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if out != "HELLO!" {
		t.Errorf("got %q want %q", out, "HELLO!")
	}
}

func TestParseExpressionRejectsTrailingTokens(t *testing.T) {
	_, err := ParseExpression(`x y`)
	if err == nil {
		t.Fatal("expected parse error for trailing tokens")
	}
}

func TestEnvironmentParseExpressionUsesEnv(t *testing.T) {
	// A custom tag registered on env should use env.ParseExpression so
	// the parser binds the same env (matters once env-scoped state
	// affects parsing — for now this just verifies the API is wired).
	env := NewEnvironment()
	env.RegisterTag("upcase", func(markup string) (TagRenderer, error) {
		expr, err := env.ParseExpression(markup)
		if err != nil {
			return nil, err
		}
		return upcaseRenderer{expr}, nil
	})
	tmpl, err := env.Parse(`{% upcase "hi" %}`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	out, _ := tmpl.Render(nil)
	if out != "HI" {
		t.Errorf("got %q want %q", out, "HI")
	}
}

func TestParseExpressionLiteral(t *testing.T) {
	expr, err := ParseExpression(`42`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if _, ok := expr.(*LiteralExpr); !ok {
		t.Errorf("got %T, want *LiteralExpr", expr)
	}
}

func TestTagContextStrictVariables(t *testing.T) {
	env := NewEnvironment()
	var sawStrict bool
	env.RegisterTag("probe", func(markup string) (TagRenderer, error) {
		return tagFn(func(w io.Writer, ctx TagContext) error {
			sawStrict = ctx.StrictVariables()
			return nil
		}), nil
	})
	tmpl, _ := env.Parse(`{% probe %}`)
	_, _ = tmpl.Render(nil, StrictVariables())
	if !sawStrict {
		t.Error("StrictVariables() = false in tag, want true")
	}
}

func TestParseExpressionPropagatesEvalErrors(t *testing.T) {
	env := NewEnvironment()
	env.RegisterTag("upcase", func(markup string) (TagRenderer, error) {
		expr, err := ParseExpression(markup)
		if err != nil {
			return nil, err
		}
		return upcaseRenderer{expr}, nil
	})
	tmpl, _ := env.Parse(`{% upcase missing %}`)
	_, err := tmpl.Render(nil, StrictVariables())
	if err == nil {
		t.Fatal("expected strict-vars error")
	}
	if !strings.Contains(err.Error(), "missing") {
		t.Errorf("error %q does not mention missing var", err.Error())
	}
}

type tagFn func(w io.Writer, ctx TagContext) error

func (f tagFn) Render(w io.Writer, ctx TagContext) error { return f(w, ctx) }
