package summary_text

import (
	"html"
	"regexp"
	"strings"

	"github.com/rs/xid"

	"github.com/Southclaws/storyden/app/resources/datagraph"
)

func Clip(s string, max int) string {
	s = strings.Join(strings.Fields(s), " ")
	if r := []rune(s); len(r) > max {
		s = string(r[:max]) + "…"
	}
	return s
}

func DisplayName(stored string) string {
	if id, rest, ok := strings.Cut(stored, "-"); ok {
		if _, err := xid.FromString(id); err == nil {
			return rest
		}
	}
	return stored
}

func SanitiseHTML(answer string) (string, error) {
	answer = strings.TrimSpace(answer)
	answer = strings.TrimPrefix(answer, "```html")
	answer = strings.TrimPrefix(answer, "```")
	answer = strings.TrimSuffix(answer, "```")
	answer = strings.TrimSpace(answer)
	if answer == "" {
		return "", nil
	}

	content, err := datagraph.NewRichText(answer)
	if err != nil {
		return "", err
	}
	return content.HTML(), nil
}

var tagRe = regexp.MustCompile(`<[^>]*>`)

func StripTags(s string) string {
	return html.UnescapeString(tagRe.ReplaceAllString(s, " "))
}
