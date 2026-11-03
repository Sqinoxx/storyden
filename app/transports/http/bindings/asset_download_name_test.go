package bindings

import (
	"testing"

	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"

	"github.com/Southclaws/storyden/app/resources/asset"
	"github.com/Southclaws/storyden/internal/mime"
)

func testAsset(original string, mimeType string) *asset.Asset {
	return &asset.Asset{
		ID:   asset.NewID(),
		Name: asset.NewExistingFilename(xid.New(), original),
		MIME: mime.New(mimeType),
	}
}

func TestAssetDownloadName(t *testing.T) {
	pdf := testAsset("2024_OSCE_Zusammenfassung.pdf", "application/pdf")

	cases := []struct {
		name      string
		asset     *asset.Asset
		requested *string
		want      string
	}{
		{"derived from stored slug", pdf, nil, "2024_osce_zusammenfassung.pdf"},
		{"requested name is used", pdf, ptr("2024 OSCE Zusammenfassung.pdf"), "2024 OSCE Zusammenfassung.pdf"},
		{"requested name without extension", pdf, ptr("OSCE"), "OSCE.pdf"},
		{"requested extension cannot change the type", pdf, ptr("invoice.exe"), "invoice.pdf"},
		{"dots inside the name survive", pdf, ptr("Teil 1.2 Final.pdf"), "Teil 1.2 Final.pdf"},
		{"path separators are neutralised", pdf, ptr("../../etc/passwd"), "_.._etc_passwd.pdf"},
		{"blank request falls back to stored name", pdf, ptr("   "), "2024_osce_zusammenfassung.pdf"},
		{"original extension kept over mime default", testAsset("photo.jpeg", "image/jpeg"), nil, "photo.jpeg"},
		{"mime extension when upload had none", testAsset("", "application/pdf"), nil, "file.pdf"},
		{"no extension known", testAsset("", "application/octet-stream"), ptr("thing.bin"), "thing"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, assetDownloadName(c.asset, c.requested))
		})
	}
}

func TestContentDisposition(t *testing.T) {
	pdf := testAsset("2024_OSCE_Zusammenfassung.pdf", "application/pdf")

	assert.Equal(t,
		`inline; filename="2024 OSCE Zusammenfassung.pdf"`,
		contentDisposition(pdf, ptr("2024 OSCE Zusammenfassung.pdf")))

	assert.Equal(t,
		`inline; filename*=utf-8''Pr%C3%BCfung.pdf`,
		contentDisposition(pdf, ptr("Prüfung.pdf")))

	assert.Equal(t,
		`inline; filename="a \"quoted\" name.pdf"`,
		contentDisposition(pdf, ptr(`a "quoted" name.pdf`)))

	assert.Equal(t,
		`attachment; filename=notes.txt`,
		contentDisposition(testAsset("notes.txt", "text/plain"), nil))
}

func ptr(s string) *string { return &s }
