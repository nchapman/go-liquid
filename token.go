package liquid

// TokenType represents the type of a token in a Liquid template.
type TokenType int

const (
	// Special tokens
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
	tokenIdent  // identifier
	tokenInt    // integer
	tokenFloat  // floating point number
	tokenString // "string" or 'string'

	// Operators
	tokenDot      // .
	tokenComma    // ,
	tokenColon    // :
	tokenPipe     // |
	tokenLBracket // [
	tokenRBracket // ]
	tokenLParen   // (
	tokenRParen   // )
	tokenRange    // ..

	// Comparison operators
	tokenEq // ==
	tokenNe // != or <>
	tokenLt // <
	tokenGt // >
	tokenLe // <=
	tokenGe // >=

	// Arithmetic operators
	tokenMinus // -

	// Assignment
	tokenAssign // =

	// Keywords
	tokenIf
	tokenElsif
	tokenElse
	tokenEndif
	tokenUnless
	tokenEndunless
	tokenCase
	tokenWhen
	tokenEndcase
	tokenFor
	tokenIn
	tokenEndfor
	tokenBreak
	tokenContinue
	tokenAssignTag // assign
	tokenCapture
	tokenEndcapture
	tokenComment
	tokenEndcomment
	tokenRaw
	tokenEndraw
	tokenAnd
	tokenOr
	tokenContains
	tokenTrue
	tokenFalse
	tokenNil
	tokenEmpty
	tokenBlank
	tokenLimit
	tokenOffset
	tokenReversed
	tokenCycle
	tokenIncrement
	tokenDecrement
)

// token represents a token in a Liquid template.
type token struct {
	typ     TokenType
	literal string
	line    int
	column  int
}

var keywords = map[string]TokenType{
	"if":         tokenIf,
	"elsif":      tokenElsif,
	"else":       tokenElse,
	"endif":      tokenEndif,
	"unless":     tokenUnless,
	"endunless":  tokenEndunless,
	"case":       tokenCase,
	"when":       tokenWhen,
	"endcase":    tokenEndcase,
	"for":        tokenFor,
	"in":         tokenIn,
	"endfor":     tokenEndfor,
	"break":      tokenBreak,
	"continue":   tokenContinue,
	"assign":     tokenAssignTag,
	"capture":    tokenCapture,
	"endcapture": tokenEndcapture,
	"comment":    tokenComment,
	"endcomment": tokenEndcomment,
	"raw":        tokenRaw,
	"endraw":     tokenEndraw,
	"and":        tokenAnd,
	"or":         tokenOr,
	"contains":   tokenContains,
	"true":       tokenTrue,
	"false":      tokenFalse,
	"nil":        tokenNil,
	"null":       tokenNil, // alias
	"empty":      tokenEmpty,
	"blank":      tokenBlank,
	"limit":      tokenLimit,
	"offset":     tokenOffset,
	"reversed":   tokenReversed,
	"cycle":      tokenCycle,
	"increment":  tokenIncrement,
	"decrement":  tokenDecrement,
}

func lookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return tokenIdent
}
