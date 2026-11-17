package thread_summary

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Southclaws/storyden/app/resources/asset"
	"github.com/Southclaws/storyden/app/resources/post/reply"
	"github.com/Southclaws/storyden/app/resources/post/thread"
	"github.com/Southclaws/storyden/app/services/summary_text"
)

// The limits add up to less than ai.PromptCharBudget together with the
// instructions and the labels around each part.
const (
	maxOpeningChars    = 1500
	maxReplyChars      = 600
	minReplyChars      = 150
	maxRepliesChars    = 3400
	maxAttachmentChars = 900
	maxAttachmentsSum  = 1800
)

type input struct {
	text        string
	replies     int
	attachments int
}

type candidate struct {
	number int
	handle string
	text   string
	likes  int
}

// selectReplies gives every reply an equal share of the budget. When there are
// too many for a readable share each, liked replies and the latest one are
// kept and the rest is sampled evenly across the thread.
func selectReplies(all []candidate) ([]candidate, int) {
	if len(all) == 0 {
		return nil, 0
	}

	keep := all
	if limit := maxRepliesChars / minReplyChars; len(all) > limit {
		chosen := map[int]bool{len(all) - 1: true}

		liked := make([]int, 0, len(all))
		for i, c := range all {
			if c.likes > 0 {
				liked = append(liked, i)
			}
		}
		sort.SliceStable(liked, func(a, b int) bool { return all[liked[a]].likes > all[liked[b]].likes })
		for _, i := range liked {
			if len(chosen) >= limit/2 {
				break
			}
			chosen[i] = true
		}

		step := float64(len(all)) / float64(limit)
		for k := 0; len(chosen) < limit && k < len(all); k++ {
			chosen[int(float64(k)*step)%len(all)] = true
		}
		for i := 0; len(chosen) < limit && i < len(all); i++ {
			chosen[i] = true
		}

		keep = make([]candidate, 0, limit)
		for i, c := range all {
			if chosen[i] {
				keep = append(keep, c)
			}
		}
	}

	share := min(maxReplyChars, max(minReplyChars, maxRepliesChars/len(keep)))
	return keep, share
}

func buildInput(t *thread.Thread, attachments []*asset.Asset) input {
	var b strings.Builder
	fmt.Fprintf(&b, "Title: %s\n\nOpening post:\n%s\n", summary_text.Clip(t.Title, 200), summary_text.Clip(t.Content.Plaintext(), maxOpeningChars))

	assets := append(append([]*asset.Asset{}, t.Assets...), attachments...)

	var all []candidate
	for _, r := range t.Replies.Items {
		text := strings.Join(strings.Fields(r.Content.Plaintext()), " ")
		if text == "" {
			continue
		}
		all = append(all, candidate{number: len(all) + 1, handle: handleOf(r), text: text, likes: r.Likes.Count})
	}

	keep, share := selectReplies(all)
	b.WriteString("\nReplies:\n")
	if len(keep) < len(all) {
		fmt.Fprintf(&b, "(%d of %d replies shown)\n", len(keep), len(all))
	}
	for _, c := range keep {
		fmt.Fprintf(&b, "[%d] %s: %s\n", c.number, c.handle, summary_text.Clip(c.text, share))
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
		if text == "" || sum+len([]rune(text)) > maxAttachmentsSum {
			continue
		}
		if attached == 0 {
			b.WriteString("\nText extracted from attached documents:\n")
		}
		attached++
		sum += len([]rune(text))
		fmt.Fprintf(&b, "[%s] %s\n", summary_text.Clip(summary_text.DisplayName(a.Name.String()), 60), text)
	}

	return input{text: b.String(), replies: len(all), attachments: attached}
}

func handleOf(r *reply.Reply) string {
	return summary_text.Clip(r.Author.Handle, 30)
}
