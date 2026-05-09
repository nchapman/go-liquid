package liquid

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestTemplateSpecs walks testdata/spec.
//
// Two flavors of scenario are supported:
//
//  1. Flat scenarios — a *.tmpl (or *.liquid) file at the top of the spec
//     directory paired with a sibling *.tmpl.out (or *.liquid.out). The
//     template may carry a `{# data: {...} #}` JSON directive supplying the
//     render data.
//
//  2. Partial scenarios — a subdirectory containing index.tmpl plus any
//     number of additional *.tmpl files used as partials. The partial name
//     in {% render %} / {% include %} is the file basename without
//     extension. expected.out holds the golden output. Data may live in the
//     index.tmpl directive or in a sibling data.json file.
func TestTemplateSpecs(t *testing.T) {
	specDir := "testdata/spec"

	entries, err := os.ReadDir(specDir)
	if err != nil {
		t.Fatalf("failed to read spec directory: %v", err)
	}

	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			// Partial scenario: must have index.tmpl + expected.out
			path := filepath.Join(specDir, name)
			if _, err := os.Stat(filepath.Join(path, "index.tmpl")); err != nil {
				continue // not a partial scenario (e.g., the i18n fixture dir)
			}
			t.Run(name, func(t *testing.T) {
				runPartialSpec(t, path)
			})
			continue
		}

		var testName string
		switch {
		case strings.HasSuffix(name, ".tmpl"):
			testName = strings.TrimSuffix(name, ".tmpl")
		case strings.HasSuffix(name, ".liquid"):
			if strings.HasSuffix(name, ".liquid.out") {
				continue
			}
			testName = strings.TrimSuffix(name, ".liquid")
		default:
			continue
		}

		t.Run(testName, func(t *testing.T) {
			runTemplateSpec(t, filepath.Join(specDir, name))
		})
	}
}

// runPartialSpec renders index.tmpl with all sibling *.tmpl files registered
// as partials (loaded by basename without extension) and compares to
// expected.out.
func runPartialSpec(t *testing.T, dir string) {
	t.Helper()
	indexPath := filepath.Join(dir, "index.tmpl")
	indexBytes, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("read %s: %v", indexPath, err)
	}
	source := string(indexBytes)

	data := extractData(t, source)
	source = removeDataDirective(source)

	// Optional data.json overrides any inline directive.
	if b, err := os.ReadFile(filepath.Join(dir, "data.json")); err == nil {
		var d map[string]any
		if err := json.Unmarshal(b, &d); err != nil {
			t.Fatalf("parse data.json: %v", err)
		}
		data = d
	}

	// Build a MapLoader from sibling *.tmpl files (excluding index.tmpl).
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("readdir %s: %v", dir, err)
	}
	loader := MapLoader{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".tmpl") || e.Name() == "index.tmpl" {
			continue
		}
		body, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatalf("read partial %s: %v", e.Name(), err)
		}
		loader[strings.TrimSuffix(e.Name(), ".tmpl")] = string(body)
	}

	tmpl, err := Parse(source)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	tmpl.WithLoader(loader)

	result, err := tmpl.Render(data)
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	expected, err := os.ReadFile(filepath.Join(dir, "expected.out"))
	if err != nil {
		t.Fatalf("read expected.out: %v", err)
	}
	if result != string(expected) {
		t.Errorf("output mismatch:\n--- expected ---\n%s\n--- actual ---\n%s", expected, result)
	}
}

func runTemplateSpec(t *testing.T, templatePath string) {
	// Read template file
	templateBytes, err := os.ReadFile(templatePath)
	if err != nil {
		t.Fatalf("failed to read template: %v", err)
	}
	templateContent := string(templateBytes)

	// Extract data from {# data: {...} #} comment
	data := extractData(t, templateContent)

	// Remove the data directive from template
	template := removeDataDirective(templateContent)

	// Render
	result, err := Render(template, data)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	// Read expected output
	expectedPath := templatePath + ".out"
	expectedBytes, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("failed to read expected output %s: %v", expectedPath, err)
	}
	expected := string(expectedBytes)

	// Compare
	if result != expected {
		t.Errorf("output mismatch:\n--- expected ---\n%s\n--- actual ---\n%s", expected, result)
	}
}

func extractData(t *testing.T, content string) map[string]any {
	// Try to find {# data: ... #} which may span multiple lines
	start := strings.Index(content, "{# data:")
	if start == -1 {
		return make(map[string]any)
	}

	// Find the closing #}
	end := strings.Index(content[start:], "#}")
	if end == -1 {
		return make(map[string]any)
	}
	end += start + 2 // include the #}

	// Extract just the JSON part
	directive := content[start:end]
	jsonStart := strings.Index(directive, "{# data:") + len("{# data:")
	jsonEnd := strings.LastIndex(directive, "#}")

	jsonStr := strings.TrimSpace(directive[jsonStart:jsonEnd])

	var data map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		t.Fatalf("failed to parse data directive: %v\nJSON: %s", err, jsonStr)
	}
	return data
}

func removeDataDirective(content string) string {
	// Find and remove {# data: ... #} which may span multiple lines
	start := strings.Index(content, "{# data:")
	if start == -1 {
		return content
	}

	end := strings.Index(content[start:], "#}")
	if end == -1 {
		return content
	}
	end += start + 2 // include the #}

	// Also remove the trailing newline if present
	if end < len(content) && content[end] == '\n' {
		end++
	}

	return content[:start] + content[end:]
}
