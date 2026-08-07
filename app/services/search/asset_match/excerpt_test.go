package asset_match

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExcerpt_MatchAtStart(t *testing.T) {
	got := Excerpt("Spanisch fuer Mediziner ist ein Wahlfach im vierten Semester des Studiums.", "Spanisch", 20)
	assert.Contains(t, got, "Spanisch")
	assert.False(t, strings.HasPrefix(got, "…"), "no leading ellipsis when the match is at the very start")
}

func TestExcerpt_MatchInMiddle(t *testing.T) {
	text := "Im Lehrplan des vierten Semesters findet sich das Fach Spanisch fuer Mediziner als Wahlpflichtfach eingetragen."
	got := Excerpt(text, "Spanisch", 15)
	assert.Contains(t, got, "Spanisch")
	assert.True(t, strings.HasPrefix(got, "…"))
	assert.True(t, strings.HasSuffix(got, "…"))
}

func TestExcerpt_MatchAtEnd(t *testing.T) {
	text := "Das Wahlpflichtfach im vierten Semester ist Spanisch"
	got := Excerpt(text, "Spanisch", 10)
	assert.Contains(t, got, "Spanisch")
	assert.False(t, strings.HasSuffix(got, "…"), "no trailing ellipsis when the match reaches the end")
}

func TestExcerpt_CaseInsensitive(t *testing.T) {
	got := Excerpt("Das Fach heisst SPANISCH und ist verpflichtend.", "spanisch", 20)
	assert.Contains(t, got, "SPANISCH")
}

func TestExcerpt_NoMatchFallsBackToLeadingWindow(t *testing.T) {
	got := Excerpt("Dies ist ein Text ohne den gesuchten Begriff irgendwo enthalten.", "Zahnmedizin", 10)
	assert.NotEmpty(t, got)
	assert.Contains(t, got, "Dies ist")
}

func TestExcerpt_EmptyText(t *testing.T) {
	assert.Empty(t, Excerpt("", "term", 10))
	assert.Empty(t, Excerpt("   ", "term", 10))
}

func TestExcerpt_WhitespaceCollapsed(t *testing.T) {
	got := Excerpt("Spanisch   fuer\n\nMediziner\tist Pflicht", "Spanisch", 30)
	assert.NotContains(t, got, "  ")
	assert.NotContains(t, got, "\n")
	assert.NotContains(t, got, "\t")
}

func TestExcerpt_UmlautCaseInsensitive(t *testing.T) {
	got := Excerpt("Der Kurs für Zahnmedizin beginnt im Wintersemester.", "ZAHNMEDIZIN", 15)
	assert.Contains(t, got, "Zahnmedizin")
}
