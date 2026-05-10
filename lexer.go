package liquid

import "strings"

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

// seekTo bulk-advances the lexer state to targetPos. Equivalent to calling
// readChar (targetPos - l.pos) times, but updates line/column in batch by
// scanning the traversed bytes for newlines.
//
// Preconditions: l.ch != 0 (i.e. l.pos < len(l.input)) and
// targetPos < len(l.input). Violating either panics with an out-of-bounds
// index — callers crossing the EOF boundary must use seekToEnd instead so
// the ch=0 / column++ semantics match readChar.
func (l *lexer) seekTo(targetPos int) {
	if targetPos <= l.pos {
		return
	}
	// readChar() iterated (targetPos - l.pos) times sets ch to each of
	// input[l.pos+1 .. targetPos] in turn. Those are the bytes that update
	// line/column.
	traversed := l.input[l.pos+1 : targetPos+1]
	if nlIdx := strings.LastIndexByte(traversed, '\n'); nlIdx >= 0 {
		l.line += strings.Count(traversed, "\n")
		l.column = len(traversed) - 1 - nlIdx
	} else {
		l.column += len(traversed)
	}
	l.pos = targetPos
	l.readPos = targetPos + 1
	l.ch = l.input[targetPos]
}

// seekToEnd bulk-advances the lexer to EOF (l.ch = 0, l.pos = len(input)).
// Equivalent to calling readChar until ch=0.
func (l *lexer) seekToEnd() {
	if l.pos >= len(l.input) {
		l.ch = 0
		l.pos = len(l.input)
		l.readPos = len(l.input) + 1
		return
	}
	// Bytes that became ch via readChar: input[l.pos+1 .. len-1] (each
	// updates line/col), then a final readChar sets ch=0 with column++.
	rest := l.input[l.pos+1:]
	if nlIdx := strings.LastIndexByte(rest, '\n'); nlIdx >= 0 {
		l.line += strings.Count(rest, "\n")
		l.column = (len(rest) - 1 - nlIdx) + 1
	} else {
		l.column += len(rest) + 1
	}
	l.pos = len(l.input)
	l.readPos = len(l.input) + 1
	l.ch = 0
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
//
// Hot path: most of a Liquid template is plain text. We avoid the per-byte
// readChar loop by jumping to the next '{' with strings.IndexByte and only
// updating line/column once over the skipped run.
func (l *lexer) nextTextToken() token {
	if l.ch == 0 {
		return token{typ: tokenEOF, line: l.line, column: l.column}
	}

	startLine := l.line
	startCol := l.column
	startPos := l.pos

	for l.ch != 0 {
		if l.ch != '{' {
			rel := strings.IndexByte(l.input[l.readPos:], '{')
			if rel < 0 {
				// No more '{' anywhere — consume rest as text.
				l.seekToEnd()
				break
			}
			l.seekTo(l.readPos + rel)
			// l.ch is now '{'.
		}

		p := l.peekChar()

		// Inline comment {# ... #} — skip entirely, then continue scanning text.
		if p == '#' {
			if l.pos > startPos {
				return token{
					typ:     tokenText,
					literal: l.input[startPos:l.pos],
					line:    startLine,
					column:  startCol,
				}
			}
			l.readChar() // {
			l.readChar() // #
			for l.ch != 0 {
				if l.ch == '#' && l.peekChar() == '}' {
					l.readChar() // #
					l.readChar() // }
					break
				}
				l.readChar()
			}
			// Trim a trailing newline so an inline comment on its own line
			// doesn't leave a blank line behind.
			if l.ch == '\n' {
				l.readChar()
			} else if l.ch == '\r' && l.peekChar() == '\n' {
				l.readChar()
				l.readChar()
			}
			startLine = l.line
			startCol = l.column
			startPos = l.pos
			continue
		}

		// Output start {{ or {{-
		if p == '{' {
			if l.pos > startPos {
				return token{
					typ:     tokenText,
					literal: l.input[startPos:l.pos],
					line:    startLine,
					column:  startCol,
				}
			}
			tokLine := l.line
			tokCol := l.column
			l.readChar() // {
			l.readChar() // {
			if l.ch == '-' {
				l.readChar()
				l.mode = modeOutput
				return token{typ: tokenOutputTrim, line: tokLine, column: tokCol}
			}
			l.mode = modeOutput
			return token{typ: tokenOutputOpen, line: tokLine, column: tokCol}
		}

		// Tag start {% or {%-
		if p == '%' {
			if l.pos > startPos {
				return token{
					typ:     tokenText,
					literal: l.input[startPos:l.pos],
					line:    startLine,
					column:  startCol,
				}
			}
			tokLine := l.line
			tokCol := l.column
			l.readChar() // {
			l.readChar() // %
			if l.ch == '-' {
				l.readChar()
				l.mode = modeTag
				return token{typ: tokenTagTrim, line: tokLine, column: tokCol}
			}
			l.mode = modeTag
			return token{typ: tokenTagOpen, line: tokLine, column: tokCol}
		}

		// Lone '{' — consume and resume scanning.
		l.readChar()
	}

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
	case '#':
		l.readChar()
		return token{typ: tokenHash, literal: "#", line: line, column: col}
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

// scanString scans a quoted string literal. Strings without escape sequences
// (the common case) are returned as direct substring slices of the input;
// only strings containing backslash escapes pay for an unescape pass.
func (l *lexer) scanString() token {
	line := l.line
	col := l.column
	quote := l.ch
	l.readChar() // consume opening quote

	startPos := l.pos
	src := l.input

	// Fast scan: walk to the closing quote or first backslash.
	for l.ch != 0 && l.ch != quote && l.ch != '\\' {
		l.readChar()
	}

	if l.ch == quote {
		// No escapes — slice the substring directly.
		literal := src[startPos:l.pos]
		l.readChar() // closing quote
		return token{typ: tokenString, literal: literal, line: line, column: col}
	}

	// Slow path: copy what we have, then process escapes byte by byte.
	literal := make([]byte, 0, len(src)-startPos)
	literal = append(literal, src[startPos:l.pos]...)
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
	l.readChar() // closing quote
	return token{typ: tokenString, literal: string(literal), line: line, column: col}
}

// scanNumber scans an integer or float literal. Operates directly on input
// indices to avoid the per-byte readChar loop.
func (l *lexer) scanNumber() token {
	line := l.line
	col := l.column
	startPos := l.pos
	src := l.input
	n := len(src)
	p := l.pos

	for p < n && isDigit(src[p]) {
		p++
	}

	isFloat := false
	if p+1 < n && src[p] == '.' && isDigit(src[p+1]) {
		isFloat = true
		p++ // consume .
		for p < n && isDigit(src[p]) {
			p++
		}
	}

	literal := src[startPos:p]
	// No newlines can appear in a number — column update is a simple delta.
	l.column += p - l.pos
	l.pos = p
	l.readPos = p + 1
	if p >= n {
		l.ch = 0
	} else {
		l.ch = src[p]
	}

	if isFloat {
		return token{typ: tokenFloat, literal: literal, line: line, column: col}
	}
	return token{typ: tokenInt, literal: literal, line: line, column: col}
}

// scanIdentifier scans an identifier. Operates directly on input indices to
// avoid the per-byte readChar loop. The first char is already known to be a
// letter or underscore.
//
// Ruby Liquid's IDENTIFIER regex allows hyphens between identifier chars
// (`my-var`), and a single trailing `?` (`available?`). Hyphens are tricky
// because `-` is also the minus operator and the trim marker, so we only
// consume a hyphen when an identifier char follows it.
func (l *lexer) scanIdentifier() token {
	line := l.line
	col := l.column
	startPos := l.pos
	src := l.input
	n := len(src)
	p := l.pos + 1 // first char is verified by caller

scan:
	for p < n {
		c := src[p]
		switch {
		case isLetter(c) || isDigit(c) || c == '_':
			p++
		// p > startPos here (p starts at l.pos+1), so the original
		// "no leading hyphen" rule holds automatically. The p+1 < n
		// guard just keeps the src[p+1] peek in bounds at end-of-input.
		case c == '-' && p+1 < n:
			nx := src[p+1]
			if isLetter(nx) || isDigit(nx) || nx == '_' {
				p += 2
			} else {
				break scan
			}
		case c == '?':
			p++
			break scan
		default:
			break scan
		}
	}

	literal := src[startPos:p]
	// No newlines can appear in an identifier.
	l.column += p - l.pos
	l.pos = p
	l.readPos = p + 1
	if p >= n {
		l.ch = 0
	} else {
		l.ch = src[p]
	}
	return token{typ: lookupIdent(literal), literal: literal, line: line, column: col}
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
// Called after {% raw %} has been parsed. Returns (content, line, col, trimRight, closed)
// where closed reports whether the matching endraw was found before EOF.
func (l *lexer) scanRawBlock() (string, int, int, bool, bool) {
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
					return l.input[startPos:savePos], startLine, startCol, trimRight, true
				}
			}

			// Not endraw, continue from after {%
		} else {
			l.readChar()
		}
	}

	// EOF reached without finding endraw
	return l.input[startPos:l.pos], startLine, startCol, false, false
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

// scanCommentBlock scans the body of {% comment %} until it finds the
// matching {% endcomment %} (or trim variant), tracking nesting so that
// `{% comment %}{% comment %}…{% endcomment %}{% endcomment %}` correctly
// pairs each opener with its closer (matches Shopify's comment.rb).
// `{% raw %}…{% endraw %}` blocks within the body are skipped over so an
// `{% endcomment %}` token inside raw text doesn't terminate prematurely.
func (l *lexer) scanCommentBlock() (string, int, int, bool) {
	startPos := l.pos
	startLine := l.line
	startCol := l.column
	depth := 1

	for l.ch != 0 {
		if l.ch != '{' || l.peekChar() != '%' {
			l.readChar()
			continue
		}
		// Save the position of this '{%' so we can skip past it if the tag
		// turns out to be irrelevant.
		savePos := l.pos
		l.readChar() // {
		l.readChar() // %
		for l.ch == ' ' || l.ch == '\t' || l.ch == '-' {
			l.readChar()
		}

		switch {
		case l.ch == 'c' && l.matchAhead("omment"):
			depth++
			for range 7 { // "comment"
				l.readChar()
			}
		case l.ch == 'e' && l.matchAhead("ndcomment"):
			depth--
			for range 10 { // "endcomment"
				l.readChar()
			}
			if depth == 0 {
				for l.ch == ' ' || l.ch == '\t' {
					l.readChar()
				}
				trimRight := false
				if l.ch == '-' {
					trimRight = true
					l.readChar()
				}
				if l.ch == '%' && l.peekChar() == '}' {
					l.readChar()
					l.readChar()
					l.mode = modeText
					return l.input[startPos:savePos], startLine, startCol, trimRight
				}
			}
		case l.ch == 'r' && l.matchAhead("aw"):
			// Skip past the raw body so its content doesn't consume our
			// nesting counter accidentally.
			for range 3 { // "raw"
				l.readChar()
			}
			// Walk to the closing %} of `{% raw %}`.
			for l.ch != 0 && !(l.ch == '%' && l.peekChar() == '}') {
				l.readChar()
			}
			if l.ch != 0 {
				l.readChar() // %
				l.readChar() // }
			}
			// Now consume up to and including {% endraw %}.
			for l.ch != 0 {
				if l.ch == '{' && l.peekChar() == '%' {
					save2 := l.pos
					l.readChar()
					l.readChar()
					for l.ch == ' ' || l.ch == '\t' || l.ch == '-' {
						l.readChar()
					}
					if l.ch == 'e' && l.matchAhead("ndraw") {
						for range 6 { // "endraw"
							l.readChar()
						}
						for l.ch != 0 && !(l.ch == '%' && l.peekChar() == '}') {
							l.readChar()
						}
						if l.ch != 0 {
							l.readChar() // %
							l.readChar() // }
						}
						break
					}
					// Not endraw — rewind to just past the {% so we don't
					// double-skip and miss content.
					_ = save2
				}
				l.readChar()
			}
		default:
			// Some other tag token — keep walking byte by byte. We are
			// already past the `{%`; the next iteration of the outer loop
			// will pick up wherever we land.
		}
	}

	// EOF reached without matching endcomment.
	return l.input[startPos:l.pos], startLine, startCol, false
}

// scanRawBodyTo scans the raw text body of a block tag until it finds
// the matching `{% endTag %}` (with optional trim markers). Returns
// (content, startLine, startCol, trimRight, closed). closed is false when
// EOF was reached without a terminator. The lexer is left in modeText
// positioned just past the close (when closed is true).
//
// `endTag` is the bare keyword (e.g. "endcomment", "enddoc"). The block
// body is captured literally — Liquid syntax inside is not interpreted.
func (l *lexer) scanRawBodyTo(endTag string) (content string, startLine, startCol int, trimRight, closed bool) {
	startPos := l.pos
	startLine = l.line
	startCol = l.column

	endTail := endTag[1:]

	for l.ch != 0 {
		if l.ch == '{' && l.peekChar() == '%' {
			savePos := l.pos

			l.readChar() // {
			l.readChar() // %

			for l.ch == ' ' || l.ch == '\t' || l.ch == '-' {
				l.readChar()
			}

			if l.ch == endTag[0] && l.matchAhead(endTail) {
				l.readChar()
				for range len(endTail) {
					l.readChar()
				}
				for l.ch == ' ' || l.ch == '\t' {
					l.readChar()
				}
				trimRight = false
				if l.ch == '-' {
					trimRight = true
					l.readChar()
				}
				if l.ch == '%' && l.peekChar() == '}' {
					l.readChar() // %
					l.readChar() // }
					l.mode = modeText
					return l.input[startPos:savePos], startLine, startCol, trimRight, true
				}
			}
		}
		l.readChar()
	}

	return l.input[startPos:l.pos], startLine, startCol, false, false
}

// scanToTagClose consumes everything up to and including the next %} or
// -%} (skipping over string literals so a `%}` inside quotes is not
// mistaken for the terminator). The caller decides whether to honor
// strings via skipStrings — inline comments treat the body as plain
// prose, so an apostrophe in "don't" must NOT begin a string.
//
// Returns (content, trimRight, closed). closed is false when EOF was
// reached without finding a tag terminator; callers should report a parse
// error in that case rather than treat the unterminated input as a
// successful empty body.
func (l *lexer) scanToTagClose(skipStrings bool) (content string, trimRight, closed bool) {
	startPos := l.pos
	for l.ch != 0 {
		if skipStrings && (l.ch == '"' || l.ch == '\'') {
			quote := l.ch
			l.readChar()
			for l.ch != 0 && l.ch != quote {
				if l.ch == '\\' && l.peekChar() != 0 {
					l.readChar()
				}
				l.readChar()
			}
			if l.ch != 0 {
				l.readChar() // closing quote
			}
			continue
		}
		if l.ch == '-' && l.peekChar() == '%' && l.peekCharN(2) == '}' {
			end := l.pos
			l.readChar() // -
			l.readChar() // %
			l.readChar() // }
			l.mode = modeText
			return l.input[startPos:end], true, true
		}
		if l.ch == '%' && l.peekChar() == '}' {
			end := l.pos
			l.readChar() // %
			l.readChar() // }
			l.mode = modeText
			return l.input[startPos:end], false, true
		}
		l.readChar()
	}
	return l.input[startPos:l.pos], false, false
}
