package library_import

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/Southclaws/opt"

	"github.com/Southclaws/storyden/app/resources/library"
	"github.com/Southclaws/storyden/app/resources/library/node_querier"
	"github.com/Southclaws/storyden/app/services/library/node_mutate"
	"github.com/Southclaws/storyden/internal/infrastructure/ai"
)

type Enricher struct {
	logger   *slog.Logger
	querier  *node_querier.Querier
	nodes    *node_mutate.Manager
	prompter ai.Prompter
}

func NewEnricher(
	logger *slog.Logger,
	querier *node_querier.Querier,
	nodes *node_mutate.Manager,
	prompter ai.Prompter,
) *Enricher {
	return &Enricher{logger: logger, querier: querier, nodes: nodes, prompter: prompter}
}

type EnrichOptions struct {
	Ledger *Ledger
	// State records which documents have already been shown to the model. It is
	// separate from the property check because the model legitimately returns
	// nothing for fields like Dozent on many documents; without it those nodes
	// would count as outstanding forever and every rerun would pay for them
	// again instead of converging.
	State  *Ledger
	Limit  int
	DryRun bool
	// Overwrite allows the model to replace property values that are already
	// set. Off by default because the import derives Typ deterministically from
	// the folder path, which is more trustworthy than an inference.
	Overwrite bool
	Progress  func(done int, name string)
}

type EnrichResult struct {
	Enriched int
	Skipped  int
	Failed   int
	// Aborted is set when the run stopped early rather than reaching the end of
	// the ledger, which on the free Gemini tier happens once the daily quota is
	// spent. The run is resumable: already-filled nodes are skipped next time.
	Aborted     bool
	AbortReason string
	Usage       ai.Usage
}

// maxConsecutiveFailures stops a run whose every request fails for the same
// systemic reason — a model name the account cannot call, a rejected schema, a
// bad key. Without it a misconfiguration walks the entire ledger at the paced
// request rate, which is hours of wall clock and, on a paid tier, real money
// spent on nothing.
const maxConsecutiveFailures = 8

// enrichFields are the properties this pass populates. A node is only skipped
// when every one of them already has a value — checking "any value present"
// would skip the entire library, since the import already fills Typ from the
// manifest rules.
var enrichFields = []string{"Fach", "Semester", "Typ", "Jahr", "Dozent"}

type classification struct {
	Titel    string   `json:"titel" jsonschema:"title=Titel,description=Ein lesbarer Titel des Dokuments ohne Dateiendung"`
	Fach     string   `json:"fach" jsonschema:"title=Fach,description=Das Fach aus der vorgegebenen Liste oder ein leerer String"`
	Semester string   `json:"semester" jsonschema:"title=Semester,description=Das Semester als Zahl von 1 bis 10 oder ein leerer String"`
	Typ      string   `json:"typ" jsonschema:"title=Typ,description=Der Dokumenttyp aus der vorgegebenen Liste oder ein leerer String"`
	Jahr     string   `json:"jahr" jsonschema:"title=Jahr,description=Das Pruefungs- oder Semesterjahr als vierstellige Zahl oder ein leerer String"`
	Dozent   string   `json:"dozent" jsonschema:"title=Dozent,description=Nachname des Dozenten oder Pruefers oder ein leerer String"`
	Tags     []string `json:"tags" jsonschema:"title=Tags,description=Bis zu vier Tags ausschliesslich aus der vorgegebenen Liste,items=string"`
}

// ocrExcerptLength bounds what is sent per document. The first few thousand
// characters of a lecture PDF carry the title, subject and term; the rest is
// body text that adds cost without improving classification.
const ocrExcerptLength = 4000

func (e *Enricher) Enrich(ctx context.Context, vocab *Vocabulary, opts EnrichOptions) (*EnrichResult, error) {
	if opts.Ledger == nil {
		return nil, fault.New("enrichment needs the import ledger to know which nodes to visit")
	}

	result := &EnrichResult{}
	allowed := allowedTags(vocab)

	done := 0
	consecutiveFailures := 0
	var lastFailure error
	for _, rec := range opts.Ledger.Records() {
		if opts.Limit > 0 && result.Enriched >= opts.Limit {
			break
		}

		done++
		if opts.Progress != nil {
			opts.Progress(done, rec.Slug)
		}

		node, err := e.querier.Get(ctx, library.NewKey(rec.Slug))
		if err != nil {
			result.Failed++
			e.logger.Warn("failed to load node for enrichment", slog.String("slug", rec.Slug), slog.String("error", err.Error()))
			continue
		}

		if !opts.Overwrite {
			alreadyVisited := opts.State != nil && opts.State.Has(rec.SHA256)
			if alreadyVisited || len(missingEnrichFields(node)) == 0 {
				result.Skipped++
				continue
			}
		}

		input := buildPrompt(node, vocab, allowed)

		out, err := ai.PromptObject(ctx, e.prompter, "Klassifiziert ein Studiendokument der Zahnmedizin.", input, classification{})
		if err != nil {
			// A spent quota would otherwise fail every remaining node in turn,
			// so stop and let the operator resume once it resets.
			if errors.Is(err, ai.ErrRateLimited) {
				result.Aborted = true
				result.AbortReason = "rate limited by the language model provider"
				e.logger.Warn("aborting enrichment, provider quota exhausted", slog.String("slug", rec.Slug))
				break
			}

			result.Failed++
			consecutiveFailures++
			lastFailure = err
			e.logger.Warn("classification failed", slog.String("slug", rec.Slug), slog.String("error", err.Error()))

			if consecutiveFailures >= maxConsecutiveFailures {
				result.Aborted = true
				result.AbortReason = fmt.Sprintf("%d consecutive failures, last error: %s", consecutiveFailures, lastFailure)
				e.logger.Error("aborting enrichment, every request is failing the same way", slog.String("error", lastFailure.Error()))
				break
			}

			continue
		}

		consecutiveFailures = 0

		if opts.DryRun {
			e.logger.Info("would enrich",
				slog.String("slug", rec.Slug),
				slog.String("titel", out.Titel),
				slog.String("fach", out.Fach),
				slog.String("typ", out.Typ),
				slog.String("jahr", out.Jahr),
			)
			result.Enriched++
			continue
		}

		if err := e.apply(ctx, node, out, vocab, allowed, opts.Overwrite); err != nil {
			result.Failed++
			e.logger.Warn("failed to write enrichment", slog.String("slug", rec.Slug), slog.String("error", err.Error()))
			continue
		}

		if opts.State != nil {
			if err := opts.State.Record(rec); err != nil {
				return nil, fault.Wrap(err, fctx.With(ctx))
			}
		}

		result.Enriched++
	}

	if reporter, ok := e.prompter.(ai.UsageReporter); ok {
		result.Usage = reporter.Usage()
	}

	return result, nil
}

func (e *Enricher) apply(ctx context.Context, node *library.Node, out *classification, vocab *Vocabulary, allowed map[string]struct{}, overwrite bool) error {
	writable := map[string]struct{}{}
	if overwrite {
		for _, f := range enrichFields {
			writable[f] = struct{}{}
		}
	} else {
		for _, f := range missingEnrichFields(node) {
			writable[f] = struct{}{}
		}
	}

	// The import already tagged the node with the subject its folder implied,
	// and that placement is how the material was actually filed. Measured
	// against 30 sample documents the model disagreed with the folder on 30% of
	// them, nearly always for the worse ("Dentale Technologie" read as
	// prothetik, PPZ as kons), and a Fach column contradicting the node's own
	// tag is visible nonsense. So the folder wins, and the model only fills the
	// gap where the folder named no subject at all.
	fach := subjectFromTags(node, vocab)
	if fach == "" {
		fach = out.Fach
	}

	values := map[string]string{}
	for name, value := range map[string]string{
		"Fach":     fach,
		"Semester": out.Semester,
		"Typ":      out.Typ,
		"Jahr":     out.Jahr,
		"Dozent":   out.Dozent,
	} {
		if value == "" {
			continue
		}
		if _, ok := writable[name]; !ok {
			continue
		}
		values[name] = value
	}

	props := library.PropertyMutationList{}
	if table, ok := node.Properties.Get(); ok && len(values) > 0 {
		props = PropertyMutations(&table.Schema, table.Properties, values)
	}

	partial := node_mutate.Partial{}

	if out.Titel != "" && out.Titel != node.Name {
		partial.Name = opt.New(out.Titel)
	}

	if len(props) > 0 {
		partial.Properties = opt.New(props)
	}

	// Tags outside the vocabulary are dropped rather than created: the point of
	// classifying against a closed set is that the set stays closed.
	//
	// Only type tags are accepted. Section and subject follow from the folder,
	// which the import already tagged, so the model can only either repeat them
	// or contradict them — and it did contradict them, tagging 5 of the first 39
	// documents both vorklinik and klinik. Type is the one axis that genuinely
	// is not always visible from the path: whether a paper carries solutions or
	// is a recalled exam only shows in the content.
	kept := []string{}
	for _, t := range out.Tags {
		normalised := strings.ToLower(strings.TrimSpace(t))
		if _, ok := allowed[normalised]; !ok {
			continue
		}
		if !isTypeTag(vocab, normalised) {
			continue
		}
		kept = append(kept, normalised)
	}

	if len(kept) > 0 {
		existing := []string{}
		for _, t := range node.Tags {
			existing = append(existing, t.Name.String())
		}
		partial.Tags = opt.New(tagNames(dedupeStrings(append(existing, kept...))))
	}

	if !partial.Name.Ok() && !partial.Properties.Ok() && !partial.Tags.Ok() {
		return nil
	}

	if _, err := e.nodes.Update(ctx, library.NewKey(node.GetSlug()), partial); err != nil {
		return fault.Wrap(err, fctx.With(ctx))
	}

	return nil
}

// isTypeTag reports whether a vocabulary tag names a document type, the only
// axis the model is trusted to contribute.
func isTypeTag(vocab *Vocabulary, tag string) bool {
	for _, term := range vocab.Types {
		if term.Tag == tag {
			return true
		}
	}
	return false
}

// subjectFromTags returns the display name of the subject the node is tagged
// with, which the import derived from the folder path. Empty when the folder
// named no subject the vocabulary knows.
func subjectFromTags(node *library.Node, vocab *Vocabulary) string {
	for _, t := range node.Tags {
		name := t.Name.String()
		for _, term := range vocab.Subjects {
			if term.Tag != name {
				continue
			}
			if term.Display != "" {
				return term.Display
			}
			return term.Tag
		}
	}

	return ""
}

func buildPrompt(node *library.Node, vocab *Vocabulary, allowed map[string]struct{}) string {
	b := strings.Builder{}

	b.WriteString("Klassifiziere das folgende Studiendokument aus dem Zahnmedizinstudium.\n\n")
	b.WriteString("Erlaubte Faecher: " + strings.Join(termTags(vocab.Subjects), ", ") + "\n")
	b.WriteString("Erlaubte Typen: " + strings.Join(termTags(vocab.Types), ", ") + "\n")
	b.WriteString("Erlaubte Abschnitte: " + strings.Join(termTags(vocab.Sections), ", ") + "\n\n")
	b.WriteString("Verwende fuer Tags ausschliesslich Werte aus diesen Listen. ")
	b.WriteString("Wenn ein Feld nicht sicher aus dem Dokument hervorgeht, gib einen leeren String zurueck. ")
	b.WriteString("Rate nicht.\n\n")

	b.WriteString("Dateiname: " + node.Name + "\n")

	if parent, ok := node.Parent.Get(); ok {
		b.WriteString("Ordner: " + parent.Name + "\n")
	}

	for _, a := range node.Assets {
		if text, ok := a.OCRText.Get(); ok && text != "" {
			b.WriteString("\nTextauszug:\n")
			b.WriteString(truncate(text, ocrExcerptLength))
			break
		}
	}

	return b.String()
}

// missingEnrichFields reports which of the enrichable properties are still
// empty on a node. A node with no property table at all counts as fully
// missing, so it is offered to the model rather than silently skipped.
func missingEnrichFields(node *library.Node) []string {
	filled := map[string]struct{}{}

	if table, ok := node.Properties.Get(); ok {
		for _, p := range table.Properties {
			if v, ok := p.Value.Get(); ok && v != "" {
				filled[p.Field.Name] = struct{}{}
			}
		}
	}

	missing := []string{}
	for _, name := range enrichFields {
		if _, ok := filled[name]; !ok {
			missing = append(missing, name)
		}
	}

	return missing
}

func allowedTags(vocab *Vocabulary) map[string]struct{} {
	out := map[string]struct{}{}
	for _, t := range vocab.Tags() {
		out[t] = struct{}{}
	}
	return out
}

func termTags(terms []Term) []string {
	out := make([]string, 0, len(terms))
	for _, t := range terms {
		out = append(out, t.Tag)
	}
	return out
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
