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

// The season keyword must not be glued to a preceding letter (so "Boss 22" is
// not a summer term) but may follow an underscore or dash as in file names.
var (
	winterPattern = regexp.MustCompile(`(?i)(?:^|[^\p{L}])((?:wintersemester|wise|ws)[\s._-]*((?:19|20)?\d{2})(?:[\s./_-]*((?:19|20)?\d{2}))?)(?:\D|$)`)
	summerPattern = regexp.MustCompile(`(?i)(?:^|[^\p{L}])((?:sommersemester|sose|ss)[\s._-]*((?:19|20)?\d{2}))(?:\D|$)`)
)

type match struct {
	term     semester.Term
	evidence string
}

func detect(text string, now time.Time) []match {
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
		if t, ok := winterTerm(m[2], m[3]); ok {
			add(t, m[1])
		}
	}

	for _, m := range summerPattern.FindAllStringSubmatch(text, -1) {
		if y, ok := year(m[2]); ok {
			add(semester.Term{Year: y}, m[1])
		}
	}

	return found
}

func winterTerm(first, second string) (semester.Term, bool) {
	// "WS2223" is the compact form of 22/23, but "WS 2021" could equally be
	// 20/21 or the year 2021, so that reading is too ambiguous to suggest.
	if second == "" && len(first) == 4 {
		a, _ := strconv.Atoi(first[:2])
		b, _ := strconv.Atoi(first[2:])
		if (a+1)%100 == b {
			if a == 19 || a == 20 {
				return semester.Term{}, false
			}
			return semester.Term{Year: 2000 + a, Winter: true}, true
		}
	}

	start, ok := year(first)
	if !ok {
		return semester.Term{}, false
	}

	if second != "" {
		end, ok := year(second)
		if !ok || end != start+1 {
			return semester.Term{}, false
		}
	}

	return semester.Term{Year: start, Winter: true}, true
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
