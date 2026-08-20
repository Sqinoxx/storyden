package library_import

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func realManifest(t *testing.T) *Manifest {
	t.Helper()

	data, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "import-manifest.yaml"))
	require.NoError(t, err)

	m, err := ParseManifest(data)
	require.NoError(t, err)

	return m
}

func realPlanner(t *testing.T) *Planner {
	t.Helper()
	return NewPlanner(realManifest(t), realVocabulary(t))
}

func planOne(t *testing.T, path string) (*Plan, PlannedFile) {
	t.Helper()

	plan, err := realPlanner(t).Plan([]Entry{{Path: path, Size: 1, SHA256: "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789", MIME: "application/pdf"}})
	require.NoError(t, err)
	require.Len(t, plan.Files, 1)

	return plan, plan.Files[0]
}

func containerPath(t *testing.T, plan *Plan, slug string) []string {
	t.Helper()

	for _, c := range plan.Containers {
		if c.Slug == slug {
			return c.Path
		}
	}

	t.Fatalf("container %q not in plan", slug)
	return nil
}

func TestPlanMapsRealSourcePathsOntoTheTree(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		path     string
		wantTree []string
		wantTags []string
	}{
		{
			name:     "vorklinik altklausur",
			path:     "Altklausuren/Vorklinik (1.-4. Semester)/Physiologie/Physio 2019.pdf",
			wantTree: []string{"Bibliothek", "Vorklinik", "Physiologie", "Altklausuren"},
			wantTags: []string{"vorklinik", "physiologie", "altklausur"},
		},
		{
			name:     "klinik pruefung with a messy subject folder",
			path:     "Klausuren&Prüfungen/Klinik/MKG (Radio, AnEx, Auscultando, Spritzenkurs)/probe.pdf",
			wantTree: []string{"Bibliothek", "Klinik", "MKG", "Altklausuren"},
			wantTags: []string{"klinik", "mkg", "altklausur"},
		},
		{
			name:     "zwischenklinik folds onto z2",
			path:     "Altklausuren/Zwischenklinik (5.-6. Semester)/Radiologie für Zahnmediziner (5. Semester)/rad.pdf",
			wantTree: []string{"Bibliothek", "Z2", "Radiologie", "Altklausuren"},
			wantTags: []string{"z2", "radiologie", "altklausur"},
		},
		{
			name:     "lecture material",
			path:     "Vorlesungen&Unterlagen/Klinik/Kons/Kurs/endo.pdf",
			wantTree: []string{"Bibliothek", "Klinik", "Kons", "Vorlesungen"},
			wantTags: []string{"klinik", "kons", "vorlesung"},
		},
		{
			name:     "stex summaries get their own section",
			path:     "Vorlesungen&Unterlagen/STEX Zusammenfassungen/Kons/Kons STEX.pdf",
			wantTree: []string{"Bibliothek", "Staatsexamen", "Kons", "Zusammenfassungen"},
			wantTags: []string{"stex", "kons", "zusammenfassung"},
		},
		{
			name:     "anki decks branch off",
			path:     "Zusammenfassungen, Vorlesungen, Anki-Decks/Ankis/Vorklinik/Biochemie/deck.apkg",
			wantTree: []string{"Bibliothek", "Anki-Decks", "Vorklinik", "Biochemie"},
			wantTags: []string{"vorklinik", "biochemie", "anki"},
		},
		{
			name:     "altprotokolle",
			path:     "Altprotokolle/Z1/protokoll.pdf",
			wantTree: []string{"Bibliothek", "Z1", "Gedächtnisprotokolle"},
			wantTags: []string{"z1", "gedächtnisprotokoll"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			plan, file := planOne(t, tt.path)

			assert.Equal(t, tt.wantTree, containerPath(t, plan, file.ParentSlug))
			assert.Equal(t, tt.wantTags, file.Tags)
			assert.Equal(t, "published", file.Visibility)
		})
	}
}

func TestPlanNamesLeafNodesWithoutTheExtension(t *testing.T) {
	t.Parallel()

	_, file := planOne(t, "Altklausuren/Vorklinik (1.-4. Semester)/Physiologie/Physio 2019.pdf")

	assert.Equal(t, "Physio 2019", file.Name)
}

func TestPlanBuildsTheFullContainerChain(t *testing.T) {
	t.Parallel()

	plan, _ := planOne(t, "Altklausuren/Vorklinik (1.-4. Semester)/Physiologie/Physio 2019.pdf")

	var slugs []string
	for _, c := range plan.Containers {
		slugs = append(slugs, c.Slug)
	}

	assert.Equal(t, []string{"bibliothek", "vorklinik", "vorklinik-physiologie", "vorklinik-physiologie-altklausuren"}, slugs)
}

// Parents must precede children so ingest can create the tree in one pass.
func TestPlanOrdersContainersParentsFirst(t *testing.T) {
	t.Parallel()

	plan, err := realPlanner(t).Plan([]Entry{
		{Path: "Vorlesungen&Unterlagen/Klinik/Kons/a.pdf", SHA256: "aa11", MIME: "application/pdf"},
		{Path: "Altklausuren/Vorklinik (1.-4. Semester)/Physiologie/b.pdf", SHA256: "bb22", MIME: "application/pdf"},
	})
	require.NoError(t, err)

	seen := map[string]struct{}{}
	for _, c := range plan.Containers {
		if c.ParentSlug != "" {
			_, ok := seen[c.ParentSlug]
			assert.True(t, ok, "container %q appears before its parent %q", c.Slug, c.ParentSlug)
		}
		seen[c.Slug] = struct{}{}
	}
}

func TestPlanAssignsChildSchemaToContainers(t *testing.T) {
	t.Parallel()

	plan, file := planOne(t, "Altklausuren/Vorklinik (1.-4. Semester)/Physiologie/Physio 2019.pdf")

	for _, c := range plan.Containers {
		if c.Slug == file.ParentSlug {
			assert.Equal(t, []string{"Fach", "Semester", "Typ", "Jahr", "Dozent"}, c.ChildSchema)
			return
		}
	}

	t.Fatal("parent container missing from plan")
}

// Identical filenames in different folders are extremely common in the source
// tree and the slug column is unique, so the planner has to keep them apart.
func TestPlanGeneratesUniqueSlugs(t *testing.T) {
	t.Parallel()

	plan, err := realPlanner(t).Plan([]Entry{
		{Path: "Altklausuren/Vorklinik (1.-4. Semester)/Physiologie/Klausur.pdf", SHA256: "aa11bb22cc33dd44ee55ff66aa11bb22cc33dd44ee55ff66aa11bb22cc33dd44"},
		{Path: "Altklausuren/Vorklinik (1.-4. Semester)/Anatomie/Klausur.pdf", SHA256: "1122334455667788990011223344556677889900112233445566778899001122"},
		{Path: "Klausuren&Prüfungen/Vorklinik/Physiologie/Klausur.pdf", SHA256: "ffeeddccbbaa99887766554433221100ffeeddccbbaa99887766554433221100"},
	})
	require.NoError(t, err)

	seen := map[string]struct{}{}
	for _, f := range plan.Files {
		_, dup := seen[f.Slug]
		assert.False(t, dup, "duplicate slug %q", f.Slug)
		seen[f.Slug] = struct{}{}
	}

	assert.Len(t, seen, 3)
}

func TestPlanSlugsAreStableAcrossRuns(t *testing.T) {
	t.Parallel()

	entries := []Entry{
		{Path: "Altklausuren/Vorklinik (1.-4. Semester)/Physiologie/Klausur.pdf", SHA256: "aa11bb22cc33dd44ee55ff66aa11bb22cc33dd44ee55ff66aa11bb22cc33dd44"},
		{Path: "Klausuren&Prüfungen/Vorklinik/Physiologie/Klausur.pdf", SHA256: "ffeeddccbbaa99887766554433221100ffeeddccbbaa99887766554433221100"},
	}

	first, err := realPlanner(t).Plan(entries)
	require.NoError(t, err)
	second, err := realPlanner(t).Plan(entries)
	require.NoError(t, err)

	require.Len(t, first.Files, 2)
	for i := range first.Files {
		assert.Equal(t, first.Files[i].Slug, second.Files[i].Slug)
	}
}

func TestPlanCountsCatchAllAndUnresolvedTerms(t *testing.T) {
	t.Parallel()

	plan, err := realPlanner(t).Plan([]Entry{
		{Path: "Irgendwas/komisch/datei.pdf", SHA256: "aa11"},
		{Path: "Altklausuren/Vorklinik (1.-4. Semester)/Doktorarbeitsvorlage/x.pdf", SHA256: "bb22"},
	})
	require.NoError(t, err)

	assert.Equal(t, 1, plan.CatchAll)
	assert.Contains(t, plan.Unresolved, "subject:Doktorarbeitsvorlage")
}

// An unknown folder still becomes a node, but it must not invent a tag —
// otherwise a single stray directory pollutes the closed vocabulary.
func TestPlanDoesNotMintTagsForUnknownFolders(t *testing.T) {
	t.Parallel()

	plan, file := planOne(t, "Vorlesungen&Unterlagen/Klinik/Doktorarbeitsvorlage/vorlage.doc")

	assert.Equal(t, []string{"Bibliothek", "Klinik", "Doktorarbeitsvorlage", "Vorlesungen"}, containerPath(t, plan, file.ParentSlug))
	assert.Equal(t, []string{"klinik", "vorlesung"}, file.Tags)
	assert.Contains(t, plan.Unresolved, "subject:Doktorarbeitsvorlage")
}

func TestPlanResolvesTermsFiledOnTheWrongAxis(t *testing.T) {
	t.Parallel()

	_, file := planOne(t, "Klausuren&Prüfungen/Klinik/STEX/protokoll.pdf")

	assert.Contains(t, file.Tags, "stex")
}

// A file lying directly in a top-level source folder must not become a
// section container named after itself.
func TestPlanFilesLooseFilesUnderSonstiges(t *testing.T) {
	t.Parallel()

	plan, file := planOne(t, "Klausuren&Prüfungen/Zahni-Kreuz-Apps.docx")

	assert.Equal(t, []string{"Bibliothek", "Sonstiges"}, containerPath(t, plan, file.ParentSlug))
	assert.Zero(t, plan.CatchAll)

	for _, c := range plan.Containers {
		assert.NotContains(t, c.Name, ".docx", "a filename leaked into the tree as %q", c.Name)
	}
}
