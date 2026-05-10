package liquid

import (
	"fmt"
	"strings"
)

// ParseExpression parses markup as a single Liquid expression against
// the default environment. It delegates to Default().ParseExpression;
// see that method for full documentation.
func ParseExpression(markup string) (Expression, error) {
	return Default().ParseExpression(markup)
}

// ParseExpression parses markup as a single Liquid expression — a
// variable reference, literal, comparison, or filter chain. Use it
// inside a TagParser so a custom tag registered on this Environment
// can accept Liquid syntax in its arguments without re-implementing
// the parser:
//
//	env.RegisterTag("greet", func(markup string) (liquid.TagRenderer, error) {
//	    expr, err := env.ParseExpression(markup)
//	    if err != nil { return nil, err }
//	    return greetRenderer{expr}, nil
//	})
//
// Evaluate the returned Expression at render time via TagContext.Eval.
//
// The whole markup must reduce to a single expression; trailing tokens
// produce a parse error. Whitespace at either end is trimmed. The
// parser is strict — there is no equivalent of Ruby's lax-mode
// safe_parse_expression that returns nil on bad input.
func (e *Environment) ParseExpression(markup string) (Expression, error) {
	env := e
	if env == nil {
		env = Default()
	}
	// Tag markup lives inside `{% %}`, so the lexer needs modeTag —
	// modeOutput would terminate on `}}` and miss `%}`-bearing input.
	// Either way our markup has the delimiters stripped already, but
	// modeTag is the semantically correct choice if the markup contains
	// `}}` inside a string literal or hash argument.
	l := newLexer(strings.TrimSpace(markup))
	l.mode = modeTag
	p := &parser{l: l, env: env}
	p.nextToken()
	expr, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	if p.curToken.typ != tokenEOF {
		return nil, fmt.Errorf("liquid: unexpected %q after expression in %q",
			p.curToken.literal, markup)
	}
	return expr, nil
}
