package pdf_summary

import (
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/Southclaws/storyden/app/services/account/semester"
	"github.com/Southclaws/storyden/app/services/semester_suggest"
)

var docTypes = []struct {
	keys  []string
	label string
}{
	{[]string{"gedächtnisprotokoll", "protokoll"}, "Protokoll"},
	{[]string{"altklausur", "klausur", "prüfung", "testat", "exam"}, "Altklausur / Prüfung"},
	{[]string{"zusammenfassung", "summary", "lernzettel"}, "Zusammenfassung"},
	{[]string{"skript", "vorlesung", "folien", "lecture", "script"}, "Skript / Vorlesung"},
}

func docType(name, text string) string {
	hay := strings.ToLower(name + "\n" + firstRunes(text, 3000))
	for _, d := range docTypes {
		for _, k := range d.keys {
			if strings.Contains(hay, k) {
				return d.label
			}
		}
	}
	return ""
}

func firstRunes(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n])
	}
	return s
}

func buildPreview(name, text string, sel selection, now time.Time) string {
	var facts []string
	if t := docType(name, text); t != "" {
		facts = append(facts, "Art: "+t)
	}
	if term, ok := semester_suggest.DetectTerm(name+"\n"+firstRunes(text, 3000), now); ok {
		facts = append(facts, "Semester: "+termLabel(term))
	}
	if sel.words > 0 {
		facts = append(facts, fmt.Sprintf("Umfang: ca. %d Wörter", roundWords(sel.words)))
	}

	var b strings.Builder
	section := func(title string, items []string) {
		if len(items) == 0 {
			return
		}
		fmt.Fprintf(&b, "<p><strong>%s</strong></p><ul>", title)
		for _, it := range items {
			fmt.Fprintf(&b, "<li>%s</li>", html.EscapeString(it))
		}
		b.WriteString("</ul>")
	}

	section("Dokument", facts)
	section("Inhalt", sel.headings)
	if len(sel.terms) > 0 {
		fmt.Fprintf(&b, "<p><strong>Stichworte</strong></p><p>%s</p>", html.EscapeString(strings.Join(sel.terms, ", ")))
	}
	return b.String()
}

func termLabel(t semester.Term) string {
	if t.Winter {
		return fmt.Sprintf("WS %02d/%02d", t.Year%100, (t.Year+1)%100)
	}
	return fmt.Sprintf("SoSe %02d", t.Year%100)
}

func roundWords(n int) int {
	switch {
	case n < 100:
		return n
	case n < 1000:
		return (n + 50) / 100 * 100
	default:
		return (n + 500) / 1000 * 1000
	}
}
