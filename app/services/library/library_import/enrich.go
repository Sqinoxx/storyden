package library_import

import (
	"context"
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
	Ledger   *Ledger
	Limit    int
	DryRun   bool
	Progress func(done int, name string)
}

type EnrichResult struct {
	Enriched int
	Skipped  int
	Failed   int
}

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

		if hasFilledProperties(node) {
			result.Skipped++
			continue
		}

		input := buildPrompt(node, vocab, allowed)

		out, err := ai.PromptObject(ctx, e.prompter, "Klassifiziert ein Studiendokument der Zahnmedizin.", input, classification{})
		if err != nil {
			result.Failed++
			e.logger.Warn("classification failed", slog.String("slug", rec.Slug), slog.String("error", err.Error()))
			continue
		}

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

		if err := e.apply(ctx, node, out, allowed); err != nil {
			result.Failed++
			e.logger.Warn("failed to write enrichment", slog.String("slug", rec.Slug), slog.String("error", err.Error()))
			continue
		}

		result.Enriched++
	}

	return result, nil
}

func (e *Enricher) apply(ctx context.Context, node *library.Node, out *classification, allowed map[string]struct{}) error {
	values := map[string]string{}
	for name, value := range map[string]string{
		"Fach":     out.Fach,
		"Semester": out.Semester,
		"Typ":      out.Typ,
		"Jahr":     out.Jahr,
		"Dozent":   out.Dozent,
	} {
		if value != "" {
			values[name] = value
		}
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
	kept := []string{}
	for _, t := range out.Tags {
		normalised := strings.ToLower(strings.TrimSpace(t))
		if _, ok := allowed[normalised]; ok {
			kept = append(kept, normalised)
		}
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

func hasFilledProperties(node *library.Node) bool {
	table, ok := node.Properties.Get()
	if !ok {
		return false
	}

	for _, p := range table.Properties {
		if v, ok := p.Value.Get(); ok && v != "" {
			return true
		}
	}

	return false
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
