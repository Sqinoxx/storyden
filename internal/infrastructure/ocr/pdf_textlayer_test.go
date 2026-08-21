package ocr

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// makeTextPDF builds a minimal, valid, single-page, uncompressed PDF whose
// content stream renders the given text with the Tj operator. It's built at
// test time (with correctly computed xref offsets) rather than committed as
// a binary fixture.
func makeTextPDF(t *testing.T, text string) []byte {
	t.Helper()

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

func makeEmptyPDF(t *testing.T) []byte {
	t.Helper()
	return makeTextPDF(t, "")
}

// makeMultiPageTextPDF builds a valid PDF with one page per entry in texts,
// each rendering its text with the Tj operator, with correctly computed
// xref offsets.
func makeMultiPageTextPDF(t *testing.T, texts []string) []byte {
	t.Helper()

	kids := make([]string, len(texts))
	for k := range texts {
		kids[k] = fmt.Sprintf("%d 0 R", 4+2*k)
	}

	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		fmt.Sprintf("<< /Type /Pages /Kids [%s] /Count %d >>", strings.Join(kids, " "), len(texts)),
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
	}

	escaper := strings.NewReplacer(`\`, `\\`, `(`, `\(`, `)`, `\)`)
	for k, text := range texts {
		contentObjNum := 5 + 2*k
		objects = append(objects,
			fmt.Sprintf("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 3 0 R >> >> /Contents %d 0 R >>", contentObjNum))

		streamBody := fmt.Sprintf("BT /F1 24 Tf 72 712 Td (%s) Tj ET\n", escaper.Replace(text))
		objects = append(objects, fmt.Sprintf("<< /Length %d >>\nstream\n%sendstream", len(streamBody), streamBody))
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

func TestExtractPDFTextLayer_RoundTrip(t *testing.T) {
	data := makeTextPDF(t, "Auslandssemester Spanisch fuer Mediziner im vierten Semester")

	text, pages, err := extractPDFTextLayer(context.Background(), data)
	require.NoError(t, err)
	assert.Equal(t, 1, pages)
	assert.Contains(t, text, "Spanisch")
}

func TestExtractPDFTextLayer_CorruptDoesNotPanic(t *testing.T) {
	garbage := bytes.Repeat([]byte{0x00, 0xFF, 0x13, 0x37}, 256)

	assert.NotPanics(t, func() {
		_, _, _ = extractPDFTextLayer(context.Background(), garbage)
	})
}

func TestExtractPDFTextLayer_EmptyHasNoUsableText(t *testing.T) {
	data := makeEmptyPDF(t)

	text, _, err := extractPDFTextLayer(context.Background(), data)
	require.NoError(t, err)
	assert.False(t, hasUsableText(text))
}

func TestExtractPDFTextLayer_MultiPageRoundTrip(t *testing.T) {
	data := makeMultiPageTextPDF(t, []string{"Anatomie", "Physiologie", "Biochemie"})

	text, pages, err := extractPDFTextLayer(context.Background(), data)
	require.NoError(t, err)
	assert.Equal(t, 3, pages)
	assert.Contains(t, text, "Anatomie")
	assert.Contains(t, text, "Physiologie")
	assert.Contains(t, text, "Biochemie")
}

func TestExtractPDFTextLayer_CancelledContextStopsBeforeFirstPage(t *testing.T) {
	data := makeMultiPageTextPDF(t, []string{"Anatomie", "Physiologie", "Biochemie"})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	text, pages, err := extractPDFTextLayer(ctx, data)
	require.NoError(t, err)
	assert.Equal(t, 0, pages)
	assert.Empty(t, text)
}

func TestHasUsableText(t *testing.T) {
	cases := []struct {
		name string
		text string
		want bool
	}{
		{"prose", "This is a normal sentence with enough words to count as real text.", true},
		{"empty", "", false},
		{"too short", "just three words", false},
		{"mojibake", "����������", false},
		{"whitespace only", "     \n\t   ", false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, hasUsableText(c.text))
		})
	}
}
