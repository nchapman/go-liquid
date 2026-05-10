package liquid

// TokenType represents the type of a token in a Liquid template.
type TokenType int

// Tokens fall into three groups:
//
//   - Structural — delimiters, punctuation, literals, identifiers. Emitted
//     directly by the lexer.
//   - Expression-context keywords — `and`, `or`, `contains`, `true`, `false`,
//     `nil`/`null`, `empty`, `blank`. These are globally reserved because
//     they have no other valid interpretation inside an expression.
//   - Tag/sub-keyword words — `if`, `for`, `assign`, `render`, `with`, `as`,
//     `limit`, `endif`, etc. These are NOT reserved tokens; the lexer emits
//     them as `tokenIdent` and the parser recognizes them by literal text in
//     the structural positions where they are meaningful. This lets users
//     write things like `{{ raw }}` or `{% for if in items %}` where the word
//     is plainly a variable name.
const (
	tokenEOF TokenType = iota
	tokenIllegal
	tokenText // raw text outside of tags

	// Delimiters
	tokenOutputOpen  // {{
	tokenOutputClose // }}
	tokenTagOpen     // {%
	tokenTagClose    // %}
	tokenOutputTrim  // {{-
	tokenOutputTrimR // -}}
	tokenTagTrim     // {%-
	tokenTagTrimR    // -%}

	// Literals
	tokenIdent
	tokenInt
	tokenFloat
	tokenString

	// Operators / punctuation
	tokenDot      // .
	tokenComma    // ,
	tokenColon    // :
	tokenPipe     // |
	tokenLBracket // [
	tokenRBracket // ]
	tokenLParen   // (
	tokenRParen   // )
	tokenRange    // ..
	tokenHash     // # (only meaningful at the start of a tag: {% # comment %})

	// Comparison
	tokenEq // ==
	tokenNe // != or <>
	tokenLt // <
	tokenGt // >
	tokenLe // <=
	tokenGe // >=

	// Arithmetic
	tokenMinus // -

	// Assignment
	tokenAssign // =

	// Expression-context keywords (truly reserved everywhere)
	tokenAnd
	tokenOr
	tokenContains
	tokenTrue
	tokenFalse
	tokenNil
	tokenEmpty
	tokenBlank
)

// token is a single lexed unit.
type token struct {
	typ     TokenType
	literal string
	line    int
	column  int
}

// lookupIdent classifies an identifier literal as either an
// expression-context keyword (truly reserved everywhere) or a plain
// identifier. Tag names (`if`, `for`, `endif`, etc.) and sub-keywords
// (`in`, `with`, `as`, `limit`, `offset`, `reversed`, `when`) are NOT
// reserved here — they remain plain identifiers and are recognized by
// the parser only at structural positions.
//
// A length-bucketed switch is faster than a map lookup for this small,
// fixed set, and identifiers are scanned in the lexer's hot path.
func lookupIdent(ident string) TokenType {
	switch len(ident) {
	case 2:
		if ident == "or" {
			return tokenOr
		}
	case 3:
		switch ident {
		case "and":
			return tokenAnd
		case "nil":
			return tokenNil
		}
	case 4:
		switch ident {
		case "true":
			return tokenTrue
		case "null":
			return tokenNil
		}
	case 5:
		switch ident {
		case "false":
			return tokenFalse
		case "empty":
			return tokenEmpty
		case "blank":
			return tokenBlank
		}
	case 8:
		if ident == "contains" {
			return tokenContains
		}
	}
	return tokenIdent
}
