package summary_text

import "strings"

var (
	germanWords  = []string{"der", "die", "das", "und", "ist", "nicht", "mit", "für", "auf", "eine", "einen", "wird", "werden", "sind", "bei", "zur", "zum", "oder", "auch", "sich"}
	englishWords = []string{"the", "and", "is", "are", "of", "to", "with", "for", "this", "that", "which", "from", "it", "on", "as", "be", "by", "was", "were", "an"}
)

const minLanguageHits = 4

// Language guesses "de" or "en" from common function words and returns an
// empty string when the text is too short or mixed to tell.
func Language(text string) string {
	de, en := 0, 0
	for _, w := range strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !(r >= 'a' && r <= 'z') && r != 'ä' && r != 'ö' && r != 'ü' && r != 'ß'
	}) {
		for _, g := range germanWords {
			if w == g {
				de++
			}
		}
		for _, e := range englishWords {
			if w == e {
				en++
			}
		}
	}

	switch {
	case de >= minLanguageHits && de >= 2*en:
		return "de"
	case en >= minLanguageHits && en >= 2*de:
		return "en"
	default:
		return ""
	}
}

// Valid reports whether a sanitised summary has the requested list structure
// and was not written in another language than its source.
func Valid(html, source string) bool {
	if !strings.Contains(html, "<li>") {
		return false
	}

	want, got := Language(source), Language(StripTags(html))
	return want == "" || got == "" || want == got
}
