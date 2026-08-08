package drive_test

import (
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/internal/infrastructure/gdrive"
	"github.com/Southclaws/storyden/internal/infrastructure/gdrive/gdrivetest"
)

// The fixture models the shape that matters for authorisation: one folder an
// administrator registers, and a second folder the service account can also see
// but nobody registered. Nothing under the latter may ever be reachable.
const (
	rootDriveID    = "root0000000000000001"
	subDriveID     = "sub00000000000000001"
	notesFileID    = "file0000000000000001"
	deepFileID     = "file0000000000000002"
	nativeDocID    = "file0000000000000003"
	outsideRootID  = "outside000000000001"
	outsideFileID  = "file0000000000000004"
	notesContent   = "seminar notes"
	deepContent    = "%PDF-1.4 deep"
	outsideContent = "confidential"
)

func fakeDrive() *gdrivetest.Client {
	return gdrivetest.New().
		AddFolder(rootDriveID, "Course material", "").
		AddFolder(subDriveID, "Semester 1", rootDriveID).
		AddFile(notesFileID, "notes.txt", "text/plain", rootDriveID, []byte(notesContent)).
		AddFile(nativeDocID, "Syllabus", gdrive.MimeDocument, rootDriveID, nil).
		AddFile(deepFileID, "deep.pdf", "application/pdf", subDriveID, []byte(deepContent)).
		AddFolder(outsideRootID, "Private", "").
		AddFile(outsideFileID, "secret.pdf", "application/pdf", outsideRootID, []byte(outsideContent))
}

// withFakeDrive swaps the real Drive client out of the graph. The rest of the
// stack, including every authorisation check, runs exactly as in production.
func withFakeDrive(client *gdrivetest.Client) fx.Option {
	return fx.Decorate(func(gdrive.Client) gdrive.Client { return client })
}
