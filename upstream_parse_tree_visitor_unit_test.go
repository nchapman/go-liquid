package liquid

import "testing"

// Upstream parity: unit/parse_tree_visitor_test.rb
//
// Ruby's Liquid::ParseTreeVisitor walks the AST returning each node's
// children — it's an introspection helper for tooling. go-liquid does
// not expose an AST visitor API.

func TestUpstream_ParseTreeVisitor_Variable(t *testing.T) { t.Skip("AST visitor not exposed") }
func TestUpstream_ParseTreeVisitor_VaribleWithFilter(t *testing.T) {
	t.Skip("AST visitor not exposed")
}
func TestUpstream_ParseTreeVisitor_DynamicVariable(t *testing.T) {
	t.Skip("AST visitor not exposed")
}
func TestUpstream_ParseTreeVisitor_Echo(t *testing.T)          { t.Skip("AST visitor not exposed") }
func TestUpstream_ParseTreeVisitor_IfCondition(t *testing.T)   { t.Skip("AST visitor not exposed") }
func TestUpstream_ParseTreeVisitor_ComplexIfCondition(t *testing.T) {
	t.Skip("AST visitor not exposed")
}
func TestUpstream_ParseTreeVisitor_IfBody(t *testing.T)        { t.Skip("AST visitor not exposed") }
func TestUpstream_ParseTreeVisitor_UnlessCondition(t *testing.T) {
	t.Skip("AST visitor not exposed")
}
func TestUpstream_ParseTreeVisitor_ComplexUnlessCondition(t *testing.T) {
	t.Skip("AST visitor not exposed")
}
func TestUpstream_ParseTreeVisitor_UnlessBody(t *testing.T)     { t.Skip("AST visitor not exposed") }
func TestUpstream_ParseTreeVisitor_ElsifCondition(t *testing.T) { t.Skip("AST visitor not exposed") }
func TestUpstream_ParseTreeVisitor_ComplexElsifCondition(t *testing.T) {
	t.Skip("AST visitor not exposed")
}
func TestUpstream_ParseTreeVisitor_ElsifBody(t *testing.T)    { t.Skip("AST visitor not exposed") }
func TestUpstream_ParseTreeVisitor_ElseBody(t *testing.T)     { t.Skip("AST visitor not exposed") }
func TestUpstream_ParseTreeVisitor_CaseLeft(t *testing.T)     { t.Skip("AST visitor not exposed") }
func TestUpstream_ParseTreeVisitor_CaseCondition(t *testing.T) { t.Skip("AST visitor not exposed") }
func TestUpstream_ParseTreeVisitor_CaseWhenBody(t *testing.T) { t.Skip("AST visitor not exposed") }
func TestUpstream_ParseTreeVisitor_CaseElseBody(t *testing.T) { t.Skip("AST visitor not exposed") }
func TestUpstream_ParseTreeVisitor_ForIn(t *testing.T)        { t.Skip("AST visitor not exposed") }
func TestUpstream_ParseTreeVisitor_ForLimit(t *testing.T)     { t.Skip("AST visitor not exposed") }
func TestUpstream_ParseTreeVisitor_ForOffset(t *testing.T)    { t.Skip("AST visitor not exposed") }
func TestUpstream_ParseTreeVisitor_ForBody(t *testing.T)      { t.Skip("AST visitor not exposed") }
func TestUpstream_ParseTreeVisitor_ForRange(t *testing.T)     { t.Skip("AST visitor not exposed") }
func TestUpstream_ParseTreeVisitor_TablerowIn(t *testing.T)   { t.Skip("AST visitor not exposed") }
func TestUpstream_ParseTreeVisitor_TablerowLimit(t *testing.T) { t.Skip("AST visitor not exposed") }
func TestUpstream_ParseTreeVisitor_TablerowOffset(t *testing.T) {
	t.Skip("AST visitor not exposed")
}
func TestUpstream_ParseTreeVisitor_TablerowBody(t *testing.T) { t.Skip("AST visitor not exposed") }
func TestUpstream_ParseTreeVisitor_Cycle(t *testing.T)        { t.Skip("AST visitor not exposed") }
func TestUpstream_ParseTreeVisitor_Assign(t *testing.T)       { t.Skip("AST visitor not exposed") }
func TestUpstream_ParseTreeVisitor_Capture(t *testing.T)      { t.Skip("AST visitor not exposed") }
func TestUpstream_ParseTreeVisitor_Include(t *testing.T)      { t.Skip("AST visitor not exposed") }
func TestUpstream_ParseTreeVisitor_IncludeWith(t *testing.T)  { t.Skip("AST visitor not exposed") }
func TestUpstream_ParseTreeVisitor_IncludeFor(t *testing.T)   { t.Skip("AST visitor not exposed") }
func TestUpstream_ParseTreeVisitor_RenderWith(t *testing.T)   { t.Skip("AST visitor not exposed") }
func TestUpstream_ParseTreeVisitor_RenderFor(t *testing.T)    { t.Skip("AST visitor not exposed") }
func TestUpstream_ParseTreeVisitor_PreserveTreeStructure(t *testing.T) {
	t.Skip("AST visitor not exposed")
}
