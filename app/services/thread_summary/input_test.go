package thread_summary

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Southclaws/opt"
	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Southclaws/storyden/app/resources/asset"
	"github.com/Southclaws/storyden/app/resources/datagraph"
	"github.com/Southclaws/storyden/app/resources/pagination"
	"github.com/Southclaws/storyden/app/resources/post"
	"github.com/Southclaws/storyden/app/resources/post/reply"
	"github.com/Southclaws/storyden/app/resources/post/thread"
	"github.com/Southclaws/storyden/internal/infrastructure/ai"
)

func content(t *testing.T, s string) datagraph.Content {
	c, err := datagraph.NewRichText("<p>" + s + "</p>")
	require.NoError(t, err)
	return c
}

func TestBuildInputBudget(t *testing.T) {
	var replies []*reply.Reply
	for range 20 {
		replies = append(replies, &reply.Reply{Post: post.Post{ID: post.ID(xid.New()), Content: content(t, strings.Repeat("a", 1000))}})
	}
	replies = append(replies, &reply.Reply{Post: post.Post{ID: post.ID(xid.New()), Content: content(t, "kurz")}})

	thr := &thread.Thread{
		Title:   "T",
		Post:    post.Post{Content: content(t, strings.Repeat("o", 5000))},
		Replies: pagination.Result[*reply.Reply]{Items: replies},
	}

	pdf := func(name, text string) *asset.Asset {
		return &asset.Asset{ID: asset.AssetID(xid.New()), Name: asset.NewFilename(name), OCRText: opt.New(text)}
	}
	big := pdf("a.pdf", strings.Repeat("x", 4000))
	in := buildInput(thr, []*asset.Asset{big, big, pdf("b.pdf", strings.Repeat("y", 4000)), pdf("c.pdf", "z")})

	assert.Equal(t, 21, in.replies)
	assert.Equal(t, 2, in.attachments)
	assert.Contains(t, in.text, "kurz")
	assert.Less(t, len(in.text), 11000)
	assert.Equal(t, 1, strings.Count(in.text, "[a-pdf]"))
	assert.NotContains(t, in.text, "[b-pdf]")
	assert.Contains(t, in.text, "[c-pdf] z")
}

func TestBuildInputPrefersStoredSummary(t *testing.T) {
	thr := &thread.Thread{
		Title: "T",
		Post:  post.Post{Content: content(t, "Frage")},
		Replies: pagination.Result[*reply.Reply]{Items: []*reply.Reply{
			{Post: post.Post{ID: post.ID(xid.New()), Content: content(t, "Antwort")}},
		}},
	}
	a := &asset.Asset{
		ID:      asset.AssetID(xid.New()),
		Name:    asset.NewFilename("klausur.pdf"),
		OCRText: opt.New(strings.Repeat("roher text ", 500)),
		Summary: opt.New(asset.Summary{HTML: "<ul><li>Kurzfassung der Klausur</li></ul>"}),
	}

	in := buildInput(thr, []*asset.Asset{a})
	assert.Contains(t, in.text, "Kurzfassung der Klausur")
	assert.NotContains(t, in.text, "roher text")
	assert.NotContains(t, in.text, "<li>")
}

func longReply(t *testing.T, marker string, likes int) *reply.Reply {
	r := &reply.Reply{Post: post.Post{ID: post.ID(xid.New()), Content: content(t, marker+" "+strings.Repeat("Wurzelkanalbehandlung ", 80))}}
	r.Likes.Count = likes
	r.Author.Handle = "studi"
	return r
}

func TestBuildInputKeepsLateAndLikedRepliesOfLongThreads(t *testing.T) {
	var replies []*reply.Reply
	for i := range 120 {
		likes := 0
		if i == 77 {
			likes = 9
		}
		replies = append(replies, longReply(t, fmt.Sprintf("marker%03d", i), likes))
	}

	thr := &thread.Thread{
		Title:   "T",
		Post:    post.Post{Content: content(t, "Frage")},
		Replies: pagination.Result[*reply.Reply]{Items: replies},
	}
	in := buildInput(thr, nil)

	assert.Equal(t, 120, in.replies)
	assert.Contains(t, in.text, "marker000")
	assert.Contains(t, in.text, "marker077")
	assert.Contains(t, in.text, "marker119")
	assert.Contains(t, in.text, "of 120 replies shown")
	assert.Contains(t, in.text, "[120] studi:")
}

func TestBuildInputSharesBudgetEvenly(t *testing.T) {
	var replies []*reply.Reply
	for i := range 10 {
		replies = append(replies, longReply(t, fmt.Sprintf("marker%03d", i), 0))
	}
	thr := &thread.Thread{
		Title:   "T",
		Post:    post.Post{Content: content(t, "Frage")},
		Replies: pagination.Result[*reply.Reply]{Items: replies},
	}
	in := buildInput(thr, nil)

	for i := range 10 {
		assert.Contains(t, in.text, fmt.Sprintf("marker%03d", i))
	}
	assert.NotContains(t, in.text, "replies shown")
}

func TestPromptFitsContextBudget(t *testing.T) {
	huge := strings.Repeat("Parodontitis ", 1000)
	for _, count := range []int{5, 10, 22, 23, 60, 200} {
		var replies []*reply.Reply
		for range count {
			r := &reply.Reply{Post: post.Post{ID: post.ID(xid.New()), Content: content(t, huge)}}
			r.Author.Handle = strings.Repeat("h", 80)
			replies = append(replies, r)
		}
		thr := &thread.Thread{
			Title:   strings.Repeat("T", 500),
			Post:    post.Post{Content: content(t, huge)},
			Replies: pagination.Result[*reply.Reply]{Items: replies},
		}
		var files []*asset.Asset
		for range 5 {
			files = append(files, &asset.Asset{ID: asset.AssetID(xid.New()), Name: asset.NewFilename(strings.Repeat("n", 200) + ".pdf"), OCRText: opt.New(huge)})
		}

		in := buildInput(thr, files)
		assert.LessOrEqual(t, len([]rune(instructions))+len([]rune(in.text)), ai.PromptCharBudget, "%d replies", count)
	}
}
