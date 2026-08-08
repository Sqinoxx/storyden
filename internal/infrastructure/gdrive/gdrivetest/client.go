// Package gdrivetest provides an in-memory Google Drive stand-in so the folder
// browser can be tested without network access or real credentials.
package gdrivetest

import (
	"context"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/Southclaws/fault/ftag"

	"github.com/Southclaws/storyden/internal/infrastructure/gdrive"
)

var _ gdrive.Client = (*Client)(nil)

type Entry struct {
	File    gdrive.File
	Content []byte
}

type Client struct {
	mu      sync.Mutex
	entries map[string]*Entry
}

func New() *Client {
	return &Client{entries: map[string]*Entry{}}
}

// AddFolder registers a folder. Pass an empty parent for a Drive root.
func (c *Client) AddFolder(id, name, parent string) *Client {
	return c.add(id, name, gdrive.MimeFolder, parent, nil)
}

func (c *Client) AddFile(id, name, mime, parent string, content []byte) *Client {
	return c.add(id, name, mime, parent, content)
}

func (c *Client) add(id, name, mime, parent string, content []byte) *Client {
	c.mu.Lock()
	defer c.mu.Unlock()

	parents := []string{}
	if parent != "" {
		parents = []string{parent}
	}

	c.entries[id] = &Entry{
		File: gdrive.File{
			ID:           id,
			Name:         name,
			MimeType:     mime,
			Size:         int64(len(content)),
			ModifiedTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			Parents:      parents,
		},
		Content: content,
	}

	return c
}

func (c *Client) Enabled() bool { return true }

func (c *Client) List(ctx context.Context, folderID string, pageToken string) (*gdrive.Listing, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.entries[folderID]; !ok {
		return nil, notFound(ctx)
	}

	files := []gdrive.File{}
	for _, e := range c.entries {
		for _, p := range e.File.Parents {
			if p == folderID {
				files = append(files, e.File)
			}
		}
	}

	// Mirror Drive's "folder,name" ordering so tests can assert on position.
	sortEntries(files)

	return &gdrive.Listing{Files: files}, nil
}

func (c *Client) Get(ctx context.Context, fileID string) (*gdrive.File, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	e, ok := c.entries[fileID]
	if !ok {
		return nil, notFound(ctx)
	}

	f := e.File

	return &f, nil
}

func (c *Client) Open(ctx context.Context, f *gdrive.File) (io.ReadCloser, string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	e, ok := c.entries[f.ID]
	if !ok {
		return nil, "", notFound(ctx)
	}

	if f.IsNative() {
		return io.NopCloser(strings.NewReader("%PDF-1.4 exported")), "application/pdf", nil
	}

	return io.NopCloser(strings.NewReader(string(e.Content))), e.File.MimeType, nil
}

func sortEntries(files []gdrive.File) {
	rank := func(f gdrive.File) int {
		if f.IsFolder() {
			return 0
		}
		return 1
	}

	for i := 1; i < len(files); i++ {
		for j := i; j > 0; j-- {
			a, b := files[j-1], files[j]
			if rank(a) < rank(b) || (rank(a) == rank(b) && a.Name <= b.Name) {
				break
			}
			files[j-1], files[j] = b, a
		}
	}
}

func notFound(ctx context.Context) error {
	return fault.New("not found", fctx.With(ctx), ftag.With(ftag.NotFound))
}
