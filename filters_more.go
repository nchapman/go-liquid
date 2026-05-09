package liquid

import (
	"encoding/base64"
	"net/url"
	"regexp"
	"strings"
)

// String/encoding filters

// filterEscapeOnce HTML-escapes input but leaves existing entity references
// (named or numeric) intact. Mirrors Shopify's HTML_ESCAPE_ONCE_REGEXP.
func filterEscapeOnce(input any, _ ...any) any {
	s := toString(input)
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '<':
			b.WriteString("&lt;")
		case '>':
			b.WriteString("&gt;")
		case '"':
			b.WriteString("&quot;")
		case '\'':
			b.WriteString("&#39;")
		case '&':
			if isExistingEntity(s[i+1:]) {
				b.WriteByte('&')
			} else {
				b.WriteString("&amp;")
			}
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

// isExistingEntity reports whether s starts with an HTML entity body
// followed by a semicolon: either letters or `#` + digits.
func isExistingEntity(s string) bool {
	if len(s) < 2 {
		return false
	}
	i := 0
	if s[0] == '#' {
		i = 1
		for i < len(s) && s[i] >= '0' && s[i] <= '9' {
			i++
		}
		if i == 1 {
			return false
		}
	} else {
		for i < len(s) && ((s[i] >= 'a' && s[i] <= 'z') || (s[i] >= 'A' && s[i] <= 'Z')) {
			i++
		}
		if i == 0 {
			return false
		}
	}
	return i < len(s) && s[i] == ';'
}

// filterURLEncode percent-encodes input using form encoding (spaces become
// '+'), matching Shopify's CGI.escape behavior.
func filterURLEncode(input any, _ ...any) any {
	if input == nil {
		return nil
	}
	return url.QueryEscape(toString(input))
}

// filterURLDecode reverses filterURLEncode. Invalid input returns the
// original string unchanged.
func filterURLDecode(input any, _ ...any) any {
	if input == nil {
		return nil
	}
	s := toString(input)
	out, err := url.QueryUnescape(s)
	if err != nil {
		return s
	}
	return out
}

var (
	stripScriptRE  = regexp.MustCompile(`(?is)<script.*?</script>`)
	stripStyleRE   = regexp.MustCompile(`(?is)<style.*?</style>`)
	stripCommentRE = regexp.MustCompile(`(?s)<!--.*?-->`)
	stripTagRE     = regexp.MustCompile(`(?s)<.*?>`)
	whitespaceRE   = regexp.MustCompile(`\s+`)
)

// filterStripHTML removes <script>/<style>/comment blocks then any tag.
func filterStripHTML(input any, _ ...any) any {
	s := toString(input)
	s = stripScriptRE.ReplaceAllString(s, "")
	s = stripStyleRE.ReplaceAllString(s, "")
	s = stripCommentRE.ReplaceAllString(s, "")
	s = stripTagRE.ReplaceAllString(s, "")
	return s
}

// filterStripNewlines removes \r and \n from the input.
func filterStripNewlines(input any, _ ...any) any {
	s := toString(input)
	return strings.NewReplacer("\r\n", "", "\n", "", "\r", "").Replace(s)
}

// filterSquish trims and collapses runs of whitespace into a single space.
func filterSquish(input any, _ ...any) any {
	if input == nil {
		return nil
	}
	s := strings.TrimSpace(toString(input))
	return whitespaceRE.ReplaceAllString(s, " ")
}

// Base64 filters

func filterBase64Encode(input any, _ ...any) any {
	return base64.StdEncoding.EncodeToString([]byte(toString(input)))
}

func filterBase64Decode(input any, _ ...any) any {
	out, err := base64.StdEncoding.DecodeString(toString(input))
	if err != nil {
		return filterErrorf("base64_decode: invalid base64 input")
	}
	return string(out)
}

func filterBase64URLSafeEncode(input any, _ ...any) any {
	return base64.URLEncoding.EncodeToString([]byte(toString(input)))
}

func filterBase64URLSafeDecode(input any, _ ...any) any {
	out, err := base64.URLEncoding.DecodeString(toString(input))
	if err != nil {
		return filterErrorf("base64_url_safe_decode: invalid base64 input")
	}
	return string(out)
}

// filterReplaceLast replaces the last occurrence of args[0] with args[1].
func filterReplaceLast(input any, args ...any) any {
	s := toString(input)
	if len(args) < 2 {
		return s
	}
	old := toString(args[0])
	if old == "" {
		return s
	}
	idx := strings.LastIndex(s, old)
	if idx < 0 {
		return s
	}
	return s[:idx] + toString(args[1]) + s[idx+len(old):]
}

// filterRemoveLast removes the last occurrence of args[0].
func filterRemoveLast(input any, args ...any) any {
	if len(args) == 0 {
		return toString(input)
	}
	return filterReplaceLast(input, args[0], "")
}

// Array filters

// filterReject keeps items whose property does NOT match the target value.
// Mirror of filterWhere.
func filterReject(input any, args ...any) any {
	slice := toSlice(input)
	if slice == nil || len(args) == 0 {
		return nil
	}
	prop := toString(args[0])
	var target any = true
	if len(args) > 1 {
		target = args[1]
	}
	var out []any
	for _, item := range slice {
		if !equalValues(getProperty(item, prop), target) {
			out = append(out, item)
		}
	}
	return out
}

// filterHas reports whether any item has a matching property value.
func filterHas(input any, args ...any) any {
	slice := toSlice(input)
	if slice == nil || len(args) == 0 {
		return false
	}
	prop := toString(args[0])
	var target any = true
	if len(args) > 1 {
		target = args[1]
	}
	for _, item := range slice {
		if equalValues(getProperty(item, prop), target) {
			return true
		}
	}
	return false
}

// filterFindIndex returns the 0-based index of the first matching item, or
// nil if none match (matching Shopify's behavior so `default:` works).
func filterFindIndex(input any, args ...any) any {
	slice := toSlice(input)
	if slice == nil || len(args) == 0 {
		return nil
	}
	prop := toString(args[0])
	var target any = true
	if len(args) > 1 {
		target = args[1]
	}
	for i, item := range slice {
		if equalValues(getProperty(item, prop), target) {
			return i
		}
	}
	return nil
}
