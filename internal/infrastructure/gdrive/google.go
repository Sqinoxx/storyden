package gdrive

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Southclaws/dt"
	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/Southclaws/fault/fmsg"
	"github.com/Southclaws/fault/ftag"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

const listFields = "nextPageToken, files(id, name, mimeType, size, modifiedTime, parents)"

const fileFields = "id, name, mimeType, size, modifiedTime, parents"

// listPageSize is Drive's maximum. Folder listings are cached, so fewer, larger
// pages cost less quota than paging politely.
const listPageSize = 1000

type googleClient struct {
	svc *drive.Service
}

func newGoogleClient(ctx context.Context, credentials []byte) (Client, error) {
	creds, err := google.CredentialsFromJSON(ctx, credentials, drive.DriveReadonlyScope)
	if err != nil {
		return nil, fault.Wrap(err,
			fmsg.WithDesc("parse service account credentials",
				"The Google Drive service account key is not a valid credentials file."))
	}

	svc, err := drive.NewService(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, fault.Wrap(err, fmsg.WithDesc("create drive service", "Could not connect to the Google Drive API."))
	}

	return &googleClient{svc: svc}, nil
}

func (c *googleClient) Enabled() bool { return true }

func (c *googleClient) List(ctx context.Context, folderID string, pageToken string) (*Listing, error) {
	ctx = fctx.WithMeta(ctx, "drive_folder_id", folderID)

	call := c.svc.Files.List().
		Q(listQuery(folderID)).
		Fields(listFields).
		OrderBy("folder,name").
		PageSize(listPageSize).
		SupportsAllDrives(true).
		IncludeItemsFromAllDrives(true).
		Context(ctx)

	if pageToken != "" {
		call = call.PageToken(pageToken)
	}

	res, err := call.Do()
	if err != nil {
		return nil, translateError(ctx, err)
	}

	return &Listing{
		Files:         dt.Map(res.Files, mapFile),
		NextPageToken: res.NextPageToken,
	}, nil
}

func (c *googleClient) Get(ctx context.Context, fileID string) (*File, error) {
	ctx = fctx.WithMeta(ctx, "drive_file_id", fileID)

	res, err := c.svc.Files.Get(fileID).
		Fields(fileFields).
		SupportsAllDrives(true).
		Context(ctx).
		Do()
	if err != nil {
		return nil, translateError(ctx, err)
	}

	f := mapFile(res)

	return &f, nil
}

func (c *googleClient) Open(ctx context.Context, f *File) (io.ReadCloser, string, error) {
	ctx = fctx.WithMeta(ctx, "drive_file_id", f.ID)

	if f.IsFolder() {
		return nil, "", fault.New("cannot open a folder",
			fctx.With(ctx),
			ftag.With(ftag.InvalidArgument),
			fmsg.WithDesc("open folder", "Folders cannot be downloaded."))
	}

	if f.IsNative() {
		exportAs, ok := ExportFormat(f.MimeType)
		if !ok {
			return nil, "", fault.New("no export format for native file",
				fctx.With(ctx),
				ftag.With(ftag.InvalidArgument),
				fmsg.WithDesc("unsupported native mime",
					"This Google Drive file type cannot be downloaded or previewed."))
		}

		res, err := c.svc.Files.Export(f.ID, exportAs).Context(ctx).Download()
		if err != nil {
			return nil, "", translateError(ctx, err)
		}

		return res.Body, exportAs, nil
	}

	res, err := c.svc.Files.Get(f.ID).
		SupportsAllDrives(true).
		Context(ctx).
		Download()
	if err != nil {
		return nil, "", translateError(ctx, err)
	}

	return res.Body, f.MimeType, nil
}

func listQuery(folderID string) string {
	// Drive query strings have no parameter binding, so the ID is escaped by
	// hand. IDs are opaque alphanumerics in practice, but a folder ID reaches
	// here from an administrator-supplied link.
	escaped := strings.NewReplacer(`\`, `\\`, `'`, `\'`).Replace(folderID)

	return "'" + escaped + "' in parents and trashed = false"
}

func mapFile(in *drive.File) File {
	modified, err := time.Parse(time.RFC3339, in.ModifiedTime)
	if err != nil {
		modified = time.Time{}
	}

	return File{
		ID:           in.Id,
		Name:         in.Name,
		MimeType:     in.MimeType,
		Size:         in.Size,
		ModifiedTime: modified,
		Parents:      in.Parents,
	}
}

// translateError collapses Drive's 403 and 404 into NotFound. A folder the
// service account has not been shared with is indistinguishable from one that
// does not exist, and saying so would confirm the existence of files outside
// this installation's reach.
func translateError(ctx context.Context, err error) error {
	var apiErr *googleapi.Error
	if !errors.As(err, &apiErr) {
		return fault.Wrap(err, fctx.With(ctx))
	}

	switch apiErr.Code {
	case http.StatusNotFound, http.StatusForbidden:
		return fault.Wrap(err,
			fctx.With(ctx),
			ftag.With(ftag.NotFound),
			fmsg.WithDesc("drive resource unavailable",
				"This file or folder is not available. It may have been removed, or it is no longer shared with this site."))

	case http.StatusTooManyRequests:
		return fault.Wrap(err,
			fctx.With(ctx),
			ftag.With(ftag.Internal),
			fmsg.WithDesc("drive quota exceeded",
				"Google Drive is rate limiting this site. Please try again shortly."))

	default:
		return fault.Wrap(err, fctx.With(ctx))
	}
}
