package liquid

// Node is the interface for all AST nodes.
type Node interface {
	node()
	Pos() (line, column int)
}

// Template represents a parsed Liquid template.
type templateAST struct {
	nodes []Node
}

func (t *templateAST) node()                   {}
func (t *templateAST) Pos() (line, column int) { return 1, 1 }

// TextNode represents raw text content.
type TextNode struct {
	Text   string
	Line   int
	Column int
}

func (n *TextNode) node()                   {}
func (n *TextNode) Pos() (line, column int) { return n.Line, n.Column }

// OutputNode represents {{ expression }}.
type OutputNode struct {
	Expr   Expression
	Line   int
	Column int
}

func (n *OutputNode) node()                   {}
func (n *OutputNode) Pos() (line, column int) { return n.Line, n.Column }

// Expression is the interface for expression nodes.
type Expression interface {
	Node
	expr()
}

// IdentExpr represents an identifier like "name" or "item".
type IdentExpr struct {
	Name   string
	Line   int
	Column int
}

func (e *IdentExpr) node()                   {}
func (e *IdentExpr) expr()                   {}
func (e *IdentExpr) Pos() (line, column int) { return e.Line, e.Column }

// LiteralExpr represents a literal value (string, int, float, bool, nil).
type LiteralExpr struct {
	Value  any
	Line   int
	Column int
}

func (e *LiteralExpr) node()                   {}
func (e *LiteralExpr) expr()                   {}
func (e *LiteralExpr) Pos() (line, column int) { return e.Line, e.Column }

// DotExpr represents object.property access.
type DotExpr struct {
	Object   Expression
	Property string
	Line     int
	Column   int
}

func (e *DotExpr) node()                   {}
func (e *DotExpr) expr()                   {}
func (e *DotExpr) Pos() (line, column int) { return e.Line, e.Column }

// IndexExpr represents array[index] access.
type IndexExpr struct {
	Object Expression
	Index  Expression
	Line   int
	Column int
}

func (e *IndexExpr) node()                   {}
func (e *IndexExpr) expr()                   {}
func (e *IndexExpr) Pos() (line, column int) { return e.Line, e.Column }

// FilterExpr represents expr | filter or expr | filter: arg.
type FilterExpr struct {
	Input  Expression
	Name   string
	Args   []Expression
	Line   int
	Column int
}

func (e *FilterExpr) node()                   {}
func (e *FilterExpr) expr()                   {}
func (e *FilterExpr) Pos() (line, column int) { return e.Line, e.Column }

// BinaryExpr represents binary operations (==, !=, <, >, etc).
type BinaryExpr struct {
	Left     Expression
	Operator string
	Right    Expression
	Line     int
	Column   int
}

func (e *BinaryExpr) node()                   {}
func (e *BinaryExpr) expr()                   {}
func (e *BinaryExpr) Pos() (line, column int) { return e.Line, e.Column }

// RangeExpr represents (start..end) range.
type RangeExpr struct {
	Start  Expression
	End    Expression
	Line   int
	Column int
}

func (e *RangeExpr) node()                   {}
func (e *RangeExpr) expr()                   {}
func (e *RangeExpr) Pos() (line, column int) { return e.Line, e.Column }

// IfTag represents {% if %}...{% endif %}.
type IfTag struct {
	Condition     Expression
	ThenBranch    []Node
	ElsifBranches []struct {
		Condition Expression
		Body      []Node
	}
	ElseBranch []Node
	Line       int
	Column     int
}

func (t *IfTag) node()                   {}
func (t *IfTag) Pos() (line, column int) { return t.Line, t.Column }

// UnlessTag represents {% unless %}...{% endunless %}.
type UnlessTag struct {
	Condition  Expression
	Body       []Node
	ElseBranch []Node
	Line       int
	Column     int
}

func (t *UnlessTag) node()                   {}
func (t *UnlessTag) Pos() (line, column int) { return t.Line, t.Column }

// CaseTag represents {% case %}...{% endcase %}.
type CaseTag struct {
	Value  Expression
	Whens  []WhenClause
	Else   []Node
	Line   int
	Column int
}

type WhenClause struct {
	Values []Expression
	Body   []Node
}

func (t *CaseTag) node()                   {}
func (t *CaseTag) Pos() (line, column int) { return t.Line, t.Column }

// ForTag represents {% for item in collection %}...{% endfor %}.
type ForTag struct {
	Variable   string
	Collection Expression
	Body       []Node
	ElseBody   []Node // rendered when collection is empty
	Limit      Expression
	Offset     Expression
	Reversed   bool
	Line       int
	Column     int
}

func (t *ForTag) node()                   {}
func (t *ForTag) Pos() (line, column int) { return t.Line, t.Column }

// BreakTag represents {% break %}.
type BreakTag struct {
	Line   int
	Column int
}

func (t *BreakTag) node()                   {}
func (t *BreakTag) Pos() (line, column int) { return t.Line, t.Column }

// ContinueTag represents {% continue %}.
type ContinueTag struct {
	Line   int
	Column int
}

func (t *ContinueTag) node()                   {}
func (t *ContinueTag) Pos() (line, column int) { return t.Line, t.Column }

// AssignTag represents {% assign var = expr %}.
type AssignTag struct {
	Variable string
	Value    Expression
	Line     int
	Column   int
}

func (t *AssignTag) node()                   {}
func (t *AssignTag) Pos() (line, column int) { return t.Line, t.Column }

// CaptureTag represents {% capture var %}...{% endcapture %}.
type CaptureTag struct {
	Variable string
	Body     []Node
	Line     int
	Column   int
}

func (t *CaptureTag) node()                   {}
func (t *CaptureTag) Pos() (line, column int) { return t.Line, t.Column }

// CommentTag represents {% comment %}...{% endcomment %}.
type CommentTag struct {
	Content string
	Line    int
	Column  int
}

func (t *CommentTag) node()                   {}
func (t *CommentTag) Pos() (line, column int) { return t.Line, t.Column }

// RawTag represents {% raw %}...{% endraw %}.
type RawTag struct {
	Content string
	Line    int
	Column  int
}

func (t *RawTag) node()                   {}
func (t *RawTag) Pos() (line, column int) { return t.Line, t.Column }

// CycleTag represents {% cycle "a", "b", "c" %} or {% cycle "group": "a", "b", "c" %}.
type CycleTag struct {
	GroupName string       // optional group name
	Values    []Expression // values to cycle through
	Line      int
	Column    int
}

func (t *CycleTag) node()                   {}
func (t *CycleTag) Pos() (line, column int) { return t.Line, t.Column }

// IncrementTag represents {% increment var %}.
type IncrementTag struct {
	Variable string
	Line     int
	Column   int
}

func (t *IncrementTag) node()                   {}
func (t *IncrementTag) Pos() (line, column int) { return t.Line, t.Column }

// DecrementTag represents {% decrement var %}.
type DecrementTag struct {
	Variable string
	Line     int
	Column   int
}

func (t *DecrementTag) node()                   {}
func (t *DecrementTag) Pos() (line, column int) { return t.Line, t.Column }
