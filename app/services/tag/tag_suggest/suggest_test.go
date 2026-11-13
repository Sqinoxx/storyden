package tag_suggest

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseMeanings(t *testing.T) {
	got := ParseMeanings("endo: root canals\n  KFO :  braces, brackets \nkaputt\n: no tag\nleer:\nurl: see http://x.y")
	assert.Equal(t, map[string]string{
		"endo": "root canals",
		"kfo":  "braces, brackets",
		"url":  "see http://x.y",
	}, got)
}

func TestCosine(t *testing.T) {
	assert.InDelta(t, 1.0, cosine([]float32{1, 2}, []float32{2, 4}), 1e-9)
	assert.InDelta(t, 0.0, cosine([]float32{1, 0}, []float32{0, 1}), 1e-9)
	assert.Zero(t, cosine([]float32{1}, []float32{1, 2}))
	assert.Zero(t, cosine([]float32{0, 0}, []float32{1, 2}))
}

func TestVote(t *testing.T) {
	n := func(sim float64, tags ...string) Neighbour {
		return Neighbour{Similarity: sim, Tags: tags}
	}

	t.Run("majority_of_similar_threads", func(t *testing.T) {
		got := Vote([]Neighbour{
			n(0.9, "endo", "wurzelkanal"),
			n(0.8, "Endo"),
			n(0.7, "kfo"),
			n(0.6),
		})
		assert.Equal(t, []string{"endo"}, got)
	})

	t.Run("single_thread_is_not_enough", func(t *testing.T) {
		assert.Empty(t, Vote([]Neighbour{n(0.95, "endo"), n(0.6, "kfo")}))
	})

	t.Run("minority_tag_is_dropped", func(t *testing.T) {
		got := Vote([]Neighbour{
			n(0.9, "endo", "anatomie"),
			n(0.9, "endo"),
			n(0.9, "endo"),
			n(0.9, "endo"),
			n(0.9, "endo", "anatomie"),
			n(0.9, "endo"),
		})
		assert.Equal(t, []string{"endo"}, got)
	})

	t.Run("at_most_three_ordered_by_score", func(t *testing.T) {
		got := Vote([]Neighbour{
			n(0.9, "a", "b", "c", "d"),
			n(0.8, "a", "b", "c", "d"),
			n(0.7, "b", "c", "d"),
			n(0.6, "c", "d"),
			n(0.5, "d"),
		})
		assert.Equal(t, []string{"d", "c", "b"}, got)
	})

	t.Run("nothing_tagged", func(t *testing.T) {
		assert.Empty(t, Vote([]Neighbour{n(0.9), n(0.8)}))
		assert.Empty(t, Vote(nil))
	})
}

func TestMerge(t *testing.T) {
	assert.Equal(t, []string{"endo", "kfo", "pharma"}, merge(3, []string{"endo", "kfo"}, []string{"Endo", "pharma", "anatomie"}))
	assert.Empty(t, merge(3))
}
