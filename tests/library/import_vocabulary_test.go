package library_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"github.com/Southclaws/opt"
	"github.com/Southclaws/storyden/app/resources/account/account_writer"
	"github.com/Southclaws/storyden/app/resources/library"
	"github.com/Southclaws/storyden/app/resources/library/node_querier"
	"github.com/Southclaws/storyden/app/resources/rbac"
	"github.com/Southclaws/storyden/app/resources/seed"
	"github.com/Southclaws/storyden/app/resources/tag/tag_querier"
	"github.com/Southclaws/storyden/app/resources/tag/tag_ref"
	"github.com/Southclaws/storyden/app/services/authentication/session"
	"github.com/Southclaws/storyden/app/services/library/library_import"
	"github.com/Southclaws/storyden/app/services/library/node_mutate"
	"github.com/Southclaws/storyden/internal/config"
	"github.com/Southclaws/storyden/internal/integration"
	"github.com/Southclaws/storyden/internal/integration/e2e"
)

// The three typo corrections run against a live vocabulary, so an existing page
// tagged with the misspelling has to come out the other side still tagged.
func TestLibraryImportVocabularyRenamePreservesLinks(t *testing.T) {
	cfg := &config.Config{OCREnabled: false, OCRBackfillEnabled: false}

	_, vocab := loadImportConfig(t)

	integration.Test(t, cfg, e2e.Setup(), fx.Invoke(func(
		ctx context.Context,
		lc fx.Lifecycle,
		aw *account_writer.Writer,
		ingester *library_import.Ingester,
		nodes *node_mutate.Manager,
		nq *node_querier.Querier,
		tq *tag_querier.Querier,
	) {
		lc.Append(fx.StartHook(func() {
			a := assert.New(t)
			r := require.New(t)

			accCtx, acc := e2e.WithAccount(ctx, aw, seed.Account_001_Odin)
			adminCtx := session.WithAccountPermissions(accCtx, *acc, rbac.NewList(rbac.PermissionAdministrator))

			// A page tagged the way the live database currently has it.
			node, err := nodes.Create(adminCtx, acc.ID, "PA-Sammlung", node_mutate.Partial{
				Tags: opt.New(tag_ref.Names{tag_ref.NewName("parodondologie"), tag_ref.NewName("kfo")}),
			})
			r.NoError(err)

			actions, err := ingester.ApplyVocabulary(adminCtx, vocab, false)
			r.NoError(err)
			a.NotEmpty(actions)

			reloaded, err := nq.Get(adminCtx, library.NewKey(node.GetSlug()))
			r.NoError(err)

			names := []string{}
			for _, tg := range reloaded.Tags {
				names = append(names, tg.Name.String())
			}

			a.Contains(names, "parodontologie", "the rename must carry the existing link across")
			a.NotContains(names, "parodondologie")
			a.Contains(names, "kfo", "unrelated tags must be untouched")

			// Every vocabulary term should now exist so classification has a
			// closed set to work against.
			all, err := tq.List(adminCtx)
			r.NoError(err)

			existing := map[string]struct{}{}
			for _, tg := range all {
				existing[tg.Name.String()] = struct{}{}
			}

			for _, want := range vocab.Tags() {
				_, ok := existing[want]
				a.True(ok, "vocabulary tag %q was not created", want)
			}
		}))
	}))
}

// Running the vocabulary pass twice must not fail on the second run: the
// misspelled tags are gone by then and the correct ones already exist.
func TestLibraryImportVocabularyIsIdempotent(t *testing.T) {
	cfg := &config.Config{OCREnabled: false, OCRBackfillEnabled: false}

	_, vocab := loadImportConfig(t)

	integration.Test(t, cfg, e2e.Setup(), fx.Invoke(func(
		ctx context.Context,
		lc fx.Lifecycle,
		aw *account_writer.Writer,
		ingester *library_import.Ingester,
		tq *tag_querier.Querier,
	) {
		lc.Append(fx.StartHook(func() {
			a := assert.New(t)
			r := require.New(t)

			accCtx, acc := e2e.WithAccount(ctx, aw, seed.Account_001_Odin)
			adminCtx := session.WithAccountPermissions(accCtx, *acc, rbac.NewList(rbac.PermissionAdministrator))

			_, err := ingester.ApplyVocabulary(adminCtx, vocab, false)
			r.NoError(err)

			before, err := tq.List(adminCtx)
			r.NoError(err)

			_, err = ingester.ApplyVocabulary(adminCtx, vocab, false)
			r.NoError(err)

			after, err := tq.List(adminCtx)
			r.NoError(err)

			a.Equal(len(before), len(after), "a second run must not create more tags")
		}))
	}))
}

func TestLibraryImportVocabularyDryRunWritesNothing(t *testing.T) {
	cfg := &config.Config{OCREnabled: false, OCRBackfillEnabled: false}

	_, vocab := loadImportConfig(t)

	integration.Test(t, cfg, e2e.Setup(), fx.Invoke(func(
		ctx context.Context,
		lc fx.Lifecycle,
		aw *account_writer.Writer,
		ingester *library_import.Ingester,
		tq *tag_querier.Querier,
	) {
		lc.Append(fx.StartHook(func() {
			a := assert.New(t)
			r := require.New(t)

			accCtx, acc := e2e.WithAccount(ctx, aw, seed.Account_001_Odin)
			adminCtx := session.WithAccountPermissions(accCtx, *acc, rbac.NewList(rbac.PermissionAdministrator))

			before, err := tq.List(adminCtx)
			r.NoError(err)

			actions, err := ingester.ApplyVocabulary(adminCtx, vocab, true)
			r.NoError(err)
			a.NotEmpty(actions, "a dry run should still report what it would do")

			after, err := tq.List(adminCtx)
			r.NoError(err)
			a.Equal(len(before), len(after))
		}))
	}))
}
