package library_import

import (
	"testing"

	"github.com/Southclaws/opt"
	"github.com/stretchr/testify/assert"

	"github.com/Southclaws/storyden/app/resources/library"
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
