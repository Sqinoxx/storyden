package library_import

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"sort"
	"sync"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fmsg"
)

type LedgerRecord struct {
	SHA256  string `json:"sha256"`
	Path    string `json:"path"`
	NodeID  string `json:"node_id"`
	AssetID string `json:"asset_id"`
	Slug    string `json:"slug"`
}

// Ledger is an append-only record of what has already been ingested. A run
// over tens of thousands of files and tens of gigabytes will be interrupted;
// replaying the ledger on start is what makes a rerun cheap instead of
// destructive.
type Ledger struct {
	mu   sync.Mutex
	file *os.File
	done map[string]LedgerRecord
}

func OpenLedger(path string) (*Ledger, error) {
	done := map[string]LedgerRecord{}

	if existing, err := os.Open(path); err == nil {
		scanner := bufio.NewScanner(existing)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for scanner.Scan() {
			var rec LedgerRecord
			if err := json.Unmarshal(scanner.Bytes(), &rec); err != nil {
				continue
			}
			done[rec.SHA256] = rec
		}
		existing.Close()
		if err := scanner.Err(); err != nil {
			return nil, fault.Wrap(err, fmsg.With("failed to read import ledger"))
		}
	} else if !os.IsNotExist(err) {
		return nil, fault.Wrap(err, fmsg.With("failed to open import ledger"))
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fault.Wrap(err, fmsg.With("failed to open import ledger for writing"))
	}

	return &Ledger{file: f, done: done}, nil
}

func (l *Ledger) Has(sha string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	_, ok := l.done[sha]
	return ok
}

func (l *Ledger) Get(sha string) (LedgerRecord, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	rec, ok := l.done[sha]
	return rec, ok
}

func (l *Ledger) Record(rec LedgerRecord) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	line, err := json.Marshal(rec)
	if err != nil {
		return fault.Wrap(err)
	}

	if _, err := l.file.Write(append(line, '\n')); err != nil {
		return fault.Wrap(err)
	}

	if err := l.file.Sync(); err != nil {
		return fault.Wrap(err)
	}

	l.done[rec.SHA256] = rec

	return nil
}

// Records returns the ingested files in a stable order so a resumed
// enrichment pass visits them the same way every time.
func (l *Ledger) Records() []LedgerRecord {
	l.mu.Lock()
	defer l.mu.Unlock()

	out := make([]LedgerRecord, 0, len(l.done))
	for _, rec := range l.done {
		out = append(out, rec)
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })

	return out
}

func (l *Ledger) Len() int {
	l.mu.Lock()
	defer l.mu.Unlock()

	return len(l.done)
}

func (l *Ledger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	return l.file.Close()
}

var _ io.Closer = (*Ledger)(nil)
