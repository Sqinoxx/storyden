package library_import

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Southclaws/opt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Southclaws/storyden/app/resources/library"
	"github.com/Southclaws/storyden/app/resources/tag/tag_ref"
)

func nodeWithProperties(values map[string]string) *library.Node {
	table := library.PropertyTable{}

	for name, value := range values {
		field := library.PropertySchemaField{Name: name}
		table.Schema.Fields = append(table.Schema.Fields, &field)
		table.Properties = append(table.Properties, &library.Property{
			Field: field,
			Value: opt.New(value),
		})
	}

	return &library.Node{Properties: opt.New(table)}
}

// The import fills Typ on every node from the manifest rules. Treating that as
// "already enriched" is what would have made a full run a silent no-op.
func TestMissingEnrichFieldsIgnoresTypFromTheImport(t *testing.T) {
	t.Parallel()

	got := missingEnrichFields(nodeWithProperties(map[string]string{"Typ": "Altklausur"}))

	assert.ElementsMatch(t, []string{"Fach", "Semester", "Jahr", "Dozent"}, got)
	assert.NotContains(t, got, "Typ", "a value that is already set is not missing")
}

func TestMissingEnrichFieldsEmptyWhenEverythingIsSet(t *testing.T) {
	t.Parallel()

	got := missingEnrichFields(nodeWithProperties(map[string]string{
		"Fach": "kons", "Semester": "5", "Typ": "Altklausur", "Jahr": "2019", "Dozent": "Weber",
	}))

	assert.Empty(t, got, "a fully populated node is the only one that may be skipped")
}

// Blank values must not count as filled, otherwise a node the model had nothing
// to say about would never be revisited.
func TestMissingEnrichFieldsTreatsBlanksAsMissing(t *testing.T) {
	t.Parallel()

	got := missingEnrichFields(nodeWithProperties(map[string]string{
		"Fach": "", "Semester": "", "Typ": "Altklausur", "Jahr": "", "Dozent": "",
	}))

	assert.ElementsMatch(t, []string{"Fach", "Semester", "Jahr", "Dozent"}, got)
}

func TestMissingEnrichFieldsHandlesNodeWithoutProperties(t *testing.T) {
	t.Parallel()

	assert.ElementsMatch(t, enrichFields, missingEnrichFields(&library.Node{}))
}

func testVocabulary(t *testing.T) *Vocabulary {
	t.Helper()

	data, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "import-vocab.yaml"))
	require.NoError(t, err)

	v, err := ParseVocabulary(data)
	require.NoError(t, err)

	return v
}

func nodeWithTags(names ...string) *library.Node {
	node := &library.Node{}
	for _, n := range names {
		node.Tags = append(node.Tags, &tag_ref.Tag{Name: tag_ref.NewName(n)})
	}
	return node
}

// The folder placement is how the material was actually filed, so it decides
// the subject. The model disagreed with it on 30% of a real sample, nearly
// always for the worse.
func TestSubjectFromTagsPrefersTheFolderSubject(t *testing.T) {
	t.Parallel()

	vocab := testVocabulary(t)

	assert.Equal(t, "MKG", subjectFromTags(nodeWithTags("klinik", "mkg", "altklausur"), vocab))
	assert.Equal(t, "Dentale Technologie", subjectFromTags(nodeWithTags("vorklinik", "dentale-technologie"), vocab))
}

// Where the folder named no subject the model is the only source available.
func TestSubjectFromTagsEmptyWithoutASubjectTag(t *testing.T) {
	t.Parallel()

	vocab := testVocabulary(t)

	assert.Empty(t, subjectFromTags(nodeWithTags("klinik", "altklausur"), vocab), "section and type tags are not subjects")
	assert.Empty(t, subjectFromTags(&library.Node{}, vocab))
}
