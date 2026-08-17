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
	ent_category "github.com/Southclaws/storyden/internal/ent/category"
	ent_post "github.com/Southclaws/storyden/internal/ent/post"
	ent_session "github.com/Southclaws/storyden/internal/ent/session"
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

// HourOfDayPoint is login activity for one hour of the day (0-23, UTC).
type HourOfDayPoint struct {
	Hour  int
	Count int
}

// WeekdayPoint is login activity for one day of the week. Weekday runs 1
// (Monday) to 7 (Sunday).
type WeekdayPoint struct {
	Weekday int
	Count   int
}

// CategoryPoint is thread activity for one forum category.
type CategoryPoint struct {
	CategoryID  xid.ID
	Name        string
	ThreadCount int
}

type Totals struct {
	Accounts          int
	Threads           int
	Replies           int
	Categories        int
	ActiveAccounts7d  int
	ActiveAccounts30d int
	SessionsActive    int
	SessionsExpired   int
	SessionsRevoked   int
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
	LoginsDaily           []SeriesPoint
	LoginsMonthly         []SeriesPoint
	LoginsYearly          []SeriesPoint
	LoginsByHour          []HourOfDayPoint
	LoginsByWeekday       []WeekdayPoint
	ActiveAccountsDaily   []SeriesPoint
	ActiveAccountsMonthly []SeriesPoint
	ActiveAccountsYearly  []SeriesPoint
	AssetsDaily           []SeriesPoint
	AssetsMonthly         []SeriesPoint
	AssetsYearly          []SeriesPoint
	TopCategories         []CategoryPoint
}

const (
	dailyBuckets         = 30
	monthlyBuckets       = 12
	yearlyBuckets        = 5
	semesterBuckets      = 8
	topContributorsLimit = 10
	topCategoriesLimit   = 8

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

	sessionsActive, err := q.db.Session.Query().
		Where(ent_session.RevokedAtIsNil(), ent_session.ExpiresAtGTE(now)).
		Count(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	sessionsExpired, err := q.db.Session.Query().
		Where(ent_session.RevokedAtIsNil(), ent_session.ExpiresAtLT(now)).
		Count(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	sessionsRevoked, err := q.db.Session.Query().
		Where(ent_session.RevokedAtNotNil()).
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

	var sessionRows []createdAtRow
	err = q.db.Session.Query().
		Where(ent_session.CreatedAtGTE(historyStart)).
		Select(ent_session.FieldCreatedAt).
		Scan(ctx, &sessionRows)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	var activityRows []accountActivityRow
	err = q.db.Post.Query().
		Where(ent_post.CreatedAtGTE(historyStart)).
		Select(ent_post.FieldAccountPosts, ent_post.FieldCreatedAt).
		Scan(ctx, &activityRows)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	accountsDaily, accountsMonthly, accountsYearly := bucketByCalendar(rowTimes(accountRows), now)
	threadTimes := rowTimes(threadRows)
	threadsDaily, threadsMonthly, threadsYearly := bucketByCalendar(threadTimes, now)

	sessionTimes := rowTimes(sessionRows)
	loginsDaily, loginsMonthly, loginsYearly := bucketByCalendar(sessionTimes, now)
	loginsByHour := bucketByHour(sessionTimes)
	loginsByWeekday := bucketByWeekday(sessionTimes)

	activeAccountsDaily, activeAccountsMonthly, activeAccountsYearly := bucketUniqueByCalendar(activityRows, now)

	threadsByFachsemester, topContributors, err := q.getAuthorActivity(ctx, historyStart, now)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	assetsByFachsemester, assetsDaily, assetsMonthly, assetsYearly, err := q.getAssetActivity(ctx, historyStart, now)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	topCategories, err := q.getCategoryActivity(ctx, historyStart)
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
			SessionsActive:    sessionsActive,
			SessionsExpired:   sessionsExpired,
			SessionsRevoked:   sessionsRevoked,
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
		LoginsDaily:           loginsDaily,
		LoginsMonthly:         loginsMonthly,
		LoginsYearly:          loginsYearly,
		LoginsByHour:          loginsByHour,
		LoginsByWeekday:       loginsByWeekday,
		ActiveAccountsDaily:   activeAccountsDaily,
		ActiveAccountsMonthly: activeAccountsMonthly,
		ActiveAccountsYearly:  activeAccountsYearly,
		AssetsDaily:           assetsDaily,
		AssetsMonthly:         assetsMonthly,
		AssetsYearly:          assetsYearly,
		TopCategories:         topCategories,
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
// material. It also buckets the same rows into a daily/monthly/yearly upload
// series, since the rows are already loaded here.
func (q *Querier) getAssetActivity(ctx context.Context, historyStart, now time.Time) (fachsemester []FachsemesterPoint, daily, monthly, yearly []SeriesPoint, err error) {
	assets, err := q.db.Asset.Query().
		Where(ent_asset.CreatedAtGTE(historyStart)).
		Select(ent_asset.FieldCreatedAt, ent_asset.FieldAccountID).
		WithOwner(func(aq *ent.AccountQuery) {
			aq.Select(ent_account.FieldID, ent_account.FieldMetadata)
		}).
		All(ctx)
	if err != nil {
		return nil, nil, nil, nil, fault.Wrap(err, fctx.With(ctx))
	}

	fachsemesterCounts := make(map[int]int)
	assetTimes := make([]time.Time, len(assets))

	for i, a := range assets {
		assetTimes[i] = a.CreatedAt

		owner := a.Edges.Owner
		if owner == nil {
			continue
		}

		fachsemesterCounts[currentSemester(owner.Metadata, now)]++
	}

	daily, monthly, yearly = bucketByCalendar(assetTimes, now)

	return fachsemesterPointsFromCounts(fachsemesterCounts), daily, monthly, yearly, nil
}

// getCategoryActivity ranks forum categories by how many threads were
// created in them within the history window, so admins can see which
// categories draw the most engagement.
func (q *Querier) getCategoryActivity(ctx context.Context, historyStart time.Time) ([]CategoryPoint, error) {
	threads, err := q.db.Post.Query().
		Where(ent_post.RootPostIDIsNil(), ent_post.CreatedAtGTE(historyStart), ent_post.CategoryIDNotNil()).
		Select(ent_post.FieldCategoryID).
		WithCategory(func(cq *ent.CategoryQuery) {
			cq.Select(ent_category.FieldID, ent_category.FieldName)
		}).
		All(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	counts := make(map[xid.ID]*CategoryPoint)

	for _, t := range threads {
		cat := t.Edges.Category
		if cat == nil {
			continue
		}

		cp, ok := counts[cat.ID]
		if !ok {
			cp = &CategoryPoint{CategoryID: cat.ID, Name: cat.Name}
			counts[cat.ID] = cp
		}

		cp.ThreadCount++
	}

	points := make([]CategoryPoint, 0, len(counts))
	for _, cp := range counts {
		points = append(points, *cp)
	}
	sort.Slice(points, func(i, j int) bool {
		if points[i].ThreadCount != points[j].ThreadCount {
			return points[i].ThreadCount > points[j].ThreadCount
		}
		return points[i].Name < points[j].Name
	})
	if len(points) > topCategoriesLimit {
		points = points[:topCategoriesLimit]
	}

	return points, nil
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

type accountActivityRow struct {
	AccountPosts xid.ID    `json:"account_posts"`
	CreatedAt    time.Time `json:"created_at"`
}

// bucketUniqueByCalendar buckets (account, timestamp) rows into zero-filled
// daily, monthly and yearly series ending on the bucket containing now,
// counting each distinct account once per bucket rather than once per row.
func bucketUniqueByCalendar(rows []accountActivityRow, now time.Time) (daily, monthly, yearly []SeriesPoint) {
	daySets := make(map[time.Time]map[xid.ID]bool)
	monthSets := make(map[time.Time]map[xid.ID]bool)
	yearSets := make(map[time.Time]map[xid.ID]bool)

	addUnique := func(sets map[time.Time]map[xid.ID]bool, bucket time.Time, id xid.ID) {
		set, ok := sets[bucket]
		if !ok {
			set = make(map[xid.ID]bool)
			sets[bucket] = set
		}
		set[id] = true
	}

	for _, r := range rows {
		t := r.CreatedAt.UTC()
		addUnique(daySets, time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC), r.AccountPosts)
		addUnique(monthSets, time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC), r.AccountPosts)
		addUnique(yearSets, time.Date(t.Year(), time.January, 1, 0, 0, 0, 0, time.UTC), r.AccountPosts)
	}

	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	for i := dailyBuckets - 1; i >= 0; i-- {
		day := today.AddDate(0, 0, -i)
		daily = append(daily, SeriesPoint{Date: day, Count: len(daySets[day])})
	}

	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	for i := monthlyBuckets - 1; i >= 0; i-- {
		month := monthStart.AddDate(0, -i, 0)
		monthly = append(monthly, SeriesPoint{Date: month, Count: len(monthSets[month])})
	}

	yearStart := time.Date(now.Year(), time.January, 1, 0, 0, 0, 0, time.UTC)
	for i := yearlyBuckets - 1; i >= 0; i-- {
		year := yearStart.AddDate(-i, 0, 0)
		yearly = append(yearly, SeriesPoint{Date: year, Count: len(yearSets[year])})
	}

	return daily, monthly, yearly
}

// bucketByHour groups timestamps by hour of day (0-23, UTC), zero-filled.
func bucketByHour(times []time.Time) []HourOfDayPoint {
	counts := make(map[int]int)
	for _, t := range times {
		counts[t.UTC().Hour()]++
	}

	points := make([]HourOfDayPoint, 24)
	for h := 0; h < 24; h++ {
		points[h] = HourOfDayPoint{Hour: h, Count: counts[h]}
	}

	return points
}

// weekdayOrder lists weekdays Monday-first, so the returned series reads
// naturally for a German audience where the week starts on Monday.
var weekdayOrder = [7]time.Weekday{
	time.Monday, time.Tuesday, time.Wednesday, time.Thursday,
	time.Friday, time.Saturday, time.Sunday,
}

// bucketByWeekday groups timestamps by weekday (1=Monday..7=Sunday, UTC),
// zero-filled.
func bucketByWeekday(times []time.Time) []WeekdayPoint {
	counts := make(map[time.Weekday]int)
	for _, t := range times {
		counts[t.UTC().Weekday()]++
	}

	points := make([]WeekdayPoint, 7)
	for i, wd := range weekdayOrder {
		points[i] = WeekdayPoint{Weekday: i + 1, Count: counts[wd]}
	}

	return points
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
