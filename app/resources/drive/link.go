package drive

import (
	"net/url"
	"regexp"
	"strings"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fmsg"
	"github.com/Southclaws/fault/ftag"
)

// driveIDPattern matches the opaque identifiers Drive puts in share links. The
// lower bound rejects fragments of a URL that got through the parser by
// accident, such as "u" or "0" from a /drive/u/0/ prefix.
var driveIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{10,}$`)

var errInvalidLink = fmsg.WithDesc("parse drive folder link",
	"That does not look like a Google Drive folder link. Copy the address of the folder from your browser, or use the folder's ID.")

// ParseFolderID accepts a Google Drive folder share link in any of the shapes
// Google hands out, or a bare folder ID, and returns the ID.
func ParseFolderID(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fault.New("empty drive folder link", ftag.With(ftag.InvalidArgument), errInvalidLink)
	}

	if driveIDPattern.MatchString(trimmed) && !strings.Contains(trimmed, "/") {
		return trimmed, nil
	}

	u, err := url.Parse(trimmed)
	if err != nil {
		return "", fault.Wrap(err, ftag.With(ftag.InvalidArgument), errInvalidLink)
	}

	if id := u.Query().Get("id"); driveIDPattern.MatchString(id) {
		return id, nil
	}

	segments := strings.Split(strings.Trim(u.Path, "/"), "/")
	for i, segment := range segments {
		if segment != "folders" {
			continue
		}

		if i+1 < len(segments) && driveIDPattern.MatchString(segments[i+1]) {
			return segments[i+1], nil
		}
	}

	// A trailing segment that looks like an ID covers link shapes not listed
	// above, which Google has changed more than once.
	if last := segments[len(segments)-1]; driveIDPattern.MatchString(last) {
		return last, nil
	}

	return "", fault.New("no folder id in link", ftag.With(ftag.InvalidArgument), errInvalidLink)
}
