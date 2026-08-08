// Package drive_browse serves the member-facing side of the Google Drive
// browser. Its central responsibility is that nothing outside a folder an
// administrator explicitly registered is ever readable, even though the service
// account can see every folder shared with it.
package drive_browse

import (
	"context"
	"io"

	"github.com/Southclaws/dt"
	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/Southclaws/fault/fmsg"
	"github.com/Southclaws/fault/ftag"

	"github.com/Southclaws/storyden/app/resources/drive"
	"github.com/Southclaws/storyden/app/resources/rbac"
	"github.com/Southclaws/storyden/app/services/authentication/session"
	"github.com/Southclaws/storyden/internal/config"
	"github.com/Southclaws/storyden/internal/infrastructure/cache"
	"github.com/Southclaws/storyden/internal/infrastructure/gdrive"
)

type Crumb struct {
	ID   string
	Name string
}

type Contents struct {
	Folder      *drive.Folder
	Breadcrumbs []Crumb
	Entries     []gdrive.File
	NextPage    string
}

type Browser struct {
	repo   *drive.Repository
	client gdrive.Client
	cache  *listCache
}

func New(repo *drive.Repository, client gdrive.Client, store cache.Store, cfg config.Config) *Browser {
	return &Browser{
		repo:   repo,
		client: client,
		cache:  newListCache(store, cfg.GoogleDriveCacheTTL),
	}
}

func (b *Browser) Enabled() bool { return b.client.Enabled() }

// ListRoots returns the registered folders the caller is allowed to see.
func (b *Browser) ListRoots(ctx context.Context) (drive.Folders, error) {
	all, err := b.repo.List(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	admin := isAdmin(ctx)
	_, member := session.GetOptAccountID(ctx).Get()

	return dt.Filter(all, func(f *drive.Folder) bool {
		return permitted(f.Visibility, member, admin)
	}), nil
}

func (b *Browser) Contents(ctx context.Context, id drive.FolderID, childID string, pageToken string) (*Contents, error) {
	root, err := b.resolveRoot(ctx, id)
	if err != nil {
		return nil, err
	}

	target := root.DriveID
	// The root crumb carries the administrator's chosen name rather than the
	// folder's name in Drive, which is what members see everywhere else.
	crumbs := []Crumb{{ID: root.DriveID, Name: root.Name}}

	if childID != "" && childID != root.DriveID {
		// Ancestry runs before anything else is reported about the target, so
		// that an ID outside the registered folder is answered identically
		// whether it names a file, a folder, or nothing at all.
		descendants, err := b.ancestry(ctx, root.DriveID, childID)
		if err != nil {
			return nil, err
		}

		child, err := b.client.Get(ctx, childID)
		if err != nil {
			return nil, fault.Wrap(err, fctx.With(ctx))
		}

		// Listing a file ID would otherwise succeed and render as an empty
		// folder, because Drive answers a parent query for it with no rows.
		if !child.IsFolder() {
			return nil, fault.New("not a folder",
				fctx.With(ctx),
				ftag.With(ftag.InvalidArgument),
				fmsg.WithDesc("list file", "That is a file, not a folder."))
		}

		target = childID
		crumbs = append(crumbs, descendants...)
	}

	listing, err := b.list(ctx, target, pageToken)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return &Contents{
		Folder:      root,
		Breadcrumbs: crumbs,
		Entries:     listing.Files,
		NextPage:    listing.NextPageToken,
	}, nil
}

// Open streams a file's contents. The returned MIME type is what the bytes
// actually are, which for Google-native documents is the exported format.
func (b *Browser) Open(ctx context.Context, id drive.FolderID, fileID string) (*gdrive.File, io.ReadCloser, string, error) {
	root, err := b.resolveRoot(ctx, id)
	if err != nil {
		return nil, nil, "", err
	}

	f, err := b.client.Get(ctx, fileID)
	if err != nil {
		return nil, nil, "", fault.Wrap(err, fctx.With(ctx))
	}

	if f.IsFolder() {
		return nil, nil, "", fault.New("cannot download a folder",
			fctx.With(ctx),
			ftag.With(ftag.InvalidArgument),
			fmsg.WithDesc("download folder", "Folders cannot be downloaded."))
	}

	if _, err := b.ancestry(ctx, root.DriveID, fileID); err != nil {
		return nil, nil, "", err
	}

	body, mime, err := b.client.Open(ctx, f)
	if err != nil {
		return nil, nil, "", fault.Wrap(err, fctx.With(ctx))
	}

	return f, body, mime, nil
}

func (b *Browser) resolveRoot(ctx context.Context, id drive.FolderID) (*drive.Folder, error) {
	folder, err := b.repo.Get(ctx, id)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	_, member := session.GetOptAccountID(ctx).Get()

	if !permitted(folder.Visibility, member, isAdmin(ctx)) {
		// Reported as missing rather than forbidden so the existence of an
		// admin-only folder is not observable to members.
		return nil, fault.New("drive folder not visible",
			fctx.With(ctx),
			ftag.With(ftag.NotFound),
			fmsg.WithDesc("visibility", "This folder is not available."))
	}

	return folder, nil
}

func (b *Browser) list(ctx context.Context, folderID string, pageToken string) (*gdrive.Listing, error) {
	if cached, ok := b.cache.getListing(ctx, folderID, pageToken); ok {
		return cached, nil
	}

	listing, err := b.client.List(ctx, folderID, pageToken)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	b.cache.setListing(ctx, folderID, pageToken, listing)

	return listing, nil
}

func permitted(v drive.DriveVisibility, member bool, admin bool) bool {
	switch v {
	case drive.DriveVisibilityPublic:
		return true

	case drive.DriveVisibilityMember:
		return member || admin

	default:
		return admin
	}
}

func isAdmin(ctx context.Context) bool {
	perms, err := session.GetPermissions(ctx)
	if err != nil {
		return false
	}

	return perms.HasAny(rbac.PermissionManageSettings, rbac.PermissionAdministrator)
}
