package liquid

import (
	"fmt"
	"strconv"
	"strings"
)

// parser parses a Liquid template into an AST.
type parser struct {
	l                  *lexer
	curToken           token
	trimNextText       bool // trim leading whitespace from next text node (set by -%} tags)
	pendingTagTrimLeft bool // current tag began with `{%-`; consulted by unknown-tag fallback
	env                *Environment
	warnings           []Warning
}

func newParserWithEnv(input string, env *Environment) *parser {
	if env == nil {
		env = Default()
	}
	p := &parser{l: newLexer(input), env: env}
	p.nextToken()
	return p
}

func (p *parser) nextToken() {
	p.curToken = p.l.nextToken()
}

func (p *parser) parse() (*templateAST, error) {
	nodes, err := p.parseNodes(func() bool { return p.curToken.typ == tokenEOF })
	if err != nil {
		return nil, err
	}
	return &templateAST{nodes: nodes}, nil
}

func (p *parser) parseNodes(endCondition func() bool) ([]Node, error) {
	var nodes []Node

	for !endCondition() && p.curToken.typ != tokenEOF {
		node, trimLeft, trimRight, err := p.parseNodeWithTrim()
		if err != nil {
			return nil, err
		}
		if node == nil {
			continue
		}
		if trimLeft {
			trimPrevTextRight(nodes)
		}
		if p.trimNextText {
			trimTextNodeLeft(node)
			p.trimNextText = false
		}
		nodes = append(nodes, node)
		if trimRight {
			p.trimNextText = true
		}
	}

	// A `{%-` terminator (e.g. `{%- endfor`) needs to trim trailing whitespace
	// from the last text node we emitted before it.
	if p.curToken.typ == tokenTagTrim {
		trimPrevTextRight(nodes)
	}
	return nodes, nil
}

// trimPrevTextRight strips trailing whitespace from the last node when it is
// a TextNode. No-op otherwise.
func trimPrevTextRight(nodes []Node) {
	if len(nodes) == 0 {
		return
	}
	if t, ok := nodes[len(nodes)-1].(*TextNode); ok {
		t.Text = trimTrailingWhitespace(t.Text)
	}
}

// trimTextNodeLeft strips leading whitespace from node when it is a TextNode.
// No-op otherwise.
func trimTextNodeLeft(node Node) {
	if t, ok := node.(*TextNode); ok {
		t.Text = trimLeadingWhitespace(t.Text)
	}
}

// trimTrailingWhitespace removes trailing whitespace including newlines
func trimTrailingWhitespace(s string) string {
	i := len(s)
	for i > 0 {
		r := s[i-1]
		if r != ' ' && r != '\t' && r != '\n' && r != '\r' {
			break
		}
		i--
	}
	return s[:i]
}

// trimLeadingWhitespace removes leading whitespace including newlines
func trimLeadingWhitespace(s string) string {
	i := 0
	for i < len(s) {
		r := s[i]
		if r != ' ' && r != '\t' && r != '\n' && r != '\r' {
			break
		}
		i++
	}
	return s[i:]
}

func (p *parser) parseNodeWithTrim() (node Node, trimLeft, trimRight bool, err error) {
	switch p.curToken.typ {
	case tokenEOF:
		return nil, false, false, nil // returns nil node and nil error - signals end of parsing
	case tokenText:
		node = &TextNode{
			Text:   p.curToken.literal,
			Line:   p.curToken.line,
			Column: p.curToken.column,
		}
		p.nextToken()
		return node, false, false, nil
	case tokenOutputOpen, tokenOutputTrim:
		trimLeft = p.curToken.typ == tokenOutputTrim
		node, trimRight, err = p.parseOutputWithTrim()
		// Ruby quirk: `{{-}}` — `{{-` immediately followed by `}}` with no
		// body — trims BOTH sides. Ruby's whitespace_handler looks at
		// token[-3], which for the 5-char `{{-}}` is the `-`, so trim_right
		// becomes true. We mirror that here.
		if trimLeft && !trimRight {
			if o, ok := node.(*OutputNode); ok && o != nil && o.Expr == nil {
				trimRight = true
			}
		}
		return node, trimLeft, trimRight, err
	case tokenTagOpen, tokenTagTrim:
		trimLeft = p.curToken.typ == tokenTagTrim
		node, trimRight, err = p.parseTagWithTrim()
		return node, trimLeft, trimRight, err
	default:
		err = newParseError(p.curToken.line, p.curToken.column,
			"unexpected token: %q", p.curToken.literal)
		p.nextToken()
		return nil, false, false, err
	}
}

func (p *parser) parseOutputWithTrim() (Node, bool, error) {
	line := p.curToken.line
	column := p.curToken.column
	p.nextToken()

	// Empty output `{{}}` (and `{{- -}}`) is permitted by upstream Liquid and
	// renders the empty string. Skip expression parsing in that case.
	var expr Expression
	if p.curToken.typ != tokenOutputClose && p.curToken.typ != tokenOutputTrimR {
		var err error
		expr, err = p.parseExpression()
		if err != nil {
			return nil, false, err
		}
	}

	trimRight := p.curToken.typ == tokenOutputTrimR
	if p.curToken.typ != tokenOutputClose && p.curToken.typ != tokenOutputTrimR {
		return nil, false, newParseError(p.curToken.line, p.curToken.column,
			"expected }}, got %q", p.curToken.literal)
	}
	p.nextToken()

	return &OutputNode{
		Expr:   expr,
		Line:   line,
		Column: column,
	}, trimRight, nil
}

func (p *parser) parseTagWithTrim() (Node, bool, error) {
	p.trimNextText = false // reset so we capture only this tag's closing trim state
	trimLeft := p.curToken.typ == tokenTagTrim
	p.pendingTagTrimLeft = trimLeft
	node, err := p.parseTag()
	p.pendingTagTrimLeft = false
	return node, p.trimNextText, err
}

// parseTag dispatches a tag by name. Tag names are not globally reserved
// tokens; the lexer emits them as plain identifiers. They are recognized
// only here, immediately after `{%`.
//
//nolint:gocyclo,cyclop // Tag-name dispatch; width matches the Liquid spec.
func (p *parser) parseTag() (Node, error) {
	p.nextToken() // consume {% or {%-

	// `{% # ... %}` is an inline comment — Shopify Liquid recognizes any tag
	// whose body starts with `#` as a comment to be discarded entirely.
	if p.curToken.typ == tokenHash {
		return p.parseInlineCommentTag()
	}

	if p.curToken.typ != tokenIdent {
		return nil, newParseError(p.curToken.line, p.curToken.column,
			"expected tag name, got %q", p.curToken.literal)
	}
	switch p.curToken.literal {
	case "if":
		return p.parseIfTag()
	case "unless":
		return p.parseUnlessTag()
	case "case":
		return p.parseCaseTag()
	case "for":
		return p.parseForTag()
	case "break":
		return p.parseBreakTag()
	case "continue":
		return p.parseContinueTag()
	case "assign":
		return p.parseAssignTag()
	case "capture":
		return p.parseCaptureTag()
	case "comment":
		return p.parseCommentTag()
	case "raw":
		return p.parseRawTag()
	case "cycle":
		return p.parseCycleTag()
	case "increment":
		return p.parseIncrementTag()
	case "decrement":
		return p.parseDecrementTag()
	case "render":
		return p.parsePartialTag(true)
	case "include":
		return p.parsePartialTag(false)
	case "echo":
		return p.parseEchoTag()
	case "liquid":
		return p.parseLiquidTag()
	case "tablerow":
		return p.parseTablerowTag()
	case "ifchanged":
		return p.parseIfchangedTag()
	case "doc":
		return p.parseDocTag()
	}
	// Custom tag plugins (RegisterTag / RegisterBlock) — fall through here
	// before reporting an unknown-tag error, so user code can extend the
	// language without forking the parser.
	if node, ok, err := p.parseCustomTag(); ok {
		return node, err
	}
	return p.handleUnknownTag()
}

// handleUnknownTag dispatches the unknown-tag fork according to the
// active environment's ErrorMode. Strict (default) returns a parse
// error matching prior behavior. Warn records a Warning and emits the
// raw `{% NAME ... %}` source span as text. Lax does the same but
// without recording a warning.
func (p *parser) handleUnknownTag() (Node, error) {
	name := p.curToken.literal
	line, column := p.curToken.line, p.curToken.column
	mode := ErrorModeStrict
	if p.env != nil {
		mode = p.env.ErrorMode()
	}
	if mode == ErrorModeStrict {
		return nil, newParseError(line, column, "unknown tag: %q", name)
	}
	trimLeft := p.pendingTagTrimLeft
	// Consume the rest of the tag verbatim so the output preserves the source.
	p.l.mode = modeText
	markup, trimRight, closed := p.l.scanToTagClose(true)
	if !closed {
		return nil, newParseError(line, column,
			"unterminated {%% %s ... %%}: expected %%}", name)
	}
	p.trimNextText = trimRight
	p.nextToken()
	if mode == ErrorModeWarn {
		p.warnings = append(p.warnings, Warning{
			Message: fmt.Sprintf("unknown tag: %q", name),
			Line:    line,
			Column:  column,
		})
	}
	openDelim, closeDelim := "{%", "%}"
	if trimLeft {
		openDelim = "{%-"
	}
	if trimRight {
		closeDelim = "-%}"
	}
	// Preserve interior whitespace verbatim — markup starts with whatever
	// separator the source had between the tag name and its body, so we
	// don't insert any extra space ourselves.
	raw := openDelim + " " + name + markup + closeDelim
	return &TextNode{Text: raw, Line: line, Column: column}, nil
}

// parseCustomTag dispatches to a registered TagParser for the current tag
// name. Returns ok=false if no plugin is registered, leaving the caller to
// emit its standard "unknown tag" error. On success, the markup string from
// just past the tag name to the closing %} is handed to the registered
// parser; for block tags, the body is then parsed up to {% endNAME %}.
func (p *parser) parseCustomTag() (Node, bool, error) {
	name := p.curToken.literal
	parse, isBlock, ok := p.env.lookupCustomTag(name)
	if !ok {
		return nil, false, nil
	}
	line, column := p.curToken.line, p.curToken.column

	// Mirror parseLiquidTag's lexer handoff: l.pos is one byte past the
	// tag name, so flipping the lexer to text mode and calling
	// scanToTagClose captures exactly the raw markup the plugin needs.
	p.l.mode = modeText
	markup, trimRight, closed := p.l.scanToTagClose(true)
	if !closed {
		return nil, true, newParseError(line, column,
			"unterminated {%% %s ... %%}: expected %%}", name)
	}
	p.trimNextText = trimRight
	p.nextToken() // refresh curToken from past the closing %}

	renderer, err := parse(markup)
	if err != nil {
		return nil, true, &ParseError{
			Message: fmt.Sprintf("tag %q: %v", name, err),
			Line:    line,
			Column:  column,
		}
	}

	if !isBlock {
		return &customTagNode{
			name:     name,
			renderer: renderer,
			line:     line,
			column:   column,
		}, true, nil
	}

	endTag := "end" + name
	body, err := p.parseNodes(func() bool {
		return p.isTagKeyword(endTag)
	})
	if err != nil {
		return nil, true, err
	}
	if !p.isTagKeyword(endTag) {
		return nil, true, newParseError(p.curToken.line, p.curToken.column,
			"expected %s", endTag)
	}
	p.nextToken() // {%
	p.nextToken() // endNAME
	if err := p.expectTagClose(); err != nil {
		return nil, true, err
	}
	return &customBlockNode{
		name:     name,
		renderer: renderer,
		body:     body,
		line:     line,
		column:   column,
	}, true, nil
}

// parseTablerowTag handles
//
//	{% tablerow VAR in COLLECTION [cols: N] [limit: M] [offset: K] %}
//	  body
//	{% endtablerow %}
//
// The body is rendered once per item, wrapped in <td>/<tr> markup. cols
// (default = collection length) controls how many cells appear per row.
func (p *parser) parseTablerowTag() (Node, error) {
	line := p.curToken.line
	column := p.curToken.column
	p.nextToken() // consume "tablerow"

	if p.curToken.typ != tokenIdent {
		return nil, newParseError(p.curToken.line, p.curToken.column,
			"expected variable name in tablerow, got %q", p.curToken.literal)
	}
	varName := p.curToken.literal
	p.nextToken()

	if !p.isWord("in") {
		return nil, newParseError(p.curToken.line, p.curToken.column,
			"expected 'in' after tablerow variable, got %q", p.curToken.literal)
	}
	p.nextToken()

	collection, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}

	tag := &TablerowTag{
		Variable:   varName,
		Collection: collection,
		Line:       line,
		Column:     column,
	}

	if err := p.parseTablerowAttrs(tag); err != nil {
		return nil, err
	}

	if err := p.expectTagClose(); err != nil {
		return nil, err
	}

	tag.Body, err = p.parseNodes(func() bool {
		return p.isTagKeyword("endtablerow")
	})
	if err != nil {
		return nil, err
	}
	if !p.isTagKeyword("endtablerow") {
		return nil, newParseError(p.curToken.line, p.curToken.column, "expected endtablerow")
	}
	p.nextToken() // {%
	p.nextToken() // endtablerow
	if err := p.expectTagClose(); err != nil {
		return nil, err
	}
	return tag, nil
}

// parseTablerowAttrs consumes the optional `cols:`, `limit:`, `offset:`,
// `range:` modifiers (and tolerated commas) that may follow `tablerow var
// in coll`. The shape mirrors `for`'s parameter loop, minus `reversed`.
// `range:` is accept-and-discard for Shopify parity.
func (p *parser) parseTablerowAttrs(tag *TablerowTag) error {
	for {
		switch {
		case p.isWord("cols"):
			expr, err := p.parseForKeywordArg("cols")
			if err != nil {
				return err
			}
			tag.Cols = expr
		case p.isWord("limit"):
			expr, err := p.parseForKeywordArg("limit")
			if err != nil {
				return err
			}
			tag.Limit = expr
		case p.isWord("offset"):
			expr, err := p.parseForKeywordArg("offset")
			if err != nil {
				return err
			}
			tag.Offset = expr
		case p.isWord("range"):
			// Shopify accepts `range:` as a tablerow attribute but its
			// renderer ignores it. Parse-and-discard for parity.
			if _, err := p.parseForKeywordArg("range"); err != nil {
				return err
			}
		case p.curToken.typ == tokenComma:
			p.nextToken()
		default:
			return nil
		}
	}
}

// parseIfchangedTag handles {% ifchanged %}body{% endifchanged %}. The
// body is rendered every iteration; the evaluator suppresses output that
// matches the previous emission for the same block.
func (p *parser) parseIfchangedTag() (Node, error) {
	line := p.curToken.line
	column := p.curToken.column
	p.nextToken() // consume "ifchanged"
	if err := p.expectTagClose(); err != nil {
		return nil, err
	}
	body, err := p.parseNodes(func() bool {
		return p.isTagKeyword("endifchanged")
	})
	if err != nil {
		return nil, err
	}
	if !p.isTagKeyword("endifchanged") {
		return nil, newParseError(p.curToken.line, p.curToken.column, "expected endifchanged")
	}
	p.nextToken() // {%
	p.nextToken() // endifchanged
	if err := p.expectTagClose(); err != nil {
		return nil, err
	}
	return &IfchangedTag{Body: body, Line: line, Column: column}, nil
}

// parseDocTag handles {% doc %}content{% enddoc %}. Body is captured as
// raw text (Liquid syntax inside is not interpreted) and discarded at
// render time.
func (p *parser) parseDocTag() (Node, error) {
	line := p.curToken.line
	column := p.curToken.column
	p.nextToken() // consume "doc"
	if err := p.validateTagClose(); err != nil {
		return nil, err
	}
	p.l.mode = modeText
	content, _, _, _, ok := p.l.scanRawBodyTo("enddoc")
	if !ok {
		return nil, &ParseError{
			Message: "unterminated {% doc %} block: expected enddoc",
			Line:    line,
			Column:  column,
		}
	}
	p.nextToken() // refresh after lexer mode switch
	return &DocTag{Content: content, Line: line, Column: column}, nil
}

// parseEchoTag handles {% echo expr | filter1 | filter2 %}. The body is an
// expression (with optional filter chain) and the result is rendered, so
// {% echo x | upcase %} is equivalent to {{ x | upcase }}. This tag is the
// canonical way to emit output inside a {% liquid %} block, where the
// {{ ... }} delimiter form isn't available.
func (p *parser) parseEchoTag() (Node, error) {
	line := p.curToken.line
	column := p.curToken.column
	p.nextToken() // consume "echo"

	expr, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	if err := p.expectTagClose(); err != nil {
		return nil, err
	}
	return &OutputNode{Expr: expr, Line: line, Column: column}, nil
}

// parseInlineCommentTag handles {% # comment text %}. Anything after the `#`
// up to %} is discarded.
func (p *parser) parseInlineCommentTag() (Node, error) {
	line := p.curToken.line
	column := p.curToken.column
	// Switch lexer to text mode and consume to the closing tag delimiter.
	// The ws/expression lexer has already consumed the `#`; everything that
	// remains in the tag body is comment content.
	p.l.mode = modeText
	// Comment body is treated as prose — a `'` in "don't" must not begin a
	// string literal — so we don't skip over strings.
	content, trimRight, closed := p.l.scanToTagClose(false)
	if !closed {
		return nil, &ParseError{
			Message: "unterminated inline comment ({% # ... %}): expected %}",
			Line:    line,
			Column:  column,
		}
	}
	p.trimNextText = trimRight
	p.nextToken() // refresh
	// If the comment spans multiple lines, every non-blank line AFTER
	// the opening `#`-line must itself begin with `#`. Mirrors Ruby's
	// InlineComment#parse check. The opening `#` is already consumed,
	// so the first split chunk is the tail of that line and is
	// exempt.
	if strings.ContainsRune(content, '\n') {
		for i, ln := range strings.Split(content, "\n") {
			if i == 0 {
				continue
			}
			s := strings.TrimSpace(ln)
			if s == "" {
				continue
			}
			if !strings.HasPrefix(s, "#") {
				return nil, &ParseError{
					Message: "Each line of comments must be prefixed by the '#' character",
					Line:    line,
					Column:  column,
				}
			}
		}
	}
	return &CommentTag{Content: content, Line: line, Column: column}, nil
}

// parseLiquidTag handles {% liquid ... %}. The body is a sequence of tag
// statements, one per line, written without their own {% %} delimiters.
// We re-emit them as a synthetic source string and parse it as a
// sub-template so every existing tag parser keeps working unchanged.
//
// Important: we must NOT call p.nextToken() after consuming "liquid",
// because in modeTag the lexer would skip whitespace (including the
// newline that begins the body) and consume the first body word as a
// token, leaving l.pos in the middle of the body. Switch the lexer to
// text mode immediately so the body capture starts at the byte right
// after "liquid".
func (p *parser) parseLiquidTag() (Node, error) {
	line := p.curToken.line
	column := p.curToken.column

	// Switch the lexer to text mode in place. l.pos currently points at the
	// first character after "liquid"; scanToTagClose reads from there.
	p.l.mode = modeText
	body, trimRight, closed := p.l.scanToTagClose(true)
	if !closed {
		return nil, &ParseError{
			Message: "unterminated {% liquid %} block: expected %}",
			Line:    line,
			Column:  column,
		}
	}
	p.trimNextText = trimRight

	// Build a synthetic source: each non-blank line becomes its own
	// {% line %}. Blank lines are dropped. Block-form tags whose body
	// spans multiple lines (raw/endraw, comment/endcomment) cannot appear
	// here without producing nonsense — reject them with a clear error so
	// authors aren't surprised by silent garbled output. `liquid` is also
	// disallowed (no nested liquid blocks).
	var sb strings.Builder
	for _, raw := range strings.Split(body, "\n") {
		stripped := strings.TrimSpace(raw)
		// Ruby tolerates any number of leading `liquid` keywords on a line —
		// they are no-ops, so `liquid liquid echo "x"` is just `echo "x"`.
		// Strip them before classifying the line.
		for stripped == "liquid" || strings.HasPrefix(stripped, "liquid ") || strings.HasPrefix(stripped, "liquid\t") {
			if stripped == "liquid" {
				stripped = ""
				break
			}
			stripped = strings.TrimSpace(stripped[len("liquid"):])
		}
		if stripped == "" {
			continue
		}
		first, _, _ := strings.Cut(stripped, " ")
		switch first {
		case "raw", "endraw", "comment", "endcomment":
			return nil, &ParseError{
				Message: "tag '" + first + "' is not allowed inside {% liquid %} block",
				Line:    line,
				Column:  column,
			}
		}
		sb.WriteString("{% ")
		sb.WriteString(stripped)
		sb.WriteString(" %}")
	}

	sub := newParserWithEnv(sb.String(), p.env)
	ast, err := sub.parse()
	if err != nil {
		return nil, &ParseError{
			Message: "in {% liquid %} block: " + err.Error(),
			Line:    line,
			Column:  column,
		}
	}

	p.nextToken() // refresh after lexer mode switch
	return &LiquidTag{Body: ast.nodes, Line: line, Column: column}, nil
}

func (p *parser) parseIfTag() (Node, error) {
	line := p.curToken.line
	column := p.curToken.column
	p.nextToken() // consume 'if'

	condition, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	err = p.expectTagClose()
	if err != nil {
		return nil, err
	}

	tag := &IfTag{
		Condition: condition,
		Line:      line,
		Column:    column,
	}

	// Parse then branch
	tag.ThenBranch, err = p.parseNodes(func() bool {
		return p.isTagKeywordIn("elsif", "else", "endif")
	})
	if err != nil {
		return nil, err
	}

	// Parse elsif branches
	for p.isTagKeyword("elsif") {
		p.nextToken() // consume {%
		p.nextToken() // consume elsif

		var elsifCond Expression
		elsifCond, err = p.parseExpression()
		if err != nil {
			return nil, err
		}

		err = p.expectTagClose()
		if err != nil {
			return nil, err
		}

		var elsifBody []Node
		elsifBody, err = p.parseNodes(func() bool {
			return p.isTagKeywordIn("elsif", "else", "endif")
		})
		if err != nil {
			return nil, err
		}

		tag.ElsifBranches = append(tag.ElsifBranches, struct {
			Condition Expression
			Body      []Node
		}{elsifCond, elsifBody})
	}

	// Parse else branch
	if p.isTagKeyword("else") {
		p.nextToken() // consume {%
		p.nextToken() // consume else

		err = p.expectTagClose()
		if err != nil {
			return nil, err
		}

		tag.ElseBranch, err = p.parseNodes(func() bool {
			return p.isTagKeyword("endif")
		})
		if err != nil {
			return nil, err
		}
	}

	// Consume endif
	if !p.isTagKeyword("endif") {
		return nil, newParseError(p.curToken.line, p.curToken.column, "expected endif")
	}
	p.nextToken() // consume {%
	p.nextToken() // consume endif
	if err := p.expectTagClose(); err != nil {
		return nil, err
	}

	return tag, nil
}

func (p *parser) parseUnlessTag() (Node, error) {
	line := p.curToken.line
	column := p.curToken.column
	p.nextToken() // consume 'unless'

	condition, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	err = p.expectTagClose()
	if err != nil {
		return nil, err
	}

	tag := &UnlessTag{
		Condition: condition,
		Line:      line,
		Column:    column,
	}

	// Parse body
	tag.Body, err = p.parseNodes(func() bool {
		return p.isTagKeywordIn("else", "endunless")
	})
	if err != nil {
		return nil, err
	}

	// Parse else branch
	if p.isTagKeyword("else") {
		p.nextToken() // consume {%
		p.nextToken() // consume else

		err = p.expectTagClose()
		if err != nil {
			return nil, err
		}

		tag.ElseBranch, err = p.parseNodes(func() bool {
			return p.isTagKeyword("endunless")
		})
		if err != nil {
			return nil, err
		}
	}

	// Consume endunless
	if !p.isTagKeyword("endunless") {
		return nil, newParseError(p.curToken.line, p.curToken.column, "expected endunless")
	}
	p.nextToken() // consume {%
	p.nextToken() // consume endunless
	err = p.expectTagClose()
	if err != nil {
		return nil, err
	}

	return tag, nil
}

func (p *parser) parseCaseTag() (Node, error) {
	line := p.curToken.line
	column := p.curToken.column
	p.nextToken() // consume 'case'

	value, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	// Tolerate trailing args after the case value, e.g. `{% case 1 bar %}`.
	err = p.drainAndExpectTagClose()
	if err != nil {
		return nil, err
	}

	tag := &CaseTag{
		Value:  value,
		Line:   line,
		Column: column,
	}

	// Skip any whitespace/text between case and first when
	for p.curToken.typ == tokenText {
		p.nextToken()
	}

	// Parse when clauses
	for p.isTagKeyword("when") {
		p.nextToken() // consume {%
		p.nextToken() // consume when

		// `when` accepts multiple values separated by either `,` or `or`:
		// `{% when 1, 2, 3 %}` and `{% when 1 or 2 or 3 %}` both match any
		// of the listed values. Parse below the logical-operator layer so
		// `or` is consumed here as a separator rather than folded into the
		// expression.
		var values []Expression
		for {
			var val Expression
			val, err = p.parseContains()
			if err != nil {
				return nil, err
			}
			values = append(values, val)

			if p.curToken.typ == tokenComma || p.curToken.typ == tokenOr {
				p.nextToken()
				continue
			}
			break
		}

		// Tolerate trailing args after the final when value, e.g.
		// `{% when 1 bar %}`.
		err = p.drainAndExpectTagClose()
		if err != nil {
			return nil, err
		}

		var body []Node
		body, err = p.parseNodes(func() bool {
			return p.isTagKeywordIn("when", "else", "endcase")
		})
		if err != nil {
			return nil, err
		}

		tag.Whens = append(tag.Whens, WhenClause{Values: values, Body: body})
	}

	// Parse else
	if p.isTagKeyword("else") {
		p.nextToken() // consume {%
		p.nextToken() // consume else

		err = p.expectTagClose()
		if err != nil {
			return nil, err
		}

		tag.Else, err = p.parseNodes(func() bool {
			return p.isTagKeyword("endcase")
		})
		if err != nil {
			return nil, err
		}
	}

	// Consume endcase
	if !p.isTagKeyword("endcase") {
		return nil, newParseError(p.curToken.line, p.curToken.column, "expected endcase")
	}
	p.nextToken() // consume {%
	p.nextToken() // consume endcase
	if err := p.expectTagClose(); err != nil {
		return nil, err
	}

	return tag, nil
}

func (p *parser) parseForTag() (Node, error) {
	line := p.curToken.line
	column := p.curToken.column
	p.nextToken() // consume 'for'

	if p.curToken.typ != tokenIdent {
		return nil, newParseError(p.curToken.line, p.curToken.column,
			"expected variable name, got %q", p.curToken.literal)
	}
	varName := p.curToken.literal
	p.nextToken()

	if !p.isWord("in") {
		return nil, newParseError(p.curToken.line, p.curToken.column,
			"expected 'in', got %q", p.curToken.literal)
	}
	p.nextToken()

	collection, err := p.parseForCollection(line, column)
	if err != nil {
		return nil, err
	}

	tag := &ForTag{
		Variable:   varName,
		Collection: collection,
		Line:       line,
		Column:     column,
		LoopName:   computeForloopName(varName, collection),
	}

	if err := p.parseForParams(tag); err != nil {
		return nil, err
	}

	if err := p.expectTagClose(); err != nil {
		return nil, err
	}

	tag.Body, err = p.parseNodes(func() bool {
		return p.isTagKeywordIn("else", "endfor")
	})
	if err != nil {
		return nil, err
	}

	// Parse else branch (for empty collection)
	if p.isTagKeyword("else") {
		p.nextToken() // consume {%
		p.nextToken() // consume else

		err = p.expectTagClose()
		if err != nil {
			return nil, err
		}

		tag.ElseBody, err = p.parseNodes(func() bool {
			return p.isTagKeyword("endfor")
		})
		if err != nil {
			return nil, err
		}
	}

	// Consume endfor
	if !p.isTagKeyword("endfor") {
		return nil, newParseError(p.curToken.line, p.curToken.column, "expected endfor")
	}
	p.nextToken() // consume {%
	p.nextToken() // consume endfor
	err = p.expectTagClose()
	if err != nil {
		return nil, err
	}

	return tag, nil
}

func (p *parser) parseBreakTag() (Node, error) {
	line := p.curToken.line
	column := p.curToken.column
	p.nextToken() // consume 'break'

	if err := p.expectTagClose(); err != nil {
		return nil, err
	}

	return &BreakTag{Line: line, Column: column}, nil
}

// parseForCollection parses the iteration source of a `for` tag: either a
// `(start..end)` range or an arbitrary primary expression.
func (p *parser) parseForCollection(line, column int) (Expression, error) {
	if p.curToken.typ != tokenLParen {
		return p.parsePrimary()
	}
	p.nextToken() // consume (
	start, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}
	if p.curToken.typ != tokenRange {
		return nil, newParseError(p.curToken.line, p.curToken.column,
			"expected '..' in range, got %q", p.curToken.literal)
	}
	p.nextToken() // consume ..
	// Tolerate extra dots (Ruby lax `(1...5)` is treated as `(1..5)`).
	for p.curToken.typ == tokenDot {
		p.nextToken()
	}
	end, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}
	if p.curToken.typ != tokenRParen {
		return nil, newParseError(p.curToken.line, p.curToken.column,
			"expected ')', got %q", p.curToken.literal)
	}
	p.nextToken() // consume )
	return &RangeExpr{Start: start, End: end, Line: line, Column: column}, nil
}

// parseForParams consumes the optional `limit:`, `offset:`, and `reversed`
// modifiers that may follow `for var in coll`. These are recognized
// contextually (by literal) so they remain valid variable names elsewhere.
func (p *parser) parseForParams(tag *ForTag) error {
	for {
		// Ruby allows optional commas between params: `for i in array, limit: 4, offset: 2`.
		if p.curToken.typ == tokenComma {
			p.nextToken()
			continue
		}
		switch {
		case p.isWord("limit"):
			limit, err := p.parseForKeywordArg("limit")
			if err != nil {
				return err
			}
			tag.Limit = limit
		case p.isWord("offset"):
			p.nextToken()
			if p.curToken.typ != tokenColon {
				return newParseError(p.curToken.line, p.curToken.column,
					"expected ':' after offset")
			}
			p.nextToken()
			// Shopify accepts `offset: continue` to resume from where the
			// previous render of this same for-tag stopped (pagination).
			// Repeated `offset:` — last wins (Ruby attribute-hash semantics).
			if p.isWord("continue") {
				tag.OffsetContinue = true
				tag.Offset = nil
				p.nextToken()
				continue
			}
			offset, err := p.parsePrimary()
			if err != nil {
				return err
			}
			tag.Offset = offset
			tag.OffsetContinue = false
		case p.isWord("reversed"):
			p.nextToken()
			tag.Reversed = true
		default:
			return nil
		}
	}
}

// parseForKeywordArg consumes `name: <primary>` and returns the parsed
// expression. The current token must be the keyword.
func (p *parser) parseForKeywordArg(name string) (Expression, error) {
	p.nextToken() // consume keyword
	if p.curToken.typ != tokenColon {
		return nil, newParseError(p.curToken.line, p.curToken.column,
			"expected ':' after %s", name)
	}
	p.nextToken()
	return p.parsePrimary()
}

func (p *parser) parseContinueTag() (Node, error) {
	line := p.curToken.line
	column := p.curToken.column
	p.nextToken() // consume 'continue'

	if err := p.expectTagClose(); err != nil {
		return nil, err
	}

	return &ContinueTag{Line: line, Column: column}, nil
}

func (p *parser) parseAssignTag() (Node, error) {
	line := p.curToken.line
	column := p.curToken.column
	p.nextToken() // consume 'assign'

	if p.curToken.typ != tokenIdent {
		return nil, newParseError(p.curToken.line, p.curToken.column,
			"Syntax Error in 'assign' tag - Valid syntax: assign [var] = [source]")
	}
	varName := p.curToken.literal
	p.nextToken()

	if p.curToken.typ != tokenAssign {
		return nil, newParseError(p.curToken.line, p.curToken.column,
			"Syntax Error in 'assign' tag - Valid syntax: assign [var] = [source]")
	}
	p.nextToken()

	value, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	if err := p.expectTagClose(); err != nil {
		return nil, err
	}

	return &AssignTag{
		Variable: varName,
		Value:    value,
		Line:     line,
		Column:   column,
	}, nil
}

func (p *parser) parseCaptureTag() (Node, error) {
	line := p.curToken.line
	column := p.curToken.column
	p.nextToken() // consume 'capture'

	if p.curToken.typ != tokenIdent {
		return nil, newParseError(p.curToken.line, p.curToken.column,
			"expected variable name, got %q", p.curToken.literal)
	}
	varName := p.curToken.literal
	p.nextToken()

	if err := p.expectTagClose(); err != nil {
		return nil, err
	}

	body, err := p.parseNodes(func() bool {
		return p.isTagKeyword("endcapture")
	})
	if err != nil {
		return nil, err
	}

	// Consume endcapture
	if !p.isTagKeyword("endcapture") {
		return nil, newParseError(p.curToken.line, p.curToken.column, "expected endcapture")
	}
	p.nextToken() // consume {%
	p.nextToken() // consume endcapture
	if err := p.expectTagClose(); err != nil {
		return nil, err
	}

	return &CaptureTag{
		Variable: varName,
		Body:     body,
		Line:     line,
		Column:   column,
	}, nil
}

func (p *parser) parseCommentTag() (Node, error) {
	line := p.curToken.line
	column := p.curToken.column
	p.nextToken() // consume 'comment'

	// Ruby tolerates trailing tokens like `{% comment foo %}`; drain them.
	p.drainToTagClose()

	if err := p.validateTagClose(); err != nil {
		return nil, err
	}

	// Set lexer to text mode and scan to endcomment
	p.l.mode = modeText
	content, _, _, trimRight := p.l.scanCommentBlock()
	p.trimNextText = trimRight // propagate closing tag's trim state
	p.nextToken()              // refresh cur token

	return &CommentTag{
		Content: content,
		Line:    line,
		Column:  column,
	}, nil
}

func (p *parser) parseRawTag() (Node, error) {
	line := p.curToken.line
	column := p.curToken.column
	p.nextToken() // consume 'raw'

	if err := p.validateTagClose(); err != nil {
		return nil, err
	}

	// Set lexer to text mode and scan to endraw
	p.l.mode = modeText
	content, _, _, trimRight, closed := p.l.scanRawBlock()
	if !closed {
		return nil, newParseError(line, column, "unterminated {%% raw %%} block")
	}
	p.trimNextText = trimRight // propagate closing tag's trim state
	p.nextToken()              // refresh cur token

	return &RawTag{
		Content: content,
		Line:    line,
		Column:  column,
	}, nil
}

func (p *parser) parseCycleTag() (Node, error) {
	line := p.curToken.line
	column := p.curToken.column
	p.nextToken() // consume 'cycle'

	tag := &CycleTag{
		Line:   line,
		Column: column,
	}

	// Empty argument list is a syntax error.
	if p.curToken.typ == tokenTagClose || p.curToken.typ == tokenTagTrimR {
		return nil, newParseError(line, column,
			"Syntax Error in 'cycle' - Valid syntax: cycle [name :] var [, var2, var3 ...]")
	}

	// Check for named cycle: {% cycle 'group': 'a', 'b' %} or {% cycle var: 'a', 'b' %}
	firstExpr, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}

	if p.curToken.typ == tokenColon {
		// Named cycle - the first expression is the group name.
		// Keep both the literal-string fast path (for keying when literal) and
		// the full expression (so variable lookups evaluate at render time).
		tag.GroupExpr = firstExpr
		if lit, ok := firstExpr.(*LiteralExpr); ok {
			if name, ok := lit.Value.(string); ok {
				tag.GroupName = name
			}
		}
		p.nextToken() // consume ':'

		// Parse first actual value
		firstExpr, err = p.parsePrimary()
		if err != nil {
			return nil, err
		}
	}

	tag.Values = append(tag.Values, firstExpr)

	// Parse remaining values separated by commas, allowing a trailing comma.
	for p.curToken.typ == tokenComma {
		p.nextToken() // consume ','
		if p.curToken.typ == tokenTagClose || p.curToken.typ == tokenTagTrimR {
			break
		}
		expr, err := p.parsePrimary()
		if err != nil {
			return nil, err
		}
		tag.Values = append(tag.Values, expr)
	}

	if err := p.expectTagClose(); err != nil {
		return nil, err
	}

	return tag, nil
}

func (p *parser) parseIncrementTag() (Node, error) {
	line := p.curToken.line
	column := p.curToken.column
	p.nextToken() // consume 'increment'

	if p.curToken.typ != tokenIdent {
		return nil, newParseError(p.curToken.line, p.curToken.column,
			"expected variable name, got %q", p.curToken.literal)
	}

	varName := p.curToken.literal
	p.nextToken() // consume variable name

	if err := p.expectTagClose(); err != nil {
		return nil, err
	}

	return &IncrementTag{
		Variable: varName,
		Line:     line,
		Column:   column,
	}, nil
}

func (p *parser) parseDecrementTag() (Node, error) {
	line := p.curToken.line
	column := p.curToken.column
	p.nextToken() // consume 'decrement'

	if p.curToken.typ != tokenIdent {
		return nil, newParseError(p.curToken.line, p.curToken.column,
			"expected variable name, got %q", p.curToken.literal)
	}

	varName := p.curToken.literal
	p.nextToken() // consume variable name

	if err := p.expectTagClose(); err != nil {
		return nil, err
	}

	return &DecrementTag{
		Variable: varName,
		Line:     line,
		Column:   column,
	}, nil
}

// parsePartialTag parses {% render %} or {% include %}. When isolated is
// true the tag is render (isolated scope); otherwise it is include
// (inherits parent scope).
func (p *parser) parsePartialTag(isolated bool) (Node, error) {
	line := p.curToken.line
	column := p.curToken.column
	p.nextToken() // consume render/include

	// {% render %} requires a string-literal partial name (Ruby parity:
	// dynamically-chosen render targets are forbidden). {% include %}
	// accepts either a literal or a variable expression resolved at
	// render time.
	var (
		name        string
		templateVar Expression
	)
	if p.curToken.typ == tokenString {
		name = p.curToken.literal
		p.nextToken()
	} else if !isolated {
		expr, err := p.parsePrimary()
		if err != nil {
			return nil, newParseError(p.curToken.line, p.curToken.column,
				"expected partial name as string literal or variable, got %q", p.curToken.literal)
		}
		templateVar = expr
	} else {
		return nil, newParseError(p.curToken.line, p.curToken.column,
			"expected partial name as string literal, got %q", p.curToken.literal)
	}

	var (
		withExpr, forExpr   Expression
		withAlias, forAlias string
		args                []NamedArg
	)

	// Optional `with EXPR [as ALIAS]` or `for EXPR [as ALIAS]`. These
	// keywords are recognized contextually so they remain usable as variable
	// names elsewhere.
	switch {
	case p.isWord("with"):
		p.nextToken()
		expr, err := p.parsePrimary()
		if err != nil {
			return nil, err
		}
		withExpr = expr
		if p.isWord("as") {
			p.nextToken()
			if p.curToken.typ != tokenIdent {
				return nil, newParseError(p.curToken.line, p.curToken.column,
					"expected alias after 'as', got %q", p.curToken.literal)
			}
			withAlias = p.curToken.literal
			p.nextToken()
		}
	case p.isWord("for"):
		p.nextToken()
		expr, err := p.parsePrimary()
		if err != nil {
			return nil, err
		}
		forExpr = expr
		if p.isWord("as") {
			p.nextToken()
			if p.curToken.typ != tokenIdent {
				return nil, newParseError(p.curToken.line, p.curToken.column,
					"expected alias after 'as', got %q", p.curToken.literal)
			}
			forAlias = p.curToken.literal
			p.nextToken()
		}
	}

	// Optional named args: [, key: value]*  (also allowed without leading comma
	// when there is no with/for clause)
	for p.curToken.typ == tokenComma || p.curToken.typ == tokenIdent {
		if p.curToken.typ == tokenComma {
			p.nextToken()
		}
		if p.curToken.typ != tokenIdent {
			break
		}
		key := p.curToken.literal
		p.nextToken()
		if p.curToken.typ != tokenColon {
			return nil, newParseError(p.curToken.line, p.curToken.column,
				"expected ':' after named arg %q", key)
		}
		p.nextToken()
		val, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		args = append(args, NamedArg{Name: key, Value: val})
	}

	if err := p.expectTagClose(); err != nil {
		return nil, err
	}

	if isolated {
		return &RenderTag{
			Template:  name,
			With:      withExpr,
			WithAlias: withAlias,
			For:       forExpr,
			ForAlias:  forAlias,
			Args:      args,
			Line:      line,
			Column:    column,
		}, nil
	}
	return &IncludeTag{
		Template:     name,
		TemplateExpr: templateVar,
		With:         withExpr,
		WithAlias:    withAlias,
		For:          forExpr,
		ForAlias:     forAlias,
		Args:         args,
		Line:         line,
		Column:       column,
	}, nil
}

// parseExpression parses an expression with optional `| filter` chain.
func (p *parser) parseExpression() (Expression, error) {
	// Tolerate leading empty pipes (`{{|x|}}`): Ruby :lax silently accepts
	// them — the variable parser treats the spurious `|` as noise.
	for p.curToken.typ == tokenPipe {
		p.nextToken()
	}
	expr, err := p.parseOr()
	if err != nil {
		return nil, err
	}
	for p.curToken.typ == tokenPipe {
		// Trailing empty pipes (`{{x |}}`) and consecutive empty pipes
		// (`{{x ||a}}`) are also tolerated. Stop applying filters when
		// the next token isn't a filter name.
		if !p.peekTokenIs(tokenIdent) {
			p.nextToken()
			continue
		}
		expr, err = p.parseFilter(expr)
		if err != nil {
			return nil, err
		}
	}
	return expr, nil
}

// parseFilter consumes one `| name[: arg, key: val, ...]` segment and wraps
// the input expression in a FilterExpr. The current token must be the pipe.
func (p *parser) parseFilter(input Expression) (Expression, error) {
	line := p.curToken.line
	column := p.curToken.column
	p.nextToken() // consume |

	if p.curToken.typ != tokenIdent {
		return nil, newParseError(p.curToken.line, p.curToken.column,
			"expected filter name, got %q", p.curToken.literal)
	}
	filterName := p.curToken.literal
	p.nextToken()

	args, kwargs, err := p.parseFilterArgs()
	if err != nil {
		return nil, err
	}
	return &FilterExpr{
		Input:  input,
		Name:   filterName,
		Args:   args,
		Kwargs: kwargs,
		Line:   line,
		Column: column,
	}, nil
}

// parseFilterArgs parses the optional `: arg1, arg2, k: v, ...` argument
// list following a filter name. Once a kwarg appears, all subsequent args
// must be kwargs (matching Shopify, avoiding interleave ambiguity).
func (p *parser) parseFilterArgs() ([]Expression, []NamedArg, error) {
	if p.curToken.typ != tokenColon {
		return nil, nil, nil
	}
	p.nextToken() // consume :

	var args []Expression
	var kwargs []NamedArg
	for {
		isKwarg := p.curToken.typ == tokenIdent && p.peekTokenIs(tokenColon)
		if isKwarg {
			key := p.curToken.literal
			p.nextToken() // consume key
			p.nextToken() // consume :
			val, err := p.parseOr()
			if err != nil {
				return nil, nil, err
			}
			kwargs = append(kwargs, NamedArg{Name: key, Value: val})
		} else {
			if len(kwargs) > 0 {
				return nil, nil, newParseError(p.curToken.line, p.curToken.column,
					"positional filter arg cannot follow named arg")
			}
			arg, err := p.parseOr()
			if err != nil {
				return nil, nil, err
			}
			args = append(args, arg)
		}
		if p.curToken.typ != tokenComma {
			return args, kwargs, nil
		}
		p.nextToken() // consume comma
	}
}

// parseOr is the entry point for `and`/`or` chains. Liquid does not give
// `and` higher precedence than `or` (unlike most languages); the chain is
// evaluated as a single right-associative sequence so that
// `F and T or T` matches Shopify's "false" (short-circuit on F) rather
// than the C-style `(F and T) or T` = "true". See condition.rb's evaluate
// loop for the canonical semantics.
func (p *parser) parseOr() (Expression, error) {
	return p.parseLogical()
}

// parseLogical builds a right-associative tree of `and`/`or` operators.
// Both bind looser than every other operator (comparison, contains, etc.).
func (p *parser) parseLogical() (Expression, error) {
	left, err := p.parseContains()
	if err != nil {
		return nil, err
	}

	// Ruby lax tolerance: `&&` and `||` are silently *stripped* — the
	// operator and its right-hand operand are dropped, the left side
	// stands alone. Counter-intuitive, but it matches Ruby Liquid's
	// QuotedFragment-based tolerance (the second `&`/`|` and following
	// expression fall outside the recognized token grammar).
	if doubled, _ := p.matchDoubledLogicalOp(); doubled {
		p.nextToken()
		p.nextToken()
		// Discard the right operand (we still parse it so we advance past
		// it, but throw the result away).
		if _, err := p.parseLogical(); err != nil {
			return nil, err
		}
		return left, nil
	}

	if p.curToken.typ == tokenAnd || p.curToken.typ == tokenOr {
		op := "and"
		if p.curToken.typ == tokenOr {
			op = "or"
		}
		line := p.curToken.line
		column := p.curToken.column
		p.nextToken()

		right, err := p.parseLogical()
		if err != nil {
			return nil, err
		}
		return &BinaryExpr{Left: left, Operator: op, Right: right, Line: line, Column: column}, nil
	}

	return left, nil
}

// parseContains parses "contains" expressions.
func (p *parser) parseContains() (Expression, error) {
	left, err := p.parseComparison()
	if err != nil {
		return nil, err
	}

	if p.curToken.typ == tokenContains {
		line := p.curToken.line
		column := p.curToken.column
		p.nextToken()

		right, err := p.parseComparison()
		if err != nil {
			return nil, err
		}
		left = &BinaryExpr{Left: left, Operator: "contains", Right: right, Line: line, Column: column}
	}

	return left, nil
}

// parseComparison parses comparison expressions.
func (p *parser) parseComparison() (Expression, error) {
	left, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}

	for {
		var op string
		switch p.curToken.typ {
		case tokenEq:
			op = "=="
		case tokenNe:
			op = "!="
		case tokenLt:
			op = "<"
		case tokenGt:
			op = ">"
		case tokenLe:
			op = "<="
		case tokenGe:
			op = ">="
		default:
			return left, nil
		}

		line := p.curToken.line
		column := p.curToken.column
		p.nextToken()

		right, err := p.parsePrimary()
		if err != nil {
			return nil, err
		}
		left = &BinaryExpr{Left: left, Operator: op, Right: right, Line: line, Column: column}
	}
}

// parsePrimary parses primary expressions with member access.
func (p *parser) parsePrimary() (Expression, error) {
	expr, err := p.parseAtom()
	if err != nil {
		return nil, err
	}

	for {
		switch p.curToken.typ {
		case tokenDot:
			line := p.curToken.line
			column := p.curToken.column
			p.nextToken()

			if p.curToken.typ != tokenIdent {
				return nil, newParseError(p.curToken.line, p.curToken.column,
					"expected property name, got %q", p.curToken.literal)
			}
			prop := p.curToken.literal
			p.nextToken()
			expr = &DotExpr{Object: expr, Property: prop, Line: line, Column: column}

		case tokenLBracket:
			line := p.curToken.line
			column := p.curToken.column
			p.nextToken()

			index, err := p.parseExpression()
			if err != nil {
				return nil, err
			}

			if p.curToken.typ != tokenRBracket {
				return nil, newParseError(p.curToken.line, p.curToken.column,
					"expected ']', got %q", p.curToken.literal)
			}
			p.nextToken()
			expr = &IndexExpr{Object: expr, Index: index, Line: line, Column: column}

		default:
			// Ruby lax tolerance: `foo=>bar` (fat-arrow leaking from Ruby
			// hash syntax) — drop `=>` and the trailing primary. Tests in
			// upstream cycle/include/render/tablerow exercises this.
			if p.curToken.typ == tokenAssign {
				s := p.snapshot()
				p.nextToken()
				if p.curToken.typ == tokenGt {
					p.nextToken()
					if _, err := p.parsePrimary(); err != nil {
						return nil, err
					}
					continue
				}
				p.restore(s)
			}
			return expr, nil
		}
	}
}

// parseAtom parses atomic expressions (literals, identifiers, parenthesized expressions).
func (p *parser) parseAtom() (Expression, error) {
	line := p.curToken.line
	column := p.curToken.column

	switch p.curToken.typ {
	case tokenIdent:
		name := p.curToken.literal
		p.nextToken()
		return &IdentExpr{Name: name, Line: line, Column: column}, nil

	case tokenString:
		value := p.curToken.literal
		p.nextToken()
		return &LiteralExpr{Value: value, Line: line, Column: column}, nil

	case tokenInt:
		value := parseInt(p.curToken.literal)
		p.nextToken()
		return &LiteralExpr{Value: value, Line: line, Column: column}, nil

	case tokenFloat:
		value := parseFloat(p.curToken.literal)
		p.nextToken()
		return &LiteralExpr{Value: value, Line: line, Column: column}, nil

	case tokenTrue:
		p.nextToken()
		return &LiteralExpr{Value: true, Line: line, Column: column}, nil

	case tokenFalse:
		p.nextToken()
		return &LiteralExpr{Value: false, Line: line, Column: column}, nil

	case tokenNil:
		p.nextToken()
		return &LiteralExpr{Value: nil, Line: line, Column: column}, nil

	case tokenEmpty, tokenBlank:
		// `blank` and `empty` normally resolve to the special literal sentinels.
		// Upstream allows them to be used as identifiers when followed by a
		// member-access (`.`) or index (`[`) — `{{ blank.x }}` looks up a
		// variable named `blank`. Emit an IdentExpr in that case; parsePrimary's
		// loop will then attach the Dot/Index chain.
		name := p.curToken.literal
		isEmpty := p.curToken.typ == tokenEmpty
		p.nextToken()
		if p.curToken.typ == tokenDot || p.curToken.typ == tokenLBracket {
			return &IdentExpr{Name: name, Line: line, Column: column}, nil
		}
		if isEmpty {
			return &LiteralExpr{Value: emptyValue{}, Line: line, Column: column}, nil
		}
		return &LiteralExpr{Value: blankValue{}, Line: line, Column: column}, nil

	case tokenLParen:
		// `(start..end)` is the canonical range form. We also accept
		// `(expr)` as a grouping construct: this is needed so that
		// `{% if (a == b and c == d) %}` parses (Ruby lax mode silently
		// permits this; literal Ruby Liquid rejected it).
		p.nextToken() // consume (
		first, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		if p.curToken.typ == tokenRange {
			p.nextToken() // consume ..
			// Tolerate extra dots: `(1...5)` lexes `1`, `..`, `.`, `5`.
			// Ruby lax mode silently accepts the third dot. Drain any
			// dots before the end bound.
			for p.curToken.typ == tokenDot {
				p.nextToken()
			}
			end, err := p.parseOr()
			if err != nil {
				return nil, err
			}
			if p.curToken.typ != tokenRParen {
				return nil, newParseError(p.curToken.line, p.curToken.column,
					"expected ')' to close range, got %q", p.curToken.literal)
			}
			p.nextToken()
			return &RangeExpr{Start: first, End: end, Line: line, Column: column}, nil
		}
		if p.curToken.typ != tokenRParen {
			return nil, newParseError(p.curToken.line, p.curToken.column,
				"expected ')' to close group, got %q", p.curToken.literal)
		}
		p.nextToken()
		return first, nil

	case tokenMinus:
		// Unary minus for negative numbers
		p.nextToken()
		if p.curToken.typ == tokenInt {
			value := -parseInt(p.curToken.literal)
			p.nextToken()
			return &LiteralExpr{Value: value, Line: line, Column: column}, nil
		}
		if p.curToken.typ == tokenFloat {
			value := -parseFloat(p.curToken.literal)
			p.nextToken()
			return &LiteralExpr{Value: value, Line: line, Column: column}, nil
		}
		return nil, newParseError(p.curToken.line, p.curToken.column,
			"expected number after '-', got %q", p.curToken.literal)

	default:
		return nil, newParseError(line, column,
			"unexpected token in expression: %q", p.curToken.literal)
	}
}

// isWord reports whether the current token is the identifier `word`. Used
// to recognize context-sensitive keywords (`in`, `with`, `as`, `limit`,
// `offset`, `reversed`) that are not globally reserved.
func (p *parser) isWord(word string) bool {
	return p.curToken.typ == tokenIdent && p.curToken.literal == word
}

// parserSnapshot captures the parser+lexer state needed to peek ahead
// without consuming input. Used by peekTokenIs and isTagKeyword for the
// one-token lookahead they each need.
type parserSnapshot struct {
	tok          token
	pos, readPos int
	ch           byte
	line, column int
	mode         lexerMode
}

func (p *parser) snapshot() parserSnapshot {
	return parserSnapshot{
		tok:     p.curToken,
		pos:     p.l.pos,
		readPos: p.l.readPos,
		ch:      p.l.ch,
		line:    p.l.line,
		column:  p.l.column,
		mode:    p.l.mode,
	}
}

func (p *parser) restore(s parserSnapshot) {
	p.curToken = s.tok
	p.l.pos = s.pos
	p.l.readPos = s.readPos
	p.l.ch = s.ch
	p.l.line = s.line
	p.l.column = s.column
	p.l.mode = s.mode
}

// matchDoubledLogicalOp reports whether the current+peek tokens form
// `&&` or `||` (Ruby lax mode treats both as noise to skip — the operator
// and its right operand are dropped, leaving only the left side).
// Returns the conceptual operator name ("and"/"or") for callers that
// want to log or otherwise distinguish the two.
func (p *parser) matchDoubledLogicalOp() (matched bool, op string) {
	switch {
	case p.curToken.typ == tokenIllegal && p.curToken.literal == "&":
		s := p.snapshot()
		p.nextToken()
		hit := p.curToken.typ == tokenIllegal && p.curToken.literal == "&"
		p.restore(s)
		if hit {
			return true, "and"
		}
	case p.curToken.typ == tokenPipe:
		s := p.snapshot()
		p.nextToken()
		hit := p.curToken.typ == tokenPipe
		p.restore(s)
		if hit {
			return true, "or"
		}
	}
	return false, ""
}

// peekTokenIs reports whether the token immediately following curToken
// has the given type. State is saved and restored so callers see no side
// effects.
//
// Explicit restore (rather than `defer p.restore`) — this is on the
// hot path of every parseNodes loop iteration through if/for/case
// bodies, and defer's frame setup measurably slows parsing.
func (p *parser) peekTokenIs(typ TokenType) bool {
	s := p.snapshot()
	p.nextToken()
	matched := p.curToken.typ == typ
	p.restore(s)
	return matched
}

// isTagKeyword peeks past `{%` (or `{%-`) to see whether the next token
// is the identifier `word`. Used to detect end-of-block markers like
// `endif`, `endfor`, `else`, `when` from within a parseNodes loop
// without consuming the tag delimiter.
func (p *parser) isTagKeyword(word string) bool {
	if p.curToken.typ != tokenTagOpen && p.curToken.typ != tokenTagTrim {
		return false
	}
	s := p.snapshot()
	p.nextToken()
	matched := p.curToken.typ == tokenIdent && p.curToken.literal == word
	p.restore(s)
	return matched
}

// isTagKeywordIn is isTagKeyword's batched form: one snapshot/peek/
// restore cycle answers whether the peek matches any of `words`. The
// parseNodes endCondition for if/unless/case blocks asks about three
// possible terminator keywords on every iteration, so collapsing them
// into a single peek measurably reduces parse time on large templates.
func (p *parser) isTagKeywordIn(words ...string) bool {
	if p.curToken.typ != tokenTagOpen && p.curToken.typ != tokenTagTrim {
		return false
	}
	s := p.snapshot()
	p.nextToken()
	matched := false
	if p.curToken.typ == tokenIdent {
		lit := p.curToken.literal
		for _, w := range words {
			if lit == w {
				matched = true
				break
			}
		}
	}
	p.restore(s)
	return matched
}

// expectTagClose expects and consumes a tag close token (%} or -%}).
// Sets trimNextText so the next text node will be trimmed if -%} was used.
func (p *parser) expectTagClose() error {
	if p.curToken.typ != tokenTagClose && p.curToken.typ != tokenTagTrimR {
		return newParseError(p.curToken.line, p.curToken.column,
			"expected %%}, got %q", p.curToken.literal)
	}
	p.trimNextText = p.curToken.typ == tokenTagTrimR
	p.nextToken()
	return nil
}

// drainAndExpectTagClose silently swallows any extra tokens before `%}`.
// Used by tags whose Ruby parsers tolerate trailing garbage (e.g.
// `{% case 1 bar %}` or `{% when 1 bar %}`). Mirrors Ruby :lax/:strict
// tolerance — :strict2 would reject these.
func (p *parser) drainAndExpectTagClose() error {
	for p.curToken.typ != tokenTagClose &&
		p.curToken.typ != tokenTagTrimR &&
		p.curToken.typ != tokenEOF {
		p.nextToken()
	}
	return p.expectTagClose()
}

// validateTagClose validates but doesn't advance past a tag close token (%} or -%}).
// Used for raw and comment tags where we need to directly scan for the end tag.
// Sets trimNextText so the next text node will be trimmed if -%} was used.
func (p *parser) validateTagClose() error {
	if p.curToken.typ != tokenTagClose && p.curToken.typ != tokenTagTrimR {
		return newParseError(p.curToken.line, p.curToken.column,
			"expected %%}, got %q", p.curToken.literal)
	}
	p.trimNextText = p.curToken.typ == tokenTagTrimR
	// Don't call nextToken() - we'll scan raw content directly
	return nil
}

// drainToTagClose silently skips any tokens before the closing %}/-%} without
// consuming the close itself. Used by tags whose Ruby parser tolerates trailing
// garbage and which then need to hand control back to the lexer for raw-text
// scanning (e.g. `{% comment foo %}`, `{% raw bar %}`).
func (p *parser) drainToTagClose() {
	for p.curToken.typ != tokenTagClose &&
		p.curToken.typ != tokenTagTrimR &&
		p.curToken.typ != tokenEOF {
		p.nextToken()
	}
}

// parseInt parses an integer literal.
func parseInt(s string) int64 {
	n, _ := strconv.ParseInt(s, 10, 64)
	return n
}

// parseFloat parses a float literal.
func parseFloat(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

// Special values for empty and blank comparisons.
type (
	emptyValue struct{}
	blankValue struct{}
)

func (e emptyValue) String() string { return "empty" }
func (b blankValue) String() string { return "blank" }
