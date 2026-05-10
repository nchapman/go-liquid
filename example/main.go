// Package main demonstrates basic use of the go-liquid template engine.
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/nchapman/go-liquid"
)

const source = `
Hi {{ user.name | capitalize }},

You have {{ items | size }} items in your cart:
{% for item in items -%}
  - {{ item.name }} (${{ item.price | round: 2 }})
{% endfor -%}

Total: ${{ total | round: 2 }}
`

func main() {
	tmpl, err := liquid.Parse(source)
	if err != nil {
		log.Fatal(err)
	}

	data := map[string]any{
		"user": map[string]any{"name": "alice"},
		"items": []map[string]any{
			{"name": "Widget", "price": 9.99},
			{"name": "Gadget", "price": 24.5},
		},
		"total": 34.49,
	}

	if err := tmpl.RenderTo(os.Stdout, data); err != nil {
		log.Fatal(err)
	}
	fmt.Println()
}
