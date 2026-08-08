package gdrive

import (
	"context"
	"io"
	"strings"
	"time"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/Southclaws/fault/ftag"
)

// MockCredentialsValue selects an in-memory Drive with a small fixed tree,
// instead of talking to Google. It exists so the browser can be exercised
// end-to-end and demonstrated locally without a Google Cloud project, following
// the same convention as EMAIL_PROVIDER=mock and LANGUAGE_MODEL_PROVIDER=mock.
const MockCredentialsValue = "mock"

const (
	MockRootFolderID = "mock-root-folder"
	MockSubFolderID  = "mock-sub-folder"
	MockTextFileID   = "mock-notes-file"
	MockPDFFileID    = "mock-handout-file"
)

var mockModified = time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)

var mockFiles = map[string]File{
	MockRootFolderID: {ID: MockRootFolderID, Name: "Mock Drive", MimeType: MimeFolder},
	MockSubFolderID:  {ID: MockSubFolderID, Name: "Handouts", MimeType: MimeFolder, Parents: []string{MockRootFolderID}, ModifiedTime: mockModified},
	MockTextFileID:   {ID: MockTextFileID, Name: "notes.txt", MimeType: "text/plain", Size: 22, Parents: []string{MockRootFolderID}, ModifiedTime: mockModified},
	MockPDFFileID:    {ID: MockPDFFileID, Name: "handout.pdf", MimeType: "application/pdf", Size: 17, Parents: []string{MockSubFolderID}, ModifiedTime: mockModified},
}

var mockContents = map[string]string{
	MockTextFileID: "mock google drive file",
	MockPDFFileID:  "%PDF-1.4 mock doc",
}

type mock struct{}

func (m *mock) Enabled() bool { return true }

func (m *mock) List(ctx context.Context, folderID string, pageToken string) (*Listing, error) {
	if _, ok := mockFiles[folderID]; !ok {
		return nil, mockNotFound(ctx)
	}

	files := []File{}
	for _, id := range []string{MockSubFolderID, MockTextFileID, MockPDFFileID} {
		f := mockFiles[id]
		if len(f.Parents) > 0 && f.Parents[0] == folderID {
			files = append(files, f)
		}
	}

	return &Listing{Files: files}, nil
}

func (m *mock) Get(ctx context.Context, fileID string) (*File, error) {
	f, ok := mockFiles[fileID]
	if !ok {
		return nil, mockNotFound(ctx)
	}

	return &f, nil
}

func (m *mock) Open(ctx context.Context, f *File) (io.ReadCloser, string, error) {
	content, ok := mockContents[f.ID]
	if !ok {
		return nil, "", mockNotFound(ctx)
	}

	return io.NopCloser(strings.NewReader(content)), f.MimeType, nil
}

func mockNotFound(ctx context.Context) error {
	return fault.New("mock drive file not found", fctx.With(ctx), ftag.With(ftag.NotFound))
}
