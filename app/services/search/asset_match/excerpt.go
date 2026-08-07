package asset_match

import (
	"regexp"
	"strings"
	"unicode"
)

var collapseWhitespace = regexp.MustCompile(`\s+`)

// snapMargin bounds how far window expansion will look for a word boundary
// before giving up and cutting mid-word.
const snapMargin = 20

// Excerpt returns a whitespace-collapsed window of roughly radius runes on
// either side of the first case-insensitive occurrence of term in text,
// expanded to nearby word boundaries and marked with "…" where truncated.
// If term isn't found, it falls back to a leading window so callers always
// get a non-empty preview rather than nothing.
func Excerpt(text, term string, radius int) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}

	runes := []rune(text)

	idx := indexFold(runes, term)
	if idx < 0 {
		return collapseAndTrim(takeWindow(runes, 0, min(len(runes), radius*2)))
	}

	termLen := len([]rune(term))
	start := snapToBoundary(runes, idx-radius, -1)
	end := snapToBoundary(runes, idx+termLen+radius, 1)

	excerpt := collapseAndTrim(takeWindow(runes, start, end))

	if start > 0 {
		excerpt = "…" + excerpt
	}
	if end < len(runes) {
		excerpt += "…"
	}

	return excerpt
}

func takeWindow(runes []rune, start, end int) string {
	if start < 0 {
		start = 0
	}
	if end > len(runes) {
		end = len(runes)
	}
	if start >= end {
		return ""
	}
	return string(runes[start:end])
}

func collapseAndTrim(s string) string {
	return strings.TrimSpace(collapseWhitespace.ReplaceAllString(s, " "))
}

// snapToBoundary nudges pos towards the nearest whitespace rune, searching
// in the given direction (-1 towards the start, +1 towards the end) up to
// snapMargin runes, so excerpts don't cut words in half when a boundary is
// nearby. Falls back to the raw position if none is found within range.
func snapToBoundary(runes []rune, pos, direction int) int {
	if pos < 0 {
		return 0
	}
	if pos > len(runes) {
		return len(runes)
	}

	for i := 0; i < snapMargin; i++ {
		p := pos + i*direction
		if p < 0 || p >= len(runes) {
			break
		}
		if unicode.IsSpace(runes[p]) {
			if direction < 0 {
				return p + 1
			}
			return p
		}
	}

	return pos
}

// indexFold returns the rune index of the first case-insensitive occurrence
// of term within runes, or -1. Operating rune-wise (rather than
// strings.Index(strings.ToLower(...))) avoids byte-offset drift for runes
// whose lowercase form has a different UTF-8 length.
func indexFold(runes []rune, term string) int {
	termRunes := []rune(strings.ToLower(term))
	n := len(termRunes)
	if n == 0 {
		return -1
	}

	for i := 0; i+n <= len(runes); i++ {
		match := true
		for j := 0; j < n; j++ {
			if unicode.ToLower(runes[i+j]) != termRunes[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}

	return -1
}
