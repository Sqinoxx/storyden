package statistics_querier

import (
	"testing"
	"time"

	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
)

func TestWindowTrend(t *testing.T) {
	now := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	window := 30 * 24 * time.Hour

	times := []time.Time{
		now,
		now.Add(-time.Hour),
		now.Add(-window),
		now.Add(-window - time.Second),
		now.Add(-2 * window),
		now.Add(-2*window - time.Second),
		now.Add(time.Hour),
	}

	assert.Equal(t, TrendPoint{Current: 3, Previous: 2}, windowTrend(times, now, window))
}

func TestUniqueWindowTrend(t *testing.T) {
	now := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	window := 30 * 24 * time.Hour
	a, b := xid.New(), xid.New()

	rows := []accountActivityRow{
		{AccountPosts: a, CreatedAt: now.Add(-time.Hour)},
		{AccountPosts: a, CreatedAt: now.Add(-2 * time.Hour)},
		{AccountPosts: b, CreatedAt: now.Add(-3 * time.Hour)},
		{AccountPosts: a, CreatedAt: now.Add(-window - time.Hour)},
	}

	assert.Equal(t, TrendPoint{Current: 2, Previous: 1}, uniqueWindowTrend(rows, now, window))
}

func TestBucketByHourUsesCommunityTimeZone(t *testing.T) {
	summer := time.Date(2026, 7, 1, 23, 30, 0, 0, time.UTC)
	winter := time.Date(2026, 1, 1, 23, 30, 0, 0, time.UTC)

	points := bucketByHour([]time.Time{summer, winter}, communityLocation)

	assert.Len(t, points, 24)
	assert.Equal(t, 1, points[1].Count, "CEST is UTC+2")
	assert.Equal(t, 1, points[0].Count, "CET is UTC+1")
	assert.Equal(t, 0, points[23].Count)
}

func TestBucketByWeekdayUsesCommunityTimeZone(t *testing.T) {
	sundayNightUTC := time.Date(2026, 10, 4, 23, 30, 0, 0, time.UTC)

	points := bucketByWeekday([]time.Time{sundayNightUTC}, communityLocation)

	assert.Equal(t, 1, points[0].Count, "Sunday 23:30 UTC is Monday in Berlin")
	assert.Equal(t, 0, points[6].Count)
}

func TestTopEmojis(t *testing.T) {
	rows := []reactRow{{Emoji: "🦷"}, {Emoji: "🦷"}, {Emoji: "👍"}}

	assert.Equal(t, []EmojiPoint{{Emoji: "🦷", Count: 2}, {Emoji: "👍", Count: 1}}, topEmojis(rows))
}
