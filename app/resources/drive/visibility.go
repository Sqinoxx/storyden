package drive

import "github.com/Southclaws/storyden/internal/ent/drivefolder"

//go:generate go run -mod=mod github.com/Southclaws/enumerator

type driveVisibilityEnum string

const (
	driveVisibilityPublic driveVisibilityEnum = "public"
	driveVisibilityMember driveVisibilityEnum = "member"
	driveVisibilityAdmin  driveVisibilityEnum = "admin"
)

func NewDriveVisibilityFromEnt(in drivefolder.Visibility) DriveVisibility {
	return DriveVisibility{driveVisibilityEnum(in)}
}

func (r DriveVisibility) Ent() drivefolder.Visibility {
	return drivefolder.Visibility(r.v)
}
