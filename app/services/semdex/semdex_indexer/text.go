package semdex_indexer

import (
	"strings"

	"github.com/Southclaws/storyden/app/services/summary_text"
)

const (
	maxBodyChars        = 2500
	maxReplyChars       = 200
	maxRepliesChars     = 2000
	maxAttachmentChars  = 600
	maxAttachmentsChars = 1200
)

type attachment struct {
	name    string
	summary string
	text    string
}

// threadText is what a thread is embedded as. The title comes first and the
// rest is bounded so the embedding model never truncates the important start.
func threadText(title, body string, tags, replies []string, attachments []attachment) string {
	var b strings.Builder

	title = strings.TrimSpace(title)
	if title != "" {
		b.WriteString(title)
		b.WriteString("\n\n")
	}
	b.WriteString(summary_text.Clip(body, maxBodyChars))

	if len(tags) > 0 {
		b.WriteString("\n\nTags: ")
		b.WriteString(strings.Join(tags, ", "))
	}

	used := 0
	for _, r := range replies {
		r = summary_text.Clip(r, maxReplyChars)
		if r == "" {
			continue
		}
		if used+len(r) > maxRepliesChars {
			break
		}
		if used == 0 {
			b.WriteString("\n\nAntworten:")
		}
		used += len(r)
		b.WriteString("\n- ")
		b.WriteString(r)
	}

	used = 0
	for _, a := range attachments {
		text := a.summary
		if text != "" {
			text = summary_text.StripTags(text)
		} else {
			text = a.text
		}
		text = summary_text.Clip(text, maxAttachmentChars)
		if text == "" {
			continue
		}
		if used+len(text) > maxAttachmentsChars {
			break
		}
		if used == 0 {
			b.WriteString("\n\nAnhänge:")
		}
		used += len(text)
		b.WriteString("\n- ")
		b.WriteString(summary_text.DisplayName(a.name))
		b.WriteString(": ")
		b.WriteString(text)
	}

	return b.String()
}
