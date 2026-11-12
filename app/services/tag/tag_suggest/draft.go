package tag_suggest

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"

	"github.com/Southclaws/storyden/internal/infrastructure/ai"
)

const (
	minTagThreads   = 3
	maxDraftsPerRun = 25
	sampleThreads   = 8
	maxMeaningChars = 120
	draftTimeout    = 30 * time.Second
)

type MeaningsDraft struct {
	Meanings string
	Added    []string
	Skipped  []string
}

func (s *Suggester) DraftMeanings(ctx context.Context) (*MeaningsDraft, error) {
	set, err := s.settings.Get(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	raw := rawMeanings(set.Metadata.OrZero())
	draft := &MeaningsDraft{Meanings: raw, Added: []string{}, Skipped: []string{}}

	if _, ok := s.prompter.(*ai.OpenAI); !ok {
		return draft, nil
	}

	known := ParseMeanings(raw)

	tags, err := s.tags.List(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	var lines []string
	attempted := 0
	for _, t := range tags {
		if attempted >= maxDraftsPerRun {
			break
		}
		name := t.Name.String()
		if t.ItemCount < minTagThreads || strings.Contains(name, ":") {
			continue
		}
		if _, ok := known[strings.ToLower(name)]; ok {
			continue
		}

		samples, err := s.tags.SampleThreads(ctx, t.Name, sampleThreads)
		if err != nil {
			return nil, fault.Wrap(err, fctx.With(ctx))
		}
		if len(samples) < minTagThreads {
			continue
		}

		attempted++

		titles := make([]string, len(samples))
		for i, sm := range samples {
			titles[i] = "- " + oneLine(sm.Title+" — "+sm.Short, 200)
		}

		meaning, ok := s.describe(ctx, name, titles)
		if !ok {
			draft.Skipped = append(draft.Skipped, name)
			continue
		}

		lines = append(lines, name+": "+meaning)
		draft.Added = append(draft.Added, name)
	}

	draft.Meanings = AppendMeanings(raw, lines)
	return draft, nil
}

const describeTemplate = `In a German dentistry students' forum, threads are tagged "%s". Recent threads with this tag:
%s

Describe in at most 8 English words what the tag "%s" stands for, e.g. "endodontics (root canal treatment)". Set confident to false if the tag could mean several different things or the threads do not make its meaning clear.`

func (s *Suggester) describe(ctx context.Context, tag string, titles []string) (string, bool) {
	ctx, cancel := context.WithTimeout(ctx, draftTimeout)
	defer cancel()

	schema := map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"meaning", "confident"},
		"properties": map[string]any{
			"meaning":   map[string]any{"type": "string"},
			"confident": map[string]any{"type": "boolean"},
		},
	}

	var res struct {
		Meaning   string `json:"meaning"`
		Confident bool   `json:"confident"`
	}
	prompt := fmt.Sprintf(describeTemplate, tag, strings.Join(titles, "\n"), tag)
	if err := ai.PromptSchema(ctx, s.prompter, "Describe a forum tag", prompt, schema, &res); err != nil {
		s.logger.Warn("tag meaning draft failed", slog.String("tag", tag), slog.String("error", err.Error()))
		return "", false
	}

	meaning := oneLine(res.Meaning, maxMeaningChars)
	if !res.Confident || meaning == "" {
		return "", false
	}
	return meaning, true
}

func AppendMeanings(raw string, lines []string) string {
	raw = strings.TrimRight(raw, " \t\r\n")
	if len(lines) == 0 {
		return raw
	}
	added := strings.Join(lines, "\n")
	if raw == "" {
		return added
	}
	return raw + "\n" + added
}

func oneLine(s string, max int) string {
	s = strings.Join(strings.Fields(s), " ")
	if r := []rune(s); len(r) > max {
		s = string(r[:max])
	}
	return s
}
