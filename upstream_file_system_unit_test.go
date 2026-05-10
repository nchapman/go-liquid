//go:build ruby_internals

package liquid

import "testing"

// Upstream parity: unit/file_system_unit_test.rb
//
// Ruby's Liquid::FileSystem / Liquid::LocalFileSystem with custom
// template_filename_patterns is not modeled in go-liquid. go-liquid
// uses Loader (MapLoader / WithLoader) — a simpler interface.

func TestUpstream_FileSystemUnit_Default(t *testing.T) {
	t.Skip("Ruby BlankFileSystem default; go-liquid requires WithLoader for {% include %}/{% render %}")
}

func TestUpstream_FileSystemUnit_Local(t *testing.T) {
	t.Skip("Ruby LocalFileSystem (filesystem-backed) not modeled; go-liquid uses Loader interface")
}

func TestUpstream_FileSystemUnit_CustomTemplateFilenamePatterns(t *testing.T) {
	t.Skip("Ruby Liquid::LocalFileSystem.pattern (filename pattern interpolation) not modeled")
}
