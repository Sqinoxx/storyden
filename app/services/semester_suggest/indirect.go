package semester_suggest

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Southclaws/storyden/app/services/account/semester"
)

var (
	relativePattern = regexp.MustCompile(`(?i)(?:^|[^\p{L}])((letzte[nmrs]?|vergangene[nmrs]?|vorige[nmrs]?|diese[nmrs]?|aktuelle[nmrs]?|laufende[nmrs]?|kommende[nmrs]?|nächste[nmrs]?|naechste[nmrs]?)\s+(wintersemester|sommersemester|semester|wise|sose))(?:[^\p{L}]|$)`)
	datePattern     = regexp.MustCompile(`(?:^|\D)((\d{1,2})\.(\d{1,2})\.((?:19|20)?\d{2}))(?:\D|$)`)
	monthPattern    = regexp.MustCompile(`(?i)(?:^|[^\p{L}])((januar|jänner|februar|märz|maerz|april|mai|juni|juli|august|september|oktober|november|dezember)\s+((?:19|20)\d{2}))(?:\D|$)`)
	seasonPattern   = regexp.MustCompile(`(?i)(?:^|[^\p{L}])((sommer|winter)\s+((?:19|20)\d{2})(?:\s*/\s*((?:19|20)?\d{2}))?)(?:\D|$)`)
)

var months = map[string]time.Month{
	"januar": time.January, "jänner": time.January, "februar": time.February,
	"märz": time.March, "maerz": time.March, "april": time.April, "mai": time.May,
	"juni": time.June, "juli": time.July, "august": time.August,
	"september": time.September, "oktober": time.October,
	"november": time.November, "dezember": time.December,
}

// Future terms are dropped: those are plans or deadlines, never the date of
// shared exam material.
func detectIndirect(text string, now time.Time) []match {
	var found []match
	seen := map[semester.Term]bool{}
	current := semester.TermFor(now)

	add := func(t semester.Term, evidence string) {
		if !inRange(t, now) || current.Elapsed(t) > 0 || seen[t] {
			return
		}
		seen[t] = true
		found = append(found, match{term: t, evidence: strings.TrimSpace(evidence)})
	}

	for _, m := range relativePattern.FindAllStringSubmatch(text, -1) {
		if t, ok := relativeTerm(current, strings.ToLower(m[2]), strings.ToLower(m[3])); ok {
			add(t, m[1])
		}
	}

	for _, m := range datePattern.FindAllStringSubmatch(text, -1) {
		d, _ := strconv.Atoi(m[2])
		mo, _ := strconv.Atoi(m[3])
		y, ok := year(m[4])
		if !ok || mo < 1 || mo > 12 || d < 1 || d > 31 {
			continue
		}
		date := time.Date(y, time.Month(mo), d, 0, 0, 0, 0, time.UTC)
		if date.After(now) {
			continue
		}
		add(semester.TermFor(date), m[1])
	}

	for _, m := range monthPattern.FindAllStringSubmatch(text, -1) {
		y, _ := strconv.Atoi(m[3])
		date := time.Date(y, months[strings.ToLower(m[2])], 1, 0, 0, 0, 0, time.UTC)
		if date.After(now) {
			continue
		}
		add(semester.TermFor(date), m[1])
	}

	for _, m := range seasonPattern.FindAllStringSubmatch(text, -1) {
		y, _ := strconv.Atoi(m[3])
		if strings.ToLower(m[2]) == "sommer" {
			if m[4] == "" {
				add(semester.Term{Year: y}, m[1])
			}
			continue
		}
		// "Winter 2023" alone could mean 2022/23 or 2023/24.
		if m[4] == "" {
			continue
		}
		if end, ok := year(m[4]); ok && end == y+1 {
			add(semester.Term{Year: y, Winter: true}, m[1])
		}
	}

	return found
}

func relativeTerm(current semester.Term, when, season string) (semester.Term, bool) {
	var step int
	switch {
	case strings.HasPrefix(when, "letzte"), strings.HasPrefix(when, "vergangene"), strings.HasPrefix(when, "vorige"):
		step = -1
	case strings.HasPrefix(when, "kommende"), strings.HasPrefix(when, "nächste"), strings.HasPrefix(when, "naechste"):
		step = 1
	default:
		step = 0
	}

	var winter *bool
	switch season {
	case "wintersemester", "wise":
		w := true
		winter = &w
	case "sommersemester", "sose":
		w := false
		winter = &w
	}

	if step == 0 {
		if winter != nil && *winter != current.Winter {
			return semester.Term{}, false
		}
		return current, true
	}

	t := shift(current, step)
	if winter != nil && t.Winter != *winter {
		t = shift(t, step)
	}
	return t, true
}

func shift(t semester.Term, by int) semester.Term {
	ordinal := t.Year*2 + boolInt(t.Winter) + by
	return semester.Term{Year: ordinal / 2, Winter: ordinal%2 == 1}
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
