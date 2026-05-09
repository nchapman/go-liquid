package liquid

// lexerMode represents the current lexing mode.
type lexerMode int

const (
	modeText   lexerMode = iota // outside of any tags
	modeOutput                  // inside {{ }}
	modeTag                     // inside {% %}
)

// lexer tokenizes a Liquid template string.
type lexer struct {
	input   string
	pos     int       // current position in input
	readPos int       // next position to read
	ch      byte      // current character
	line    int       // current line (1-indexed)
	column  int       // current column (1-indexed)
	mode    lexerMode // current lexing mode
}

func newLexer(input string) *lexer {
	l := &lexer{
		input:  input,
		line:   1,
		column: 0,
	}
	l.readChar()
	return l
}

func (l *lexer) readChar() {
	if l.readPos >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.readPos]
	}
	l.pos = l.readPos
	l.readPos++
	if l.ch == '\n' {
		l.line++
		l.column = 0
	} else {
		l.column++
	}
}

func (l *lexer) peekChar() byte {
	if l.readPos >= len(l.input) {
		return 0
	}
	return l.input[l.readPos]
}

func (l *lexer) peekCharN(n int) byte {
	pos := l.readPos + n - 1
	if pos >= len(l.input) {
		return 0
	}
	return l.input[pos]
}

func (l *lexer) nextToken() token {
	switch l.mode {
	case modeText:
		return l.nextTextToken()
	case modeOutput:
		return l.nextOutputToken()
	case modeTag:
		return l.nextTagToken()
	default:
		return token{typ: tokenEOF, line: l.line, column: l.column}
	}
}

// nextTextToken scans text until we hit {{ or {% or EOF.
func (l *lexer) nextTextToken() token {
	if l.ch == 0 {
		return token{typ: tokenEOF, line: l.line, column: l.column}
	}

	startLine := l.line
	startCol := l.column
	startPos := l.pos

	for l.ch != 0 {
		// Check for inline comment {# ... #} - skip entirely
		if l.ch == '{' && l.peekChar() == '#' {
			// If we have accumulated text, return it first
			if l.pos > startPos {
				return token{
					typ:     tokenText,
					literal: l.input[startPos:l.pos],
					line:    startLine,
					column:  startCol,
				}
			}
			// Skip the entire comment
			l.readChar() // consume {
			l.readChar() // consume #
			for l.ch != 0 {
				if l.ch == '#' && l.peekChar() == '}' {
					l.readChar() // consume #
					l.readChar() // consume }
					break
				}
				l.readChar()
			}
			// Also consume trailing newline if the comment was on its own line
			// This prevents blank lines from inline comments
			if l.ch == '\n' {
				l.readChar()
			} else if l.ch == '\r' && l.peekChar() == '\n' {
				l.readChar()
				l.readChar()
			}
			// Continue scanning text after the comment
			startLine = l.line
			startCol = l.column
			startPos = l.pos
			continue
		}

		// Check for output start {{ or {{-
		if l.ch == '{' && l.peekChar() == '{' {
			// If we have accumulated text, return it first
			if l.pos > startPos {
				return token{
					typ:     tokenText,
					literal: l.input[startPos:l.pos],
					line:    startLine,
					column:  startCol,
				}
			}
			// Check for trim marker
			tokLine := l.line
			tokCol := l.column
			l.readChar() // consume first {
			l.readChar() // consume second {
			if l.ch == '-' {
				l.readChar() // consume -
				l.mode = modeOutput
				return token{typ: tokenOutputTrim, line: tokLine, column: tokCol}
			}
			l.mode = modeOutput
			return token{typ: tokenOutputOpen, line: tokLine, column: tokCol}
		}

		// Check for tag start {% or {%-
		if l.ch == '{' && l.peekChar() == '%' {
			// If we have accumulated text, return it first
			if l.pos > startPos {
				return token{
					typ:     tokenText,
					literal: l.input[startPos:l.pos],
					line:    startLine,
					column:  startCol,
				}
			}
			// Check for trim marker
			tokLine := l.line
			tokCol := l.column
			l.readChar() // consume {
			l.readChar() // consume %
			if l.ch == '-' {
				l.readChar() // consume -
				l.mode = modeTag
				return token{typ: tokenTagTrim, line: tokLine, column: tokCol}
			}
			l.mode = modeTag
			return token{typ: tokenTagOpen, line: tokLine, column: tokCol}
		}

		l.readChar()
	}

	// Return any remaining text
	if l.pos > startPos {
		return token{
			typ:     tokenText,
			literal: l.input[startPos:l.pos],
			line:    startLine,
			column:  startCol,
		}
	}

	return token{typ: tokenEOF, line: l.line, column: l.column}
}

// nextOutputToken scans tokens inside {{ }}.
func (l *lexer) nextOutputToken() token {
	l.skipWhitespace()

	tokLine := l.line
	tokCol := l.column

	// Check for close }} or -}}
	if l.ch == '-' && l.peekChar() == '}' && l.peekCharN(2) == '}' {
		l.readChar() // consume -
		l.readChar() // consume }
		l.readChar() // consume }
		l.mode = modeText
		return token{typ: tokenOutputTrimR, line: tokLine, column: tokCol}
	}
	if l.ch == '}' && l.peekChar() == '}' {
		l.readChar() // consume }
		l.readChar() // consume }
		l.mode = modeText
		return token{typ: tokenOutputClose, line: tokLine, column: tokCol}
	}

	return l.scanExpression(tokLine, tokCol)
}

// nextTagToken scans tokens inside {% %}.
func (l *lexer) nextTagToken() token {
	l.skipWhitespace()

	tokLine := l.line
	tokCol := l.column

	// Check for close %} or -%}
	if l.ch == '-' && l.peekChar() == '%' && l.peekCharN(2) == '}' {
		l.readChar() // consume -
		l.readChar() // consume %
		l.readChar() // consume }
		l.mode = modeText
		return token{typ: tokenTagTrimR, line: tokLine, column: tokCol}
	}
	if l.ch == '%' && l.peekChar() == '}' {
		l.readChar() // consume %
		l.readChar() // consume }
		l.mode = modeText
		return token{typ: tokenTagClose, line: tokLine, column: tokCol}
	}

	return l.scanExpression(tokLine, tokCol)
}

// scanExpression scans expression tokens common to both output and tag modes.
func (l *lexer) scanExpression(line, col int) token {
	if l.ch == 0 {
		return token{typ: tokenEOF, line: line, column: col}
	}

	switch l.ch {
	case '.':
		if l.peekChar() == '.' {
			l.readChar()
			l.readChar()
			return token{typ: tokenRange, literal: "..", line: line, column: col}
		}
		l.readChar()
		return token{typ: tokenDot, literal: ".", line: line, column: col}
	case ',':
		l.readChar()
		return token{typ: tokenComma, literal: ",", line: line, column: col}
	case ':':
		l.readChar()
		return token{typ: tokenColon, literal: ":", line: line, column: col}
	case '|':
		l.readChar()
		return token{typ: tokenPipe, literal: "|", line: line, column: col}
	case '[':
		l.readChar()
		return token{typ: tokenLBracket, literal: "[", line: line, column: col}
	case ']':
		l.readChar()
		return token{typ: tokenRBracket, literal: "]", line: line, column: col}
	case '(':
		l.readChar()
		return token{typ: tokenLParen, literal: "(", line: line, column: col}
	case ')':
		l.readChar()
		return token{typ: tokenRParen, literal: ")", line: line, column: col}
	case '=':
		if l.peekChar() == '=' {
			l.readChar()
			l.readChar()
			return token{typ: tokenEq, literal: "==", line: line, column: col}
		}
		l.readChar()
		return token{typ: tokenAssign, literal: "=", line: line, column: col}
	case '!':
		if l.peekChar() == '=' {
			l.readChar()
			l.readChar()
			return token{typ: tokenNe, literal: "!=", line: line, column: col}
		}
		l.readChar()
		return token{typ: tokenIllegal, literal: "!", line: line, column: col}
	case '<':
		if l.peekChar() == '=' {
			l.readChar()
			l.readChar()
			return token{typ: tokenLe, literal: "<=", line: line, column: col}
		}
		if l.peekChar() == '>' {
			l.readChar()
			l.readChar()
			return token{typ: tokenNe, literal: "<>", line: line, column: col}
		}
		l.readChar()
		return token{typ: tokenLt, literal: "<", line: line, column: col}
	case '>':
		if l.peekChar() == '=' {
			l.readChar()
			l.readChar()
			return token{typ: tokenGe, literal: ">=", line: line, column: col}
		}
		l.readChar()
		return token{typ: tokenGt, literal: ">", line: line, column: col}
	case '-':
		l.readChar()
		return token{typ: tokenMinus, literal: "-", line: line, column: col}
	case '"', '\'':
		return l.scanString()
	default:
		if isDigit(l.ch) {
			return l.scanNumber()
		}
		if isLetter(l.ch) || l.ch == '_' {
			return l.scanIdentifier()
		}
		ch := l.ch
		l.readChar()
		return token{typ: tokenIllegal, literal: string(ch), line: line, column: col}
	}
}

func (l *lexer) scanString() token {
	line := l.line
	col := l.column
	quote := l.ch
	l.readChar() // consume opening quote

	var literal []byte
	for l.ch != 0 && l.ch != quote {
		if l.ch == '\\' && l.peekChar() != 0 {
			l.readChar()
			switch l.ch {
			case 'n':
				literal = append(literal, '\n')
			case 't':
				literal = append(literal, '\t')
			case 'r':
				literal = append(literal, '\r')
			case '\\':
				literal = append(literal, '\\')
			case '"':
				literal = append(literal, '"')
			case '\'':
				literal = append(literal, '\'')
			default:
				literal = append(literal, '\\', l.ch)
			}
		} else {
			literal = append(literal, l.ch)
		}
		l.readChar()
	}
	l.readChar() // consume closing quote
	return token{typ: tokenString, literal: string(literal), line: line, column: col}
}

func (l *lexer) scanNumber() token {
	line := l.line
	col := l.column
	startPos := l.pos
	isFloat := false

	for isDigit(l.ch) {
		l.readChar()
	}

	if l.ch == '.' && isDigit(l.peekChar()) {
		isFloat = true
		l.readChar() // consume .
		for isDigit(l.ch) {
			l.readChar()
		}
	}

	literal := l.input[startPos:l.pos]
	if isFloat {
		return token{typ: tokenFloat, literal: literal, line: line, column: col}
	}
	return token{typ: tokenInt, literal: literal, line: line, column: col}
}

func (l *lexer) scanIdentifier() token {
	line := l.line
	col := l.column
	startPos := l.pos

	for isLetter(l.ch) || isDigit(l.ch) || l.ch == '_' || l.ch == '?' {
		l.readChar()
	}

	literal := l.input[startPos:l.pos]
	typ := lookupIdent(literal)
	return token{typ: typ, literal: literal, line: line, column: col}
}

func (l *lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\r' || l.ch == '\n' {
		l.readChar()
	}
}

func isLetter(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
}

func isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}

// scanRawBlock scans until we find {% endraw %} or {%- endraw -%} and returns the raw content.
// Called after {% raw %} has been parsed. Returns (content, line, col, trimRight).
func (l *lexer) scanRawBlock() (string, int, int, bool) {
	startPos := l.pos
	startLine := l.line
	startCol := l.column

	for l.ch != 0 {
		// Look for {% endraw %} or {%- endraw -%}
		if l.ch == '{' && l.peekChar() == '%' {
			savePos := l.pos

			l.readChar() // {
			l.readChar() // %

			// Skip optional - and whitespace
			for l.ch == ' ' || l.ch == '\t' || l.ch == '-' {
				l.readChar()
			}

			// Check for "endraw"
			if l.ch == 'e' && l.matchAhead("ndraw") {
				l.readChar() // consume 'e'
				for range 5 {
					l.readChar()
				}

				// Skip whitespace
				for l.ch == ' ' || l.ch == '\t' {
					l.readChar()
				}

				// Check for -%} or %}
				trimRight := false
				if l.ch == '-' {
					trimRight = true
					l.readChar()
				}
				if l.ch == '%' && l.peekChar() == '}' {
					l.readChar() // %
					l.readChar() // }
					l.mode = modeText
					return l.input[startPos:savePos], startLine, startCol, trimRight
				}
			}

			// Not endraw, continue from after {%
		} else {
			l.readChar()
		}
	}

	// EOF reached without finding endraw
	return l.input[startPos:l.pos], startLine, startCol, false
}

// matchAhead checks if the next n characters match the given string.
func (l *lexer) matchAhead(s string) bool {
	for i := range len(s) {
		pos := l.readPos + i
		if pos >= len(l.input) || l.input[pos] != s[i] {
			return false
		}
	}
	return true
}

// scanCommentBlock scans until we find {% endcomment %} or {%- endcomment -%} and returns the content.
// Called after {% comment %} has been parsed. Returns (content, line, col, trimRight).
func (l *lexer) scanCommentBlock() (string, int, int, bool) {
	startPos := l.pos
	startLine := l.line
	startCol := l.column

	for l.ch != 0 {
		// Look for {% endcomment %} or {%- endcomment -%}
		if l.ch == '{' && l.peekChar() == '%' {
			savePos := l.pos

			l.readChar() // {
			l.readChar() // %

			// Skip whitespace and optional -
			for l.ch == ' ' || l.ch == '\t' || l.ch == '-' {
				l.readChar()
			}

			// Check for "endcomment"
			if l.ch == 'e' && l.matchAhead("ndcomment") {
				l.readChar() // consume 'e'
				for range 9 {
					l.readChar()
				}

				// Skip whitespace
				for l.ch == ' ' || l.ch == '\t' {
					l.readChar()
				}

				// Check for -%} or %}
				trimRight := false
				if l.ch == '-' {
					trimRight = true
					l.readChar()
				}
				if l.ch == '%' && l.peekChar() == '}' {
					l.readChar() // %
					l.readChar() // }
					l.mode = modeText
					return l.input[startPos:savePos], startLine, startCol, trimRight
				}
			}

			// Not endcomment, continue searching
		}
		l.readChar()
	}

	// EOF reached without finding endcomment
	return l.input[startPos:l.pos], startLine, startCol, false
}
