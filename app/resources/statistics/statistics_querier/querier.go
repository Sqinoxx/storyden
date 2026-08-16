package statistics_querier

import (
	"context"
	"sort"
	"time"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/rs/xid"

	"github.com/Southclaws/storyden/app/services/account/semester"
	"github.com/Southclaws/storyden/internal/ent"
	ent_account "github.com/Southclaws/storyden/internal/ent/account"
	ent_asset "github.com/Southclaws/storyden/internal/ent/asset"
	ent_post "github.com/Southclaws/storyden/internal/ent/post"
)

type Querier struct {
	db *ent.Client
}

func New(db *ent.Client) *Querier {
	return &Querier{db}
}

type SeriesPoint struct {
	Date  time.Time
	Count int
}

type SemesterPoint struct {
	Term  semester.Term
	Count int
}

// FachsemesterPoint is activity for one study-semester cohort. Semester 0
// means the members have no recorded study semester; semester.Finished means
// they've progressed beyond the degree's final semester ("Fertig").
type FachsemesterPoint struct {
	Semester int
	Count    int
}

type Contributor struct {
	AccountID    xid.ID
	Handle       string
	Name         string
	Semester     int
	ThreadCount  int
	LastThreadAt time.Time
}

type Totals struct {
	Accounts          int
	Threads           int
	Replies           int
	Categories        int
	ActiveAccounts7d  int
	ActiveAccounts30d int
}

type Statistics struct {
	Totals                Totals
	AccountsDaily         []SeriesPoint
	AccountsMonthly       []SeriesPoint
	AccountsYearly        []SeriesPoint
	ThreadsDaily          []SeriesPoint
	ThreadsMonthly        []SeriesPoint
	ThreadsYearly         []SeriesPoint
	ThreadsBySemester     []SemesterPoint
	ThreadsByFachsemester []FachsemesterPoint
	AssetsByFachsemester  []FachsemesterPoint
	TopContributors       []Contributor
}

const (
	dailyBuckets         = 30
	monthlyBuckets       = 12
	yearlyBuckets        = 5
	semesterBuckets      = 8
	topContributorsLimit = 10

	// historyWindow bounds every timestamp query to the widest granularity
	// (yearly), so a single indexed range scan per entity supplies the daily,
	// monthly and yearly series without re-querying the table three times.
	historyWindow = yearlyBuckets * 366 * 24 * time.Hour
)

// Get computes usage statistics for the admin dashboard. Totals are plain
// indexed counts; time series are built from a single bounded, single-column
// selection of created_at timestamps per entity, bucketed in Go, so growth
// of the underlying tables only grows the query result by one time.Time per
// row within the history window rather than by query count.
func (q *Querier) Get(ctx context.Context) (*Statistics, error) {
	now := time.Now().UTC()
	historyStart := now.Add(-historyWindow)

	accountsTotal, err := q.db.Account.Query().Count(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	threadsTotal, err := q.db.Post.Query().Where(ent_post.RootPostIDIsNil()).Count(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	repliesTotal, err := q.db.Post.Query().Where(ent_post.RootPostIDNotNil()).Count(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	categoriesTotal, err := q.db.Category.Query().Count(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	active7, err := q.db.Account.Query().
		Where(ent_account.HasPostsWith(ent_post.CreatedAtGTE(now.Add(-7 * 24 * time.Hour)))).
		Count(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	active30, err := q.db.Account.Query().
		Where(ent_account.HasPostsWith(ent_post.CreatedAtGTE(now.Add(-30 * 24 * time.Hour)))).
		Count(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	var accountRows []createdAtRow
	err = q.db.Account.Query().
		Where(ent_account.CreatedAtGTE(historyStart)).
		Select(ent_account.FieldCreatedAt).
		Scan(ctx, &accountRows)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	var threadRows []createdAtRow
	err = q.db.Post.Query().
		Where(ent_post.RootPostIDIsNil(), ent_post.CreatedAtGTE(historyStart)).
		Select(ent_post.FieldCreatedAt).
		Scan(ctx, &threadRows)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	accountsDaily, accountsMonthly, accountsYearly := bucketByCalendar(rowTimes(accountRows), now)
	threadTimes := rowTimes(threadRows)
	threadsDaily, threadsMonthly, threadsYearly := bucketByCalendar(threadTimes, now)

	threadsByFachsemester, topContributors, err := q.getAuthorActivity(ctx, historyStart, now)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	assetsByFachsemester, err := q.getAssetActivity(ctx, historyStart, now)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return &Statistics{
		Totals: Totals{
			Accounts:          accountsTotal,
			Threads:           threadsTotal,
			Replies:           repliesTotal,
			Categories:        categoriesTotal,
			ActiveAccounts7d:  active7,
			ActiveAccounts30d: active30,
		},
		AccountsDaily:         accountsDaily,
		AccountsMonthly:       accountsMonthly,
		AccountsYearly:        accountsYearly,
		ThreadsDaily:          threadsDaily,
		ThreadsMonthly:        threadsMonthly,
		ThreadsYearly:         threadsYearly,
		ThreadsBySemester:     bucketBySemester(threadTimes, now),
		ThreadsByFachsemester: threadsByFachsemester,
		AssetsByFachsemester:  assetsByFachsemester,
		TopContributors:       topContributors,
	}, nil
}

// getAuthorActivity reads each thread's author (handle, name, recorded study
// semester) in a single eager-loaded query, restricted to the columns
// actually used, and aggregates it in Go into a per-cohort breakdown and a
// leaderboard of the most active members. This tells admins which semester
// cohorts are carrying the most (or least) of the discussion, and who
// specifically is behind those numbers.
func (q *Querier) getAuthorActivity(ctx context.Context, historyStart, now time.Time) ([]FachsemesterPoint, []Contributor, error) {
	threads, err := q.db.Post.Query().
		Where(ent_post.RootPostIDIsNil(), ent_post.CreatedAtGTE(historyStart)).
		Select(ent_post.FieldCreatedAt, ent_post.FieldAccountPosts).
		WithAuthor(func(aq *ent.AccountQuery) {
			aq.Select(
				ent_account.FieldID,
				ent_account.FieldHandle,
				ent_account.FieldName,
				ent_account.FieldMetadata,
			)
		}).
		All(ctx)
	if err != nil {
		return nil, nil, fault.Wrap(err, fctx.With(ctx))
	}

	fachsemesterCounts := make(map[int]int)
	contributors := make(map[xid.ID]*Contributor)

	for _, t := range threads {
		author := t.Edges.Author
		if author == nil {
			continue
		}

		sem := currentSemester(author.Metadata, now)
		fachsemesterCounts[sem]++

		c, ok := contributors[author.ID]
		if !ok {
			c = &Contributor{
				AccountID: author.ID,
				Handle:    author.Handle,
				Name:      author.Name,
				Semester:  sem,
			}
			contributors[author.ID] = c
		}

		c.ThreadCount++
		if t.CreatedAt.After(c.LastThreadAt) {
			c.LastThreadAt = t.CreatedAt
		}
	}

	fachsemesterPoints := fachsemesterPointsFromCounts(fachsemesterCounts)

	topContributors := make([]Contributor, 0, len(contributors))
	for _, c := range contributors {
		topContributors = append(topContributors, *c)
	}
	sort.Slice(topContributors, func(i, j int) bool {
		if topContributors[i].ThreadCount != topContributors[j].ThreadCount {
			return topContributors[i].ThreadCount > topContributors[j].ThreadCount
		}
		return topContributors[i].LastThreadAt.After(topContributors[j].LastThreadAt)
	})
	if len(topContributors) > topContributorsLimit {
		topContributors = topContributors[:topContributorsLimit]
	}

	return fachsemesterPoints, topContributors, nil
}

// getAssetActivity mirrors getAuthorActivity but for uploaded files, so
// admins can see which semester cohorts are contributing the most (or least)
// material.
func (q *Querier) getAssetActivity(ctx context.Context, historyStart, now time.Time) ([]FachsemesterPoint, error) {
	assets, err := q.db.Asset.Query().
		Where(ent_asset.CreatedAtGTE(historyStart)).
		Select(ent_asset.FieldCreatedAt, ent_asset.FieldAccountID).
		WithOwner(func(aq *ent.AccountQuery) {
			aq.Select(ent_account.FieldID, ent_account.FieldMetadata)
		}).
		All(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	fachsemesterCounts := make(map[int]int)

	for _, a := range assets {
		owner := a.Edges.Owner
		if owner == nil {
			continue
		}

		fachsemesterCounts[currentSemester(owner.Metadata, now)]++
	}

	return fachsemesterPointsFromCounts(fachsemesterCounts), nil
}

// fachsemesterPointsFromCounts zero-fills a per-semester count map into the
// standard bucket order: unknown (0), each real semester (Min..Max), then
// finished (semester.Finished).
func fachsemesterPointsFromCounts(counts map[int]int) []FachsemesterPoint {
	points := make([]FachsemesterPoint, 0, semester.Max+3)
	points = append(points, FachsemesterPoint{Semester: 0, Count: counts[0]})
	for s := semester.Min; s <= semester.Max; s++ {
		points = append(points, FachsemesterPoint{Semester: s, Count: counts[s]})
	}
	points = append(points, FachsemesterPoint{Semester: semester.Finished, Count: counts[semester.Finished]})

	return points
}

// currentSemester projects an account's recorded semester forward to now, the
// same way account reads do, so cohorts reflect where members stand today
// rather than where they stood when metadata was last written. Zero means no
// semester is recorded.
func currentSemester(metadata map[string]any, now time.Time) int {
	academic, ok := semester.FromMetadata(metadata)
	if !ok {
		return 0
	}

	return semester.Current(academic.Semester, academic.Term, now)
}

type createdAtRow struct {
	CreatedAt time.Time `json:"created_at"`
}

func rowTimes(rows []createdAtRow) []time.Time {
	times := make([]time.Time, len(rows))
	for i, r := range rows {
		times[i] = r.CreatedAt
	}
	return times
}

// bucketByCalendar buckets timestamps into zero-filled daily, monthly and
// yearly series ending on the bucket containing now.
func bucketByCalendar(times []time.Time, now time.Time) (daily, monthly, yearly []SeriesPoint) {
	dayCounts := make(map[time.Time]int)
	monthCounts := make(map[time.Time]int)
	yearCounts := make(map[time.Time]int)

	for _, t := range times {
		t = t.UTC()
		dayCounts[time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)]++
		monthCounts[time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)]++
		yearCounts[time.Date(t.Year(), time.January, 1, 0, 0, 0, 0, time.UTC)]++
	}

	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	for i := dailyBuckets - 1; i >= 0; i-- {
		day := today.AddDate(0, 0, -i)
		daily = append(daily, SeriesPoint{Date: day, Count: dayCounts[day]})
	}

	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	for i := monthlyBuckets - 1; i >= 0; i-- {
		month := monthStart.AddDate(0, -i, 0)
		monthly = append(monthly, SeriesPoint{Date: month, Count: monthCounts[month]})
	}

	yearStart := time.Date(now.Year(), time.January, 1, 0, 0, 0, 0, time.UTC)
	for i := yearlyBuckets - 1; i >= 0; i-- {
		year := yearStart.AddDate(-i, 0, 0)
		yearly = append(yearly, SeriesPoint{Date: year, Count: yearCounts[year]})
	}

	return daily, monthly, yearly
}

// bucketBySemester buckets timestamps into zero-filled German academic terms
// ending on the term containing now.
func bucketBySemester(times []time.Time, now time.Time) []SemesterPoint {
	counts := make(map[semester.Term]int)
	for _, t := range times {
		counts[semester.TermFor(t.UTC())]++
	}

	terms := make([]semester.Term, semesterBuckets)
	term := semester.TermFor(now)
	for i := semesterBuckets - 1; i >= 0; i-- {
		terms[i] = term
		term = prevTerm(term)
	}

	points := make([]SemesterPoint, 0, semesterBuckets)
	for _, t := range terms {
		points = append(points, SemesterPoint{Term: t, Count: counts[t]})
	}

	return points
}

func prevTerm(t semester.Term) semester.Term {
	if t.Winter {
		return semester.Term{Year: t.Year, Winter: false}
	}

	return semester.Term{Year: t.Year - 1, Winter: true}
}
