package bindings

import (
	"mime"
	"path"
	"regexp"
	"strings"
	"unicode"

	"github.com/rs/xid"

	"github.com/Southclaws/storyden/app/resources/asset"
)

const maxDownloadNameRunes = 200

// Uploads are stored as a slug, so "Report.pdf" becomes "<xid>-report-pdf".
// The trailing segment is recovered as the extension the same way the web
// client does it, which keeps both download paths agreeing on the name.
var slugExtensionPattern = regexp.MustCompile(`^[a-z0-9]{1,6}$`)

func contentDisposition(a *asset.Asset, requested *string) string {
	disposition := "attachment"
	if inlineRenderableMIMEs[strings.ToLower(a.MIME.String())] {
		disposition = "inline"
	}

	formatted := mime.FormatMediaType(disposition, map[string]string{
		"filename": assetDownloadName(a, requested),
	})
	if formatted == "" {
		return disposition
	}

	return formatted
}

func assetDownloadName(a *asset.Asset, requested *string) string {
	base, ext := splitStoredAssetName(a)

	if requested != nil {
		if name := sanitiseDownloadName(*requested); name != "" {
			base = name
			if e := path.Ext(name); slugExtensionPattern.MatchString(strings.ToLower(strings.TrimPrefix(e, "."))) {
				base = strings.TrimSuffix(name, e)
			}
		}
	}

	base = strings.Trim(base, ". ")
	if base == "" {
		base = "file"
	}

	return base + ext
}

func splitStoredAssetName(a *asset.Asset) (string, string) {
	slug := a.Name.String()
	if id, rest, ok := strings.Cut(slug, "-"); ok {
		if _, err := xid.FromString(id); err == nil {
			slug = rest
		}
	} else if _, err := xid.FromString(slug); err == nil {
		slug = ""
	}

	ext := a.MIME.Extension()
	if i := strings.LastIndex(slug, "-"); i > 0 && slugExtensionPattern.MatchString(slug[i+1:]) {
		ext = "." + slug[i+1:]
		slug = slug[:i]
	}

	return strings.ReplaceAll(slug, "-", " "), ext
}

func sanitiseDownloadName(in string) string {
	cleaned := strings.Map(func(r rune) rune {
		switch {
		case r == '/' || r == '\\':
			return '_'
		case unicode.IsControl(r), r == unicode.ReplacementChar:
			return -1
		}
		return r
	}, in)

	cleaned = strings.TrimSpace(cleaned)

	if runes := []rune(cleaned); len(runes) > maxDownloadNameRunes {
		cleaned = string(runes[:maxDownloadNameRunes])
	}

	return cleaned
}
