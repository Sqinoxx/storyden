package semdex_indexer

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestThreadTextIncludesRepliesTagsAndAttachments(t *testing.T) {
	got := threadText(
		"Hirnnerven Eselsbrücke",
		"Kennt jemand eine?",
		[]string{"anatomie", "physikum"},
		[]string{"Onkel Otto orgelt", "", "Danke!"},
		[]attachment{
			{name: "abc-altklausur.pdf", summary: "<ul><li>Fragen zum N. trigeminus</li></ul>", text: "roh"},
			{name: "abc-skript.pdf", text: "Skript Neuroanatomie"},
		},
	)

	assert.True(t, strings.HasPrefix(got, "Hirnnerven Eselsbrücke\n\nKennt jemand eine?"))
	assert.Contains(t, got, "Tags: anatomie, physikum")
	assert.Contains(t, got, "Antworten:\n- Onkel Otto orgelt\n- Danke!")
	assert.Contains(t, got, "Fragen zum N. trigeminus")
	assert.NotContains(t, got, "<li>")
	assert.NotContains(t, got, "roh")
	assert.Contains(t, got, "Skript Neuroanatomie")
}

func TestThreadTextIsBounded(t *testing.T) {
	long := strings.Repeat("Wurzelkanal ", 2000)
	replies := make([]string, 100)
	for i := range replies {
		replies[i] = long
	}

	got := threadText("Titel", long, nil, replies, []attachment{{name: "a.pdf", text: long}, {name: "b.pdf", text: long}, {name: "c.pdf", text: long}})

	assert.Less(t, len([]rune(got)), 6500)
	assert.True(t, strings.HasPrefix(got, "Titel\n\n"))
}

func TestThreadTextPlain(t *testing.T) {
	assert.Equal(t, "Titel\n\nText", threadText("Titel", "Text", nil, nil, nil))
}
