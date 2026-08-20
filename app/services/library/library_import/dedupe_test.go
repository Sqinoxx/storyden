package library_import

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDedupePrefersTheNonDemotedBranch(t *testing.T) {
	t.Parallel()

	demote := []string{"Zusammenfassungen, Vorlesungen, Anki-Decks/Vorlesungen & Unterlagen"}

	entries := []Entry{
		{Path: "Zusammenfassungen, Vorlesungen, Anki-Decks/Vorlesungen & Unterlagen/Klinik/Kons/Kons STEX.pdf", Size: 299, SHA256: "a"},
		{Path: "Vorlesungen&Unterlagen/Klinik/Kons/Sonstiges/Zusammenfassungen/Kons STEX.pdf", Size: 299, SHA256: "a"},
		{Path: "Vorlesungen&Unterlagen/STEX Zusammenfassungen/Kons/Kons STEX.pdf", Size: 299, SHA256: "a"},
	}

	got := Dedupe(entries, demote)

	require.Len(t, got.Canonical, 1)
	assert.Equal(t, "Vorlesungen&Unterlagen/STEX Zusammenfassungen/Kons/Kons STEX.pdf", got.Canonical[0].Path)
	assert.EqualValues(t, 598, got.SavedBytes)

	require.Len(t, got.Groups, 1)
	assert.Len(t, got.Groups[0].Duplicates, 2)
}

func TestDedupeKeepsDistinctContent(t *testing.T) {
	t.Parallel()

	entries := []Entry{
		{Path: "a/x.pdf", Size: 10, SHA256: "a"},
		{Path: "b/y.pdf", Size: 20, SHA256: "b"},
	}

	got := Dedupe(entries, nil)

	assert.Len(t, got.Canonical, 2)
	assert.Empty(t, got.Groups)
	assert.Zero(t, got.SavedBytes)
}

// Same name and size but different bytes must stay two separate documents.
func TestDedupeDoesNotCollapseOnNameAlone(t *testing.T) {
	t.Parallel()

	entries := []Entry{
		{Path: "Altklausuren/Kons/klausur.pdf", Size: 100, SHA256: "a"},
		{Path: "Altklausuren/MKG/klausur.pdf", Size: 100, SHA256: "b"},
	}

	assert.Len(t, Dedupe(entries, nil).Canonical, 2)
}

func TestDedupeIsDeterministic(t *testing.T) {
	t.Parallel()

	entries := []Entry{
		{Path: "b/deep/nested/file.pdf", Size: 1, SHA256: "a"},
		{Path: "a/file.pdf", Size: 1, SHA256: "a"},
	}

	first := Dedupe(entries, nil)
	second := Dedupe([]Entry{entries[1], entries[0]}, nil)

	assert.Equal(t, first.Canonical[0].Path, second.Canonical[0].Path)
	assert.Equal(t, "a/file.pdf", first.Canonical[0].Path)
}
