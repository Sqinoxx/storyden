package asset_ref_test

import (
	"fmt"
	"testing"

	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Southclaws/storyden/app/resources/asset/asset_ref"
	"github.com/Southclaws/storyden/app/resources/datagraph"
)

func TestExtractAssetIDsFromRefs(t *testing.T) {
	id := xid.New()
	validRef := fmt.Sprintf("http://localhost:8000/api/assets/%s-lehrplan-pdf", id.String())
	relativeRef := fmt.Sprintf("/api/assets/%s-lehrplan-pdf", id.String())

	cases := []struct {
		name string
		refs []string
		want []string
	}{
		{"absolute URL", []string{validRef}, []string{id.String()}},
		{"relative path", []string{relativeRef}, []string{id.String()}},
		{"external URL rejected", []string{"https://evil.example/" + id.String() + "-lehrplan-pdf"}, nil},
		{"garbage rejected", []string{"not a url at all", "", "/assets/"}, nil},
		{"malformed filename ignored", []string{"/api/assets/not-a-valid-xid"}, nil},
		{"duplicates deduped", []string{validRef, validRef}, []string{id.String()}},
		{"query and fragment stripped", []string{validRef + "?download=1#frag"}, []string{id.String()}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := asset_ref.ExtractAssetIDsFromRefs(c.refs)
			gotStrs := make([]string, len(got))
			for i, id := range got {
				gotStrs[i] = id.String()
			}
			if c.want == nil {
				assert.Empty(t, gotStrs)
			} else {
				assert.Equal(t, c.want, gotStrs)
			}
		})
	}
}

func TestExtractAssetIDs_FromContent(t *testing.T) {
	id1 := xid.New()
	id2 := xid.New()

	html := fmt.Sprintf(
		`<p>See attached <a href="/api/assets/%s-lehrplan-pdf" data-type="file-attachment">lehrplan.pdf</a></p><img src="http://localhost:8000/api/assets/%s-photo-png" alt="photo" />`,
		id1.String(), id2.String(),
	)

	content, err := datagraph.NewRichText(html)
	require.NoError(t, err)

	ids := asset_ref.ExtractAssetIDs(content)
	require.Len(t, ids, 2)
	assert.Equal(t, id1.String(), ids[0].String())
	assert.Equal(t, id2.String(), ids[1].String())
}

func TestExtractAssetIDs_EmptyContent(t *testing.T) {
	content, err := datagraph.NewRichText("<p>just some text, no attachments</p>")
	require.NoError(t, err)

	assert.Empty(t, asset_ref.ExtractAssetIDs(content))
}
