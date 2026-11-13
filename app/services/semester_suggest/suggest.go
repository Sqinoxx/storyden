// Terms are always computed in code. Small models are confidently wrong at
// term arithmetic, so the model only confirms that an indirect reference such
// as a date is about the post's own exam or course.
package semester_suggest

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/services/account/semester"
	"github.com/Southclaws/storyden/internal/infrastructure/ai"
)

const (
	minConfidence = 90
	maxInputChars = 4000
	modelTimeout  = 20 * time.Second
)

type Source string

const (
	SourcePattern Source = "pattern"
	SourceModel   Source = "model"
)

type Suggestion struct {
	Term     semester.Term
	Source   Source
	Evidence string
}

type Input struct {
	Title     string
	Body      string
	Filenames []string
}

func (i Input) text() string {
	parts := []string{i.Title, i.Body}
	parts = append(parts, i.Filenames...)
	text := strings.Join(parts, "\n")
	if len(text) > maxInputChars {
		text = text[:maxInputChars]
	}
	return text
}

type Suggester struct {
	logger   *slog.Logger
	prompter ai.Prompter
	now      func() time.Time
}

func Build() fx.Option {
	return fx.Provide(New)
}

func New(logger *slog.Logger, prompter ai.Prompter) *Suggester {
	return &Suggester{logger: logger, prompter: prompter, now: time.Now}
}

func (s *Suggester) Suggest(ctx context.Context, in Input) (*Suggestion, error) {
	text := in.text()
	if strings.TrimSpace(text) == "" {
		return nil, nil
	}

	now := s.now()

	switch found := detect(text, now); len(found) {
	case 1:
		return &Suggestion{Term: found[0].term, Source: SourcePattern, Evidence: found[0].evidence}, nil
	case 0:
	default:
		return nil, nil
	}

	candidates := detectIndirect(text, now)
	if len(candidates) != 1 {
		return nil, nil
	}

	if !ai.IsOpenAI(s.prompter) {
		return nil, nil
	}

	ctx, cancel := context.WithTimeout(ctx, modelTimeout)
	defer cancel()

	ok, err := s.confirm(ctx, text, candidates[0].evidence)
	if err != nil {
		s.logger.Warn("semester suggestion model call failed", slog.String("error", err.Error()))
		return nil, nil
	}
	if !ok {
		return nil, nil
	}

	return &Suggestion{Term: candidates[0].term, Source: SourceModel, Evidence: candidates[0].evidence}, nil
}

type confirmation struct {
	Category   string `json:"category" jsonschema:"enum=exam,enum=other"`
	Confidence int    `json:"confidence" jsonschema:"description=Confidence from 0 to 100"`
}

const confirmTemplate = `German forum post: %s

In this post, is "%s" the time of an exam, test or course the post is about? Category "exam" if yes, "other" if it is anything else (birthday, deadline, start of studies, plans). Give confidence 0-100.`

func (s *Suggester) confirm(ctx context.Context, text, phrase string) (bool, error) {
	ans, err := ai.PromptObject(ctx, s.prompter, "Confirm a time reference in a forum post", fmt.Sprintf(confirmTemplate, text, phrase), confirmation{})
	if err != nil {
		return false, fault.Wrap(err, fctx.With(ctx))
	}

	return accepted(*ans), nil
}

func accepted(c confirmation) bool {
	return strings.EqualFold(strings.TrimSpace(c.Category), "exam") && c.Confidence >= minConfidence
}
