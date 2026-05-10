package liquid

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Loader resolves a partial name (as written in {% render %} / {% include %})
// into template source. Implementations must be safe for concurrent use.
type Loader interface {
	Load(name string) (source string, err error)
}

// MapLoader is an in-memory Loader. Useful for tests and embedded templates.
type MapLoader map[string]string

// Load returns the source registered for name, or an error if absent.
func (m MapLoader) Load(name string) (string, error) {
	if src, ok := m[name]; ok {
		return src, nil
	}
	return "", fmt.Errorf("template %q not found", name)
}

// FileSystemLoader reads partials from an fs.FS rooted at Root. If Ext is
// non-empty it is appended to the requested name (e.g. ".liquid"). Path
// traversal escapes (".." segments, absolute paths) are rejected.
type FileSystemLoader struct {
	FS  fs.FS
	Ext string
}

// NewFileSystemLoader returns a loader that reads from the given directory.
func NewFileSystemLoader(root, ext string) *FileSystemLoader {
	return &FileSystemLoader{FS: os.DirFS(root), Ext: ext}
}

// Load implements Loader.
func (f *FileSystemLoader) Load(name string) (string, error) {
	if f.FS == nil {
		return "", fmt.Errorf("FileSystemLoader: FS not set")
	}
	clean, err := safePartialPath(name, f.Ext)
	if err != nil {
		return "", err
	}
	b, err := fs.ReadFile(f.FS, clean)
	if err != nil {
		// Don't wrap the underlying *fs.PathError: on real filesystems it
		// embeds the absolute on-disk path, which leaks the loader's root.
		// Preserve fs.ErrNotExist for callers that type-check via errors.Is.
		if errors.Is(err, fs.ErrNotExist) {
			return "", fmt.Errorf("partial %q: %w", name, fs.ErrNotExist)
		}
		return "", fmt.Errorf("partial %q: load failed", name)
	}
	return string(b), nil
}

// safePartialPath rejects names that would escape the loader's root.
func safePartialPath(name, ext string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("partial name is empty")
	}
	if filepath.IsAbs(name) || strings.HasPrefix(name, "/") {
		return "", fmt.Errorf("partial name %q must be relative", name)
	}
	clean := filepath.ToSlash(filepath.Clean(name))
	if clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("partial name %q escapes root", name)
	}
	if ext != "" && !strings.HasSuffix(clean, ext) {
		clean += ext
	}
	return clean, nil
}

// partialCache memoizes parsed partials per (Loader, name). It lives on a
// Template and is shared with any partials it loads, so a partial that
// includes another partial reuses the same cache.
type partialCache struct {
	loader Loader
	env    *Environment
	mu     sync.Mutex
	parsed map[string]*Template
}

func newPartialCache(loader Loader, env *Environment) *partialCache {
	if env == nil {
		env = Default()
	}
	return &partialCache{loader: loader, env: env, parsed: make(map[string]*Template)}
}

func (c *partialCache) get(name string) (*Template, error) {
	if c == nil || c.loader == nil {
		return nil, fmt.Errorf("no loader configured for partial %q", name)
	}
	c.mu.Lock()
	if t, ok := c.parsed[name]; ok {
		c.mu.Unlock()
		return t, nil
	}
	c.mu.Unlock()

	src, err := c.loader.Load(name)
	if err != nil {
		return nil, err
	}
	t, err := c.env.Parse(src)
	if err != nil {
		// Surface the partial name on the underlying ParseError so
		// errors.As can extract it; falls back to a wrap for non-ParseError
		// failures (which Parse doesn't currently produce, but keep the
		// belt-and-suspenders).
		pe := &ParseError{}
		if errors.As(err, &pe) {
			pe.TemplateName = name
			return nil, pe
		}
		return nil, fmt.Errorf("partial %q: %w", name, err)
	}
	t.name = name
	t.partials.Store(c) // share cache with nested partials

	c.mu.Lock()
	if existing, ok := c.parsed[name]; ok {
		c.mu.Unlock()
		return existing, nil // race: someone else parsed it first
	}
	c.parsed[name] = t
	c.mu.Unlock()
	return t, nil
}
