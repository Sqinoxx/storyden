package asset_import

import (
	"bytes"
	"context"
	"fmt"
	"io"
	stdmime "mime"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/Southclaws/fault/fmsg"
	"github.com/Southclaws/fault/ftag"

	"github.com/Southclaws/storyden/app/resources/asset"
	"github.com/Southclaws/storyden/app/resources/settings"
	"github.com/Southclaws/storyden/app/services/asset/asset_upload"
	"github.com/Southclaws/storyden/internal/config"
	"github.com/Southclaws/storyden/internal/infrastructure/httpsafe"
	"github.com/Southclaws/storyden/internal/mime"
)

const (
	fetchTimeout          = 60 * time.Second
	defaultMaxUploadBytes = 50 * 1024 * 1024
	userAgent             = `Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36`
)

var allowedMIMEs = map[string]string{
	"application/pdf":    "pdf",
	"application/msword": "doc",
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document":   "docx",
	"application/vnd.ms-powerpoint":                                             "ppt",
	"application/vnd.openxmlformats-officedocument.presentationml.presentation": "pptx",
	"application/vnd.ms-excel":                                                  "xls",
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":         "xlsx",
}

var (
	errNotAFile = fmsg.WithDesc("not a file",
		"Unter diesem Link liegt keine Datei, sondern eine Webseite. Ist der Link öffentlich freigegeben?")
	errUnsupported = fmsg.WithDesc("unsupported type",
		"Nur PDF-, Word-, PowerPoint- und Excel-Dateien können über einen Link hochgeladen werden.")
	errTooLarge = fmsg.WithDesc("too large",
		"Die Datei ist größer als das Upload-Limit.")
	errUnreachable = fmsg.WithDesc("unreachable",
		"Die Datei konnte nicht heruntergeladen werden.")
	errInvalidURL = fmsg.WithDesc("invalid url",
		"Das ist kein gültiger Link.")
)

type Importer struct {
	client   *http.Client
	uploader *asset_upload.Uploader
	settings *settings.SettingsRepository
	cfg      config.Config
}

func New(uploader *asset_upload.Uploader, settings *settings.SettingsRepository, cfg config.Config) *Importer {
	return NewWithClient(httpsafe.NewClient(httpsafe.Config{Timeout: fetchTimeout}), uploader, settings, cfg)
}

func NewWithClient(client *http.Client, uploader *asset_upload.Uploader, settings *settings.SettingsRepository, cfg config.Config) *Importer {
	return &Importer{
		client:   client,
		uploader: uploader,
		settings: settings,
		cfg:      cfg,
	}
}

type Imported struct {
	Asset *asset.Asset
	Name  string
}

func (i *Importer) Import(ctx context.Context, raw string) (*Imported, error) {
	u, err := Resolve(raw)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx), ftag.With(ftag.InvalidArgument), errInvalidURL)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx), ftag.With(ftag.InvalidArgument), errInvalidURL)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := i.client.Do(req)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx), ftag.With(ftag.InvalidArgument), errUnreachable)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		ctx = fctx.WithMeta(ctx, "status", resp.Status)
		return nil, fault.New("import source did not return the file", fctx.With(ctx), ftag.With(ftag.InvalidArgument), errUnreachable)
	}

	if ct, _, _ := stdmime.ParseMediaType(resp.Header.Get("Content-Type")); ct == "text/html" {
		return nil, fault.New("import source returned a web page", fctx.With(ctx), ftag.With(ftag.InvalidArgument), errNotAFile)
	}

	maxBytes := i.maxUploadBytes(ctx)
	if resp.ContentLength > maxBytes {
		return nil, fault.New("import source exceeds upload limit", fctx.With(ctx), ftag.With(ftag.InvalidArgument), errTooLarge)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx), ftag.With(ftag.InvalidArgument), errUnreachable)
	}
	if int64(len(data)) > maxBytes {
		return nil, fault.New("import source exceeds upload limit", fctx.With(ctx), ftag.With(ftag.InvalidArgument), errTooLarge)
	}

	mt, _, err := mime.Detect(bytes.NewReader(data))
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx), ftag.With(ftag.InvalidArgument), errUnreachable)
	}

	ext, ok := allowedMIMEs[mt.String()]
	if !ok {
		ctx = fctx.WithMeta(ctx, "mime", mt.String())
		return nil, fault.New("import source is not an allowed document type", fctx.With(ctx), ftag.With(ftag.InvalidArgument), errUnsupported)
	}

	name := filename(resp.Header.Get("Content-Disposition"), u, ext)

	a, err := i.uploader.Upload(ctx, bytes.NewReader(data), int64(len(data)), asset.NewFilename(name), asset_upload.Options{})
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return &Imported{Asset: a, Name: name}, nil
}

func (i *Importer) maxUploadBytes(ctx context.Context) int64 {
	limit := int64(defaultMaxUploadBytes)
	if i.cfg.MaxUploadSizeMB > 0 {
		limit = int64(i.cfg.MaxUploadSizeMB) * 1024 * 1024
	}

	if i.settings == nil {
		return limit
	}

	s, err := i.settings.Get(ctx)
	if err != nil {
		return limit
	}
	if v, ok := s.Services.OrZero().Assets.OrZero().MaxUploadSizeMB.Get(); ok && v > 0 {
		return int64(v) * 1024 * 1024
	}

	return limit
}

var googleFileID = regexp.MustCompile(`^[A-Za-z0-9_-]{10,}$`)

// Resolve turns share links into a URL that serves the file itself.
func Resolve(raw string) (*url.URL, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return nil, err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fault.Newf("unsupported scheme %q", u.Scheme)
	}
	if u.Host == "" {
		return nil, fault.New("missing host")
	}

	segments := strings.Split(strings.Trim(u.Path, "/"), "/")
	idAfter := func(marker string) string {
		for i, s := range segments {
			if s == marker && i+1 < len(segments) && googleFileID.MatchString(segments[i+1]) {
				return segments[i+1]
			}
		}
		return ""
	}

	switch u.Host {
	case "drive.google.com":
		id := u.Query().Get("id")
		if !googleFileID.MatchString(id) {
			id = idAfter("d")
		}
		if id == "" {
			return u, nil
		}
		return &url.URL{
			Scheme:   "https",
			Host:     "drive.usercontent.google.com",
			Path:     "/download",
			RawQuery: url.Values{"id": {id}, "export": {"download"}}.Encode(),
		}, nil

	case "docs.google.com":
		if len(segments) < 3 {
			return u, nil
		}
		kind := segments[0]
		if kind != "document" && kind != "presentation" && kind != "spreadsheets" {
			return u, nil
		}
		id := idAfter("d")
		if id == "" {
			return u, nil
		}
		return &url.URL{
			Scheme:   "https",
			Host:     "docs.google.com",
			Path:     fmt.Sprintf("/%s/d/%s/export", kind, id),
			RawQuery: url.Values{"format": {"pdf"}}.Encode(),
		}, nil
	}

	return u, nil
}

func filename(contentDisposition string, u *url.URL, ext string) string {
	if _, params, err := stdmime.ParseMediaType(contentDisposition); err == nil {
		if name := strings.TrimSpace(params["filename"]); name != "" {
			return path.Base(name)
		}
	}

	if base := path.Base(u.Path); base != "." && base != "/" && strings.Contains(base, ".") {
		if unescaped, err := url.PathUnescape(base); err == nil {
			return unescaped
		}
		return base
	}

	return "dokument." + ext
}
