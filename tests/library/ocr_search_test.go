package library_test

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
// binary fixture. Mirrors tests/thread/search's helper of the same name.
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

// TestLibrarySearch_OCRCompoundWordPrefix covers searching a library node by
// a word contained inside a German compound word in its attached PDF's OCR
// text. German freely compounds words ("Anatomieklausur" is one word, not
// "Anatomie" + "Klausur" separately) — a naive whole-word full text search
// would miss this, which is exactly the bug prefix matching in
// node_search.ocrTextMatches exists to avoid.
func TestLibrarySearch_OCRCompoundWordPrefix(t *testing.T) {
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

			// "Anatomie" only ever appears as a prefix of the compound word
			// "Anatomieklausur" in the document text, never standalone.
			stem := "Anatomie"
			compound := stem + "klausur" + uuid.NewString()[:8]
			asset := uploadTestPDF(t, root, cl, adminSession,
				"Fragen zur "+compound+" im Wintersemester")

			assetID, err := xid.FromString(asset.Id)
			r.NoError(err)
			r.NoError(proc.ProcessAsset(root, assetID))

			visibility := openapi.VisibilityPublished
			assetIDs := openapi.AssetIDs{asset.Id}
			name := "Klausur " + uuid.NewString()
			nodeResp, err := cl.NodeCreateWithResponse(root, openapi.NodeInitialProps{
				Name:       name,
				Visibility: &visibility,
				AssetIds:   &assetIDs,
			}, adminSession)
			tests.Ok(t, err, nodeResp)

			searchResp, err := cl.DatagraphSearchWithResponse(root, &openapi.DatagraphSearchParams{
				Q:    opt.New(openapi.SearchQuery(stem)).Ptr(),
				Kind: opt.New([]openapi.DatagraphItemKind{openapi.DatagraphItemKindNode}).Ptr(),
			}, adminSession)
			tests.Ok(t, err, searchResp)

			found := false
			for _, item := range searchResp.JSON200.Items {
				nodeItem, err := item.AsDatagraphItemNode()
				if err != nil || nodeItem.Ref.Id != nodeResp.JSON200.Id {
					continue
				}
				found = true
			}
			r.True(found, "expected library node with compound word %q in its PDF to be found by searching %q", compound, stem)
		}))
	}))
}
