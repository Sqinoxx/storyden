package ocr

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"github.com/gabriel-vasile/mimetype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Southclaws/storyden/internal/config"
)

func makeZip(t *testing.T, entries [][2]string) []byte {
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

func makeDOCX(t *testing.T, paragraphs ...string) []byte {
	t.Helper()

	var body strings.Builder
	for _, p := range paragraphs {
		fmt.Fprintf(&body, `<w:p><w:r><w:t xml:space="preserve">%s</w:t></w:r></w:p>`, p)
	}

	return makeZip(t, [][2]string{
		{"[Content_Types].xml", `<?xml version="1.0"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/></Types>`},
		{"_rels/.rels", `<?xml version="1.0"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"/>`},
		{"word/document.xml", `<?xml version="1.0"?><w:document xmlns:w="` + nsWordML + `"><w:body>` + body.String() + `<w:sectPr/></w:body></w:document>`},
	})
}

func slideXML(text string) string {
	return `<?xml version="1.0"?><p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:a="` + nsDrawingML + `"><p:cSld><p:spTree><p:sp><p:txBody><a:p><a:r><a:t>` + text + `</a:t></a:r></a:p></p:txBody></p:sp></p:spTree></p:cSld></p:sld>`
}

func makePPTX(t *testing.T, slides map[int]string) []byte {
	t.Helper()

	entries := [][2]string{
		{"[Content_Types].xml", `<?xml version="1.0"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Override PartName="/ppt/presentation.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.presentation.main+xml"/></Types>`},
		{"_rels/.rels", `<?xml version="1.0"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"/>`},
		{"ppt/presentation.xml", `<?xml version="1.0"?><p:presentation xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main"/>`},
	}
	for n, text := range slides {
		entries = append(entries, [2]string{fmt.Sprintf("ppt/slides/slide%d.xml", n), slideXML(text)})
	}

	return makeZip(t, entries)
}

func makeODT(t *testing.T, content string) []byte {
	t.Helper()

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.CreateHeader(&zip.FileHeader{Name: "mimetype", Method: zip.Store})
	require.NoError(t, err)
	_, err = w.Write([]byte(MimeODT))
	require.NoError(t, err)
	w, err = zw.Create("content.xml")
	require.NoError(t, err)
	_, err = w.Write([]byte(`<?xml version="1.0"?><office:document-content xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0" xmlns:text="` + nsODFText + `"><office:body><office:text>` + content + `</office:text></office:body></office:document-content>`))
	require.NoError(t, err)
	require.NoError(t, zw.Close())

	return buf.Bytes()
}

func extract(t *testing.T, data []byte, mime string) (Result, error) {
	t.Helper()
	c := NewLayeredClient(slog.Default(), nil, nil, config.Config{})
	return c.ExtractText(context.Background(), data, mime)
}

func TestDocument_DOCX(t *testing.T) {
	data := makeDOCX(t, "Altklausur Pharmakologie", "Frage 1: Wirkmechanismus von Aspirin")

	assert.Equal(t, MimeDOCX, mimetype.Detect(data).String())

	result, err := extract(t, data, MimeDOCX)
	require.NoError(t, err)
	assert.Equal(t, "office-xml", result.Engine)
	assert.Equal(t, "Altklausur Pharmakologie\nFrage 1: Wirkmechanismus von Aspirin", result.Text)
}

func TestDocument_PPTXKeepsSlideOrder(t *testing.T) {
	data := makePPTX(t, map[int]string{10: "Zehn", 2: "Zwei", 1: "Eins"})

	assert.Equal(t, MimePPTX, mimetype.Detect(data).String())

	result, err := extract(t, data, MimePPTX)
	require.NoError(t, err)
	assert.Equal(t, "Eins\n\nZwei\n\nZehn", result.Text)
}

func TestDocument_ODT(t *testing.T) {
	data := makeODT(t, `<text:h>Kariologie</text:h><text:p>Fluorid<text:s/>hemmt <text:span>Demineralisation</text:span></text:p>`)

	assert.Equal(t, MimeODT, mimetype.Detect(data).String())

	result, err := extract(t, data, MimeODT)
	require.NoError(t, err)
	assert.Equal(t, "odf-xml", result.Engine)
	assert.Equal(t, "Kariologie\nFluorid hemmt Demineralisation", result.Text)
}

func TestDocument_RTF(t *testing.T) {
	data := []byte(`{\rtf1\ansi{\fonttbl{\f0 Arial;}}{\colortbl;\red0\green0\blue0;}{\*\generator Word;}\f0 Parodontitis\par Gr\'fcne Zahnb\u252?rste\par}`)

	result, err := extract(t, data, "text/rtf")
	require.NoError(t, err)
	assert.Equal(t, "Parodontitis\nGrüne Zahnbürste", result.Text)
}

func TestDocument_PlainTextWithCharsetAndLatin1(t *testing.T) {
	result, err := extract(t, []byte("Zahnärztliche Prothetik"), "text/plain; charset=utf-8")
	require.NoError(t, err)
	assert.Equal(t, "Zahnärztliche Prothetik", result.Text)

	result, err = extract(t, []byte("Name;Br\xfccke\n"), "text/csv")
	require.NoError(t, err)
	assert.Equal(t, "Name;Brücke", result.Text)
}

func TestDocument_CorruptOrEmptyIsNoText(t *testing.T) {
	_, err := extract(t, []byte("PK\x03\x04 not really a zip"), MimeDOCX)
	assert.True(t, errors.Is(err, ErrNoText))

	_, err = extract(t, makeDOCX(t), MimeDOCX)
	assert.True(t, errors.Is(err, ErrNoText))

	_, err = extract(t, []byte("   \n"), "text/plain")
	assert.True(t, errors.Is(err, ErrNoText))
}

func TestDocument_DecompressionIsBounded(t *testing.T) {
	huge := strings.Repeat("A", maxZipEntryBytes+1)
	data := makeZip(t, [][2]string{
		{"word/document.xml", `<w:document xmlns:w="` + nsWordML + `"><w:body><w:p><w:r><w:t>` + huge + `</w:t></w:r></w:p></w:body></w:document>`},
	})

	_, err := extract(t, data, MimeDOCX)
	assert.True(t, errors.Is(err, ErrNoText))
}

func TestIsDocumentMIME(t *testing.T) {
	assert.True(t, IsDocumentMIME("text/plain; charset=utf-8"))
	assert.True(t, IsDocumentMIME(MimePPTX))
	assert.False(t, IsDocumentMIME("application/zip"))
	assert.False(t, IsDocumentMIME("image/svg+xml"))
}
