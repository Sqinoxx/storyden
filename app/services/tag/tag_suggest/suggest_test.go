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

func TestAppendMeanings(t *testing.T) {
	assert.Equal(t, "", AppendMeanings("", nil))
	assert.Equal(t, "endo: root canals", AppendMeanings("endo: root canals\n\n", nil))
	assert.Equal(t, "kfo: braces", AppendMeanings("  \n", []string{"kfo: braces"}))
	assert.Equal(t, "endo: root canals\nkfo: braces\npa: gums", AppendMeanings("endo: root canals\n", []string{"kfo: braces", "pa: gums"}))
}

func TestOneLine(t *testing.T) {
	assert.Equal(t, "a b c", oneLine(" a\n b\t c ", 10))
	assert.Equal(t, "äöü", oneLine("äöüß", 3))
}
