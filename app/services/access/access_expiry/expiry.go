package access_expiry

import (
	"context"
	"log/slog"
	"time"

	"github.com/Southclaws/opt"
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/resources/access/access_repo"
	"github.com/Southclaws/storyden/app/resources/account"
	"github.com/Southclaws/storyden/app/resources/account/notification"
	"github.com/Southclaws/storyden/app/services/notification/notify"
)

const (
	sweepInterval = time.Hour
	NoticePeriod  = 14 * 24 * time.Hour
)

func Build() fx.Option {
	return fx.Options(
		fx.Provide(New),
		fx.Invoke(func(*Expiry) {}),
	)
}

// Expiry reminds members shortly before a grant lapses so they can pay for
// the next term in time. Access itself ends at read time, not here.
type Expiry struct {
	logger   *slog.Logger
	repo     *access_repo.Repository
	notifier *notify.Notifier
}

func New(
	ctx context.Context,
	lc fx.Lifecycle,
	logger *slog.Logger,
	repo *access_repo.Repository,
	notifier *notify.Notifier,
) *Expiry {
	e := &Expiry{
		logger:   logger,
		repo:     repo,
		notifier: notifier,
	}

	jobCtx, cancel := context.WithCancel(ctx)

	lc.Append(fx.StartHook(func() error {
		go e.run(jobCtx)
		return nil
	}))

	lc.Append(fx.StopHook(func() error {
		cancel()
		return nil
	}))

	return e
}

func (e *Expiry) run(ctx context.Context) {
	e.Sweep(ctx, time.Now())

	ticker := time.NewTicker(sweepInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			e.Sweep(ctx, time.Now())
		}
	}
}

func (e *Expiry) Sweep(ctx context.Context, now time.Time) int {
	grants, err := e.repo.ListExpiringUnnotified(ctx, now, now.Add(NoticePeriod))
	if err != nil {
		e.logger.Error("failed to list expiring access grants", slog.String("error", err.Error()))
		return 0
	}

	notified := 0
	for _, g := range grants {
		if err := e.notifier.Send(ctx, g.AccountID, opt.NewEmpty[account.AccountID](), notification.EventAccessExpiring, nil); err != nil {
			e.logger.Error("failed to send access expiry notification", slog.String("error", err.Error()))
			continue
		}
		if err := e.repo.MarkExpiryNotified(ctx, g.ID, now); err != nil {
			e.logger.Error("failed to mark access expiry notified", slog.String("error", err.Error()))
			continue
		}
		notified++
	}

	return notified
}
