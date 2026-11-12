package thread_summary

import (
	"fmt"
	"strings"

	"github.com/rs/xid"

	"github.com/Southclaws/storyden/app/resources/asset"
	"github.com/Southclaws/storyden/app/resources/post/thread"
)

const (
	maxOpeningChars    = 2000
	maxReplyChars      = 600
	maxRepliesChars    = 4500
	maxAttachmentChars = 1500
	maxAttachmentsSum  = 3000
)

type input struct {
	text        string
	replies     int
	attachments int
}

func buildInput(t *thread.Thread, attachments []*asset.Asset) input {
	var b strings.Builder
	fmt.Fprintf(&b, "Title: %s\n\nOpening post:\n%s\n", t.Title, clip(t.Content.Plaintext(), maxOpeningChars))

	assets := append(append([]*asset.Asset{}, t.Assets...), attachments...)

	b.WriteString("\nReplies:\n")
	used := 0
	replies := 0
	for _, r := range t.Replies.Items {
		text := clip(r.Content.Plaintext(), maxReplyChars)
		if text == "" {
			continue
		}
		replies++
		if used+len(text) > maxRepliesChars {
			continue
		}
		used += len(text)
		fmt.Fprintf(&b, "[%d] %s: %s\n", replies, r.Author.Handle, text)
	}

	seen := map[asset.AssetID]bool{}
	attached := 0
	sum := 0
	for _, a := range assets {
		if a == nil || seen[a.ID] {
			continue
		}
		seen[a.ID] = true
		text := clip(a.OCRText.OrZero(), maxAttachmentChars)
		if text == "" || sum+len(text) > maxAttachmentsSum {
			continue
		}
		if attached == 0 {
			b.WriteString("\nText extracted from attached documents:\n")
		}
		attached++
		sum += len(text)
		fmt.Fprintf(&b, "[%s] %s\n", displayName(a.Name.String()), text)
	}

	return input{text: b.String(), replies: replies, attachments: attached}
}

func clip(s string, max int) string {
	s = strings.Join(strings.Fields(s), " ")
	if r := []rune(s); len(r) > max {
		s = string(r[:max]) + "…"
	}
	return s
}

func displayName(stored string) string {
	if id, rest, ok := strings.Cut(stored, "-"); ok {
		if _, err := xid.FromString(id); err == nil {
			return rest
		}
	}
	return stored
}
