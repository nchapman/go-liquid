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

// expressionKeywords maps the small set of words that are reserved in every
// position because they have no other valid expression interpretation. Tag
// names (`if`, `for`, `endif`, etc.) and sub-keywords (`in`, `with`, `as`,
// `limit`, `offset`, `reversed`, `when`) are NOT here — they remain plain
// identifiers and are recognized by the parser only at structural positions.
var expressionKeywords = map[string]TokenType{
	"and":      tokenAnd,
	"or":       tokenOr,
	"contains": tokenContains,
	"true":     tokenTrue,
	"false":    tokenFalse,
	"nil":      tokenNil,
	"null":     tokenNil,
	"empty":    tokenEmpty,
	"blank":    tokenBlank,
}

func lookupIdent(ident string) TokenType {
	if tok, ok := expressionKeywords[ident]; ok {
		return tok
	}
	return tokenIdent
}
