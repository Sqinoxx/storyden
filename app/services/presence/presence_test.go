package presence

import (
	"context"
	"testing"
	"time"

	"github.com/Southclaws/opt"
	"github.com/rs/xid"
	"github.com/stretchr/testify/require"

	"github.com/Southclaws/storyden/app/resources/account"
	"github.com/Southclaws/storyden/app/resources/post/category"
	"github.com/Southclaws/storyden/internal/infrastructure/cache/cachetest"
)

func TestTracker(t *testing.T) {
	ctx := context.Background()

	newTracker := func(now *time.Time) *Tracker {
		tr := New(cachetest.New())
		tr.now = func() time.Time { return *now }
		return tr
	}

	accA := account.AccountID(xid.New())
	accB := account.AccountID(xid.New())
	accC := account.AccountID(xid.New())
	catX := category.CategoryID(xid.New())
	catY := category.CategoryID(xid.New())

	t.Run("dedupes_repeated_heartbeats", func(t *testing.T) {
		r := require.New(t)
		now := time.Unix(1_000_000_020, 0)
		tr := newTracker(&now)

		for range 3 {
			_, err := tr.Heartbeat(ctx, accA, opt.NewEmpty[category.CategoryID]())
			r.NoError(err)
		}

		c, err := tr.Heartbeat(ctx, accA, opt.NewEmpty[category.CategoryID]())
		r.NoError(err)
		r.Equal(1, c.Online)
		r.False(c.InCategory.Ok())
	})

	t.Run("counts_others_in_same_category", func(t *testing.T) {
		r := require.New(t)
		now := time.Unix(1_000_000_020, 0)
		tr := newTracker(&now)

		_, err := tr.Heartbeat(ctx, accB, opt.New(catX))
		r.NoError(err)
		_, err = tr.Heartbeat(ctx, accC, opt.New(catY))
		r.NoError(err)

		c, err := tr.Heartbeat(ctx, accA, opt.New(catX))
		r.NoError(err)
		r.Equal(3, c.Online)
		r.Equal(1, c.InCategory.OrZero())

		c, err = tr.Heartbeat(ctx, accC, opt.New(catY))
		r.NoError(err)
		r.Equal(0, c.InCategory.OrZero())
	})

	t.Run("previous_bucket_still_counts_and_newer_category_wins", func(t *testing.T) {
		r := require.New(t)
		now := time.Unix(1_000_000_020, 0)
		tr := newTracker(&now)

		_, err := tr.Heartbeat(ctx, accB, opt.New(catX))
		r.NoError(err)

		now = now.Add(bucketSize)

		c, err := tr.Heartbeat(ctx, accA, opt.New(catX))
		r.NoError(err)
		r.Equal(2, c.Online)
		r.Equal(1, c.InCategory.OrZero())

		_, err = tr.Heartbeat(ctx, accB, opt.New(catY))
		r.NoError(err)

		c, err = tr.Heartbeat(ctx, accA, opt.New(catX))
		r.NoError(err)
		r.Equal(2, c.Online)
		r.Equal(0, c.InCategory.OrZero())
	})

	t.Run("expires_after_two_buckets", func(t *testing.T) {
		r := require.New(t)
		now := time.Unix(1_000_000_020, 0)
		tr := newTracker(&now)

		_, err := tr.Heartbeat(ctx, accB, opt.New(catX))
		r.NoError(err)

		now = now.Add(2 * bucketSize)

		c, err := tr.Heartbeat(ctx, accA, opt.New(catX))
		r.NoError(err)
		r.Equal(1, c.Online)
		r.Equal(0, c.InCategory.OrZero())
	})
}
