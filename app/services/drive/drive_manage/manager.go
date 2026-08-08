// Package drive_manage handles the administrative side of the Google Drive
// browser: which folders an installation exposes and to whom.
package drive_manage

import (
	"context"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/Southclaws/fault/fmsg"
	"github.com/Southclaws/fault/ftag"
	"github.com/Southclaws/opt"

	"github.com/Southclaws/storyden/app/resources/drive"
	"github.com/Southclaws/storyden/app/resources/rbac"
	"github.com/Southclaws/storyden/app/services/authentication/session"
	"github.com/Southclaws/storyden/internal/infrastructure/gdrive"
)

type Manager struct {
	repo   *drive.Repository
	client gdrive.Client
}

func New(repo *drive.Repository, client gdrive.Client) *Manager {
	return &Manager{repo: repo, client: client}
}

// Configured reports whether credentials are present. Registering folders is
// allowed without them so an administrator can prepare the list, but nothing
// inside can be read until they are set.
func (m *Manager) Configured() bool { return m.client.Enabled() }

func (m *Manager) List(ctx context.Context) (drive.Folders, error) {
	if err := authorise(ctx); err != nil {
		return nil, err
	}

	folders, err := m.repo.List(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return folders, nil
}

func (m *Manager) Create(ctx context.Context, name string, link string, visibility drive.DriveVisibility, p drive.Partial) (*drive.Folder, error) {
	if err := authorise(ctx); err != nil {
		return nil, err
	}

	accountID, err := session.GetAccountID(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	driveID, err := drive.ParseFolderID(link)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	if err := m.assertReachable(ctx, driveID); err != nil {
		return nil, err
	}

	p.DriveID = opt.NewEmpty[string]()

	folder, err := m.repo.Create(ctx, accountID, name, driveID, visibility, p)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return folder, nil
}

// Update accepts the link in its raw form so callers do not have to know how
// Drive share URLs are shaped. A link is only re-validated when it changes.
func (m *Manager) Update(ctx context.Context, id drive.FolderID, link opt.Optional[string], p drive.Partial) (*drive.Folder, error) {
	if err := authorise(ctx); err != nil {
		return nil, err
	}

	if raw, ok := link.Get(); ok {
		driveID, err := drive.ParseFolderID(raw)
		if err != nil {
			return nil, fault.Wrap(err, fctx.With(ctx))
		}

		existing, err := m.repo.Get(ctx, id)
		if err != nil {
			return nil, fault.Wrap(err, fctx.With(ctx))
		}

		if driveID != existing.DriveID {
			if err := m.assertReachable(ctx, driveID); err != nil {
				return nil, err
			}

			p.DriveID = opt.New(driveID)
		}
	}

	folder, err := m.repo.Update(ctx, id, p)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return folder, nil
}

func (m *Manager) Delete(ctx context.Context, id drive.FolderID) error {
	if err := authorise(ctx); err != nil {
		return err
	}

	if err := m.repo.Delete(ctx, id); err != nil {
		return fault.Wrap(err, fctx.With(ctx))
	}

	return nil
}

// assertReachable fails early with an actionable message. Without it the folder
// would save cleanly and then appear permanently empty to members, which gives
// an administrator nothing to act on.
func (m *Manager) assertReachable(ctx context.Context, driveID string) error {
	if !m.client.Enabled() {
		return fault.New("google drive not configured",
			fctx.With(ctx),
			ftag.With(ftag.InvalidArgument),
			fmsg.WithDesc("gdrive disabled",
				"Google Drive is not configured on this installation. A service account key must be set first."))
	}

	f, err := m.client.Get(ctx, driveID)
	if err != nil {
		return fault.Wrap(err,
			fctx.With(ctx),
			ftag.With(ftag.InvalidArgument),
			fmsg.WithDesc("drive folder unreachable",
				"This folder could not be read. Share it with the service account's email address in Google Drive, then try again."))
	}

	if !f.IsFolder() {
		return fault.New("drive id is not a folder",
			fctx.With(ctx),
			ftag.With(ftag.InvalidArgument),
			fmsg.WithDesc("not a folder", "That link points to a file, not a folder."))
	}

	return nil
}

func authorise(ctx context.Context) error {
	if err := session.Authorise(ctx, nil, rbac.PermissionManageSettings, rbac.PermissionAdministrator); err != nil {
		return fault.Wrap(err, fctx.With(ctx))
	}

	return nil
}
