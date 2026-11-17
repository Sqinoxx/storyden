package semester_suggest

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Southclaws/storyden/app/services/account/semester"
)

const (
	termsBack    = 16
	termsForward = 1
)

const (
	winterKeywords = `wintersemester|winter[\s_-]semester|wintersem|wisem|wi[._-]?se|w\.s\.|ws`
	summerKeywords = `sommersemester|sommer[\s_-]semester|sommersem|sosem|so[._-]?se|s\.s\.|ss`
	yearDigits     = `((?:19|20)?\d{2})`
	keywordSep     = `[\s._'’´` + "`" + `-]*`
	yearSep        = `([\s./_-]*)`
)

// Short keywords like "ss" must not be glued to a preceding letter, otherwise
// "Boss 22" or "news24" would be terms. Glued forms are split up front only
// where they are unambiguous: CamelCase ("KlausurWS24") and long keywords.
var (
	winterPattern        = regexp.MustCompile(`(?i)(?:^|[^\p{L}])(((?:` + winterKeywords + `)` + keywordSep + yearDigits + `)(?:` + yearSep + yearDigits + `)?)(?:\D|$)`)
	summerPattern        = regexp.MustCompile(`(?i)(?:^|[^\p{L}])((?:` + summerKeywords + `)` + keywordSep + yearDigits + `)(?:\D|$)`)
	winterPairPattern    = regexp.MustCompile(`(?i)(?:^|[^\p{L}])(winter` + keywordSep + yearDigits + `[\s./_-]+` + yearDigits + `)(?:\D|$)`)
	winterReversePattern = regexp.MustCompile(`(?i)(?:^|[^\d./\p{L}])(` + yearDigits + `(?:` + yearSep + yearDigits + `)?[\s._-]*(?:` + winterKeywords + `))(?:[^\p{L}]|$)`)
	summerReversePattern = regexp.MustCompile(`(?i)(?:^|[^\d./\p{L}])(` + yearDigits + `[\s._-]*(?:` + summerKeywords + `))(?:[^\p{L}]|$)`)

	camelKeyword = regexp.MustCompile(`(\p{Ll})(WS|SS|WiSe|SoSe|Winter|Sommer)`)
	gluedKeyword = regexp.MustCompile(`(?i)(\p{L})(wintersem|sommersem)`)
)

type match struct {
	term     semester.Term
	evidence string
}

func normalise(text string) string {
	text = camelKeyword.ReplaceAllString(text, "${1} ${2}")
	return gluedKeyword.ReplaceAllString(text, "${1} ${2}")
}

func detect(text string, now time.Time) []match {
	text = normalise(text)

	var found []match
	seen := map[semester.Term]bool{}

	add := func(t semester.Term, evidence string) {
		if !inRange(t, now) || seen[t] {
			return
		}
		seen[t] = true
		found = append(found, match{term: t, evidence: strings.TrimSpace(evidence)})
	}

	for _, m := range winterPattern.FindAllStringSubmatch(text, -1) {
		if t, pair, ok := winterTerm(m[3], m[4], m[5]); ok {
			if pair {
				add(t, m[1])
			} else {
				add(t, m[2])
			}
		}
	}

	for _, m := range winterPairPattern.FindAllStringSubmatch(text, -1) {
		if t, pair, ok := winterTerm(m[2], "/", m[3]); ok && pair {
			add(t, m[1])
		}
	}

	for _, m := range winterReversePattern.FindAllStringSubmatch(text, -1) {
		if t, pair, ok := winterTerm(m[2], "/", m[4]); ok && (pair || m[4] == "") {
			add(t, m[1])
		}
	}

	for _, m := range summerPattern.FindAllStringSubmatch(text, -1) {
		if y, ok := year(m[2]); ok {
			add(semester.Term{Year: y}, m[1])
		}
	}

	for _, m := range summerReversePattern.FindAllStringSubmatch(text, -1) {
		if y, ok := year(m[2]); ok {
			add(semester.Term{Year: y}, m[1])
		}
	}

	return found
}

// winterTerm reports whether the second number was used as the end year. A
// second number that does not follow the first is usually unrelated, such as
// a date in "WS24_14.02.2025", unless the two are explicitly joined by "/".
func winterTerm(first, sep, second string) (semester.Term, bool, bool) {
	if second != "" {
		start, ok1 := year(first)
		end, ok2 := year(second)
		if ok1 && ok2 && end == start+1 {
			return semester.Term{Year: start, Winter: true}, true, true
		}
		if strings.Contains(sep, "/") {
			return semester.Term{}, false, false
		}
	}

	// "WS2223" is the compact form of 22/23, but "WS 2021" could equally be
	// 20/21 or the year 2021, so that reading is too ambiguous to suggest.
	if len(first) == 4 {
		a, _ := strconv.Atoi(first[:2])
		b, _ := strconv.Atoi(first[2:])
		if (a+1)%100 == b {
			if a == 19 || a == 20 {
				return semester.Term{}, false, false
			}
			return semester.Term{Year: 2000 + a, Winter: true}, false, true
		}
	}

	start, ok := year(first)
	if !ok {
		return semester.Term{}, false, false
	}

	return semester.Term{Year: start, Winter: true}, false, true
}

func year(raw string) (int, bool) {
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0, false
	}

	switch len(raw) {
	case 2:
		return 2000 + n, true
	case 4:
		return n, true
	default:
		return 0, false
	}
}

func inRange(t semester.Term, now time.Time) bool {
	current := semester.TermFor(now)
	elapsed := t.Elapsed(current)
	return elapsed >= -termsForward && elapsed <= termsBack
}

func DetectTerm(text string, now time.Time) (semester.Term, bool) {
	found := detect(text, now)
	if len(found) != 1 {
		return semester.Term{}, false
	}
	return found[0].term, true
}
