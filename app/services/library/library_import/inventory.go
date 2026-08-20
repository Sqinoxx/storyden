package library_import

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/Southclaws/fault/fmsg"

	"github.com/Southclaws/storyden/internal/mime"
)

type Entry struct {
	Path    string    `json:"path"`
	Size    int64     `json:"size"`
	SHA256  string    `json:"sha256"`
	MIME    string    `json:"mime"`
	ModTime time.Time `json:"mod_time"`
}

type Inventory struct {
	Root      string  `json:"root"`
	Entries   []Entry `json:"-"`
	TotalSize int64   `json:"total_size"`
}

type ScanOptions struct {
	// Demote lists path prefixes that lose the tie-break when the same content
	// appears more than once. The source tree keeps a near-complete second copy
	// of the lecture material under a differently named branch.
	Demote []string

	// Skip lists path prefixes excluded from the inventory entirely.
	Skip []string

	// NoHash skips content hashing, which makes a coverage check over a large
	// tree take seconds instead of hours. Deduplication needs hashes, so a
	// no-hash inventory is only useful for reviewing manifest rules.
	NoHash bool

	Progress func(files int, bytes int64)
}

func Scan(ctx context.Context, root string, opts ScanOptions) (*Inventory, error) {
	root = filepath.Clean(root)

	inv := &Inventory{Root: root}

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if d.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return fault.Wrap(err, fctx.With(ctx))
		}
		rel = filepath.ToSlash(rel)

		if hasAnyPrefix(rel, opts.Skip) || isIgnoredFile(rel) {
			return nil
		}

		entry, err := describeFile(path, rel, opts.NoHash)
		if err != nil {
			return fault.Wrap(err, fctx.With(ctx), fmsg.With("failed to read "+rel))
		}

		inv.Entries = append(inv.Entries, *entry)
		inv.TotalSize += entry.Size

		if opts.Progress != nil {
			opts.Progress(len(inv.Entries), inv.TotalSize)
		}

		return nil
	})
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	sort.Slice(inv.Entries, func(i, j int) bool { return inv.Entries[i].Path < inv.Entries[j].Path })

	return inv, nil
}

func describeFile(abs, rel string, noHash bool) (*Entry, error) {
	stat, err := os.Stat(abs)
	if err != nil {
		return nil, fault.Wrap(err)
	}

	f, err := os.Open(abs)
	if err != nil {
		return nil, fault.Wrap(err)
	}
	defer f.Close()

	mt, reader, err := mime.Detect(f)
	if err != nil {
		return nil, fault.Wrap(err)
	}

	sum := ""
	if !noHash {
		hasher := sha256.New()
		if _, err := io.Copy(hasher, reader); err != nil {
			return nil, fault.Wrap(err)
		}
		sum = hex.EncodeToString(hasher.Sum(nil))
	}

	return &Entry{
		Path:    rel,
		Size:    stat.Size(),
		SHA256:  sum,
		MIME:    mt.String(),
		ModTime: stat.ModTime(),
	}, nil
}

func hasAnyPrefix(path string, prefixes []string) bool {
	for _, p := range prefixes {
		if p == "" {
			continue
		}
		if strings.HasPrefix(foldSegment(path), foldSegment(p)) {
			return true
		}
	}
	return false
}

func isIgnoredFile(rel string) bool {
	base := strings.ToLower(filepath.Base(rel))
	return base == ".ds_store" || base == "thumbs.db" || base == "desktop.ini"
}

func (i *Inventory) WriteJSONL(w io.Writer) error {
	enc := json.NewEncoder(w)
	for _, e := range i.Entries {
		if err := enc.Encode(e); err != nil {
			return fault.Wrap(err)
		}
	}
	return nil
}

func ReadInventoryJSONL(r io.Reader) (*Inventory, error) {
	inv := &Inventory{}
	dec := json.NewDecoder(r)
	for {
		var e Entry
		if err := dec.Decode(&e); err != nil {
			if err == io.EOF {
				break
			}
			return nil, fault.Wrap(err)
		}
		inv.Entries = append(inv.Entries, e)
		inv.TotalSize += e.Size
	}
	return inv, nil
}
