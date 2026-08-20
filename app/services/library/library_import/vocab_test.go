package library_import

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func realVocabulary(t *testing.T) *Vocabulary {
	t.Helper()

	data, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "import-vocab.yaml"))
	require.NoError(t, err)

	v, err := ParseVocabulary(data)
	require.NoError(t, err)

	return v
}

func TestVocabularyResolvesRealFolderNames(t *testing.T) {
	t.Parallel()

	v := realVocabulary(t)

	tests := []struct {
		folder string
		kind   TermKind
		wantTa string
		exact  bool
	}{
		{"MKG", KindSubject, "mkg", true},
		{"MKG (Radio, AnEx, Auscultando, Spritzenkurs)", KindSubject, "mkg", true},
		{"MKG - zahnärztlich-chirurgische Propädeutik und Notfallmedizin", KindSubject, "mkg", true},
		{"Mikrobiologie (5. Semester)", KindSubject, "mikrobiologie", false},
		{"Klinische Werkstoffkunde (5.&6.)", KindSubject, "werkstoffkunde", true},
		{"Pathologie für Zahnmediziner (5. Semester)", KindSubject, "pathologie", true},
		{"PRO Zahnersatzkunde am Phantom (5. Semester)", KindSubject, "prothetik", true},
		{"Radiologie für Zahnmediziner (5. Semester)", KindSubject, "radiologie", true},
		{"Derma", KindSubject, "dermatologie", true},
		{"Innere Medizin (10. Semester)", KindSubject, "innere-medizin", false},
		{"HistoPatho", KindSubject, "histopathologie", true},
		{"Vorklinik (1.-4. Semester)", KindSection, "vorklinik", true},
		{"Zwischenklinik (5.-6. Semester)", KindSection, "z2", true},
		{"Klinik (7.-10. Semester)", KindSection, "klinik", true},
		{"Altklausuren", KindType, "altklausur", true},
		{"Klausuren&Prüfungen", KindType, "altklausur", true},
		{"Vorlesungen & Unterlagen", KindType, "vorlesung", true},
		{"Ankis", KindType, "anki", true},
	}

	for _, tt := range tests {
		t.Run(tt.folder, func(t *testing.T) {
			t.Parallel()

			got, ok := v.Lookup(tt.kind, tt.folder)
			require.True(t, ok, "no vocabulary match for %q", tt.folder)
			assert.Equal(t, tt.wantTa, got.Term.Tag)
			assert.Equal(t, tt.exact, got.Exact)
		})
	}
}

// Neuroanatomie contains "anatomie", so the longest-alias rule is what keeps
// the two subjects apart.
func TestVocabularyPrefersLongestAlias(t *testing.T) {
	t.Parallel()

	v := realVocabulary(t)

	got, ok := v.Lookup(KindSubject, "Neuroanatomie")
	require.True(t, ok)
	assert.Equal(t, "neuroanatomie", got.Term.Tag)
}

func TestVocabularyRejectsUnknownAndShortAccidentalMatches(t *testing.T) {
	t.Parallel()

	v := realVocabulary(t)

	_, ok := v.Lookup(KindSubject, "Doktorarbeitsvorlage")
	assert.False(t, ok)

	// "AC" is only three characters, so it must not match inside unrelated names.
	got, ok := v.Lookup(KindSubject, "Praktikum")
	assert.False(t, ok, "unexpected match %q", got.Term.Tag)
}

func TestVocabularyCarriesTheTypoRenames(t *testing.T) {
	t.Parallel()

	v := realVocabulary(t)

	assert.Equal(t, "parodontologie", v.Renames["parodondologie"])
	assert.Equal(t, "pharmakologie", v.Renames["pharmalogie"])
	assert.Equal(t, "gedächtnisprotokoll", v.Renames["gedächnisprotokoll"])
}

func TestVocabularyLookupAnyFallsBackAcrossAxes(t *testing.T) {
	t.Parallel()

	v := realVocabulary(t)

	// The source tree files "Ankis" and "Videos" where a subject belongs.
	_, ok := v.Lookup(KindSubject, "Ankis")
	require.False(t, ok)

	got, ok := v.LookupAny(KindSubject, "Ankis")
	require.True(t, ok)
	assert.Equal(t, "anki", got.Term.Tag)

	got, ok = v.LookupAny(KindSubject, "STEX")
	require.True(t, ok)
	assert.Equal(t, "stex", got.Term.Tag)
}

func TestVocabularyLookupAnyPrefersTheRequestedAxis(t *testing.T) {
	t.Parallel()

	v := realVocabulary(t)

	// "Staatsexamen" exists on both the section and the type axis.
	sec, ok := v.LookupAny(KindSection, "Staatsexamen")
	require.True(t, ok)
	assert.Equal(t, "stex", sec.Term.Tag)
	assert.Equal(t, KindSection, sec.Kind)

	typ, ok := v.LookupAny(KindType, "Staatsexamen")
	require.True(t, ok)
	assert.Equal(t, KindType, typ.Kind)
}
