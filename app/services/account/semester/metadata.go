package semester

import (
	"maps"
	"time"
)

// MetadataKey is the account metadata entry holding the academic record.
const MetadataKey = "academic"

const (
	fieldSemester  = "semester"
	fieldTerm      = "term"
	fieldUpdatedAt = "updated_at"
)

// Academic is the stored form: a semester plus the term it was recorded in.
// Without the term the number cannot be advanced, so the two travel together.
type Academic struct {
	Semester int
	Term     Term
}

// FromMetadata reads the academic record, reporting false when it is absent or
// unreadable. Malformed entries are ignored rather than rejected so a bad write
// from an old client cannot make a profile unloadable.
func FromMetadata(meta map[string]any) (Academic, bool) {
	raw, ok := meta[MetadataKey].(map[string]any)
	if !ok {
		return Academic{}, false
	}

	value, ok := readInt(raw[fieldSemester])
	if !ok {
		return Academic{}, false
	}

	rawTerm, ok := raw[fieldTerm].(string)
	if !ok {
		return Academic{}, false
	}

	term, err := Parse(rawTerm)
	if err != nil {
		return Academic{}, false
	}

	return Academic{Semester: value, Term: term}, true
}

// NormaliseWrite rewrites an incoming metadata patch so the semester is stamped
// with the term it was chosen in. Clients only send the number; anchoring it
// server-side is what makes the value self-advancing.
func NormaliseWrite(patch map[string]any, now time.Time) map[string]any {
	raw, ok := patch[MetadataKey].(map[string]any)
	if !ok {
		return patch
	}

	value, ok := readInt(raw[fieldSemester])
	if !ok {
		return patch
	}

	out := maps.Clone(patch)
	out[MetadataKey] = map[string]any{
		fieldSemester:  Clamp(value),
		fieldTerm:      TermFor(now).String(),
		fieldUpdatedAt: now.UTC().Format(time.RFC3339),
	}

	return out
}

// Project returns metadata with the academic semester advanced to the term
// containing now. Called on every read so the displayed value is correct even
// if the rollover job has not run.
func Project(meta map[string]any, now time.Time) map[string]any {
	academic, ok := FromMetadata(meta)
	if !ok {
		return meta
	}

	current := Current(academic.Semester, academic.Term, now)

	out := maps.Clone(meta)
	out[MetadataKey] = map[string]any{
		fieldSemester: current,
		fieldTerm:     TermFor(now).String(),
	}

	return out
}

// Advance returns the persisted form of an account whose semester has moved on,
// and whether anything actually changed.
func Advance(meta map[string]any, now time.Time) (map[string]any, bool) {
	academic, ok := FromMetadata(meta)
	if !ok {
		return meta, false
	}

	term := TermFor(now)
	if academic.Term == term {
		return meta, false
	}

	out := maps.Clone(meta)
	out[MetadataKey] = map[string]any{
		fieldSemester:  Current(academic.Semester, academic.Term, now),
		fieldTerm:      term.String(),
		fieldUpdatedAt: now.UTC().Format(time.RFC3339),
	}

	return out, true
}

// readInt copes with metadata arriving either straight from a Go caller or via
// JSON, where every number decodes as a float64.
func readInt(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int64:
		return int(n), true
	case float64:
		return int(n), true
	default:
		return 0, false
	}
}
