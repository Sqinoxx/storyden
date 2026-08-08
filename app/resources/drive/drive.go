package drive

import (
	"time"

	"github.com/rs/xid"

	"github.com/Southclaws/storyden/app/resources/account"
	"github.com/Southclaws/storyden/internal/ent"
)

type FolderID xid.ID

func (i FolderID) String() string { return xid.ID(i).String() }

// Folder is a Google Drive folder an administrator has made browsable. It holds
// no file data of its own; it is the root that authorises everything beneath it.
type Folder struct {
	ID          FolderID
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Name        string
	Description string
	DriveID     string
	Visibility  DriveVisibility
	Sort        int
	AddedBy     account.AccountID
}

type Folders []*Folder

func Map(in *ent.DriveFolder) *Folder {
	return &Folder{
		ID:          FolderID(in.ID),
		CreatedAt:   in.CreatedAt,
		UpdatedAt:   in.UpdatedAt,
		Name:        in.Name,
		Description: in.Description,
		DriveID:     in.DriveFolderID,
		Visibility:  NewDriveVisibilityFromEnt(in.Visibility),
		Sort:        in.Sort,
		AddedBy:     account.AccountID(in.AddedBy),
	}
}
