// Package gdrive provides read-only access to Google Drive through a service
// account. Folders are shared with the service account's email address by an
// administrator, which is what grants Storyden visibility of them.
package gdrive

import (
	"context"
	"encoding/base64"
	"io"
	"os"
	"strings"
	"time"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/Southclaws/fault/fmsg"
	"github.com/Southclaws/fault/ftag"
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/internal/config"
)

const (
	MimeFolder       = "application/vnd.google-apps.folder"
	MimeDocument     = "application/vnd.google-apps.document"
	MimeSpreadsheet  = "application/vnd.google-apps.spreadsheet"
	MimePresentation = "application/vnd.google-apps.presentation"
	MimeDrawing      = "application/vnd.google-apps.drawing"
)

type File struct {
	ID           string
	Name         string
	MimeType     string
	Size         int64
	ModifiedTime time.Time
	Parents      []string
}

func (f File) IsFolder() bool {
	return f.MimeType == MimeFolder
}

// IsNative reports whether the file is a Google-native document. These hold no
// bytes of their own and must be exported to a portable format to be read.
func (f File) IsNative() bool {
	return strings.HasPrefix(f.MimeType, "application/vnd.google-apps.")
}

type Listing struct {
	Files         []File
	NextPageToken string
}

// exportFormats maps Google-native documents onto a portable format that both
// the browser preview and a plain download can handle.
var exportFormats = map[string]string{
	MimeDocument:     "application/pdf",
	MimeSpreadsheet:  "application/pdf",
	MimePresentation: "application/pdf",
	MimeDrawing:      "image/png",
}

func ExportFormat(mime string) (string, bool) {
	format, ok := exportFormats[mime]

	return format, ok
}

// ServedMimeType is the type a caller will actually receive, which differs from
// the file's own type for Google-native documents. Listings report this rather
// than the Drive type so a client can decide up front whether it can render the
// file, without knowing anything about Google's export rules.
func ServedMimeType(f File) string {
	if format, ok := exportFormats[f.MimeType]; ok {
		return format
	}

	return f.MimeType
}

// Downloadable reports whether the file can be read at all. Google-native types
// with no export format, such as Forms and Sites, cannot.
func Downloadable(f File) bool {
	if f.IsFolder() {
		return false
	}

	if !f.IsNative() {
		return true
	}

	_, ok := exportFormats[f.MimeType]

	return ok
}

type Client interface {
	// Enabled reports whether credentials were configured. Callers use this to
	// hide the feature rather than surfacing errors for an unconfigured
	// instance.
	Enabled() bool

	List(ctx context.Context, folderID string, pageToken string) (*Listing, error)
	Get(ctx context.Context, fileID string) (*File, error)

	// Open returns the file contents along with the MIME type they are actually
	// served as, which differs from File.MimeType for Google-native documents.
	Open(ctx context.Context, f *File) (io.ReadCloser, string, error)
}

func Build() fx.Option {
	return fx.Provide(func(ctx context.Context, cfg config.Config) (Client, error) {
		if strings.TrimSpace(cfg.GoogleDriveServiceAccountJSON) == MockCredentialsValue {
			return &mock{}, nil
		}

		credentials, err := readCredentials(cfg)
		if err != nil {
			return nil, err
		}

		if credentials == nil {
			return &disabled{}, nil
		}

		return newGoogleClient(ctx, credentials)
	})
}

func readCredentials(cfg config.Config) ([]byte, error) {
	if raw := strings.TrimSpace(cfg.GoogleDriveServiceAccountJSON); raw != "" {
		// The key is JSON, so a leading brace means it was passed verbatim and
		// anything else is expected to be the base64 form.
		if strings.HasPrefix(raw, "{") {
			return []byte(raw), nil
		}

		decoded, err := base64.StdEncoding.DecodeString(raw)
		if err != nil {
			return nil, fault.Wrap(err,
				fmsg.WithDesc("decode service account key",
					"GOOGLE_DRIVE_SERVICE_ACCOUNT_JSON is neither raw JSON nor valid base64."))
		}

		return decoded, nil
	}

	if path := strings.TrimSpace(cfg.GoogleDriveServiceAccountFile); path != "" {
		contents, err := os.ReadFile(path)
		if err != nil {
			return nil, fault.Wrap(err,
				fmsg.WithDesc("read service account key file",
					"GOOGLE_DRIVE_SERVICE_ACCOUNT_FILE could not be read."))
		}

		return contents, nil
	}

	return nil, nil
}

type disabled struct{}

var errDisabled = fault.New("google drive is not configured",
	ftag.With(ftag.NotFound),
	fmsg.WithDesc("gdrive disabled",
		"Google Drive integration is not configured on this installation."))

func (d *disabled) Enabled() bool { return false }

func (d *disabled) List(ctx context.Context, folderID string, pageToken string) (*Listing, error) {
	return nil, fault.Wrap(errDisabled, fctx.With(ctx))
}

func (d *disabled) Get(ctx context.Context, fileID string) (*File, error) {
	return nil, fault.Wrap(errDisabled, fctx.With(ctx))
}

func (d *disabled) Open(ctx context.Context, f *File) (io.ReadCloser, string, error) {
	return nil, "", fault.Wrap(errDisabled, fctx.With(ctx))
}
