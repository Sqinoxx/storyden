package thread_summary

import (
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
