package library_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/resources/account/account_writer"
	"github.com/Southclaws/storyden/app/resources/library"
	"github.com/Southclaws/storyden/app/resources/library/node_querier"
	"github.com/Southclaws/storyden/app/resources/rbac"
	"github.com/Southclaws/storyden/app/resources/seed"
	"github.com/Southclaws/storyden/app/services/authentication/session"
	"github.com/Southclaws/storyden/app/services/library/library_import"
	"github.com/Southclaws/storyden/internal/config"
	"github.com/Southclaws/storyden/internal/infrastructure/ai"
	"github.com/Southclaws/storyden/internal/integration"
	"github.com/Southclaws/storyden/internal/integration/e2e"
)

// fakePrompter stands in for a language model. PromptObject dispatches through
// the StructuredPrompter interface, which is what makes this substitution
// possible at all. Replies are chosen from the prompt text rather than a fixed
// sequence, so assertions do not depend on the order the enricher happens to
// walk the ledger in.
type fakePrompter struct {
	ai.Mock

	mu    sync.Mutex
	calls int
	reply func(call int, input string) (string, error)
}

func (f *fakePrompter) PromptObjectJSON(ctx context.Context, description, input string, schema any) (string, error) {
	f.mu.Lock()
	call := f.calls
	f.calls++
	f.mu.Unlock()

	if f.reply == nil {
		return "{}", nil
	}

	return f.reply(call, input)
}

func (f *fakePrompter) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

func (f *fakePrompter) reset(reply func(call int, input string) (string, error)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = 0
	f.reply = reply
}

func enrichFixture(t *testing.T) string {
	t.Helper()

	root := t.TempDir()

	files := map[string]string{
		"Altklausuren/Vorklinik (1.-4. Semester)/Physiologie/Klausur 2019.pdf": "physio",
		"Altklausuren/Vorklinik (1.-4. Semester)/Anatomie/Klausur 2020.pdf":    "anatomie",
		"Altklausuren/Vorklinik (1.-4. Semester)/Biochemie/Klausur 2021.pdf":   "biochemie",
	}

	for rel, content := range files {
		abs := filepath.Join(root, filepath.FromSlash(rel))
		require.NoError(t, os.MkdirAll(filepath.Dir(abs), 0o755))
		require.NoError(t, os.WriteFile(abs, []byte(content), 0o644))
	}

	return root
}

// enrichFixtureN builds a tree of n documents, for cases that need more files
// than the consecutive-failure threshold.
func enrichFixtureN(t *testing.T, n int) string {
	t.Helper()

	root := t.TempDir()

	for i := range n {
		rel := fmt.Sprintf("Altklausuren/Vorklinik (1.-4. Semester)/Physiologie/Klausur %d.pdf", 2000+i)
		abs := filepath.Join(root, filepath.FromSlash(rel))
		require.NoError(t, os.MkdirAll(filepath.Dir(abs), 0o755))
		require.NoError(t, os.WriteFile(abs, fmt.Appendf(nil, "inhalt %d", i), 0o644))
	}

	return root
}

// This is the regression that matters most: the real import fills Typ on every
// node from the manifest rules, and the original skip check bailed as soon as
// *any* property had a value. That would have skipped all 2443 documents and
// made a full enrichment run a silent no-op.
func TestEnrichVisitsNodesThatAlreadyHaveTypSet(t *testing.T) {
	cfg := &config.Config{OCREnabled: false, OCRBackfillEnabled: false}

	root := enrichFixture(t)
	manifest, vocab := loadImportConfig(t)

	fake := &fakePrompter{reply: func(_ int, input string) (string, error) {
		if strings.Contains(input, "Klausur 2019") {
			return `{"titel":"Physiologie Klausur 2019","fach":"physiologie","semester":"2","typ":"Zusammenfassung","jahr":"2019","dozent":"Koch","tags":["physiologie","nicht-im-vokabular"]}`, nil
		}
		return "{}", nil
	}}

	integration.Test(t, cfg, e2e.Setup(),
		fx.Decorate(func(ai.Prompter) ai.Prompter { return fake }),
		fx.Invoke(func(
			ctx context.Context,
			lc fx.Lifecycle,
			aw *account_writer.Writer,
			ingester *library_import.Ingester,
			enricher *library_import.Enricher,
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

				ledger, err := library_import.OpenLedger(filepath.Join(t.TempDir(), "import-state.jsonl"))
				r.NoError(err)
				defer ledger.Close()

				_, err = ingester.Apply(adminCtx, plan, library_import.IngestOptions{Root: root, Owner: acc.ID, Ledger: ledger})
				r.NoError(err)

				// Precondition: the import behaves like production and fills Typ.
				var leafSlug string
				for _, f := range plan.Files {
					if f.Name == "Klausur 2019" {
						leafSlug = f.Slug
					}
				}
				r.NotEmpty(leafSlug)

				before, err := nq.Get(adminCtx, library.NewKey(leafSlug))
				r.NoError(err)
				beforeProps := propertyValues(t, before)
				r.Equal("Altklausur", beforeProps["Typ"], "precondition: the import must have set Typ")
				r.Empty(beforeProps["Fach"])

				state, err := library_import.OpenLedger(filepath.Join(t.TempDir(), "enrich-state.jsonl"))
				r.NoError(err)
				defer state.Close()

				result, err := enricher.Enrich(adminCtx, vocab, library_import.EnrichOptions{Ledger: ledger, State: state})
				r.NoError(err)

				a.Equal(3, result.Enriched, "nodes that already have Typ set must still be enriched")
				a.Zero(result.Skipped, "nothing may be skipped just because Typ is populated")
				a.Zero(result.Failed)
				a.Equal(3, fake.callCount(), "the model must actually have been consulted")

				after, err := nq.Get(adminCtx, library.NewKey(leafSlug))
				r.NoError(err)
				props := propertyValues(t, after)

				a.Equal("physiologie", props["Fach"], "empty fields get filled")
				a.Equal("2019", props["Jahr"])
				a.Equal("Koch", props["Dozent"])

				// Typ came from the folder path, which is deterministic and beats
				// an inference, so the model must not be able to overwrite it.
				a.Equal("Altklausur", props["Typ"], "the manifest value must survive")

				tags := tagNamesOf(after)
				a.Contains(tags, "physiologie")
				a.NotContains(tags, "nicht-im-vokabular", "tags outside the vocabulary are dropped")
			}))
		}))
}

// With a spent daily quota every remaining node would fail identically, so the
// run has to stop rather than grind through the rest of the ledger.
func TestEnrichAbortsOnRateLimitAndResumes(t *testing.T) {
	cfg := &config.Config{OCREnabled: false, OCRBackfillEnabled: false}

	root := enrichFixture(t)
	manifest, vocab := loadImportConfig(t)

	// The third call stands in for a spent daily quota.
	fake := &fakePrompter{reply: func(call int, _ string) (string, error) {
		if call >= 2 {
			return "", ai.ErrRateLimited
		}
		return `{"fach":"anatomie","jahr":"2020"}`, nil
	}}

	integration.Test(t, cfg, e2e.Setup(),
		fx.Decorate(func(ai.Prompter) ai.Prompter { return fake }),
		fx.Invoke(func(
			ctx context.Context,
			lc fx.Lifecycle,
			aw *account_writer.Writer,
			ingester *library_import.Ingester,
			enricher *library_import.Enricher,
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

				ledger, err := library_import.OpenLedger(filepath.Join(t.TempDir(), "import-state.jsonl"))
				r.NoError(err)
				defer ledger.Close()

				_, err = ingester.Apply(adminCtx, plan, library_import.IngestOptions{Root: root, Owner: acc.ID, Ledger: ledger})
				r.NoError(err)

				statePath := filepath.Join(t.TempDir(), "enrich-state.jsonl")
				state, err := library_import.OpenLedger(statePath)
				r.NoError(err)

				result, err := enricher.Enrich(adminCtx, vocab, library_import.EnrichOptions{Ledger: ledger, State: state})
				r.NoError(err, "a quota stop is an expected outcome, not an error")

				a.True(result.Aborted)
				a.NotEmpty(result.AbortReason)
				a.Equal(2, result.Enriched, "the two that succeeded before the limit are kept")
				a.Zero(result.Failed, "a quota stop must not be counted as a per-node failure")
				a.Equal(3, fake.callCount(), "the loop must stop at the limit, not carry on")

				// Resuming skips what already has values, so a rerun the next day
				// only costs quota for the nodes still outstanding.
				fake.reset(func(int, string) (string, error) {
					return `{"fach":"biochemie","jahr":"2021"}`, nil
				})

				// Reopening proves the state survives a process restart, which is
				// exactly what a next-day resume does.
				r.NoError(state.Close())
				reopened, err := library_import.OpenLedger(statePath)
				r.NoError(err)
				defer reopened.Close()
				a.Equal(2, reopened.Len(), "only the two that completed are recorded")

				second, err := enricher.Enrich(adminCtx, vocab, library_import.EnrichOptions{Ledger: ledger, State: reopened})
				r.NoError(err)

				a.False(second.Aborted)
				a.Equal(1, second.Enriched, "only the outstanding node is revisited")
				a.Equal(2, second.Skipped, "the finished ones are skipped")
				a.Equal(1, fake.callCount(), "no quota is spent on already-enriched nodes")
			}))
		}))
}

func propertyValues(t *testing.T, node *library.Node) map[string]string {
	t.Helper()

	out := map[string]string{}

	table, ok := node.Properties.Get()
	if !ok {
		return out
	}

	for _, p := range table.Properties {
		out[p.Field.Name] = p.Value.OrZero()
	}

	return out
}

func tagNamesOf(node *library.Node) []string {
	out := []string{}
	for _, t := range node.Tags {
		out = append(out, t.Name.String())
	}
	return out
}

// A wrong model name, a rejected schema or a bad key fails every request
// identically. Without a circuit breaker the run walks the whole ledger at the
// paced request rate — this actually happened, burning 155 documents of wall
// clock against a model the account could not call.
func TestEnrichAbortsWhenEveryRequestFailsTheSameWay(t *testing.T) {
	cfg := &config.Config{OCREnabled: false, OCRBackfillEnabled: false}

	// More documents than the breaker threshold, so the run has something to
	// stop short of.
	root := enrichFixtureN(t, 20)
	manifest, vocab := loadImportConfig(t)

	systemic := errors.New("NOT_FOUND: model is not available to this account")
	fake := &fakePrompter{reply: func(int, string) (string, error) {
		return "", systemic
	}}

	integration.Test(t, cfg, e2e.Setup(),
		fx.Decorate(func(ai.Prompter) ai.Prompter { return fake }),
		fx.Invoke(func(
			ctx context.Context,
			lc fx.Lifecycle,
			aw *account_writer.Writer,
			ingester *library_import.Ingester,
			enricher *library_import.Enricher,
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

				ledger, err := library_import.OpenLedger(filepath.Join(t.TempDir(), "import-state.jsonl"))
				r.NoError(err)
				defer ledger.Close()

				_, err = ingester.Apply(adminCtx, plan, library_import.IngestOptions{Root: root, Owner: acc.ID, Ledger: ledger})
				r.NoError(err)

				result, err := enricher.Enrich(adminCtx, vocab, library_import.EnrichOptions{Ledger: ledger})
				r.NoError(err)

				a.True(result.Aborted, "a systemic failure must stop the run")
				a.Contains(result.AbortReason, "consecutive failures")
				a.Zero(result.Enriched)
				a.Less(result.Failed, 20, "the run must stop well short of the full ledger")
				a.Equal(result.Failed, fake.callCount(), "no request may be made after the breaker trips")
			}))
		}))
}
