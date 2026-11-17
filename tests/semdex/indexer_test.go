package semdex_test

import (
	"context"
	"testing"

	"github.com/Southclaws/opt"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/resources/account/account_writer"
	"github.com/Southclaws/storyden/app/resources/seed"
	"github.com/Southclaws/storyden/app/services/semdex"
	"github.com/Southclaws/storyden/app/services/semdex/semdex_indexer"
	"github.com/Southclaws/storyden/app/transports/http/openapi"
	"github.com/Southclaws/storyden/internal/config"
	"github.com/Southclaws/storyden/internal/ent"
	ent_node "github.com/Southclaws/storyden/internal/ent/node"
	ent_post "github.com/Southclaws/storyden/internal/ent/post"
	"github.com/Southclaws/storyden/internal/integration"
	"github.com/Southclaws/storyden/internal/integration/e2e"
	"github.com/Southclaws/storyden/tests"
)

func TestSemdexIndexerBackfillsAndRemoves(t *testing.T) {
	t.Parallel()

	integration.Test(t, &config.Config{
		LanguageModelProvider: "mock",
		SemdexProvider:        "chromem",
		SemdexLocalPath:       t.TempDir(),
	}, e2e.Setup(), fx.Invoke(func(
		root context.Context,
		lc fx.Lifecycle,
		cl *openapi.ClientWithResponses,
		sh *e2e.SessionHelper,
		aw *account_writer.Writer,
		db *ent.Client,
		sd semdex.Semdexer,
		idx *semdex_indexer.Indexer,
	) {
		lc.Append(fx.StartHook(func() {
			r := require.New(t)

			adminCtx, _ := e2e.WithAccount(root, aw, seed.Account_001_Odin)
			memberCtx, _ := e2e.WithAccount(root, aw, seed.Account_003_Baldur)
			session := sh.WithSession(memberCtx)

			cat, err := cl.CategoryCreateWithResponse(root, openapi.CategoryInitialProps{Name: uuid.NewString(), Colour: "#000"}, sh.WithSession(adminCtx))
			tests.Ok(t, err, cat)

			create := func(vis openapi.Visibility) string {
				res, err := cl.ThreadCreateWithResponse(root, openapi.ThreadInitialProps{
					Title:      "Physikum Anatomie " + uuid.NewString(),
					Body:       opt.New("<p>Tipps zum Lernen der Hirnnerven</p>").Ptr(),
					Category:   opt.New(cat.JSON200.Id).Ptr(),
					Visibility: opt.New(vis).Ptr(),
				}, session)
				tests.Ok(t, err, res)
				return res.JSON200.Id
			}

			published1 := create(openapi.VisibilityPublished)
			published2 := create(openapi.VisibilityPublished)
			draft := create(openapi.VisibilityDraft)

			indexedAt := func(id string) *ent.Post {
				p, err := db.Post.Get(root, openapi.ParseID(id))
				r.NoError(err)
				return p
			}

			count := func() int {
				n, err := sd.(semdex.Counter).Count(root)
				r.NoError(err)
				return n
			}

			_, err = idx.Sync(root)
			r.NoError(err)

			r.NotNil(indexedAt(published1).SemdexIndexedAt)
			r.NotNil(indexedAt(published2).SemdexIndexedAt)
			r.Nil(indexedAt(draft).SemdexIndexedAt)

			remaining, err := db.Post.Query().
				Where(
					ent_post.RootPostIDIsNil(),
					ent_post.VisibilityEQ(ent_post.VisibilityPublished),
					ent_post.SemdexIndexedAtIsNil(),
				).
				Count(root)
			r.NoError(err)
			r.Zero(remaining, "every published thread is backfilled")

			before := count()
			r.GreaterOrEqual(before, 2)

			t.Run("second_sync_is_a_no_op", func(t *testing.T) {
				_, err := idx.Sync(root)
				require.NoError(t, err)
				require.Equal(t, before, count())
			})

			t.Run("unpublished_thread_is_removed", func(t *testing.T) {
				res, err := cl.ThreadUpdateWithResponse(root, published2, openapi.ThreadMutableProps{
					Visibility: opt.New(openapi.VisibilityDraft).Ptr(),
				}, session)
				tests.Ok(t, err, res)

				_, err = idx.Sync(root)
				require.NoError(t, err)

				require.Nil(t, indexedAt(published2).SemdexIndexedAt)
				require.Equal(t, before-1, count())
			})

			t.Run("wiped_index_is_rebuilt", func(t *testing.T) {
				posts, err := db.Post.Query().Where(ent_post.SemdexIndexedAtNotNil()).IDs(root)
				require.NoError(t, err)
				nodes, err := db.Node.Query().Where(ent_node.SemdexIndexedAtNotNil()).IDs(root)
				require.NoError(t, err)
				for _, id := range append(posts, nodes...) {
					_, err := sd.Delete(root, id)
					require.NoError(t, err)
				}
				require.Zero(t, count())

				require.NoError(t, idx.ResetIfIndexEmpty(root))
				require.Nil(t, indexedAt(published1).SemdexIndexedAt)

				_, err = idx.Sync(root)
				require.NoError(t, err)
				require.NotNil(t, indexedAt(published1).SemdexIndexedAt)
				require.Equal(t, before-1, count())
			})
		}))
	}))
}
