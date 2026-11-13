package pdf_summary

import (
	"math"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

const maxUnitChars = 400

type unit struct {
	text    string
	pos     int
	heading bool
	task    bool
	score   float64
}

type selection struct {
	text      string
	headings  []string
	terms     []string
	truncated bool
	words     int
}

var (
	numbered  = regexp.MustCompile(`^(\d+(\.\d+)*[.)]?|[IVX]+\.|[a-h]\))\s+\S`)
	taskStart = regexp.MustCompile(`(?i)^(aufgabe|frage|fall|kapitel|teil|abschnitt|thema|question|task|chapter)\b`)
	wordRe    = regexp.MustCompile(`[\p{L}][\p{L}\p{N}-]*[\p{L}\p{N}]`)
)

var stopwords = toSet(`aber alle allem allen aller alles also andere anderen auch auf aus bei beim bereits bitte bis dann darf darum dass davon dazu dem den denen der deren des dessen die dies diese diesem diesen dieser dieses doch dort durch eine einem einen einer eines einige etwa euch für gegen hast hat hatte hier ihre ihren ihrer immer jede jeden jeder jedes jetzt kann kein keine können man mehr mein meine mit muss müssen nach nicht noch nur oder ohne seid sein seine sich sie sind soll sollen sowie über unter viel vom von vor wann war waren warum was weil welche welcher wenn wer werden wie wieder wird wir wo zum zur zwischen sehr kann geben gibt beim bzw etc ggf usw siehe seite seiten punkte punkt bitte name datum note aufgabe aufgaben frage fragen kapitel teil abschnitt thema
about after also been before being between both could does each from have here into more most much only other over same should some such than that their them then there these they this those through under very what when where which while with would your page`)

func toSet(s string) map[string]bool {
	m := map[string]bool{}
	for _, w := range strings.Fields(s) {
		m[w] = true
	}
	return m
}

func splitUnits(text string) []string {
	var out []string
	for _, line := range strings.Split(text, "\n") {
		line = strings.Join(strings.Fields(line), " ")
		if len([]rune(line)) < 4 {
			continue
		}
		if len(line) <= maxUnitChars {
			out = append(out, line)
			continue
		}
		out = append(out, splitSentences(line)...)
	}
	return out
}

func splitSentences(s string) []string {
	var out []string
	start := 0
	runes := []rune(s)
	for i := 0; i < len(runes)-1; i++ {
		if (runes[i] == '.' || runes[i] == '!' || runes[i] == '?') && runes[i+1] == ' ' && i-start > 40 {
			out = append(out, strings.TrimSpace(string(runes[start:i+1])))
			start = i + 2
		}
	}
	if start < len(runes) {
		out = append(out, strings.TrimSpace(string(runes[start:])))
	}
	return out
}

func isHeading(s string) bool {
	r := []rune(s)
	if len(r) > 80 || strings.HasSuffix(s, ".") || strings.HasSuffix(s, ",") {
		return false
	}
	if numbered.MatchString(s) || taskStart.MatchString(s) || strings.HasSuffix(s, ":") {
		return true
	}
	letters, upper := 0, 0
	for _, c := range r {
		if unicode.IsLetter(c) {
			letters++
			if unicode.IsUpper(c) {
				upper++
			}
		}
	}
	return letters >= 4 && float64(upper)/float64(letters) >= 0.6
}

func words(s string) []string {
	var out []string
	for _, w := range wordRe.FindAllString(strings.ToLower(s), -1) {
		if len([]rune(w)) < 4 || stopwords[w] {
			continue
		}
		out = append(out, w)
	}
	return out
}

func selectText(text string, budget int) selection {
	raw := splitUnits(text)

	seen := map[string]bool{}
	var units []*unit
	for i, u := range raw {
		key := strings.ToLower(u)
		if seen[key] {
			continue
		}
		seen[key] = true
		units = append(units, &unit{
			text:    u,
			pos:     i,
			heading: isHeading(u),
			task:    taskStart.MatchString(u) || strings.HasSuffix(u, "?"),
		})
	}

	tf := map[string]int{}
	df := map[string]int{}
	total := len(strings.Fields(text))
	for _, u := range units {
		ws := words(u.text)
		uniq := map[string]bool{}
		for _, w := range ws {
			tf[w]++
			uniq[w] = true
		}
		for w := range uniq {
			df[w]++
		}
	}

	n := float64(len(units)) + 1
	weight := func(w string) float64 {
		return math.Log1p(float64(tf[w])) * math.Log(n/float64(df[w]))
	}

	for _, u := range units {
		ws := words(u.text)
		uniq := map[string]bool{}
		sum := 0.0
		for _, w := range ws {
			if !uniq[w] {
				uniq[w] = true
				sum += weight(w)
			}
		}
		score := sum / math.Sqrt(float64(len(ws))+1)
		if u.heading {
			score += 2
		}
		if u.task {
			score += 1.5
		}
		if u.pos < 3 {
			score += 1
		}
		u.score = score
	}

	ranked := append([]*unit{}, units...)
	sort.SliceStable(ranked, func(i, j int) bool { return ranked[i].score > ranked[j].score })

	used := 0
	picked := map[int]bool{}
	for _, u := range ranked {
		l := len(u.text) + 1
		if used+l > budget {
			continue
		}
		used += l
		picked[u.pos] = true
	}

	var b strings.Builder
	var headings []string
	for _, u := range units {
		if picked[u.pos] {
			b.WriteString(u.text)
			b.WriteByte('\n')
		}
		if u.heading && len(headings) < 3 && len([]rune(u.text)) >= 6 {
			headings = append(headings, strings.TrimSuffix(u.text, ":"))
		}
	}

	terms := make([]string, 0, len(tf))
	for w := range tf {
		if tf[w] >= 2 {
			terms = append(terms, w)
		}
	}
	sort.SliceStable(terms, func(i, j int) bool {
		wi, wj := weight(terms[i])*float64(tf[terms[i]]), weight(terms[j])*float64(tf[terms[j]])
		if wi == wj {
			return terms[i] < terms[j]
		}
		return wi > wj
	})
	if len(terms) > 6 {
		terms = terms[:6]
	}

	return selection{
		text:      strings.TrimSpace(b.String()),
		headings:  headings,
		terms:     terms,
		truncated: len(strings.TrimSpace(text)) > budget,
		words:     total,
	}
}
