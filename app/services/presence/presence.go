package presence

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/Southclaws/opt"
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/resources/account"
	"github.com/Southclaws/storyden/app/resources/post/category"
	"github.com/Southclaws/storyden/internal/infrastructure/cache"
)

const (
	bucketSize = 30 * time.Second
	bucketTTL  = 3 * bucketSize
	keyPrefix  = "presence:"
	noCategory = "-"
)

type Counts struct {
	Online     int
	InCategory opt.Optional[int]
}

type Tracker struct {
	store cache.Store
	now   func() time.Time
}

func Build() fx.Option {
	return fx.Provide(New)
}

func New(store cache.Store) *Tracker {
	return &Tracker{store: store, now: time.Now}
}

func (t *Tracker) Heartbeat(ctx context.Context, accountID account.AccountID, categoryID opt.Optional[category.CategoryID]) (*Counts, error) {
	current := t.bucket()
	key := bucketKey(current)

	if _, err := t.store.HIncrBy(ctx, key, field(accountID, categoryID), 1); err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	if err := t.store.Expire(ctx, key, bucketTTL); err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	accounts := map[string]map[string]struct{}{}
	for _, b := range []int64{current, current - 1} {
		entries, err := t.store.HGetAll(ctx, bucketKey(b))
		if err != nil {
			return nil, fault.Wrap(err, fctx.With(ctx))
		}

		latest := map[string]map[string]struct{}{}
		for f := range entries {
			acc, cat, ok := strings.Cut(f, ":")
			if !ok {
				continue
			}
			if _, seen := accounts[acc]; seen {
				continue
			}
			if latest[acc] == nil {
				latest[acc] = map[string]struct{}{}
			}
			latest[acc][cat] = struct{}{}
		}

		for acc, cats := range latest {
			accounts[acc] = cats
		}
	}

	counts := &Counts{Online: len(accounts)}

	if cid, ok := categoryID.Get(); ok {
		target := cid.String()
		self := accountID.String()
		n := 0
		for acc, cats := range accounts {
			if acc == self {
				continue
			}
			if _, ok := cats[target]; ok {
				n++
			}
		}
		counts.InCategory = opt.New(n)
	}

	return counts, nil
}

func (t *Tracker) bucket() int64 {
	return t.now().Unix() / int64(bucketSize/time.Second)
}

func bucketKey(b int64) string {
	return keyPrefix + strconv.FormatInt(b, 10)
}

func field(accountID account.AccountID, categoryID opt.Optional[category.CategoryID]) string {
	cat := opt.Map(categoryID, func(c category.CategoryID) string { return c.String() }).OrZero()
	if cat == "" {
		cat = noCategory
	}
	return accountID.String() + ":" + cat
}
