package library_import

import (
	"strings"
)

type Pattern struct {
	segments []string
}

func NewPattern(raw string) Pattern {
	return Pattern{segments: splitPath(raw)}
}

func splitPath(raw string) []string {
	raw = strings.ReplaceAll(raw, "\\", "/")
	parts := strings.Split(raw, "/")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// Match walks pattern segments against path segments. A `{name}` segment
// captures exactly one path segment, `*` consumes one without capturing and
// `**` consumes any number, capturing the remainder as `path` when it is the
// final segment.
func (p Pattern) Match(path string) (map[string]string, bool) {
	captures := map[string]string{}
	if !matchSegments(p.segments, splitPath(path), captures) {
		return nil, false
	}
	return captures, true
}

func matchSegments(pattern, path []string, captures map[string]string) bool {
	if len(pattern) == 0 {
		return len(path) == 0
	}

	head := pattern[0]

	if head == "**" {
		if len(pattern) == 1 {
			captures["path"] = strings.Join(path, "/")
			return true
		}
		for i := 0; i <= len(path); i++ {
			nested := map[string]string{}
			if matchSegments(pattern[1:], path[i:], nested) {
				for k, v := range nested {
					captures[k] = v
				}
				return true
			}
		}
		return false
	}

	if len(path) == 0 {
		return false
	}

	switch {
	case head == "*":
	case strings.HasPrefix(head, "{") && strings.HasSuffix(head, "}"):
		captures[strings.Trim(head, "{}")] = path[0]
	default:
		if !segmentEqual(head, path[0]) {
			return false
		}
	}

	return matchSegments(pattern[1:], path[1:], captures)
}

// segmentEqual compares literal segments loosely: the source tree mixes case
// and spacing inconsistently ("Klausuren&Prüfungen" vs "Klausuren & Prüfungen")
// and a rule should not break over that.
func segmentEqual(a, b string) bool {
	return foldSegment(a) == foldSegment(b)
}

func foldSegment(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		switch r {
		case ' ', '\t', '_', '-':
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// Expand substitutes {key} placeholders in a template from captures.
func Expand(template string, captures map[string]string) string {
	out := template
	for k, v := range captures {
		out = strings.ReplaceAll(out, "{"+k+"}", v)
	}
	return out
}
