package ocr

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fmsg"
)

const (
	MimeDOCX = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	MimePPTX = "application/vnd.openxmlformats-officedocument.presentationml.presentation"
	MimeODT  = "application/vnd.oasis.opendocument.text"
	MimeODP  = "application/vnd.oasis.opendocument.presentation"
	MimeODS  = "application/vnd.oasis.opendocument.spreadsheet"
	MimeText = "text/plain"
	MimeMD   = "text/markdown"
	MimeCSV  = "text/csv"
	MimeRTF  = "text/rtf"
	MimeRTF2 = "application/rtf"
)

const (
	maxZipEntryBytes = 20 << 20
	maxZipTotalBytes = 50 << 20
)

const (
	nsWordML    = "http://schemas.openxmlformats.org/wordprocessingml/2006/main"
	nsDrawingML = "http://schemas.openxmlformats.org/drawingml/2006/main"
	nsODFText   = "urn:oasis:names:tc:opendocument:xmlns:text:1.0"
)

var documentMIMEs = map[string]bool{
	MimeDOCX: true,
	MimePPTX: true,
	MimeODT:  true,
	MimeODP:  true,
	MimeODS:  true,
	MimeText: true,
	MimeMD:   true,
	MimeCSV:  true,
	MimeRTF:  true,
	MimeRTF2: true,
}

// BaseMIME strips parameters such as "; charset=utf-8", which the content
// sniffer appends to every text type.
func BaseMIME(m string) string {
	base, _, _ := strings.Cut(m, ";")
	return strings.ToLower(strings.TrimSpace(base))
}

// IsDocumentMIME reports whether text can be read directly out of the file's
// structure, without any image recognition.
func IsDocumentMIME(m string) bool {
	return documentMIMEs[BaseMIME(m)]
}

func extractDocument(data []byte, mimeType string) (Result, error) {
	var (
		text   string
		engine string
		err    error
	)

	switch BaseMIME(mimeType) {
	case MimeDOCX:
		text, err = extractDOCX(data)
		engine = "office-xml"
	case MimePPTX:
		text, err = extractPPTX(data)
		engine = "office-xml"
	case MimeODT, MimeODP, MimeODS:
		text, err = extractODF(data)
		engine = "odf-xml"
	case MimeRTF, MimeRTF2:
		text = extractRTF(data)
		engine = "rtf"
	case MimeText, MimeMD, MimeCSV:
		text = extractPlain(data)
		engine = "plain"
	default:
		return Result{}, fault.Wrap(ErrUnsupportedMIME)
	}

	if err != nil {
		return Result{}, fault.Wrap(ErrNoText, fmsg.With(err.Error()))
	}

	text = strings.TrimSpace(text)
	if text == "" {
		return Result{}, fault.Wrap(ErrNoText)
	}

	return Result{Text: text, Engine: engine}, nil
}

type zipDocument struct {
	files map[string]*zip.File
	read  int64
}

func openZip(data []byte) (*zipDocument, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}

	files := make(map[string]*zip.File, len(zr.File))
	for _, f := range zr.File {
		files[f.Name] = f
	}

	return &zipDocument{files: files}, nil
}

// read bounds every entry and the archive as a whole, because the declared
// sizes in a zip header are attacker-controlled and a few kilobytes can
// inflate to gigabytes.
func (z *zipDocument) readEntry(name string) ([]byte, bool, error) {
	f, ok := z.files[name]
	if !ok {
		return nil, false, nil
	}

	rc, err := f.Open()
	if err != nil {
		return nil, true, err
	}
	defer rc.Close()

	remaining := min(int64(maxZipEntryBytes), maxZipTotalBytes-z.read)
	if remaining <= 0 {
		return nil, true, fault.New("document exceeds the decompressed size limit")
	}

	b, err := io.ReadAll(io.LimitReader(rc, remaining+1))
	if err != nil {
		return nil, true, err
	}
	if int64(len(b)) > remaining {
		return nil, true, fault.New("document exceeds the decompressed size limit")
	}

	z.read += int64(len(b))

	return b, true, nil
}

func (z *zipDocument) numbered(pattern *regexp.Regexp) []string {
	type entry struct {
		name string
		n    int
	}

	var entries []entry
	for name := range z.files {
		m := pattern.FindStringSubmatch(name)
		if m == nil {
			continue
		}
		n, err := strconv.Atoi(m[1])
		if err != nil {
			continue
		}
		entries = append(entries, entry{name, n})
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].n < entries[j].n })

	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.name
	}

	return names
}

func extractDOCX(data []byte) (string, error) {
	z, err := openZip(data)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	found := false

	for _, part := range []string{"word/document.xml", "word/footnotes.xml", "word/endnotes.xml"} {
		b, ok, err := z.readEntry(part)
		if err != nil {
			return "", err
		}
		if !ok {
			continue
		}
		found = true

		text, err := ooxmlText(b, nsWordML)
		if err != nil {
			return "", err
		}
		writeSection(&sb, text)
	}

	if !found {
		return "", fault.New("docx has no word/document.xml")
	}

	return sb.String(), nil
}

var (
	slidePattern = regexp.MustCompile(`^ppt/slides/slide(\d+)\.xml$`)
	notesPattern = regexp.MustCompile(`^ppt/notesSlides/notesSlide(\d+)\.xml$`)
)

func extractPPTX(data []byte) (string, error) {
	z, err := openZip(data)
	if err != nil {
		return "", err
	}

	slides := z.numbered(slidePattern)
	if len(slides) == 0 {
		return "", fault.New("pptx has no slides")
	}

	var sb strings.Builder

	for _, part := range append(slides, z.numbered(notesPattern)...) {
		b, _, err := z.readEntry(part)
		if err != nil {
			return "", err
		}

		text, err := ooxmlText(b, nsDrawingML)
		if err != nil {
			return "", err
		}
		writeSection(&sb, text)
	}

	return sb.String(), nil
}

func writeSection(sb *strings.Builder, text string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	if sb.Len() > 0 {
		sb.WriteString("\n\n")
	}
	sb.WriteString(text)
}

// ooxmlText collects the runs of a WordprocessingML or DrawingML part. Text
// only ever lives in <t> elements of the given namespace; everything else is
// layout and styling.
func ooxmlText(data []byte, ns string) (string, error) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	dec.Strict = false

	var sb strings.Builder
	inText := 0

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Space != ns {
				continue
			}
			switch t.Name.Local {
			case "t":
				inText++
			case "tab":
				sb.WriteByte('\t')
			case "br", "cr":
				sb.WriteByte('\n')
			}

		case xml.EndElement:
			if t.Name.Space != ns {
				continue
			}
			switch t.Name.Local {
			case "t":
				inText = max(0, inText-1)
			case "p":
				sb.WriteByte('\n')
			}

		case xml.CharData:
			if inText > 0 {
				sb.Write(t)
			}
		}
	}

	return sb.String(), nil
}

func extractODF(data []byte) (string, error) {
	z, err := openZip(data)
	if err != nil {
		return "", err
	}

	b, ok, err := z.readEntry("content.xml")
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fault.New("opendocument has no content.xml")
	}

	dec := xml.NewDecoder(bytes.NewReader(b))
	dec.Strict = false

	var sb strings.Builder
	inPara := 0

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Space != nsODFText {
				continue
			}
			switch t.Name.Local {
			case "p", "h":
				inPara++
			case "s":
				sb.WriteByte(' ')
			case "tab":
				sb.WriteByte('\t')
			case "line-break":
				sb.WriteByte('\n')
			}

		case xml.EndElement:
			if t.Name.Space != nsODFText {
				continue
			}
			if t.Name.Local == "p" || t.Name.Local == "h" {
				inPara = max(0, inPara-1)
				sb.WriteByte('\n')
			}

		case xml.CharData:
			if inPara > 0 {
				sb.Write(t)
			}
		}
	}

	return sb.String(), nil
}

// extractPlain falls back to Latin-1 for files that are not valid UTF-8,
// which is what spreadsheets exported on Windows usually are.
func extractPlain(data []byte) string {
	data = bytes.TrimPrefix(data, []byte("\xEF\xBB\xBF"))

	if utf8.Valid(data) {
		return string(data)
	}

	return decodeLatin1(data)
}

func decodeLatin1(data []byte) string {
	runes := make([]rune, len(data))
	for i, b := range data {
		runes[i] = rune(b)
	}
	return string(runes)
}

var rtfSkippedDestinations = map[string]bool{
	"fonttbl":      true,
	"colortbl":     true,
	"stylesheet":   true,
	"info":         true,
	"pict":         true,
	"header":       true,
	"footer":       true,
	"listtable":    true,
	"themedata":    true,
	"datastore":    true,
	"latentstyles": true,
}

// extractRTF walks the control words of an RTF document and keeps only the
// visible text. Destination groups that hold metadata (font tables, pictures,
// "\*" extensions) are skipped wholesale.
func extractRTF(data []byte) string {
	var sb strings.Builder

	type group struct{ skip bool }
	stack := []group{{}}
	skipping := func() bool { return stack[len(stack)-1].skip }

	ucSkip := 1
	pendingSkip := 0

	for i := 0; i < len(data); i++ {
		c := data[i]

		switch c {
		case '{':
			stack = append(stack, group{skip: skipping()})
			continue
		case '}':
			if len(stack) > 1 {
				stack = stack[:len(stack)-1]
			}
			continue
		case '\r', '\n':
			continue
		case '\\':
		default:
			if pendingSkip > 0 {
				pendingSkip--
				continue
			}
			if !skipping() {
				sb.WriteByte(c)
			}
			continue
		}

		if i+1 >= len(data) {
			break
		}
		next := data[i+1]

		switch {
		case next == '\\' || next == '{' || next == '}':
			if !skipping() {
				sb.WriteByte(next)
			}
			i++

		case next == '*':
			stack[len(stack)-1].skip = true
			i++

		case next == '\'':
			if i+3 < len(data) {
				if v, err := strconv.ParseUint(string(data[i+2:i+4]), 16, 8); err == nil {
					if pendingSkip > 0 {
						pendingSkip--
					} else if !skipping() {
						sb.WriteRune(rune(v))
					}
				}
			}
			i += 3

		case isASCIILetter(next):
			j := i + 1
			for j < len(data) && isASCIILetter(data[j]) {
				j++
			}
			word := string(data[i+1 : j])

			k := j
			if k < len(data) && (data[k] == '-' || isASCIIDigit(data[k])) {
				k++
				for k < len(data) && isASCIIDigit(data[k]) {
					k++
				}
			}
			param, hasParam := 0, k > j
			if hasParam {
				param, _ = strconv.Atoi(string(data[j:k]))
			}
			if k < len(data) && data[k] == ' ' {
				k++
			}
			i = k - 1

			if rtfSkippedDestinations[word] {
				stack[len(stack)-1].skip = true
				continue
			}
			if skipping() {
				continue
			}

			switch word {
			case "par", "line", "row", "sect", "page":
				sb.WriteByte('\n')
			case "tab", "cell":
				sb.WriteByte('\t')
			case "uc":
				if hasParam {
					ucSkip = param
				}
			case "u":
				if hasParam {
					if param < 0 {
						param += 65536
					}
					sb.WriteRune(rune(param))
					pendingSkip = ucSkip
				}
			}

		default:
			i++
		}
	}

	return sb.String()
}

func isASCIILetter(c byte) bool { return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') }
func isASCIIDigit(c byte) bool  { return c >= '0' && c <= '9' }
