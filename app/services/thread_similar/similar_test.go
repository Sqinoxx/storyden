package thread_similar

import (
	"testing"

	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"

	"github.com/Southclaws/storyden/app/resources/datagraph"
)

func TestPick(t *testing.T) {
	self := xid.New()
	ref := func(id xid.ID, kind datagraph.Kind, sim float64) *datagraph.Ref {
		return &datagraph.Ref{ID: id, Kind: kind, Relevance: sim}
	}
	a, b, c, d := xid.New(), xid.New(), xid.New(), xid.New()

	got := Pick([]*datagraph.Ref{
		ref(self, datagraph.KindThread, 1),
		ref(a, datagraph.KindThread, 0.9),
		ref(b, datagraph.KindReply, 0.85),
		ref(c, datagraph.KindThread, 0.8),
		ref(d, datagraph.KindThread, 0.69),
	}, self, MinRelatedSimilarity, 5)

	assert.Equal(t, []*datagraph.Ref{ref(a, datagraph.KindThread, 0.9), ref(c, datagraph.KindThread, 0.8)}, got)
	assert.Len(t, Pick(got, xid.NilID(), 0, 1), 1)
}
