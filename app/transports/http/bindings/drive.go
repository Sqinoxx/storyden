package bindings

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/Southclaws/dt"
	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/Southclaws/fault/fmsg"
	"github.com/Southclaws/fault/ftag"
	"github.com/Southclaws/opt"
	"github.com/rs/xid"

	"github.com/Southclaws/storyden/app/resources/drive"
	"github.com/Southclaws/storyden/app/services/drive/drive_browse"
	"github.com/Southclaws/storyden/app/services/drive/drive_credentials"
	"github.com/Southclaws/storyden/app/services/drive/drive_manage"
	"github.com/Southclaws/storyden/app/transports/http/openapi"
	"github.com/Southclaws/storyden/internal/infrastructure/gdrive"
)

// driveCredentialsMaxBytes bounds the uploaded key well above any real
// service account key (a few KB) while still rejecting anything absurd
// before it's read fully into memory.
const driveCredentialsMaxBytes = 1 << 20 // 1MiB

// driveCacheControl is short and private. Listings come from an external
// service that can change at any moment, and folders may be restricted to
// members, so no shared cache may hold a copy.
const driveCacheControl = "private, max-age=60"

type Drive struct {
	manager     *drive_manage.Manager
	browser     *drive_browse.Browser
	credentials *drive_credentials.Resolver
}

func NewDrive(manager *drive_manage.Manager, browser *drive_browse.Browser, credentials *drive_credentials.Resolver) Drive {
	return Drive{manager: manager, browser: browser, credentials: credentials}
}

func (d *Drive) AdminDriveFolderList(ctx context.Context, request openapi.AdminDriveFolderListRequestObject) (openapi.AdminDriveFolderListResponseObject, error) {
	folders, err := d.manager.List(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return openapi.AdminDriveFolderList200JSONResponse{
		DriveFolderListOKJSONResponse: openapi.DriveFolderListOKJSONResponse{
			Folders:    dt.Map(folders, serialiseDriveFolder),
			Configured: d.manager.Configured(),
		},
	}, nil
}

func (d *Drive) AdminDriveFolderCreate(ctx context.Context, request openapi.AdminDriveFolderCreateRequestObject) (openapi.AdminDriveFolderCreateResponseObject, error) {
	visibility := drive.DriveVisibilityMember
	if request.Body.Visibility != nil {
		parsed, err := drive.NewDriveVisibility(string(*request.Body.Visibility))
		if err != nil {
			return nil, fault.Wrap(err, fctx.With(ctx))
		}

		visibility = parsed
	}

	folder, err := d.manager.Create(ctx, request.Body.Name, request.Body.Link, visibility, drive.Partial{
		Description: opt.NewPtr(request.Body.Description),
		Sort:        opt.NewPtr(request.Body.Sort),
	})
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return openapi.AdminDriveFolderCreate200JSONResponse{
		DriveFolderGetOKJSONResponse: openapi.DriveFolderGetOKJSONResponse(serialiseDriveFolder(folder)),
	}, nil
}

func (d *Drive) AdminDriveFolderUpdate(ctx context.Context, request openapi.AdminDriveFolderUpdateRequestObject) (openapi.AdminDriveFolderUpdateResponseObject, error) {
	id, err := xid.FromString(request.DriveFolderId)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	visibility, err := optDriveVisibility(request.Body.Visibility)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	folder, err := d.manager.Update(ctx, drive.FolderID(id), opt.NewPtr(request.Body.Link), drive.Partial{
		Name:        opt.NewPtr(request.Body.Name),
		Description: opt.NewPtr(request.Body.Description),
		Visibility:  visibility,
		Sort:        opt.NewPtr(request.Body.Sort),
	})
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return openapi.AdminDriveFolderUpdate200JSONResponse{
		DriveFolderGetOKJSONResponse: openapi.DriveFolderGetOKJSONResponse(serialiseDriveFolder(folder)),
	}, nil
}

func (d *Drive) AdminDriveFolderDelete(ctx context.Context, request openapi.AdminDriveFolderDeleteRequestObject) (openapi.AdminDriveFolderDeleteResponseObject, error) {
	id, err := xid.FromString(request.DriveFolderId)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	if err := d.manager.Delete(ctx, drive.FolderID(id)); err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return openapi.AdminDriveFolderDelete200Response{}, nil
}

func (d *Drive) AdminDriveCredentialsGet(ctx context.Context, request openapi.AdminDriveCredentialsGetRequestObject) (openapi.AdminDriveCredentialsGetResponseObject, error) {
	status, err := d.credentials.Status(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return openapi.AdminDriveCredentialsGet200JSONResponse{
		DriveCredentialsStatusOKJSONResponse: serialiseDriveCredentialsStatus(status),
	}, nil
}

func (d *Drive) AdminDriveCredentialsUpload(ctx context.Context, request openapi.AdminDriveCredentialsUploadRequestObject) (openapi.AdminDriveCredentialsUploadResponseObject, error) {
	// This route bypasses the OpenAPI validator so the body can be streamed,
	// which means its authorisation must be applied here instead.
	if err := AuthoriseOperation(ctx, "AdminDriveCredentialsUpload"); err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	limited := io.LimitReader(request.Body, driveCredentialsMaxBytes+1)

	credentials, err := io.ReadAll(limited)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx), ftag.With(ftag.InvalidArgument))
	}

	if len(credentials) > driveCredentialsMaxBytes {
		return nil, fault.New("service account key too large",
			fctx.With(ctx),
			ftag.With(ftag.InvalidArgument),
			fmsg.WithDesc("key too large", "The service account key file is too large."))
	}

	status, err := d.credentials.Upload(ctx, credentials)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return openapi.AdminDriveCredentialsUpload200JSONResponse{
		DriveCredentialsStatusOKJSONResponse: serialiseDriveCredentialsStatus(status),
	}, nil
}

func (d *Drive) AdminDriveCredentialsDelete(ctx context.Context, request openapi.AdminDriveCredentialsDeleteRequestObject) (openapi.AdminDriveCredentialsDeleteResponseObject, error) {
	status, err := d.credentials.Remove(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return openapi.AdminDriveCredentialsDelete200JSONResponse{
		DriveCredentialsStatusOKJSONResponse: serialiseDriveCredentialsStatus(status),
	}, nil
}

func serialiseDriveCredentialsStatus(s drive_credentials.Status) openapi.DriveCredentialsStatusOKJSONResponse {
	return openapi.DriveCredentialsStatusOKJSONResponse{
		Configured:          s.Configured,
		Source:              openapi.DriveCredentialsSource(s.Source),
		ServiceAccountEmail: opt.NewSafe(s.ServiceAccountEmail, s.ServiceAccountEmail != "").Ptr(),
	}
}

func (d *Drive) DriveFolderList(ctx context.Context, request openapi.DriveFolderListRequestObject) (openapi.DriveFolderListResponseObject, error) {
	folders, err := d.browser.ListRoots(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return openapi.DriveFolderList200JSONResponse{
		DriveFolderListOKJSONResponse: openapi.DriveFolderListOKJSONResponse{
			Folders:    dt.Map(folders, serialiseDriveFolder),
			Configured: d.browser.Enabled(),
		},
	}, nil
}

func (d *Drive) DriveFolderContents(ctx context.Context, request openapi.DriveFolderContentsRequestObject) (openapi.DriveFolderContentsResponseObject, error) {
	id, err := xid.FromString(request.DriveFolderId)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	contents, err := d.browser.Contents(ctx,
		drive.FolderID(id),
		opt.NewPtr(request.Params.ChildId).Or(""),
		opt.NewPtr(request.Params.PageToken).Or(""),
	)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return openapi.DriveFolderContents200JSONResponse{
		DriveFolderContentsOKJSONResponse: openapi.DriveFolderContentsOKJSONResponse{
			Folder: serialiseDriveFolder(contents.Folder),
			Breadcrumbs: dt.Map(contents.Breadcrumbs, func(c drive_browse.Crumb) openapi.DriveCrumb {
				return openapi.DriveCrumb{Id: c.ID, Name: c.Name}
			}),
			Entries:       dt.Map(contents.Entries, serialiseDriveEntry),
			NextPageToken: opt.NewSafe(contents.NextPage, contents.NextPage != "").Ptr(),
		},
	}, nil
}

func (d *Drive) DriveFileDownload(ctx context.Context, request openapi.DriveFileDownloadRequestObject) (openapi.DriveFileDownloadResponseObject, error) {
	id, err := xid.FromString(request.DriveFolderId)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	f, body, mime, err := d.browser.Open(ctx, drive.FolderID(id), request.DriveFileId)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	inline := request.Params.Disposition != nil &&
		*request.Params.Disposition == openapi.DriveFileDownloadParamsDisposition("inline")

	return openapi.DriveFileDownload200AsteriskResponse{
		DriveFileDownloadOKAsteriskResponse: openapi.DriveFileDownloadOKAsteriskResponse{
			Body:        body,
			ContentType: mime,
			// Zero suppresses the Content-Length header, so the body streams
			// chunked. Drive reports no length at all for exported documents,
			// and for the rest its metadata could disagree with the bytes that
			// actually arrive, which the server would answer by truncating the
			// response. Both paths behave the same way as a result.
			ContentLength: 0,
			Headers: openapi.DriveFileDownloadOKResponseHeaders{
				CacheControl:       driveCacheControl,
				ContentDisposition: driveContentDisposition(f.Name, mime, inline),
				// Drive content is authored outside this installation, so the
				// browser must not be allowed to reinterpret the type.
				XContentTypeOptions: "nosniff",
			},
		},
	}, nil
}

func serialiseDriveFolder(in *drive.Folder) openapi.DriveFolder {
	return openapi.DriveFolder{
		Id:            openapi.Identifier(in.ID.String()),
		CreatedAt:     in.CreatedAt,
		UpdatedAt:     in.UpdatedAt,
		Name:          in.Name,
		Description:   opt.NewSafe(in.Description, in.Description != "").Ptr(),
		DriveFolderId: in.DriveID,
		Visibility:    openapi.DriveVisibility(in.Visibility.String()),
		Sort:          in.Sort,
	}
}

func serialiseDriveEntry(in gdrive.File) openapi.DriveEntry {
	mime := gdrive.ServedMimeType(in)

	return openapi.DriveEntry{
		Id:       in.ID,
		Name:     in.Name,
		MimeType: mime,
		// A size of zero means Drive did not report one, which is the norm for
		// exported documents and for shortcuts.
		Size:        opt.NewSafe(in.Size, in.Size > 0).Ptr(),
		ModifiedAt:  opt.NewSafe(in.ModifiedTime, !in.ModifiedTime.IsZero()).Ptr(),
		IsFolder:    in.IsFolder(),
		Previewable: gdrive.Downloadable(in) && drivePreviewable(mime),
	}
}

// drivePreviewable answers whether the client's preview UI can display the file
// at all. This is deliberately wider than the inline Content-Disposition
// allowlist below: text is fetched and rendered by the client itself, which
// never asks the browser to interpret the response.
//
// HTML and SVG are excluded because both carry markup the browser would run if
// it ever did reach a document context.
func drivePreviewable(mime string) bool {
	normalised, _, _ := strings.Cut(strings.ToLower(mime), ";")
	normalised = strings.TrimSpace(normalised)

	switch normalised {
	case "text/html", "image/svg+xml":
		return false

	case "application/pdf", "application/json":
		return true
	}

	return strings.HasPrefix(normalised, "image/") || strings.HasPrefix(normalised, "text/")
}

// driveContentDisposition honours an inline request only for types a browser
// renders without executing them. Drive folders are curated by administrators
// but the files inside are not, and this endpoint shares an origin with the
// rest of the API.
func driveContentDisposition(name string, mime string, inline bool) string {
	safe := strings.ReplaceAll(name, `"`, "")

	if inline && inlineRenderableMIMEs[strings.ToLower(mime)] {
		return fmt.Sprintf(`inline; filename="%s"`, safe)
	}

	return fmt.Sprintf(`attachment; filename="%s"`, safe)
}

func optDriveVisibility(in *openapi.DriveVisibility) (opt.Optional[drive.DriveVisibility], error) {
	if in == nil {
		return opt.NewEmpty[drive.DriveVisibility](), nil
	}

	v, err := drive.NewDriveVisibility(string(*in))
	if err != nil {
		return opt.NewEmpty[drive.DriveVisibility](), err
	}

	return opt.New(v), nil
}
