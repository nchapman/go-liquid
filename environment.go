package liquid

import (
	"sync"
)

// Environment is a container for the tags, filters, and default loader that
// templates parsed through it will see. Mirrors Ruby Liquid's
// Liquid::Environment: lets a process embed multiple isolated configurations
// without leaking custom filters/tags through a process-global registry.
//
// Use NewEnvironment for an isolated scope. The package-level
// RegisterFilter/RegisterTag/Parse functions delegate to Default(), which is
// the implicit environment shared by all callers that don't construct one
// explicitly.
//
// All registration methods are safe to call concurrently with rendering.
// Lookups take a read-lock; registration takes a write-lock.
type Environment struct {
	mu         sync.RWMutex
	filters    map[string]Filter
	inlineTags map[string]TagParser
	blockTags  map[string]TagParser
	loader     Loader
	errorMode  ErrorMode
}

// NewEnvironment returns a fresh environment pre-loaded with the standard
// filters. User-registered filters and tags from Default() are NOT inherited
// — that is the whole point of having an isolated environment.
func NewEnvironment() *Environment {
	e := &Environment{
		filters:    map[string]Filter{},
		inlineTags: map[string]TagParser{},
		blockTags:  map[string]TagParser{},
	}
	registerStandardFilters(e)
	return e
}

var (
	defaultEnvOnce sync.Once
	defaultEnv     *Environment
)

// Default returns the package-wide environment that backs the package-level
// RegisterFilter/RegisterTag/Parse functions. Callers that want isolation
// should construct their own via NewEnvironment instead of mutating Default.
func Default() *Environment {
	defaultEnvOnce.Do(func() {
		defaultEnv = NewEnvironment()
	})
	return defaultEnv
}

// RegisterFilter installs a positional-argument filter. Overrides any
// existing filter (built-in or user) with the same name.
func (e *Environment) RegisterFilter(name string, fn FilterFunc) {
	e.mu.Lock()
	e.filters[name] = fn
	e.mu.Unlock()
}

// RegisterKwargFilter installs a filter that accepts named arguments. See
// RegisterFilter for override semantics.
func (e *Environment) RegisterKwargFilter(name string, fn KwargFilterFunc) {
	e.mu.Lock()
	e.filters[name] = fn
	e.mu.Unlock()
}

// RegisterFilterE installs a filter using the (any, error)-returning
// signature. Preferred for new filters: errors are first-class.
func (e *Environment) RegisterFilterE(name string, fn FilterFuncE) {
	e.mu.Lock()
	e.filters[name] = fn
	e.mu.Unlock()
}

// RegisterTag installs an inline custom tag (no body). Panics if the name
// shadows a built-in tag.
func (e *Environment) RegisterTag(name string, parse TagParser) {
	guardCustomTagName(name)
	e.mu.Lock()
	e.inlineTags[name] = parse
	e.mu.Unlock()
}

// RegisterBlock installs a block custom tag with a body terminated by
// {% endNAME %}. Panics if the name shadows a built-in tag.
func (e *Environment) RegisterBlock(name string, parse TagParser) {
	guardCustomTagName(name)
	e.mu.Lock()
	e.blockTags[name] = parse
	e.mu.Unlock()
}

// WithErrorMode sets the parse-time error mode for templates parsed
// through this environment. The default is ErrorModeStrict. See
// ErrorMode for the semantics of each setting. Returns the environment
// for chaining.
func (e *Environment) WithErrorMode(mode ErrorMode) *Environment {
	e.mu.Lock()
	e.errorMode = mode
	e.mu.Unlock()
	return e
}

// ErrorMode returns the current parse-time error mode.
func (e *Environment) ErrorMode() ErrorMode {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.errorMode
}

// WithLoader sets the default Loader used by templates parsed through this
// environment, so they can resolve {% render %} / {% include %} partials
// without an explicit per-template Template.WithLoader. Returns the
// environment for chaining.
func (e *Environment) WithLoader(l Loader) *Environment {
	e.mu.Lock()
	e.loader = l
	e.mu.Unlock()
	return e
}

// Parse parses source against this environment. Custom tags are resolved
// from the environment's registry; filters are resolved at render time
// from the same environment. If the environment has a default loader
// configured via WithLoader, the resulting template is pre-attached to it.
func (e *Environment) Parse(source string) (*Template, error) {
	p := newParserWithEnv(source, e)
	ast, err := p.parse()
	if err != nil {
		return nil, err
	}
	t := &Template{ast: ast, env: e, warnings: p.warnings}
	if l := e.currentLoader(); l != nil {
		t.WithLoader(l)
	}
	return t, nil
}

// MustParse is like Parse but panics on error.
func (e *Environment) MustParse(source string) *Template {
	t, err := e.Parse(source)
	if err != nil {
		panic(err)
	}
	return t
}

// Render parses and renders source against data in one call. For
// repeatedly-rendered templates, prefer Parse + Template.Render.
func (e *Environment) Render(source string, data any, opts ...RenderOption) (string, error) {
	t, err := e.Parse(source)
	if err != nil {
		return "", err
	}
	return t.Render(data, opts...)
}

func (e *Environment) lookupFilter(name string) (Filter, bool) {
	e.mu.RLock()
	fn, ok := e.filters[name]
	e.mu.RUnlock()
	return fn, ok
}

// lookupCustomTag returns the registered parser for name, reporting whether
// it was found and whether it expects a body (block).
func (e *Environment) lookupCustomTag(name string) (parse TagParser, isBlock, ok bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if p, found := e.blockTags[name]; found {
		return p, true, true
	}
	if p, found := e.inlineTags[name]; found {
		return p, false, true
	}
	return nil, false, false
}

func (e *Environment) currentLoader() Loader {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.loader
}
