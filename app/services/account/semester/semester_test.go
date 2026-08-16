package semester

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func date(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 12, 0, 0, 0, time.UTC)
}

func TestTermFor(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name string
		in   time.Time
		want Term
	}{
		{"last day of winter", date(2026, time.March, 31), Term{2025, true}},
		{"first day of summer", date(2026, time.April, 1), Term{2026, false}},
		{"mid summer", date(2026, time.July, 15), Term{2026, false}},
		{"last day of summer", date(2026, time.September, 30), Term{2026, false}},
		{"first day of winter", date(2026, time.October, 1), Term{2026, true}},
		{"winter spills into next year", date(2027, time.January, 5), Term{2026, true}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, TermFor(tt.in))
		})
	}
}

func TestTermOrdering(t *testing.T) {
	t.Parallel()

	a := assert.New(t)

	ss2026 := Term{2026, false}
	ws2026 := Term{2026, true}
	ss2027 := Term{2027, false}

	a.Equal(1, ss2026.Elapsed(ws2026))
	a.Equal(1, ws2026.Elapsed(ss2027))
	a.Equal(2, ss2026.Elapsed(ss2027))
	a.Equal(-1, ws2026.Elapsed(ss2026))
}

func TestCurrent(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name       string
		stored     int
		storedTerm Term
		now        time.Time
		want       int
	}{
		{"same term does not advance", 3, Term{2026, false}, date(2026, time.September, 30), 3},
		{"summer to winter advances one", 3, Term{2026, false}, date(2026, time.October, 1), 4},
		{"winter to next summer advances one", 4, Term{2026, true}, date(2027, time.April, 1), 5},
		{"multi year gap", 1, Term{2024, false}, date(2026, time.April, 1), 5},
		{"caps at eleven", 10, Term{2020, false}, date(2026, time.October, 1), Max},
		{"already at cap", Max, Term{2026, false}, date(2030, time.April, 1), Max},
		{"clock running backwards never decrements", 5, Term{2026, true}, date(2020, time.April, 1), 5},
		{"below range is lifted to one", 0, Term{2026, false}, date(2026, time.April, 1), Min},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, Current(tt.stored, tt.storedTerm, tt.now))
		})
	}
}

func TestParseRoundTrip(t *testing.T) {
	t.Parallel()

	r := require.New(t)
	a := assert.New(t)

	for _, term := range []Term{{2026, false}, {2026, true}, {1999, true}} {
		parsed, err := Parse(term.String())
		r.NoError(err)
		a.Equal(term, parsed)
	}

	a.Equal("2026-SS", Term{2026, false}.String())
	a.Equal("2026-WS", Term{2026, true}.String())
}

func TestParseRejectsGarbage(t *testing.T) {
	t.Parallel()

	for _, raw := range []string{"", "2026", "2026-XX", "abcd-SS", "2026SS"} {
		_, err := Parse(raw)
		assert.Error(t, err, "expected %q to be rejected", raw)
	}
}
