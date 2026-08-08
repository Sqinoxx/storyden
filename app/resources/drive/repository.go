package drive

import (
	"context"
	"time"

	"github.com/Southclaws/dt"
	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/Southclaws/fault/fmsg"
	"github.com/Southclaws/fault/ftag"
	"github.com/Southclaws/opt"
	"github.com/rs/xid"

	"github.com/Southclaws/storyden/app/resources/account"
	"github.com/Southclaws/storyden/internal/ent"
	ent_drivefolder "github.com/Southclaws/storyden/internal/ent/drivefolder"
)

type Repository struct {
	db *ent.Client
}

func New(db *ent.Client) *Repository {
	return &Repository{db}
}

type Partial struct {
	Name        opt.Optional[string]
	Description opt.Optional[string]
	DriveID     opt.Optional[string]
	Visibility  opt.Optional[DriveVisibility]
	Sort        opt.Optional[int]
}

func (p Partial) apply(m *ent.DriveFolderMutation) {
	p.Name.Call(m.SetName)
	p.Description.Call(m.SetDescription)
	p.DriveID.Call(m.SetDriveFolderID)
	p.Visibility.Call(func(v DriveVisibility) { m.SetVisibility(v.Ent()) })
	p.Sort.Call(m.SetSort)
}

func (r *Repository) Create(ctx context.Context, addedBy account.AccountID, name string, driveID string, visibility DriveVisibility, p Partial) (*Folder, error) {
	if err := r.assertDriveIDFree(ctx, driveID, nil); err != nil {
		return nil, err
	}

	create := r.db.DriveFolder.Create().
		SetName(name).
		SetDriveFolderID(driveID).
		SetVisibility(visibility.Ent()).
		SetAddedBy(xid.ID(addedBy))

	p.apply(create.Mutation())

	result, err := create.Save(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return Map(result), nil
}

func (r *Repository) Update(ctx context.Context, id FolderID, p Partial) (*Folder, error) {
	if driveID, ok := p.DriveID.Get(); ok {
		if err := r.assertDriveIDFree(ctx, driveID, &id); err != nil {
			return nil, err
		}
	}

	update := r.db.DriveFolder.UpdateOneID(xid.ID(id))

	p.apply(update.Mutation())

	result, err := update.Save(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fault.Wrap(err, fctx.With(ctx), ftag.With(ftag.NotFound))
		}

		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return Map(result), nil
}

func (r *Repository) List(ctx context.Context) (Folders, error) {
	result, err := r.db.DriveFolder.Query().
		Where(ent_drivefolder.DeletedAtIsNil()).
		Order(ent.Asc(ent_drivefolder.FieldSort), ent.Asc(ent_drivefolder.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return dt.Map(result, Map), nil
}

func (r *Repository) Get(ctx context.Context, id FolderID) (*Folder, error) {
	result, err := r.db.DriveFolder.Query().
		Where(
			ent_drivefolder.ID(xid.ID(id)),
			ent_drivefolder.DeletedAtIsNil(),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fault.Wrap(err, fctx.With(ctx), ftag.With(ftag.NotFound))
		}

		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return Map(result), nil
}

// Delete soft-deletes the folder. Nothing is removed from Google Drive; this
// only withdraws the folder from the site.
func (r *Repository) Delete(ctx context.Context, id FolderID) error {
	affected, err := r.db.DriveFolder.Update().
		Where(
			ent_drivefolder.ID(xid.ID(id)),
			ent_drivefolder.DeletedAtIsNil(),
		).
		SetDeletedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return fault.Wrap(err, fctx.With(ctx))
	}

	if affected == 0 {
		return fault.New("drive folder not found", fctx.With(ctx), ftag.With(ftag.NotFound))
	}

	return nil
}

// assertDriveIDFree enforces uniqueness across live rows only. A database
// constraint cannot express this, because soft-deleted rows keep their
// drive_folder_id and would otherwise block re-adding a folder that was removed.
func (r *Repository) assertDriveIDFree(ctx context.Context, driveID string, excluding *FolderID) error {
	q := r.db.DriveFolder.Query().Where(
		ent_drivefolder.DriveFolderID(driveID),
		ent_drivefolder.DeletedAtIsNil(),
	)

	if excluding != nil {
		q = q.Where(ent_drivefolder.IDNEQ(xid.ID(*excluding)))
	}

	exists, err := q.Exist(ctx)
	if err != nil {
		return fault.Wrap(err, fctx.With(ctx))
	}

	if exists {
		return fault.New("drive folder already added",
			fctx.With(ctx),
			ftag.With(ftag.AlreadyExists),
			fmsg.WithDesc("duplicate drive folder", "This Google Drive folder has already been added."))
	}

	return nil
}
