package library_import

import (
	"sort"
	"strings"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fmsg"
	"github.com/goccy/go-yaml"
)

type TermKind string

const (
	KindSection TermKind = "section"
	KindSubject TermKind = "subject"
	KindType    TermKind = "type"
)

type Term struct {
	Tag     string   `yaml:"tag"`
	Display string   `yaml:"display"`
	Aliases []string `yaml:"aliases"`
}

type Vocabulary struct {
	Renames  map[string]string `yaml:"renames"`
	Sections []Term            `yaml:"sections"`
	Subjects []Term            `yaml:"subjects"`
	Types    []Term            `yaml:"types"`
}

type Match struct {
	Term  Term
	Kind  TermKind
	Exact bool
}

func ParseVocabulary(data []byte) (*Vocabulary, error) {
	var v Vocabulary
	if err := yaml.Unmarshal(data, &v); err != nil {
		return nil, fault.Wrap(err, fmsg.With("failed to parse vocabulary file"))
	}

	for kind, terms := range map[TermKind][]Term{KindSection: v.Sections, KindSubject: v.Subjects, KindType: v.Types} {
		for _, t := range terms {
			if t.Tag == "" {
				return nil, fault.New("vocabulary term without a tag", fmsg.With(string(kind)+" term is missing its tag"))
			}
		}
	}

	return &v, nil
}

func (v *Vocabulary) terms(kind TermKind) []Term {
	switch kind {
	case KindSection:
		return v.Sections
	case KindSubject:
		return v.Subjects
	case KindType:
		return v.Types
	}
	return nil
}

// Lookup resolves a raw folder segment against one axis of the vocabulary.
// Exact folded matches win outright; otherwise the longest alias contained in
// the segment wins, which is what rescues names like
// "MKG (Radio, AnEx, Auscultando, Spritzenkurs)".
func (v *Vocabulary) Lookup(kind TermKind, raw string) (Match, bool) {
	folded := foldSegment(raw)
	if folded == "" {
		return Match{}, false
	}

	terms := v.terms(kind)

	for _, t := range terms {
		for _, candidate := range append([]string{t.Tag, t.Display}, t.Aliases...) {
			if candidate != "" && foldSegment(candidate) == folded {
				return Match{Term: t, Kind: kind, Exact: true}, true
			}
		}
	}

	best := Match{}
	bestLen := 0
	for _, t := range terms {
		for _, candidate := range append([]string{t.Tag, t.Display}, t.Aliases...) {
			c := foldSegment(candidate)
			if len(c) < minFuzzyAliasLength || !strings.Contains(folded, c) {
				continue
			}
			if len(c) > bestLen {
				best, bestLen = Match{Term: t, Kind: kind}, len(c)
			}
		}
	}

	return best, bestLen > 0
}

// minFuzzyAliasLength keeps short aliases such as "ac" or "ppz" from matching
// by accident inside unrelated folder names.
const minFuzzyAliasLength = 4

// LookupAny falls back to the other axes when a segment is not a term on the
// axis the manifest expected. The source tree routinely files a type where a
// subject belongs, as with "Ankis" or "Videos" sitting between section and
// document, and those are still worth tagging correctly.
func (v *Vocabulary) LookupAny(preferred TermKind, raw string) (Match, bool) {
	if m, ok := v.Lookup(preferred, raw); ok {
		return m, true
	}

	for _, kind := range []TermKind{KindSection, KindSubject, KindType} {
		if kind == preferred {
			continue
		}
		if m, ok := v.Lookup(kind, raw); ok {
			return m, true
		}
	}

	return Match{}, false
}

func (v *Vocabulary) Tags() []string {
	seen := map[string]struct{}{}
	for _, kind := range []TermKind{KindSection, KindSubject, KindType} {
		for _, t := range v.terms(kind) {
			seen[t.Tag] = struct{}{}
		}
	}

	out := make([]string, 0, len(seen))
	for t := range seen {
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}
