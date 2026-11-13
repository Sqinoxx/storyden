package summary_text_test

import (
	"testing"

	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Southclaws/storyden/app/services/summary_text"
)

func TestSanitise(t *testing.T) {
	out, err := summary_text.SanitiseHTML("```html\n<p>ok</p><script>x</script>\n```")
	require.NoError(t, err)
	assert.Contains(t, out, "<p>ok</p>")
	assert.NotContains(t, out, "script")
	assert.NotContains(t, out, "```")

	out, err = summary_text.SanitiseHTML("   ")
	require.NoError(t, err)
	assert.Empty(t, out)
}

func TestDisplayName(t *testing.T) {
	id := xid.New().String()
	assert.Equal(t, "klausur-pdf", summary_text.DisplayName(id+"-klausur-pdf"))
	assert.Equal(t, "plain-name", summary_text.DisplayName("plain-name"))
}

func TestStripTags(t *testing.T) {
	assert.Equal(t, "Endo &amp; KFO", summary_text.Clip(summary_text.StripTags("<ul><li>Endo &amp;amp; KFO</li></ul>"), 100))
	assert.Equal(t, "a b", summary_text.Clip(summary_text.StripTags("<p>a</p><p>b</p>"), 100))
}
