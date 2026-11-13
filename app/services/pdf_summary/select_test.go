package pdf_summary

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var topics = []string{"Endodontie", "Parodontologie", "Prothetik", "Kieferorthopädie", "Kinderzahnheilkunde", "Chirurgie", "Radiologie", "Werkstoffkunde"}

func sampleExam() string {
	var b strings.Builder
	for i, topic := range topics {
		fmt.Fprintf(&b, "Universität Musterstadt – Zahnklinik\n")
		fmt.Fprintf(&b, "Aufgabe %d: %s\n", i+1, topic)
		fmt.Fprintf(&b, "Beschreiben Sie im Bereich %s die Aufbereitung des Wurzelkanals an Zahn %d mit rotierenden Instrumenten und nennen Sie typische Komplikationen.\n", topic, 30+i)
		fmt.Fprintf(&b, "Fall %d: Die Patientin stellt sich mit Schmerzen an Zahn %d vor, die Sensibilitätsprobe ist negativ und im Röntgenbild zeigt sich eine apikale Aufhellung.\n", i+1, 40+i)
		fmt.Fprintf(&b, "Welche Diagnose stellen Sie bei Befund %d?\n", i+1)
		fmt.Fprintf(&b, "Weitere Hinweise zur Bearbeitung finden Sie auf Seite %d dieser Klausur, Bearbeitungszeit %d Minuten.\n", i+2, 10+i)
	}
	return b.String()
}

func TestSelectTextRespectsBudget(t *testing.T) {
	sel := selectText(sampleExam(), 800)
	assert.LessOrEqual(t, len(sel.text), 800)
	assert.True(t, sel.truncated)
	assert.NotEmpty(t, sel.text)
}

func TestSelectTextKeepsRepeatedHeadersOnce(t *testing.T) {
	sel := selectText(sampleExam(), 10000)
	assert.LessOrEqual(t, strings.Count(sel.text, "Zahnklinik"), 1)
}

func TestSelectTextPrefersHeadingsAndKeepsOrder(t *testing.T) {
	sel := selectText(sampleExam(), 600)
	assert.Contains(t, sel.text, "Aufgabe 1: Endodontie")

	lines := strings.Split(sel.text, "\n")
	full := sampleExam()
	last := -1
	for _, l := range lines {
		i := strings.Index(full, l)
		require.GreaterOrEqual(t, i, 0, l)
		assert.Greater(t, i, last-1)
		last = i
	}

	require.NotEmpty(t, sel.headings)
	assert.LessOrEqual(t, len(sel.headings), 3)
	assert.NotEmpty(t, sel.terms)
	assert.NotContains(t, sel.terms, "aufgabe")
}

func TestSelectTextShortInput(t *testing.T) {
	sel := selectText("Kurzer Text über Prothetik.", 2500)
	assert.False(t, sel.truncated)
	assert.Equal(t, "Kurzer Text über Prothetik.", sel.text)
}

func TestBuildPreview(t *testing.T) {
	text := "Altklausur Zahnerhaltung WS 22/23\n" + sampleExam()
	sel := selectText(text, quickBudget)
	html := buildPreview("altklausur-ze.pdf", text, sel, time.Date(2026, 11, 13, 0, 0, 0, 0, time.UTC))

	assert.Contains(t, html, "Altklausur / Prüfung")
	assert.Contains(t, html, "Semester: WS 22/23")
	assert.Contains(t, html, "Wörter")
	assert.Contains(t, html, "<strong>Inhalt</strong>")
	assert.Contains(t, html, "<strong>Stichworte</strong>")
}

func TestPreviewEscapesHTML(t *testing.T) {
	text := "AUFGABE <script>alert(1)</script>\n" + strings.Repeat("Füllungstherapie mit Komposit. ", 20)
	html := buildPreview("x.pdf", text, selectText(text, quickBudget), time.Now())
	assert.NotContains(t, html, "<script>")
}

func TestHashChangesWithText(t *testing.T) {
	assert.Equal(t, hashOf("a"), hashOf("a"))
	assert.NotEqual(t, hashOf("a"), hashOf("b"))
}

func TestBroadcastLateSubscriber(t *testing.T) {
	b := newBroadcast()
	b.push("Hallo ")
	b.push("Welt")

	got := make(chan string, 1)
	go func() {
		var out strings.Builder
		for ev := range b.subscribe(context.Background()) {
			if ev.Done {
				out.WriteString("|" + ev.HTML)
				break
			}
			out.WriteString(ev.Text)
		}
		got <- out.String()
	}()

	b.push("!")
	b.finish("<p>Hallo Welt!</p>", nil)
	assert.Equal(t, "Hallo Welt!|<p>Hallo Welt!</p>", <-got)
}
