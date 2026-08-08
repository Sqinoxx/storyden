package main

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"entgo.io/ent/schema/field"
	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fmsg"
)

// timeLayouts are the formats a SQLite datetime column may hand back. Storyden's
// SQLite databases store "2026-07-23 13:00:24.3359653+02:00" so that layout is
// tried first, the rest are defensive.
var timeLayouts = []string{
	"2006-01-02 15:04:05.999999999-07:00",
	"2006-01-02 15:04:05.999999999Z07:00",
	time.RFC3339Nano,
	"2006-01-02 15:04:05.999999999",
	"2006-01-02 15:04:05",
	"2006-01-02",
}

func convertValue(t field.Type, v any) (any, error) {
	if v == nil {
		return nil, nil
	}

	switch t {
	case field.TypeBool:
		return toBool(v)

	case field.TypeTime:
		return toTime(v)

	case field.TypeJSON:
		return toJSON(v)

	case field.TypeString, field.TypeEnum:
		return toString(v)

	default:
		if b, ok := v.([]byte); ok {
			return string(b), nil
		}
		return v, nil
	}
}

func toBool(v any) (any, error) {
	switch x := v.(type) {
	case bool:
		return x, nil

	case int64:
		return x != 0, nil

	case float64:
		return x != 0, nil

	case []byte:
		return toBool(string(x))

	case string:
		b, err := strconv.ParseBool(strings.TrimSpace(x))
		if err != nil {
			return nil, fault.Wrap(err, fmsg.With("failed to parse boolean"))
		}
		return b, nil

	default:
		return nil, fault.Newf("cannot convert %T to boolean", v)
	}
}

func toTime(v any) (any, error) {
	switch x := v.(type) {
	case time.Time:
		return x, nil

	case []byte:
		return toTime(string(x))

	case string:
		s := strings.TrimSpace(x)
		if s == "" {
			return nil, nil
		}

		for _, layout := range timeLayouts {
			if t, err := time.Parse(layout, s); err == nil {
				return t, nil
			}
		}

		return nil, fault.Newf("cannot parse timestamp %q with any known layout", s)

	default:
		return nil, fault.Newf("cannot convert %T to timestamp", v)
	}
}

// toJSON validates the payload before it reaches a Postgres jsonb column. A
// silently mangled value here breaks things far downstream — roles.permissions
// is read back via sqljson.ValueContains and drives every permission check.
func toJSON(v any) (any, error) {
	var s string

	switch x := v.(type) {
	case []byte:
		s = string(x)
	case string:
		s = x
	default:
		return nil, fault.Newf("cannot convert %T to JSON", v)
	}

	if strings.TrimSpace(s) == "" {
		return nil, nil
	}

	if !json.Valid([]byte(s)) {
		return nil, fault.Newf("value is not valid JSON: %q", truncate(s, 128))
	}

	return s, nil
}

func toString(v any) (any, error) {
	switch x := v.(type) {
	case string:
		return x, nil

	case []byte:
		return string(x), nil

	case int64:
		return strconv.FormatInt(x, 10), nil

	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64), nil

	case bool:
		return strconv.FormatBool(x), nil

	case time.Time:
		return x.Format(time.RFC3339Nano), nil

	default:
		return nil, fault.Newf("cannot convert %T to string", v)
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
