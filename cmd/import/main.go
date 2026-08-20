package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/resources/account"
	"github.com/Southclaws/storyden/app/resources/account/account_repo"
	"github.com/Southclaws/storyden/app/resources/rbac"
	"github.com/Southclaws/storyden/app/resources/visibility"
	"github.com/Southclaws/storyden/app/services/authentication/session"
	"github.com/Southclaws/storyden/app/services/library/library_import"
	"github.com/Southclaws/storyden/app/services/ocr"
	"github.com/Southclaws/storyden/internal/script"
)

const usage = `Import a directory tree into the library.

  import --phase scan   --root <dir>   inventory + hash + duplicate report
  import --phase vocab                 reconcile the tag vocabulary
  import --phase plan   --root <dir>   map the inventory onto the node tree
  import --phase apply  --root <dir>   create nodes and upload assets
  import --phase ocr                   extract text from everything pending
  import --phase enrich                fill properties from filename and text
  import --phase set-visibility        retarget every imported node's visibility

Phases are resumable and expect to be run in this order. Run apply with
OCR_ENABLED=false, then ocr with OCR_ENABLED=true and OCR_CONCURRENCY set to
the number of cores you are willing to give it.
`

type flags struct {
	phase      string
	root       string
	work       string
	manifest   string
	vocab      string
	owner      string
	limitRoot  string
	visibility string
	dryRun     bool
	noHash     bool
	limit      int
}

func main() {
	f := flags{}

	flag.StringVar(&f.phase, "phase", "", "scan | vocab | plan | apply | ocr | enrich | set-visibility")
	flag.StringVar(&f.root, "root", "", "source directory to import")
	flag.StringVar(&f.work, "work", "./import-work", "directory for inventory, plan and ledger files")
	flag.StringVar(&f.manifest, "manifest", "./import-manifest.yaml", "path to the manifest")
	flag.StringVar(&f.vocab, "vocab-file", "./import-vocab.yaml", "path to the vocabulary")
	flag.StringVar(&f.owner, "owner", "", "handle of the account that will own imported nodes")
	flag.StringVar(&f.visibility, "visibility", "", "target visibility for --phase set-visibility (draft | unlisted | review | published)")
	flag.StringVar(&f.limitRoot, "limit-root", "", "only process paths under this prefix")
	flag.BoolVar(&f.dryRun, "dry-run", false, "report what would happen without writing")
	flag.BoolVar(&f.noHash, "no-hash", false, "scan without hashing, for a fast manifest coverage check")
	flag.IntVar(&f.limit, "limit", 0, "stop after this many files (0 = no limit)")
	flag.Usage = func() { fmt.Fprint(os.Stderr, usage); flag.PrintDefaults() }
	flag.Parse()

	if f.phase == "" {
		flag.Usage()
		os.Exit(2)
	}

	if err := os.MkdirAll(f.work, 0o755); err != nil {
		fatal(err)
	}

	// scan and plan are pure file operations. Keeping them out of the fx graph
	// means a manifest coverage check needs no database and no configuration.
	switch f.phase {
	case "scan":
		if err := runScan(context.Background(), f); err != nil {
			fatal(err)
		}
		return
	case "plan":
		if _, err := buildPlan(f); err != nil {
			fatal(err)
		}
		return
	}

	script.Run(fx.Invoke(func(
		ctx context.Context,
		ingester *library_import.Ingester,
		enricher *library_import.Enricher,
		processor *ocr.Processor,
		accounts *account_repo.Repository,
	) {
		if err := run(ctx, f, ingester, enricher, processor, accounts); err != nil {
			fatal(err)
		}
	}))
}

func run(
	ctx context.Context,
	f flags,
	ingester *library_import.Ingester,
	enricher *library_import.Enricher,
	processor *ocr.Processor,
	accounts *account_repo.Repository,
) error {
	switch f.phase {
	case "vocab":
		return runVocab(ctx, f, ingester)
	case "apply":
		return runApply(ctx, f, ingester, accounts)
	case "ocr":
		return runOCR(ctx, processor)
	case "enrich":
		return runEnrich(ctx, f, enricher, accounts)
	case "set-visibility":
		return runSetVisibility(ctx, f, ingester, accounts)
	}

	return fmt.Errorf("unknown phase %q", f.phase)
}

func runScan(ctx context.Context, f flags) error {
	if f.root == "" {
		return fmt.Errorf("--root is required for scan")
	}

	manifest, err := loadManifest(f)
	if err != nil {
		return err
	}

	started := time.Now()
	last := time.Now()

	inv, err := library_import.Scan(ctx, f.root, library_import.ScanOptions{
		Demote: manifest.Defaults.Demote,
		Skip:   manifest.Defaults.Skip,
		NoHash: f.noHash,
		Progress: func(files int, bytes int64) {
			if time.Since(last) < time.Second {
				return
			}
			last = time.Now()
			fmt.Printf("\rscanned %d files, %.1f GB", files, float64(bytes)/(1<<30))
		},
	})
	if err != nil {
		return err
	}

	fmt.Printf("\rscanned %d files, %.1f GB in %s\n", len(inv.Entries), float64(inv.TotalSize)/(1<<30), time.Since(started).Round(time.Second))

	if err := writeFile(filepath.Join(f.work, "inventory.jsonl"), inv.WriteJSONL); err != nil {
		return err
	}

	if f.noHash {
		fmt.Println("no-hash mode: duplicate detection skipped")
		return nil
	}

	result := library_import.Dedupe(inv.Entries, manifest.Defaults.Demote)

	fmt.Printf("canonical %d files, %d duplicate groups, %.1f GB redundant\n",
		len(result.Canonical), len(result.Groups), float64(result.SavedBytes)/(1<<30))

	return writeFile(filepath.Join(f.work, "duplicates.report.txt"), func(w io.Writer) error {
		for _, g := range result.Groups {
			fmt.Fprintf(w, "keep %s\n", g.Canonical)
			for _, d := range g.Duplicates {
				fmt.Fprintf(w, "  drop %s\n", d)
			}
		}
		return nil
	})
}

func runVocab(ctx context.Context, f flags, ingester *library_import.Ingester) error {
	vocab, err := loadVocabulary(f)
	if err != nil {
		return err
	}

	actions, err := ingester.ApplyVocabulary(ctx, vocab, f.dryRun)
	if err != nil {
		return err
	}

	for _, a := range actions {
		fmt.Println(a)
	}

	if f.dryRun {
		fmt.Printf("dry run: %d vocabulary actions not applied\n", len(actions))
	} else {
		fmt.Printf("applied %d vocabulary actions\n", len(actions))
	}

	return nil
}

func buildPlan(f flags) (*library_import.Plan, error) {
	manifest, err := loadManifest(f)
	if err != nil {
		return nil, err
	}

	vocab, err := loadVocabulary(f)
	if err != nil {
		return nil, err
	}

	invFile, err := os.Open(filepath.Join(f.work, "inventory.jsonl"))
	if err != nil {
		return nil, fmt.Errorf("no inventory found, run --phase scan first: %w", err)
	}
	defer invFile.Close()

	inv, err := library_import.ReadInventoryJSONL(invFile)
	if err != nil {
		return nil, err
	}

	entries := inv.Entries
	if !f.noHash {
		for _, e := range entries {
			if e.SHA256 == "" {
				return nil, fmt.Errorf("inventory was written with --no-hash, rerun --phase scan without it")
			}
		}
		entries = library_import.Dedupe(entries, manifest.Defaults.Demote).Canonical
	}
	entries = filterEntries(entries, f)

	plan, err := library_import.NewPlanner(manifest, vocab).Plan(entries)
	if err != nil {
		return nil, err
	}

	reportPlan(plan)

	if err := writeFile(filepath.Join(f.work, "plan.jsonl"), plan.WriteJSONL); err != nil {
		return nil, err
	}

	return plan, nil
}

func filterEntries(entries []library_import.Entry, f flags) []library_import.Entry {
	out := entries

	if f.limitRoot != "" {
		filtered := out[:0:0]
		for _, e := range out {
			if strings.HasPrefix(e.Path, f.limitRoot) {
				filtered = append(filtered, e)
			}
		}
		out = filtered
	}

	if f.limit > 0 && len(out) > f.limit {
		out = out[:f.limit]
	}

	return out
}

func reportPlan(plan *library_import.Plan) {
	fmt.Printf("%d containers, %d files\n", len(plan.Containers), len(plan.Files))

	for _, c := range plan.Containers {
		indent := strings.Repeat("  ", len(c.Path)-1)
		count := ""
		if c.FileCount > 0 {
			count = fmt.Sprintf("  (%d)", c.FileCount)
		}
		fmt.Printf("%s%s%s\n", indent, c.Name, count)
	}

	if plan.CatchAll > 0 {
		fmt.Printf("\n%d files matched only the catch-all rule\n", plan.CatchAll)
	}

	if len(plan.Unresolved) > 0 {
		fmt.Printf("\n%d folder names no vocabulary term covers:\n", len(plan.Unresolved))
		keys := make([]string, 0, len(plan.Unresolved))
		for k := range plan.Unresolved {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool { return plan.Unresolved[keys[i]] > plan.Unresolved[keys[j]] })
		for _, k := range keys {
			fmt.Printf("  %5d  %s\n", plan.Unresolved[k], k)
		}
	}
}

func runApply(ctx context.Context, f flags, ingester *library_import.Ingester, accounts *account_repo.Repository) error {
	if f.root == "" {
		return fmt.Errorf("--root is required for apply")
	}

	// Without hashes there is no deduplication and no stable ledger key, so a
	// no-hash inventory would import the archive's 1500 redundant copies and
	// then be unable to resume.
	if f.noHash {
		return fmt.Errorf("--no-hash is only for reviewing the plan, rerun --phase scan without it before applying")
	}

	plan, err := buildPlan(f)
	if err != nil {
		return err
	}

	if f.dryRun {
		fmt.Println("\ndry run: nothing written")
		return nil
	}

	ctx, err = importerContext(ctx, accounts, f.owner)
	if err != nil {
		return err
	}

	ledger, err := library_import.OpenLedger(filepath.Join(f.work, "import-state.jsonl"))
	if err != nil {
		return err
	}
	defer ledger.Close()

	fmt.Printf("\nledger already holds %d files\n", ledger.Len())

	last := time.Now()
	result, err := ingester.Apply(ctx, plan, library_import.IngestOptions{
		Root:   f.root,
		Owner:  accountIDFromContext(ctx),
		Ledger: ledger,
		Progress: func(done, total int, path string) {
			if time.Since(last) < time.Second {
				return
			}
			last = time.Now()
			fmt.Printf("\r%d/%d  %-70.70s", done, total, path)
		},
	})
	if err != nil {
		return err
	}

	fmt.Printf("\ncontainers: %d created, %d reused. files: %d ingested, %d already done\n",
		result.ContainersCreated, result.ContainersReused, result.FilesIngested, result.FilesSkipped)

	return nil
}

func runOCR(ctx context.Context, processor *ocr.Processor) error {
	total := 0
	for {
		n, err := processor.ProcessAllPending(ctx, 200)
		if err != nil {
			return err
		}
		if n == 0 {
			break
		}
		total += n
		fmt.Printf("\rextracted text from %d assets", total)
	}

	fmt.Printf("\rextracted text from %d assets\n", total)

	return nil
}

func runSetVisibility(ctx context.Context, f flags, ingester *library_import.Ingester, accounts *account_repo.Repository) error {
	if f.visibility == "" {
		return fmt.Errorf("--visibility is required for set-visibility (draft | unlisted | review | published)")
	}

	vis, err := visibility.NewVisibility(f.visibility)
	if err != nil {
		return err
	}

	ctx, err = importerContext(ctx, accounts, f.owner)
	if err != nil {
		return err
	}

	ledger, err := library_import.OpenLedger(filepath.Join(f.work, "import-state.jsonl"))
	if err != nil {
		return err
	}
	defer ledger.Close()

	result, err := ingester.SetVisibility(ctx, vis, library_import.FixVisibilityOptions{
		Ledger: ledger,
		Progress: func(done, total int, slug string) {
			fmt.Printf("\r%d/%d  %-70.70s", done, total, slug)
		},
	})
	if err != nil {
		return err
	}

	fmt.Printf("\n%d nodes updated, %d already at %s\n", result.Updated, result.Skipped, vis)

	return nil
}

func runEnrich(ctx context.Context, f flags, enricher *library_import.Enricher, accounts *account_repo.Repository) error {
	vocab, err := loadVocabulary(f)
	if err != nil {
		return err
	}

	ctx, err = importerContext(ctx, accounts, f.owner)
	if err != nil {
		return err
	}

	ledger, err := library_import.OpenLedger(filepath.Join(f.work, "import-state.jsonl"))
	if err != nil {
		return err
	}
	defer ledger.Close()

	result, err := enricher.Enrich(ctx, vocab, library_import.EnrichOptions{
		Ledger: ledger,
		Limit:  f.limit,
		DryRun: f.dryRun,
		Progress: func(done int, name string) {
			fmt.Printf("\r%d  %-70.70s", done, name)
		},
	})
	if err != nil {
		return err
	}

	fmt.Printf("\n%d nodes enriched, %d skipped, %d failed\n", result.Enriched, result.Skipped, result.Failed)

	return nil
}

// importerContext grants the run administrator rights explicitly rather than
// relying on the account's role edges, so publishing and schema changes cannot
// fail halfway through a long run.
func importerContext(ctx context.Context, accounts *account_repo.Repository, handle string) (context.Context, error) {
	if handle == "" {
		return nil, fmt.Errorf("--owner is required, pass the handle of an admin account")
	}

	acc, exists, err := accounts.LookupByHandle(ctx, handle)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("no account with handle %q", handle)
	}

	return session.WithAccountPermissions(ctx, acc.Account, rbac.NewList(rbac.PermissionAdministrator)), nil
}

func accountIDFromContext(ctx context.Context) account.AccountID {
	id, err := session.GetAccountID(ctx)
	if err != nil {
		fatal(err)
	}
	return id
}

func loadManifest(f flags) (*library_import.Manifest, error) {
	data, err := os.ReadFile(f.manifest)
	if err != nil {
		return nil, err
	}
	return library_import.ParseManifest(data)
}

func loadVocabulary(f flags) (*library_import.Vocabulary, error) {
	data, err := os.ReadFile(f.vocab)
	if err != nil {
		return nil, err
	}
	return library_import.ParseVocabulary(data)
}

func writeFile(path string, write func(w io.Writer) error) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	return write(f)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "\nimport failed:", err)
	os.Exit(1)
}
