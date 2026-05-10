package liquid

import "testing"

// Upstream parity: unit/tags/for_tag_unit_test.rb
//
// Both upstream tests inspect template.root.nodelist[0].nodelist — AST
// introspection on the parsed ForTag node — which go-liquid does not
// expose. Behaviorally equivalent end-to-end coverage lives in
// upstream_for_tag_test.go.

func TestUpstream_ForTagUnit_ForNodelist(t *testing.T) {
	t.Skip("Ruby Template#root#nodelist AST introspection not exposed; behavior covered by integration tests")
}

func TestUpstream_ForTagUnit_ForElseNodelist(t *testing.T) {
	t.Skip("Ruby AST introspection not exposed; behavior covered by integration tests")
}
