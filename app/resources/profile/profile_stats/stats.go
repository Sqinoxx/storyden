package profile_stats

import (
	"context"
	"time"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"

	"github.com/Southclaws/storyden/app/services/account/semester"
	"github.com/Southclaws/storyden/internal/ent"
	ent_account "github.com/Southclaws/storyden/internal/ent/account"
	ent_post "github.com/Southclaws/storyden/internal/ent/post"
)

const window = 30 * 24 * time.Hour

type SemesterCount struct {
	Semester int
	Count    int
}

type Stats struct {
	Total      int
	Joined30d  int
	Active30d  int
	BySemester []SemesterCount
}

type Querier struct {
	db *ent.Client
}

func New(db *ent.Client) *Querier {
	return &Querier{db: db}
}

func (q *Querier) Get(ctx context.Context) (*Stats, error) {
	now := time.Now().UTC()
	since := now.Add(-window)

	members := func() *ent.AccountQuery {
		return q.db.Account.Query().Where(
			ent_account.KindEQ(ent_account.KindHuman),
			ent_account.DeletedAtIsNil(),
			ent_account.Not(ent_account.HandleHasPrefix("deleted-")),
		)
	}

	total, err := members().Count(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	joined, err := members().Where(ent_account.CreatedAtGTE(since)).Count(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	active, err := members().
		Where(ent_account.HasPostsWith(ent_post.CreatedAtGTE(since))).
		Count(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	accounts, err := members().Select(ent_account.FieldMetadata).All(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	counts := make(map[int]int)
	for _, a := range accounts {
		counts[currentSemester(a.Metadata, now)]++
	}

	return &Stats{
		Total:      total,
		Joined30d:  joined,
		Active30d:  active,
		BySemester: semesterCounts(counts),
	}, nil
}

func semesterCounts(counts map[int]int) []SemesterCount {
	points := make([]SemesterCount, 0, semester.Max+3)
	points = append(points, SemesterCount{Semester: 0, Count: counts[0]})
	for s := semester.Min; s <= semester.Max; s++ {
		points = append(points, SemesterCount{Semester: s, Count: counts[s]})
	}
	points = append(points, SemesterCount{Semester: semester.Finished, Count: counts[semester.Finished]})

	return points
}

func currentSemester(metadata map[string]any, now time.Time) int {
	academic, ok := semester.FromMetadata(metadata)
	if !ok {
		return 0
	}

	return semester.Current(academic.Semester, academic.Term, now)
}
