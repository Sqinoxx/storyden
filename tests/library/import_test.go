package library_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/resources/account/account_writer"
	"github.com/Southclaws/storyden/app/resources/asset/asset_querier"
	"github.com/Southclaws/storyden/app/resources/library"
	"github.com/Southclaws/storyden/app/resources/library/node_querier"
	"github.com/Southclaws/storyden/app/resources/rbac"
	"github.com/Southclaws/storyden/app/resources/seed"
	"github.com/Southclaws/storyden/app/resources/visibility"
	"github.com/Southclaws/storyden/app/services/authentication/session"
	"github.com/Southclaws/storyden/app/services/library/library_import"
	"github.com/Southclaws/storyden/internal/config"
	"github.com/Southclaws/storyden/internal/integration"
	"github.com/Southclaws/storyden/internal/integration/e2e"
)

// fixtureTree mirrors the shapes that actually occur in the source archive:
// two competing branches holding the same document, umlauts and ampersands in
// directory names, repeated filenames across subjects, and a folder no
// vocabulary term covers.
func fixtureTree(t *testing.T) string {
	t.Helper()

	root := t.TempDir()

	files := map[string]string{
		"Altklausuren/Vorklinik (1.-4. Semester)/Physiologie/Klausur 2019.pdf":                          "physio nineteen",
		"Altklausuren/Vorklinik (1.-4. Semester)/Anatomie/Klausur 2019.pdf":                             "anatomie nineteen",
		"Altklausuren/Zwischenklinik (5.-6. Semester)/Radiologie für Zahnmediziner (5. Semester)/r.pdf": "radiologie",
		"Klausuren&Prüfungen/Klinik/MKG (Radio, AnEx, Auscultando, Spritzenkurs)/mkg.pdf":               "mkg exam",
		"Vorlesungen&Unterlagen/Klinik/Kons/Kurs/endo.pdf":                                              "endodontie kurs",
		"Vorlesungen&Unterlagen/Klinik/Doktorarbeitsvorlage/vorlage.doc":                                "template",
		"Vorlesungen&Unterlagen/STEX Zusammenfassungen/Kons/Kons STEX.pdf":                              "kons stex summary",
		"Altprotokolle/Z1/protokoll.pdf":                                                                "protokoll",
		"Zusammenfassungen, Vorlesungen, Anki-Decks/Ankis/Vorklinik/Biochemie/deck.apkg":                "anki deck",

		// Same bytes as the STEX summary above but in the demoted branch, so
		// dedupe must keep the other copy and drop this one.
		"Zusammenfassungen, Vorlesungen, Anki-Decks/Vorlesungen & Unterlagen/Klinik/Kons/Kons STEX.pdf": "kons stex summary",
	}

	for rel, content := range files {
		abs := filepath.Join(root, filepath.FromSlash(rel))
		require.NoError(t, os.MkdirAll(filepath.Dir(abs), 0o755))
		require.NoError(t, os.WriteFile(abs, []byte(content), 0o644))
	}

	return root
}

func loadImportConfig(t *testing.T) (*library_import.Manifest, *library_import.Vocabulary) {
	t.Helper()

	manifestData, err := os.ReadFile(filepath.Join("..", "..", "import-manifest.yaml"))
	require.NoError(t, err)
	manifest, err := library_import.ParseManifest(manifestData)
	require.NoError(t, err)

	vocabData, err := os.ReadFile(filepath.Join("..", "..", "import-vocab.yaml"))
	require.NoError(t, err)
	vocab, err := library_import.ParseVocabulary(vocabData)
	require.NoError(t, err)

	return manifest, vocab
}

func TestLibraryImportEndToEnd(t *testing.T) {
	// OCR stays off so the import measures only itself; the assets it creates
	// are asserted to be left pending for the separate extraction phase.
	cfg := &config.Config{OCREnabled: false, OCRBackfillEnabled: false}

	root := fixtureTree(t)
	manifest, vocab := loadImportConfig(t)

	integration.Test(t, cfg, e2e.Setup(), fx.Invoke(func(
		ctx context.Context,
		lc fx.Lifecycle,
		aw *account_writer.Writer,
		ingester *library_import.Ingester,
		nq *node_querier.Querier,
		aq *asset_querier.Querier,
	) {
		lc.Append(fx.StartHook(func() {
			a := assert.New(t)
			r := require.New(t)

			accCtx, acc := e2e.WithAccount(ctx, aw, seed.Account_001_Odin)
			adminCtx := session.WithAccountPermissions(accCtx, *acc, rbac.NewList(rbac.PermissionAdministrator))

			inv, err := library_import.Scan(adminCtx, root, library_import.ScanOptions{
				Demote: manifest.Defaults.Demote,
			})
			r.NoError(err)
			a.Len(inv.Entries, 10)

			deduped := library_import.Dedupe(inv.Entries, manifest.Defaults.Demote)
			a.Len(deduped.Canonical, 9, "the duplicated STEX summary should collapse")
			r.Len(deduped.Groups, 1)
			a.Equal("Vorlesungen&Unterlagen/STEX Zusammenfassungen/Kons/Kons STEX.pdf", deduped.Groups[0].Canonical)

			plan, err := library_import.NewPlanner(manifest, vocab).Plan(deduped.Canonical)
			r.NoError(err)
			a.Zero(plan.CatchAll, "every fixture path should match a specific rule")

			ledgerPath := filepath.Join(t.TempDir(), "import-state.jsonl")
			ledger, err := library_import.OpenLedger(ledgerPath)
			r.NoError(err)
			defer ledger.Close()

			result, err := ingester.Apply(adminCtx, plan, library_import.IngestOptions{
				Root:   root,
				Owner:  acc.ID,
				Ledger: ledger,
			})
			r.NoError(err)
			a.Equal(9, result.FilesIngested)
			a.Zero(result.FilesSkipped)
			a.Positive(result.ContainersCreated)

			// The tree the manifest promises.
			for _, slug := range []string{
				"bibliothek",
				"vorklinik",
				"vorklinik-physiologie",
				"vorklinik-physiologie-altklausuren",
				"klinik-mkg-altklausuren",
				"staatsexamen-kons-zusammenfassungen",
				"anki-decks-vorklinik-biochemie",
			} {
				_, err := nq.Get(adminCtx, library.NewKey(slug))
				a.NoError(err, "expected container %q", slug)
			}

			// A type container carries the child schema, which is what turns the
			// listing into a sortable table in the directory block.
			typeNode, err := nq.Get(adminCtx, library.NewKey("vorklinik-physiologie-altklausuren"))
			r.NoError(err)
			schema, ok := typeNode.ChildProperties.Get()
			r.True(ok, "child property schema missing")
			names := []string{}
			for _, f := range schema.Fields {
				names = append(names, f.Name)
			}
			a.ElementsMatch([]string{"Fach", "Semester", "Typ", "Jahr", "Dozent"}, names)

			// A leaf: asset attached, tags on all three axes, rule properties set,
			// and left pending so the extraction phase picks it up.
			var leafSlug string
			for _, f := range plan.Files {
				if f.Entry.Path == "Altklausuren/Vorklinik (1.-4. Semester)/Physiologie/Klausur 2019.pdf" {
					leafSlug = f.Slug
				}
			}
			r.NotEmpty(leafSlug)

			leaf, err := nq.Get(adminCtx, library.NewKey(leafSlug))
			r.NoError(err)
			a.Equal("Klausur 2019", leaf.Name)
			r.Len(leaf.Assets, 1)

			tagNames := []string{}
			for _, tg := range leaf.Tags {
				tagNames = append(tagNames, tg.Name.String())
			}
			a.ElementsMatch([]string{"vorklinik", "physiologie", "altklausur"}, tagNames)

			props, ok := leaf.Properties.Get()
			r.True(ok, "leaf properties missing")
			typValue := ""
			for _, p := range props.Properties {
				if p.Field.Name == "Typ" {
					typValue = p.Value.OrZero()
				}
			}
			a.Equal("Altklausur", typValue)

			stored, err := aq.GetByID(adminCtx, leaf.Assets[0].ID)
			r.NoError(err)
			a.Equal("pending", stored.OCRStatus, "assets must be left for the extraction phase")

			// A folder outside the vocabulary becomes a node but mints no tag.
			unknown, err := nq.Get(adminCtx, library.NewKey("klinik-doktorarbeitsvorlage-vorlesungen"))
			r.NoError(err)
			a.NotNil(unknown)
		}))
	}))
}

// Rerunning apply against a populated ledger must be a no-op rather than
// duplicating the tree, because a 30 GB run will be interrupted.
func TestLibraryImportIsResumable(t *testing.T) {
	cfg := &config.Config{OCREnabled: false, OCRBackfillEnabled: false}

	root := fixtureTree(t)
	manifest, vocab := loadImportConfig(t)

	integration.Test(t, cfg, e2e.Setup(), fx.Invoke(func(
		ctx context.Context,
		lc fx.Lifecycle,
		aw *account_writer.Writer,
		ingester *library_import.Ingester,
	) {
		lc.Append(fx.StartHook(func() {
			a := assert.New(t)
			r := require.New(t)

			accCtx, acc := e2e.WithAccount(ctx, aw, seed.Account_001_Odin)
			adminCtx := session.WithAccountPermissions(accCtx, *acc, rbac.NewList(rbac.PermissionAdministrator))

			inv, err := library_import.Scan(adminCtx, root, library_import.ScanOptions{Demote: manifest.Defaults.Demote})
			r.NoError(err)

			deduped := library_import.Dedupe(inv.Entries, manifest.Defaults.Demote)
			plan, err := library_import.NewPlanner(manifest, vocab).Plan(deduped.Canonical)
			r.NoError(err)

			ledgerPath := filepath.Join(t.TempDir(), "import-state.jsonl")

			ledger, err := library_import.OpenLedger(ledgerPath)
			r.NoError(err)

			first, err := ingester.Apply(adminCtx, plan, library_import.IngestOptions{Root: root, Owner: acc.ID, Ledger: ledger})
			r.NoError(err)
			r.NoError(ledger.Close())
			a.Equal(9, first.FilesIngested)

			reopened, err := library_import.OpenLedger(ledgerPath)
			r.NoError(err)
			defer reopened.Close()
			a.Equal(9, reopened.Len(), "ledger should survive a restart")

			second, err := ingester.Apply(adminCtx, plan, library_import.IngestOptions{Root: root, Owner: acc.ID, Ledger: reopened})
			r.NoError(err)
			a.Zero(second.FilesIngested)
			a.Equal(9, second.FilesSkipped)
			a.Zero(second.ContainersCreated, "containers should be reused, not recreated")
		}))
	}))
}

// This is exactly the situation that motivated SetVisibility: a manifest
// default is discovered to be wrong only after real content is already
// imported (unlisted turned out to be excluded from search entirely), and
// re-running the whole import is not an option once assets are uploaded and
// OCR has run. The ledger has to be enough to retarget every node that was
// actually created.
func TestLibraryImportSetVisibility(t *testing.T) {
	cfg := &config.Config{OCREnabled: false, OCRBackfillEnabled: false}

	root := fixtureTree(t)

	// Deliberately decoupled from the real import-manifest.yaml: this test
	// simulates the exact situation that motivated SetVisibility (a manifest
	// default turns out to be wrong after real content already imported), so
	// it needs to control the starting visibility itself rather than track
	// whatever the shared manifest currently says.
	_, vocab := loadImportConfig(t)
	manifestData := []byte(`defaults:
  root: Bibliothek
  visibility: unlisted
rules:
  - match: "**"
    target: "{path}"
`)
	manifest, err := library_import.ParseManifest(manifestData)
	require.NoError(t, err)

	integration.Test(t, cfg, e2e.Setup(), fx.Invoke(func(
		ctx context.Context,
		lc fx.Lifecycle,
		aw *account_writer.Writer,
		ingester *library_import.Ingester,
		nq *node_querier.Querier,
	) {
		lc.Append(fx.StartHook(func() {
			a := assert.New(t)
			r := require.New(t)

			accCtx, acc := e2e.WithAccount(ctx, aw, seed.Account_001_Odin)
			adminCtx := session.WithAccountPermissions(accCtx, *acc, rbac.NewList(rbac.PermissionAdministrator))

			inv, err := library_import.Scan(adminCtx, root, library_import.ScanOptions{Demote: manifest.Defaults.Demote})
			r.NoError(err)

			deduped := library_import.Dedupe(inv.Entries, manifest.Defaults.Demote)
			plan, err := library_import.NewPlanner(manifest, vocab).Plan(deduped.Canonical)
			r.NoError(err)

			ledgerPath := filepath.Join(t.TempDir(), "import-state.jsonl")
			ledger, err := library_import.OpenLedger(ledgerPath)
			r.NoError(err)
			defer ledger.Close()

			applied, err := ingester.Apply(adminCtx, plan, library_import.IngestOptions{Root: root, Owner: acc.ID, Ledger: ledger})
			r.NoError(err)
			r.Equal(9, applied.FilesIngested)

			var leafSlug string
			for _, f := range plan.Files {
				if f.Entry.Path == "Altklausuren/Vorklinik (1.-4. Semester)/Physiologie/Klausur 2019.pdf" {
					leafSlug = f.Slug
				}
			}
			r.NotEmpty(leafSlug)

			before, err := nq.Get(adminCtx, library.NewKey(leafSlug))
			r.NoError(err)
			a.Equal(visibility.VisibilityUnlisted, before.Visibility, "the fixture manifest still defaults to unlisted")

			// A dry run must report what it would do without touching the DB —
			// this is the exact case that slipped through uncovered the first
			// time SetVisibility shipped.
			dry, err := ingester.SetVisibility(adminCtx, visibility.VisibilityPublished, library_import.FixVisibilityOptions{Ledger: ledger, DryRun: true})
			r.NoError(err)
			a.Equal(9, dry.Updated)

			stillUnlisted, err := nq.Get(adminCtx, library.NewKey(leafSlug))
			r.NoError(err)
			a.Equal(visibility.VisibilityUnlisted, stillUnlisted.Visibility, "dry run must not write anything")

			result, err := ingester.SetVisibility(adminCtx, visibility.VisibilityPublished, library_import.FixVisibilityOptions{Ledger: ledger})
			r.NoError(err)
			a.Equal(9, result.Updated)
			a.Zero(result.Skipped)

			after, err := nq.Get(adminCtx, library.NewKey(leafSlug))
			r.NoError(err)
			a.Equal(visibility.VisibilityPublished, after.Visibility)

			// A second run against the same target must be a pure no-op, since
			// a large archive's fix pass will often be interrupted and resumed.
			again, err := ingester.SetVisibility(adminCtx, visibility.VisibilityPublished, library_import.FixVisibilityOptions{Ledger: ledger})
			r.NoError(err)
			a.Zero(again.Updated)
			a.Equal(9, again.Skipped)
		}))
	}))
}
