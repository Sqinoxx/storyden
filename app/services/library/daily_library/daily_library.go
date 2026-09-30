package daily_library

import (
	"context"
	"fmt"
	"html"
	"log/slog"
	"strings"
	"time"
	_ "time/tzdata"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/resources/account"
	"github.com/Southclaws/storyden/app/resources/datagraph"
	"github.com/Southclaws/storyden/app/resources/library"
	"github.com/Southclaws/storyden/app/resources/library/node_cache"
	"github.com/Southclaws/storyden/app/resources/library/node_writer"
	"github.com/Southclaws/storyden/app/resources/mark"
	"github.com/Southclaws/storyden/app/resources/visibility"
	"github.com/Southclaws/storyden/internal/config"
	"github.com/Southclaws/storyden/internal/ent"
	ent_account "github.com/Southclaws/storyden/internal/ent/account"
	ent_node "github.com/Southclaws/storyden/internal/ent/node"
	ent_post "github.com/Southclaws/storyden/internal/ent/post"
	"github.com/Southclaws/storyden/internal/ent/predicate"
)

const (
	ArchiveSlug = "tagesarchiv"
	ArchiveName = "Tagesarchiv"
	MetadataKey = "daily_library"

	dayLayout       = "2006-01-02"
	refreshInterval = 15 * time.Minute
	refreshDays     = 7
)

func DaySlug(day time.Time) string {
	return ArchiveSlug + "-" + day.Format(dayLayout)
}

func Build() fx.Option {
	return fx.Options(
		fx.Provide(New),
		fx.Invoke(func(*Generator) {}),
	)
}

type Generator struct {
	logger     *slog.Logger
	db         *ent.Client
	nodeWriter *node_writer.Writer
	cache      *node_cache.Cache
	loc        *time.Location
}

func New(
	ctx context.Context,
	lc fx.Lifecycle,
	cfg config.Config,
	logger *slog.Logger,
	db *ent.Client,
	nodeWriter *node_writer.Writer,
	cache *node_cache.Cache,
) *Generator {
	loc, err := time.LoadLocation(cfg.DailyLibraryTimezone)
	if err != nil {
		logger.Warn("daily library: invalid time zone, falling back to UTC",
			slog.String("timezone", cfg.DailyLibraryTimezone),
			slog.String("error", err.Error()),
		)
		loc = time.UTC
	}

	g := &Generator{
		logger:     logger,
		db:         db,
		nodeWriter: nodeWriter,
		cache:      cache,
		loc:        loc,
	}

	if !cfg.DailyLibraryEnabled {
		return g
	}

	jobCtx, cancel := context.WithCancel(ctx)

	lc.Append(fx.StartHook(func() error {
		go g.run(jobCtx)
		return nil
	}))

	lc.Append(fx.StopHook(func() error {
		cancel()
		return nil
	}))

	return g
}

func (g *Generator) run(ctx context.Context) {
	g.sweepAndLog(ctx)

	ticker := time.NewTicker(refreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			g.sweepAndLog(ctx)
		}
	}
}

func (g *Generator) sweepAndLog(ctx context.Context) {
	if err := g.Sweep(ctx, time.Now()); err != nil && ctx.Err() == nil {
		g.logger.ErrorContext(ctx, "daily library sweep failed", slog.String("error", err.Error()))
	}
}

func (g *Generator) Sweep(ctx context.Context, now time.Time) error {
	owner, ok, err := g.owner(ctx)
	if err != nil {
		return fault.Wrap(err, fctx.With(ctx))
	}
	if !ok {
		return nil
	}

	archive, err := g.ensureArchive(ctx, owner)
	if err != nil {
		return fault.Wrap(err, fctx.With(ctx))
	}

	today := g.Day(now)
	from := today.AddDate(0, 0, -(refreshDays - 1))

	backfilled := archive.Metadata[MetadataKey] != nil
	if !backfilled {
		first, err := g.db.Post.Query().
			Where(threadPredicates()...).
			Order(ent.Asc(ent_post.FieldCreatedAt)).
			First(ctx)
		if err != nil && !ent.IsNotFound(err) {
			return fault.Wrap(err, fctx.With(ctx))
		}
		if first != nil {
			if d := g.Day(first.CreatedAt); d.Before(from) {
				from = d
			}
		}
	}

	for day := from; !day.After(today); day = day.AddDate(0, 0, 1) {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		if _, err := g.GenerateDay(ctx, owner, library.NodeID(archive.ID), day); err != nil {
			return fault.Wrap(err, fctx.With(ctx))
		}
	}

	if !backfilled {
		meta := archive.Metadata
		if meta == nil {
			meta = map[string]any{}
		}
		meta[MetadataKey] = map[string]any{"backfilled_at": now.UTC().Format(time.RFC3339)}

		if err := g.db.Node.UpdateOneID(archive.ID).SetMetadata(meta).Exec(ctx); err != nil {
			return fault.Wrap(err, fctx.With(ctx))
		}
	}

	return nil
}

func (g *Generator) GenerateDay(ctx context.Context, owner account.AccountID, archiveID library.NodeID, day time.Time) (bool, error) {
	day = g.Day(day)
	slug := DaySlug(day)

	threads, err := g.db.Post.Query().
		Where(threadPredicates()...).
		Where(
			ent_post.CreatedAtGTE(day.UTC()),
			ent_post.CreatedAtLT(day.AddDate(0, 0, 1).UTC()),
		).
		WithAuthor().
		Order(ent.Asc(ent_post.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return false, fault.Wrap(err, fctx.With(ctx))
	}

	existing, err := g.db.Node.Query().Where(ent_node.Slug(slug)).Only(ctx)
	if err != nil && !ent.IsNotFound(err) {
		return false, fault.Wrap(err, fctx.With(ctx))
	}

	if existing == nil && len(threads) == 0 {
		return false, nil
	}

	if existing != nil && existing.Visibility != ent_node.VisibilityReview {
		return false, nil
	}

	content, err := datagraph.NewRichText(g.render(day, threads))
	if err != nil {
		return false, fault.Wrap(err, fctx.With(ctx))
	}

	meta := map[string]any{
		MetadataKey: map[string]any{
			"date":         day.Format(dayLayout),
			"thread_count": len(threads),
		},
	}

	opts := []node_writer.Option{
		node_writer.WithContent(content),
		node_writer.WithDescription(threadCountLabel(len(threads))),
		node_writer.WithMetadata(meta),
	}

	if existing == nil {
		opts = append(opts,
			node_writer.WithParent(archiveID),
			node_writer.WithVisibility(visibility.VisibilityReview),
		)

		if _, err := g.nodeWriter.Create(ctx, owner, day.Format("02.01.2006"), mark.NewSlugFromName(slug), opts...); err != nil {
			return false, fault.Wrap(err, fctx.With(ctx))
		}

		return true, nil
	}

	if existing.Content != nil && *existing.Content == content.HTML() {
		return false, nil
	}

	if _, err := g.nodeWriter.Update(ctx, library.NewID(existing.ID), opts...); err != nil {
		return false, fault.Wrap(err, fctx.With(ctx))
	}

	if err := g.cache.Invalidate(ctx, slug); err != nil {
		return false, fault.Wrap(err, fctx.With(ctx))
	}

	return true, nil
}

func (g *Generator) ensureArchive(ctx context.Context, owner account.AccountID) (*ent.Node, error) {
	archive, err := g.db.Node.Query().Where(ent_node.Slug(ArchiveSlug)).Only(ctx)
	if err == nil {
		return archive, nil
	}
	if !ent.IsNotFound(err) {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	content, err := datagraph.NewRichText("<p>Automatisch erzeugte Tagesübersichten aller Threads.</p>")
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	created, err := g.nodeWriter.Create(ctx, owner, ArchiveName, mark.NewSlugFromName(ArchiveSlug),
		node_writer.WithContent(content),
		node_writer.WithVisibility(visibility.VisibilityReview),
	)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return g.db.Node.Get(ctx, created.Mark.ID())
}

func (g *Generator) owner(ctx context.Context) (account.AccountID, bool, error) {
	admin, err := g.db.Account.Query().
		Where(
			ent_account.DeletedAtIsNil(),
			ent_account.Admin(true),
		).
		Order(ent.Asc(ent_account.FieldCreatedAt)).
		First(ctx)
	if err == nil {
		return account.AccountID(admin.ID), true, nil
	}
	if !ent.IsNotFound(err) {
		return account.AccountID{}, false, fault.Wrap(err, fctx.With(ctx))
	}

	return account.AccountID{}, false, nil
}

func (g *Generator) Day(t time.Time) time.Time {
	t = t.In(g.loc)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, g.loc)
}

func (g *Generator) render(day time.Time, threads []*ent.Post) string {
	var b strings.Builder

	fmt.Fprintf(&b, "<p>%s vom %s.</p>", threadCountLabel(len(threads)), html.EscapeString(germanDate(day)))

	if len(threads) == 0 {
		return b.String()
	}

	b.WriteString("<ul>")
	for _, t := range threads {
		link := "/t/" + t.Slug
		fmt.Fprintf(&b, `<li><p><a href="%s">%s</a>`, html.EscapeString(link), html.EscapeString(t.Title))
		if t.Edges.Author != nil {
			fmt.Fprintf(&b, " – %s", html.EscapeString(t.Edges.Author.Name))
		}
		fmt.Fprintf(&b, ", %s Uhr</p></li>", t.CreatedAt.In(g.loc).Format("15:04"))
	}
	b.WriteString("</ul>")

	return b.String()
}

func threadPredicates() []predicate.Post {
	return []predicate.Post{
		ent_post.RootPostIDIsNil(),
		ent_post.DeletedAtIsNil(),
		ent_post.VisibilityEQ(ent_post.VisibilityPublished),
	}
}

func threadCountLabel(n int) string {
	if n == 1 {
		return "1 Thread"
	}
	return fmt.Sprintf("%d Threads", n)
}

var (
	germanWeekdays = [...]string{"Sonntag", "Montag", "Dienstag", "Mittwoch", "Donnerstag", "Freitag", "Samstag"}
	germanMonths   = [...]string{"Januar", "Februar", "März", "April", "Mai", "Juni", "Juli", "August", "September", "Oktober", "November", "Dezember"}
)

func germanDate(day time.Time) string {
	return fmt.Sprintf("%s, %d. %s %d", germanWeekdays[day.Weekday()], day.Day(), germanMonths[day.Month()-1], day.Year())
}
