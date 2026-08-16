package semester

import (
	"context"
	"log/slog"
	"time"

	"github.com/rs/xid"
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/resources/account/account_querier"
	"github.com/Southclaws/storyden/app/resources/account/account_writer"
)

const (
	rolloverInterval = 24 * time.Hour
	rolloverBatch    = 200
)

// Build wires the rollover job. fx.Invoke forces construction so the job starts
// on boot rather than only when something happens to depend on it.
func Build() fx.Option {
	return fx.Options(
		fx.Provide(NewRollover),
		fx.Invoke(func(*Rollover) {}),
	)
}

// Rollover writes each member's advanced semester back to the database once a
// day. Read paths already project the current value, so this job exists to keep
// stored data honest, not to make the feature work — if it never runs, members
// still see the right semester.
type Rollover struct {
	logger  *slog.Logger
	querier *account_querier.Querier
	writer  *account_writer.Writer
}

func NewRollover(
	ctx context.Context,
	lc fx.Lifecycle,
	logger *slog.Logger,
	querier *account_querier.Querier,
	writer *account_writer.Writer,
) *Rollover {
	r := &Rollover{
		logger:  logger,
		querier: querier,
		writer:  writer,
	}

	jobCtx, cancel := context.WithCancel(ctx)

	lc.Append(fx.StartHook(func() error {
		go r.run(jobCtx)
		return nil
	}))

	lc.Append(fx.StopHook(func() error {
		cancel()
		return nil
	}))

	return r
}

func (r *Rollover) run(ctx context.Context) {
	r.Sweep(ctx, time.Now())

	ticker := time.NewTicker(rolloverInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.Sweep(ctx, time.Now())
		}
	}
}

// Sweep advances every account whose recorded term has passed. Returns how many
// accounts were updated.
func (r *Rollover) Sweep(ctx context.Context, now time.Time) int {
	var (
		cursor  xid.ID
		updated int
	)

	for {
		if ctx.Err() != nil {
			return updated
		}

		batch, err := r.querier.ListMetadataAfter(ctx, cursor, rolloverBatch)
		if err != nil {
			r.logger.ErrorContext(ctx, "semester rollover failed to read accounts", slog.String("error", err.Error()))
			return updated
		}

		if len(batch) == 0 {
			return updated
		}

		for _, record := range batch {
			cursor = xid.ID(record.ID)

			next, changed := Advance(record.Metadata, now)
			if !changed {
				continue
			}

			if _, err := r.writer.Update(ctx, record.ID, account_writer.SetMetadata(next)); err != nil {
				r.logger.ErrorContext(ctx, "semester rollover failed to update account",
					slog.String("account_id", record.ID.String()),
					slog.String("error", err.Error()),
				)
				continue
			}

			updated++
		}

		if len(batch) < rolloverBatch {
			return updated
		}
	}
}
