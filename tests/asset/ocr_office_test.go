package asset_test

import (
	"archive/zip"
	"bytes"
	"context"
	"testing"

	"github.com/rs/xid"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/resources/account/account_writer"
	"github.com/Southclaws/storyden/app/resources/asset/asset_querier"
	"github.com/Southclaws/storyden/app/resources/asset/asset_writer"
	"github.com/Southclaws/storyden/app/resources/seed"
	"github.com/Southclaws/storyden/app/services/ocr"
	"github.com/Southclaws/storyden/app/transports/http/openapi"
	"github.com/Southclaws/storyden/internal/config"
	"github.com/Southclaws/storyden/internal/integration"
	"github.com/Southclaws/storyden/internal/integration/e2e"
)

func officeZip(t *testing.T, entries [][2]string) []byte {
	t.Helper()

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, e := range entries {
		w, err := zw.Create(e[0])
		require.NoError(t, err)
		_, err = w.Write([]byte(e[1]))
		require.NoError(t, err)
	}
	require.NoError(t, zw.Close())

	return buf.Bytes()
}

func testDOCX(t *testing.T, text string) []byte {
	return officeZip(t, [][2]string{
		{"[Content_Types].xml", `<?xml version="1.0"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"/>`},
		{"word/document.xml", `<?xml version="1.0"?><w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body><w:p><w:r><w:t>` + text + `</w:t></w:r></w:p></w:body></w:document>`},
	})
}

func testPPTX(t *testing.T, text string) []byte {
	return officeZip(t, [][2]string{
		{"[Content_Types].xml", `<?xml version="1.0"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"/>`},
		{"ppt/presentation.xml", `<?xml version="1.0"?><p:presentation xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main"/>`},
		{"ppt/slides/slide1.xml", `<?xml version="1.0"?><p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"><p:cSld><p:spTree><p:sp><p:txBody><a:p><a:r><a:t>` + text + `</a:t></a:r></a:p></p:txBody></p:sp></p:spTree></p:cSld></p:sld>`},
	})
}

func TestOCR_ExtractsTextFromOfficeAndTextFiles(t *testing.T) {
	cfg := &config.Config{
		OCREnabled:         true,
		OCRProvider:        "textlayer",
		OCRBackfillEnabled: false,
	}

	integration.Test(t, cfg, e2e.Setup(), fx.Invoke(func(
		root context.Context,
		lc fx.Lifecycle,
		cl *openapi.ClientWithResponses,
		sh *e2e.SessionHelper,
		aw *account_writer.Writer,
		aq *asset_querier.Querier,
		proc *ocr.Processor,
	) {
		lc.Append(fx.StartHook(func() {
			memberCtx, _ := e2e.WithAccount(root, aw, seed.Account_003_Baldur)
			session := sh.WithSession(memberCtx)

			cases := []struct {
				name string
				data []byte
				want string
			}{
				{"docx", testDOCX(t, "Altklausur Kieferorthopaedie Frage drei"), "Kieferorthopaedie"},
				{"pptx", testPPTX(t, "Vorlesung Endodontie Wurzelkanal"), "Endodontie"},
				{"txt", []byte("Zusammenfassung Parodontologie\nSondierungstiefe"), "Sondierungstiefe"},
			}

			for _, c := range cases {
				t.Run(c.name, func(t *testing.T) {
					r := require.New(t)

					a := uploadTestAsset(t, root, cl, session, "application/octet-stream", c.data)
					id, err := xid.FromString(a.Id)
					r.NoError(err)

					r.NoError(proc.ProcessAsset(root, id))

					got, err := aq.GetByID(root, id)
					r.NoError(err)
					r.Equal("completed", got.OCRStatus, "mime %s", got.MIME.String())
					text, ok := got.OCRText.Get()
					r.True(ok)
					r.Contains(text, c.want)
				})
			}
		}))
	}))
}

func TestOCR_RequeuesAssetsOfNewlySupportedTypes(t *testing.T) {
	cfg := &config.Config{
		OCREnabled:         true,
		OCRProvider:        "textlayer",
		OCRBackfillEnabled: false,
	}

	integration.Test(t, cfg, e2e.Setup(), fx.Invoke(func(
		root context.Context,
		lc fx.Lifecycle,
		cl *openapi.ClientWithResponses,
		sh *e2e.SessionHelper,
		aw *account_writer.Writer,
		aq *asset_querier.Querier,
		writer *asset_writer.Writer,
	) {
		lc.Append(fx.StartHook(func() {
			r := require.New(t)

			memberCtx, _ := e2e.WithAccount(root, aw, seed.Account_003_Baldur)
			session := sh.WithSession(memberCtx)

			docx := uploadTestAsset(t, root, cl, session, "application/octet-stream", testDOCX(t, "Prothetik"))
			docxID, err := xid.FromString(docx.Id)
			r.NoError(err)

			svg := uploadTestAsset(t, root, cl, session, "image/svg+xml", []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="1" height="1"/>`))
			svgID, err := xid.FromString(svg.Id)
			r.NoError(err)

			_, err = writer.UpdateOCRSkipped(root, docxID, asset_writer.UnsupportedMIMEReasonPrefix+": "+docx.MimeType)
			r.NoError(err)
			_, err = writer.UpdateOCRSkipped(root, svgID, asset_writer.UnsupportedMIMEReasonPrefix+": image/svg+xml")
			r.NoError(err)

			ids, err := writer.ResetOCRSkippedUnsupported(root, ocr.IsSupportedMIME)
			r.NoError(err)
			r.Contains(ids, docxID)
			r.NotContains(ids, svgID)

			got, err := aq.GetByID(root, docxID)
			r.NoError(err)
			r.Equal("pending", got.OCRStatus)

			got, err = aq.GetByID(root, svgID)
			r.NoError(err)
			r.Equal("skipped", got.OCRStatus)
		}))
	}))
}
