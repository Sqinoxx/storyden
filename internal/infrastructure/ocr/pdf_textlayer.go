package ocr

import (
	"bytes"
	"strings"
	"unicode"

	"github.com/Southclaws/fault"
	"github.com/ledongthuc/pdf"
)

const (
	minUsableWords = 5
	minAlnumRatio  = 0.5
)

// extractPDFTextLayer reads the embedded text layer of a PDF. It never
// panics: the underlying pdf library panics on malformed structures, so
// every stage (opening the reader, and each individual page) is guarded by
// recover. A panic on one page does not discard text already extracted from
// other pages.
func extractPDFTextLayer(data []byte) (text string, pages int, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fault.Newf("pdf text layer extraction panicked: %v", r)
		}
	}()

	rd, rerr := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if rerr != nil {
		return "", 0, fault.Wrap(rerr)
	}

	numPages := rd.NumPage()

	var buf bytes.Buffer
	fonts := make(map[string]*pdf.Font)

	for i := 1; i <= numPages; i++ {
		pageText, ok := extractPDFPage(rd, i, fonts)
		if ok {
			buf.WriteString(pageText)
			buf.WriteString("\n")
		}
	}

	return buf.String(), numPages, nil
}

// extractPDFPage extracts the plain text of a single page, recovering from
// any panic so that a single malformed page does not abort the whole
// document.
func extractPDFPage(rd *pdf.Reader, num int, fonts map[string]*pdf.Font) (text string, ok bool) {
	defer func() {
		if recover() != nil {
			text, ok = "", false
		}
	}()

	p := rd.Page(num)
	if p.V.IsNull() {
		return "", false
	}

	for _, name := range p.Fonts() {
		if _, seen := fonts[name]; !seen {
			f := p.Font(name)
			fonts[name] = &f
		}
	}

	t, err := p.GetPlainText(fonts)
	if err != nil {
		return "", false
	}

	return t, true
}

// hasUsableText reports whether extracted text looks like real prose rather
// than the near-empty or mojibake output typical of scanned pages or
// subset-embedded fonts without a usable encoding.
func hasUsableText(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}

	words := strings.Fields(s)
	if len(words) < minUsableWords {
		return false
	}

	var alnum, total int
	for _, r := range s {
		if unicode.IsSpace(r) {
			continue
		}
		total++
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			alnum++
		}
	}
	if total == 0 {
		return false
	}

	return float64(alnum)/float64(total) >= minAlnumRatio
}
