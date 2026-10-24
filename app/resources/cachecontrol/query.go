package cachecontrol

import (
	"strings"
	"time"

	"github.com/Southclaws/opt"
)

type ETag struct {
	Value string
	Time  time.Time
	Scope string
}

func NewETag(t time.Time) *ETag {
	return &ETag{
		Value: "t-" + t.UTC().Format(time.RFC3339Nano),
		Time:  t,
	}
}

// NewScopedETag binds the ETag to an identity, for resources whose content
// depends on who is asking rather than only on the URL.
func NewScopedETag(scope string, t time.Time) *ETag {
	e := NewETag(t)
	e.Value = scope + scopeSeparator + e.Value
	e.Scope = scope
	return e
}

const scopeSeparator = "|"

func ParseETag(t string) ETag {
	// Strip quotes if present (ETags are quoted in HTTP headers)
	if len(t) >= 2 && t[0] == '"' && t[len(t)-1] == '"' {
		t = t[1 : len(t)-1]
	}

	scope, ts, scoped := strings.Cut(t, scopeSeparator)
	if !scoped {
		scope, ts = "", t
	}

	// expected format: [<scope>|]t-<time in RFC3339Nano>
	if len(ts) < 3 || ts[:2] != "t-" {
		return ETag{}
	}

	parsedTime, err := time.Parse(time.RFC3339Nano, ts[2:])
	if err != nil {
		return ETag{}
	}

	return ETag{
		Value: t,
		Time:  parsedTime,
		Scope: scope,
	}
}

func (t ETag) String() string {
	// Return quoted ETag for HTTP header
	return `"` + t.Value + `"`
}

// Query represents a HTTP conditional request query.
type Query struct {
	ETag          opt.Optional[ETag]
	ModifiedSince opt.Optional[time.Time]
	scope         string
}

// NewQuery must be constructed from a HTTP request's conditional headers.
func NewQuery(ifNoneMatch opt.Optional[string], ifModifiedSince opt.Optional[time.Time]) Query {
	return Query{
		ETag:          opt.Map(ifNoneMatch, ParseETag),
		ModifiedSince: ifModifiedSince,
	}
}

// Scoped restricts the query to validators issued for the given scope. Since
// If-Modified-Since carries no identity, it is discarded, as is an ETag issued
// for any other scope, such as a different account in the same browser.
func (q Query) Scoped(scope string) Query {
	etag, ok := q.ETag.Get()
	if !ok || etag.Scope != scope {
		return Query{scope: scope}
	}

	return Query{ETag: q.ETag, scope: scope}
}

func (q Query) newETag(t time.Time) *ETag {
	if q.scope != "" {
		return NewScopedETag(q.scope, t)
	}
	return NewETag(t)
}

// NotModified takes the current updated date of a resource and returns true if
// the cache control query includes a Is-Modified-Since header and the resource
// updated date is not after the header value. True means a 304 response header.
func (q Query) Check(fn func() *time.Time) (*ETag, bool) {
	resourceUpdated := fn()
	if resourceUpdated == nil {
		return nil, false
	}

	if etag, ok := q.ETag.Get(); ok {
		modified := resourceUpdated.Compare(etag.Time)

		return q.newETag(*resourceUpdated), modified <= 0
	}

	if ms, ok := q.ModifiedSince.Get(); ok {

		// truncate the resourceUpdated to the nearest second because the actual
		// HTTP header is already truncated but the DB time is in nanoseconds.
		// If we didn't do this the resource time will always be slightly ahead.
		truncated := resourceUpdated.Truncate(time.Second)

		// If the resource update time is ahead of the HTTP Last-Modified check,
		// modified = 1, meaning the resource has been modified since the last
		// request and should be returned from the DB, instead of a 304 status.
		modified := truncated.Compare(ms)

		return q.newETag(*resourceUpdated), modified <= 0
	}

	return q.newETag(*resourceUpdated), false
}
