package thread_summary

import (
	"fmt"
	"strings"

	"github.com/Southclaws/storyden/app/resources/asset"
	"github.com/Southclaws/storyden/app/resources/post/thread"
	"github.com/Southclaws/storyden/app/services/summary_text"
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
	fmt.Fprintf(&b, "Title: %s\n\nOpening post:\n%s\n", t.Title, summary_text.Clip(t.Content.Plaintext(), maxOpeningChars))

	assets := append(append([]*asset.Asset{}, t.Assets...), attachments...)

	b.WriteString("\nReplies:\n")
	used := 0
	replies := 0
	for _, r := range t.Replies.Items {
		text := summary_text.Clip(r.Content.Plaintext(), maxReplyChars)
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
		raw := a.OCRText.OrZero()
		if sum, ok := a.Summary.Get(); ok {
			raw = summary_text.StripTags(sum.HTML)
		}
		text := summary_text.Clip(raw, maxAttachmentChars)
		if text == "" || sum+len(text) > maxAttachmentsSum {
			continue
		}
		if attached == 0 {
			b.WriteString("\nText extracted from attached documents:\n")
		}
		attached++
		sum += len(text)
		fmt.Fprintf(&b, "[%s] %s\n", summary_text.DisplayName(a.Name.String()), text)
	}

	return input{text: b.String(), replies: replies, attachments: attached}
}
