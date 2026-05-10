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
		}

		p := l.peekChar()

		// Inline comment {# ... #} — skip entirely, then continue scanning text.
		if p == '#' {
			if l.pos > startPos {
				return makeTextToken(l.input[startPos:l.pos], startLine, startCol)
			}
			l.consumeInlineComment()
			startLine, startCol, startPos = l.line, l.column, l.pos
			continue
		}

		// Output ({{...) or tag ({%...) start.
		if p == '{' || p == '%' {
			if l.pos > startPos {
				return makeTextToken(l.input[startPos:l.pos], startLine, startCol)
			}
			return l.openTagOrOutput(p)
		}

		// Lone '{' — consume and resume scanning.
		l.readChar()
	}

	if l.pos > startPos {
		return makeTextToken(l.input[startPos:l.pos], startLine, startCol)
	}
	return token{typ: tokenEOF, line: l.line, column: l.column}
}

func makeTextToken(literal string, line, column int) token {
	return token{typ: tokenText, literal: literal, line: line, column: column}
}

// consumeInlineComment skips a `{# ... #}` block. The lexer is positioned at
// `{`. Trailing line ending is consumed so an inline comment on its own line
// doesn't leave a blank line behind.
func (l *lexer) consumeInlineComment() {
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
	switch {
	case l.ch == '\n':
		l.readChar()
	case l.ch == '\r' && l.peekChar() == '\n':
		l.readChar()
		l.readChar()
	}
}

// openTagOrOutput consumes a `{{`, `{{-`, `{%`, or `{%-` opener and returns
// the corresponding token. `peek` is the second character (`{` or `%`).
func (l *lexer) openTagOrOutput(peek byte) token {
	tokLine := l.line
	tokCol := l.column
	l.readChar() // {
	l.readChar() // { or %

	trim := l.ch == '-'
	if trim {
		l.readChar()
	}

	if peek == '{' {
		l.mode = modeOutput
		if trim {
			return token{typ: tokenOutputTrim, line: tokLine, column: tokCol}
		}
		return token{typ: tokenOutputOpen, line: tokLine, column: tokCol}
	}
	l.mode = modeTag
	if trim {
		return token{typ: tokenTagTrim, line: tokLine, column: tokCol}
	}
	return token{typ: tokenTagOpen, line: tokLine, column: tokCol}
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
// The width is intrinsic to the token grammar — one case per punctuation
// class the Liquid expression syntax accepts.
//
//nolint:gocyclo,cyclop,funlen // Token-class dispatch; width matches the grammar.
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

// scanString scans a quoted string literal. Matching Ruby Liquid's
// SINGLE_STRING_LITERAL/DOUBLE_STRING_LITERAL regexes, content is taken
// verbatim — escape sequences (`\n`, `\t`, …) are NOT interpreted by the
// lexer. The first matching quote of the same kind closes the string;
// the other quote can appear inside unescaped. Filters that need to
// interpret escapes do so themselves.
func (l *lexer) scanString() token {
	line := l.line
	col := l.column
	quote := l.ch
	l.readChar() // consume opening quote

	startPos := l.pos
	src := l.input

	for l.ch != 0 && l.ch != quote {
		l.readChar()
	}

	literal := src[startPos:l.pos]
	if l.ch == quote {
		l.readChar() // closing quote
	}
	return token{typ: tokenString, literal: literal, line: line, column: col}
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

	// Ruby Liquid's NUMBER_LITERAL is /-?\d+(\.\d+)?/ — scientific notation
	// is NOT recognized. `1e5` lexes as int(1) followed by identifier `e5` to
	// stay compatible with the Ruby reference.

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
func (l *lexer) scanRawBlock() (content string, line, col int, trimRight, closed bool) {
	startPos := l.pos
	startLine := l.line
	startCol := l.column

	for l.ch != 0 {
		if l.ch != '{' || l.peekChar() != '%' {
			l.readChar()
			continue
		}
		savePos := l.pos
		l.readChar() // {
		l.readChar() // %
		l.skipTagLeadingSpace()

		if l.ch == 'e' && l.matchKeyword("ndraw") {
			l.advance(6) // past "endraw"
			if trimRight, ok := l.tryCloseTag(); ok {
				l.mode = modeText
				return l.input[startPos:savePos], startLine, startCol, trimRight, true
			}
		}
		// Not endraw — continue scanning from wherever we landed.
	}

	// EOF reached without finding endraw.
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

// matchKeyword reports whether l.ch followed by rest spells out a complete
// keyword — that is, the byte after rest is not an identifier continuation.
// Used in raw-text scanners (comment, raw) where a substring match like
// `endraw` against `{% endrawful %}` would otherwise falsely terminate the
// block. Liquid's tokenizer treats tag names as full identifiers.
func (l *lexer) matchKeyword(rest string) bool {
	if !l.matchAhead(rest) {
		return false
	}
	after := l.readPos + len(rest)
	if after >= len(l.input) {
		return true
	}
	return !isIdentByte(l.input[after])
}

// isIdentByte reports whether c can appear inside a Liquid identifier
// (letter, digit, or underscore).
func isIdentByte(c byte) bool {
	return c == '_' ||
		(c >= '0' && c <= '9') ||
		(c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z')
}

// scanCommentBlock scans the body of {% comment %} until it finds the
// matching {% endcomment %} (or trim variant), tracking nesting so that
// `{% comment %}{% comment %}…{% endcomment %}{% endcomment %}` correctly
// pairs each opener with its closer (matches Shopify's comment.rb).
// `{% raw %}…{% endraw %}` blocks within the body are skipped over so an
// `{% endcomment %}` token inside raw text doesn't terminate prematurely.
func (l *lexer) scanCommentBlock() (content string, line, col int, trimRight bool) {
	startPos := l.pos
	startLine := l.line
	startCol := l.column
	depth := 1

	for l.ch != 0 {
		if l.ch != '{' || l.peekChar() != '%' {
			l.readChar()
			continue
		}
		// Save the position of this '{%' so we can include the body up to
		// (but not including) the closing tag in the returned content.
		savePos := l.pos
		l.readChar() // {
		l.readChar() // %
		l.skipTagLeadingSpace()

		switch {
		case l.ch == 'c' && l.matchKeyword("omment"):
			depth++
			l.advance(7) // past "comment"
		case l.ch == 'e' && l.matchKeyword("ndcomment"):
			depth--
			l.advance(10) // past "endcomment"
			if depth == 0 {
				if trimRight, ok := l.tryCloseTagTolerant(); ok {
					l.mode = modeText
					return l.input[startPos:savePos], startLine, startCol, trimRight
				}
			}
		case l.ch == 'r' && l.matchKeyword("aw"):
			l.advance(3) // past "raw"
			l.skipRawSubBlock()
		default:
			// Some other tag token — keep walking byte by byte. The next
			// outer-loop iteration picks up wherever we land.
		}
	}

	// EOF reached without matching endcomment.
	return l.input[startPos:l.pos], startLine, startCol, false
}

// skipTagLeadingSpace consumes whitespace and a leading trim-dash (`-`)
// inside an already-opened `{% ...` tag, advancing l.ch to the first
// content byte. The dash is consumed because `{%-` and `{% -` are both
// valid trim markers.
func (l *lexer) skipTagLeadingSpace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '-' {
		l.readChar()
	}
}

// advance consumes n bytes from the input. Advancing happens from the
// current readPos, so to skip past a keyword whose first byte is already
// in l.ch (e.g. l.ch == 'c' for "comment"), pass the full keyword length.
func (l *lexer) advance(n int) {
	for range n {
		l.readChar()
	}
}

// skipPastTagClose advances past the next `%}` (consuming the two bytes).
// Stops at EOF without advancing further.
func (l *lexer) skipPastTagClose() {
	for l.ch != 0 && (l.ch != '%' || l.peekChar() != '}') {
		l.readChar()
	}
	if l.ch != 0 {
		l.readChar() // %
		l.readChar() // }
	}
}

// tryCloseTagTolerant is like tryCloseTag but skips arbitrary trailing tokens
// before the closing `%}`/`-%}`. Ruby Liquid tolerates `{% endcomment foo %}`
// and `{% endraw bar %}`; only :strict2 rejects them.
func (l *lexer) tryCloseTagTolerant() (trimRight, ok bool) {
	startPos := l.pos
	for l.ch != 0 {
		if l.ch == '-' && l.peekChar() == '%' {
			trimRight = true
			l.readChar()
		}
		if l.ch == '%' && l.peekChar() == '}' {
			l.readChar()
			l.readChar()
			return trimRight, true
		}
		// Don't run past a `{%` — that would consume a following tag's opener.
		if l.ch == '{' && l.peekChar() == '%' {
			l.pos = startPos
			l.readPos = startPos + 1
			l.ch = l.input[startPos]
			return false, false
		}
		l.readChar()
	}
	return false, false
}

// tryCloseTag attempts to consume the trailing whitespace, optional `-`, and
// `%}` that closes a tag. Returns (trimRight, true) on success; on failure the
// lexer position may have advanced through whitespace but the caller should
// continue scanning.
func (l *lexer) tryCloseTag() (trimRight, ok bool) {
	for l.ch == ' ' || l.ch == '\t' {
		l.readChar()
	}
	if l.ch == '-' {
		trimRight = true
		l.readChar()
	}
	if l.ch == '%' && l.peekChar() == '}' {
		l.readChar()
		l.readChar()
		return trimRight, true
	}
	return false, false
}

// skipRawSubBlock consumes the body of a `{% raw %}...{% endraw %}` block
// nested inside a comment, so that its contents don't perturb the comment's
// `endcomment` counter. The caller has already advanced past `raw`.
func (l *lexer) skipRawSubBlock() {
	l.skipPastTagClose() // close of `{% raw %}`
	for l.ch != 0 {
		if l.ch == '{' && l.peekChar() == '%' {
			l.readChar()
			l.readChar()
			l.skipTagLeadingSpace()
			if l.ch == 'e' && l.matchKeyword("ndraw") {
				l.advance(6) // past "endraw"
				l.skipPastTagClose()
				return
			}
			continue
		}
		l.readChar()
	}
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

			if l.ch == endTag[0] && l.matchKeyword(endTail) {
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
