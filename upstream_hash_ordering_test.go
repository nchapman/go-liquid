package liquid

import "testing"

// Upstream parity: integration/hash_ordering_test.rb

func TestUpstream_HashOrdering_GlobalRegisterOrder(t *testing.T) {
	// Later filter registration wins. Equivalent of Ruby's with_global_filter
	// chaining MoneyFilter then CanadianMoneyFilter.
	env := NewEnvironment()
	env.RegisterFilter("money", func(input any, args ...any) any { return " 1000$ " })
	env.RegisterFilter("money", func(input any, args ...any) any { return " 1000$ CAD " })
	tmpl, err := env.Parse("{{1000 | money}}")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	got, err := tmpl.Render(nil)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if got != " 1000$ CAD " {
		t.Fatalf("got %q", got)
	}
}
