// Package semester models German academic terms so that a member's recorded
// study semester advances on its own.
//
// A stored semester is only meaningful alongside the term it was recorded in.
// The displayed value is derived from that pair on every read, which means it
// stays correct even if the background rollover never runs.
package semester

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/ftag"
)

const (
	// Min and Max bound a dentistry degree's semester count.
	Min = 1
	Max = 11

	// Finished is returned by Current, and stored in metadata in place of a
	// semester number, once a member has progressed beyond Max ("Fertig").
	// It is a stable end state: Current short-circuits on it instead of
	// projecting further, so a finished member can never decay back into a
	// real semester number as terms keep passing.
	Finished = -1
)

// Term identifies one academic term. Year is the year the term begins in, so
// the winter term spanning 2026/27 is Term{2026, true}.
type Term struct {
	Year   int
	Winter bool
}

// TermFor returns the term containing t. Summer runs April to September,
// winter runs October to March of the following year.
func TermFor(t time.Time) Term {
	year, month := t.Year(), t.Month()

	switch {
	case month >= time.October:
		return Term{Year: year, Winter: true}
	case month <= time.March:
		return Term{Year: year - 1, Winter: true}
	default:
		return Term{Year: year, Winter: false}
	}
}

// ordinal maps a term onto a monotonically increasing number so that the
// distance between two terms is a plain subtraction.
func (t Term) ordinal() int {
	if t.Winter {
		return t.Year*2 + 1
	}

	return t.Year * 2
}

// Elapsed reports how many term boundaries lie between t and other. Negative
// when other precedes t.
func (t Term) Elapsed(other Term) int {
	return other.ordinal() - t.ordinal()
}

func (t Term) String() string {
	if t.Winter {
		return fmt.Sprintf("%d-WS", t.Year)
	}

	return fmt.Sprintf("%d-SS", t.Year)
}

func Parse(raw string) (Term, error) {
	year, season, ok := strings.Cut(raw, "-")
	if !ok {
		return Term{}, fault.New("malformed term", ftag.With(ftag.InvalidArgument))
	}

	y, err := strconv.Atoi(year)
	if err != nil {
		return Term{}, fault.Wrap(err, ftag.With(ftag.InvalidArgument))
	}

	switch strings.ToUpper(season) {
	case "WS":
		return Term{Year: y, Winter: true}, nil
	case "SS":
		return Term{Year: y, Winter: false}, nil
	default:
		return Term{}, fault.New("unknown season", ftag.With(ftag.InvalidArgument))
	}
}

// Current advances a semester recorded in storedTerm to the term containing
// now, clamped to the degree's range, or reports Finished once that would
// carry the member past Max. Terms that have not yet elapsed and clocks that
// run backwards never decrement the value.
func Current(stored int, storedTerm Term, now time.Time) int {
	if stored == Finished {
		return Finished
	}

	projected := stored + max(0, storedTerm.Elapsed(TermFor(now)))
	if projected > Max {
		return Finished
	}

	return Clamp(projected)
}

func Clamp(semester int) int {
	if semester < Min {
		return Min
	}
	if semester > Max {
		return Max
	}

	return semester
}
