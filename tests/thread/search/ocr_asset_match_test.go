package search_test

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/Southclaws/opt"
	"github.com/google/uuid"
	"github.com/rs/xid"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/resources/account/account_writer"
	"github.com/Southclaws/storyden/app/resources/seed"
	"github.com/Southclaws/storyden/app/services/ocr"
	"github.com/Southclaws/storyden/app/transports/http/openapi"
	"github.com/Southclaws/storyden/internal/config"
	"github.com/Southclaws/storyden/internal/integration"
	"github.com/Southclaws/storyden/internal/integration/e2e"
	"github.com/Southclaws/storyden/tests"
)

// buildTestPDF builds a minimal, valid, single-page, uncompressed PDF whose
// content stream renders the given text with the Tj operator. Built at test
// time (with correctly computed byte offsets) rather than committed as a
// binary fixture.
func buildTestPDF(text string) []byte {
	escaped := strings.NewReplacer(`\`, `\\`, `(`, `\(`, `)`, `\)`).Replace(text)
	streamBody := fmt.Sprintf("BT /F1 24 Tf 72 712 Td (%s) Tj ET\n", escaped)

	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 4 0 R >> >> /Contents 5 0 R >>",
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
		fmt.Sprintf("<< /Length %d >>\nstream\n%sendstream", len(streamBody), streamBody),
	}

	var buf bytes.Buffer
	buf.WriteString("%PDF-1.4\n")

	offsets := make([]int, len(objects)+1)
	for i, obj := range objects {
		offsets[i+1] = buf.Len()
		buf.WriteString(fmt.Sprintf("%d 0 obj\n%s\nendobj\n", i+1, obj))
	}

	xrefStart := buf.Len()
	buf.WriteString(fmt.Sprintf("xref\n0 %d\n", len(objects)+1))
	buf.WriteString("0000000000 65535 f \n")
	for i := 1; i <= len(objects); i++ {
		buf.WriteString(fmt.Sprintf("%010d 00000 n \n", offsets[i]))
	}

	buf.WriteString(fmt.Sprintf("trailer\n<< /Size %d /Root 1 0 R >>\n", len(objects)+1))
	buf.WriteString(fmt.Sprintf("startxref\n%d\n%%%%EOF", xrefStart))

	return buf.Bytes()
}

func uploadTestPDF(t *testing.T, ctx context.Context, cl *openapi.ClientWithResponses, session openapi.RequestEditorFn, text string) openapi.Asset {
	t.Helper()

	data := buildTestPDF(text)

	assetResp, err := cl.AssetUploadWithBodyWithResponse(
		ctx,
		&openapi.AssetUploadParams{ContentLength: int64(len(data))},
		"application/pdf",
		bytes.NewReader(data),
		session,
	)
	tests.Ok(t, err, assetResp)

	return *assetResp.JSON200
}

// TestSearch_PDFAttachment_ExplicitAssetIDs covers a thread that explicitly
// attaches an asset via asset_ids: the PDF's extracted text should surface
// the thread in search, with an excerpt from the matched asset.
func TestSearch_PDFAttachment_ExplicitAssetIDs(t *testing.T) {
	cfg := &config.Config{
		OCREnabled:         true,
		OCRProvider:        "textlayer", // pure Go, no external binaries required
		OCRBackfillEnabled: false,
	}

	integration.Test(t, cfg, e2e.Setup(), fx.Invoke(func(
		root context.Context,
		lc fx.Lifecycle,
		cl *openapi.ClientWithResponses,
		sh *e2e.SessionHelper,
		aw *account_writer.Writer,
		proc *ocr.Processor,
	) {
		lc.Append(fx.StartHook(func() {
			r := require.New(t)

			adminCtx, _ := e2e.WithAccount(root, aw, seed.Account_001_Odin)
			adminSession := sh.WithSession(adminCtx)

			catResp, err := cl.CategoryCreateWithResponse(root, openapi.CategoryInitialProps{
				Name:   "ocr-search-" + uuid.NewString(),
				Colour: "#123456",
			}, adminSession)
			tests.Ok(t, err, catResp)

			word := "Spanischkurs" + uuid.NewString()[:8]
			asset := uploadTestPDF(t, root, cl, adminSession, "Auslandssemester "+word+" fuer Mediziner im vierten Semester")

			assetID, err := xid.FromString(asset.Id)
			r.NoError(err)

			err = proc.ProcessAsset(root, assetID)
			r.NoError(err)

			assetIDs := openapi.AssetIDs{asset.Id}
			threadResp, err := cl.ThreadCreateWithResponse(root, openapi.ThreadInitialProps{
				Title:      "Auslandssemester Planung " + uuid.NewString(),
				Body:       opt.New("<p>Wer hat Erfahrung mit dem Erasmus-Programm?</p>").Ptr(),
				Category:   opt.New(catResp.JSON200.Id).Ptr(),
				Visibility: opt.New(openapi.VisibilityPublished).Ptr(),
				AssetIds:   &assetIDs,
			}, adminSession)
			tests.Ok(t, err, threadResp)

			searchResp, err := cl.DatagraphSearchWithResponse(root, &openapi.DatagraphSearchParams{
				Q: opt.New(openapi.SearchQuery(word)).Ptr(),
			}, adminSession)
			tests.Ok(t, err, searchResp)

			found := false
			for _, item := range searchResp.JSON200.Items {
				threadItem, err := item.AsDatagraphItemThread()
				if err != nil || threadItem.Ref.Id != threadResp.JSON200.Id {
					continue
				}
				found = true

				r.NotNil(threadItem.AssetMatches, "expected asset_matches to be populated")
				r.NotEmpty(*threadItem.AssetMatches)
				match := (*threadItem.AssetMatches)[0]
				r.Equal(asset.Id, match.Asset.Id)
				r.Contains(match.Excerpt, word)
			}
			r.True(found, "expected thread with matching PDF attachment to appear in search results")
		}))
	}))
}

// TestSearch_PDFAttachment_BodyDerivedLink covers a thread that references an
// uploaded PDF only via a link in its body (as the composer does for
// non-image attachments), without explicit asset_ids — the deterministic
// body-linking must attach it for search to find it.
func TestSearch_PDFAttachment_BodyDerivedLink(t *testing.T) {
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
		proc *ocr.Processor,
	) {
		lc.Append(fx.StartHook(func() {
			r := require.New(t)

			adminCtx, _ := e2e.WithAccount(root, aw, seed.Account_001_Odin)
			adminSession := sh.WithSession(adminCtx)

			catResp, err := cl.CategoryCreateWithResponse(root, openapi.CategoryInitialProps{
				Name:   "ocr-search-body-" + uuid.NewString(),
				Colour: "#654321",
			}, adminSession)
			tests.Ok(t, err, catResp)

			word := "Zahnschmerzambulanz" + uuid.NewString()[:8]
			asset := uploadTestPDF(t, root, cl, adminSession, "Sprechstunde "+word+" jeden Mittwoch von 8 bis 12 Uhr")

			assetID, err := xid.FromString(asset.Id)
			r.NoError(err)
			r.NoError(proc.ProcessAsset(root, assetID))

			body := fmt.Sprintf(
				`<p>Anbei der Aushang: <a href="%s" data-type="file-attachment" download>%s</a></p>`,
				asset.Path, asset.Filename,
			)

			threadResp, err := cl.ThreadCreateWithResponse(root, openapi.ThreadInitialProps{
				Title:      "Sprechstundenaushang " + uuid.NewString(),
				Body:       opt.New(body).Ptr(),
				Category:   opt.New(catResp.JSON200.Id).Ptr(),
				Visibility: opt.New(openapi.VisibilityPublished).Ptr(),
			}, adminSession)
			tests.Ok(t, err, threadResp)

			searchResp, err := cl.DatagraphSearchWithResponse(root, &openapi.DatagraphSearchParams{
				Q: opt.New(openapi.SearchQuery(word)).Ptr(),
			}, adminSession)
			tests.Ok(t, err, searchResp)

			found := false
			for _, item := range searchResp.JSON200.Items {
				threadItem, err := item.AsDatagraphItemThread()
				if err != nil || threadItem.Ref.Id != threadResp.JSON200.Id {
					continue
				}
				found = true
				r.NotNil(threadItem.AssetMatches)
				r.NotEmpty(*threadItem.AssetMatches)
				r.Contains((*threadItem.AssetMatches)[0].Excerpt, word)
			}
			r.True(found, "expected thread with body-linked PDF attachment to appear in search results")
		}))
	}))
}

// TestSearch_PDFAttachment_Reply covers a reply whose body links an uploaded
// PDF the same way the reply composer does — replies have no explicit
// asset_ids field, so this is the only way reply attachments become linked.
func TestSearch_PDFAttachment_Reply(t *testing.T) {
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
		proc *ocr.Processor,
	) {
		lc.Append(fx.StartHook(func() {
			r := require.New(t)

			adminCtx, _ := e2e.WithAccount(root, aw, seed.Account_001_Odin)
			adminSession := sh.WithSession(adminCtx)

			catResp, err := cl.CategoryCreateWithResponse(root, openapi.CategoryInitialProps{
				Name:   "ocr-search-reply-" + uuid.NewString(),
				Colour: "#abcdef",
			}, adminSession)
			tests.Ok(t, err, catResp)

			threadResp, err := cl.ThreadCreateWithResponse(root, openapi.ThreadInitialProps{
				Title:      "Frage zu Klausurterminen " + uuid.NewString(),
				Body:       opt.New("<p>Weiss jemand wann die Klausuren stattfinden?</p>").Ptr(),
				Category:   opt.New(catResp.JSON200.Id).Ptr(),
				Visibility: opt.New(openapi.VisibilityPublished).Ptr(),
			}, adminSession)
			tests.Ok(t, err, threadResp)

			word := "Pruefungsordnung" + uuid.NewString()[:8]
			asset := uploadTestPDF(t, root, cl, adminSession, "Gemaess "+word+" Paragraph 12 gelten folgende Fristen")

			assetID, err := xid.FromString(asset.Id)
			r.NoError(err)
			r.NoError(proc.ProcessAsset(root, assetID))

			replyBody := fmt.Sprintf(
				`<p>Hier die Regelung: <a href="%s" data-type="file-attachment" download>%s</a></p>`,
				asset.Path, asset.Filename,
			)

			replyResp, err := cl.ReplyCreateWithResponse(adminCtx, threadResp.JSON200.Slug, openapi.ReplyInitialProps{
				Body: replyBody,
			}, adminSession)
			tests.Ok(t, err, replyResp)

			searchResp, err := cl.DatagraphSearchWithResponse(root, &openapi.DatagraphSearchParams{
				Q:    opt.New(openapi.SearchQuery(word)).Ptr(),
				Kind: opt.New([]openapi.DatagraphItemKind{openapi.DatagraphItemKindReply}).Ptr(),
			}, adminSession)
			tests.Ok(t, err, searchResp)

			found := false
			for _, item := range searchResp.JSON200.Items {
				replyItem, err := item.AsDatagraphItemReply()
				if err != nil || replyItem.Ref.Id != replyResp.JSON200.Id {
					continue
				}
				found = true
				r.NotNil(replyItem.AssetMatches)
				r.NotEmpty(*replyItem.AssetMatches)
				r.Contains((*replyItem.AssetMatches)[0].Excerpt, word)
			}
			r.True(found, "expected reply with body-linked PDF attachment to appear in search results")
		}))
	}))
}
