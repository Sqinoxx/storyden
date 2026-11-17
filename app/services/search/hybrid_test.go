package search

import (
	"testing"

	"github.com/Southclaws/opt"
	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"

	"github.com/Southclaws/storyden/app/resources/account"
	"github.com/Southclaws/storyden/app/resources/datagraph"
	"github.com/Southclaws/storyden/app/resources/pagination"
	"github.com/Southclaws/storyden/app/resources/post"
	"github.com/Southclaws/storyden/app/resources/post/thread"
	"github.com/Southclaws/storyden/app/services/search/searcher"
)

func th(title string) *thread.Thread {
	return &thread.Thread{Post: post.Post{ID: post.ID(xid.New())}, Title: title}
}

func titles(items datagraph.ItemList) []string {
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = it.GetName()
	}
	return out
}

func TestFuseKeepsTitleMatchesOnTopAndAddsMeaning(t *testing.T) {
	exact := th("Hirnnerven Eselsbrücke")
	bodyOnly := th("Anatomie Lernplan")
	both := th("Physikum Nerven")
	meaning := th("Merkspruch für die zwölf Kopfnerven")

	got, added := Fuse("hirnnerven", datagraph.ItemList{bodyOnly, exact, both}, datagraph.ItemList{meaning, both})

	assert.Equal(t, "Hirnnerven Eselsbrücke", got[0].GetName())
	assert.Equal(t, []string{"Hirnnerven Eselsbrücke", "Physikum Nerven", "Anatomie Lernplan", "Merkspruch für die zwölf Kopfnerven"}, titles(got))
	assert.Equal(t, []xid.ID{meaning.GetID()}, added)
}

func TestFuseWithoutSemanticKeepsKeywordOrder(t *testing.T) {
	a, b := th("a"), th("b")
	got, added := Fuse("x", datagraph.ItemList{a, b}, nil)
	assert.Equal(t, []string{"a", "b"}, titles(got))
	assert.Empty(t, added)
}

func TestFuseOnlySemantic(t *testing.T) {
	a := th("Kofferdam anlegen")
	got, added := Fuse("Spanngummi", nil, datagraph.ItemList{a})
	assert.Equal(t, []string{"Kofferdam anlegen"}, titles(got))
	assert.Equal(t, []xid.ID{a.GetID()}, added)
}

func TestSemanticApplies(t *testing.T) {
	first := pagination.NewPageParams(1, 50)
	second := pagination.NewPageParams(2, 50)

	assert.True(t, semanticApplies("Hirnnerven", first, searcher.Options{}))
	assert.True(t, semanticApplies("Hirnnerven", first, searcher.Options{Kinds: opt.New([]datagraph.Kind{datagraph.KindThread, datagraph.KindNode})}))

	assert.False(t, semanticApplies("Hirnnerven", second, searcher.Options{}))
	assert.False(t, semanticApplies("ab", first, searcher.Options{}))
	assert.False(t, semanticApplies("Hirnnerven", first, searcher.Options{Kinds: opt.New([]datagraph.Kind{datagraph.KindNode})}))
	assert.False(t, semanticApplies("Hirnnerven", first, searcher.Options{Authors: opt.New([]account.AccountID{account.AccountID(xid.New())})}))
}
