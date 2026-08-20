package library_import

import (
	"encoding/json"
	"io"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/Southclaws/fault"

	"github.com/Southclaws/storyden/app/resources/mark"
)

type PlannedContainer struct {
	Path        []string `json:"path"`
	Slug        string   `json:"slug"`
	Name        string   `json:"name"`
	ParentSlug  string   `json:"parent_slug,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	ChildSchema []string `json:"child_schema,omitempty"`
	FileCount   int      `json:"file_count"`
}

type PlannedFile struct {
	Entry      Entry             `json:"entry"`
	ParentSlug string            `json:"parent_slug"`
	Name       string            `json:"name"`
	Slug       string            `json:"slug"`
	Tags       []string          `json:"tags,omitempty"`
	Properties map[string]string `json:"properties,omitempty"`
	Visibility string            `json:"visibility"`
	Rule       string            `json:"rule"`
}

type Plan struct {
	Containers []PlannedContainer
	Files      []PlannedFile
	// Unresolved lists segments no vocabulary axis could name, so the manifest
	// or the vocabulary can be sharpened before anything is written.
	Unresolved map[string]int
	// CatchAll counts files that only matched the final fallback rule.
	CatchAll int
}

type Planner struct {
	manifest *Manifest
	vocab    *Vocabulary
}

func NewPlanner(m *Manifest, v *Vocabulary) *Planner {
	return &Planner{manifest: m, vocab: v}
}

func (p *Planner) Plan(entries []Entry) (*Plan, error) {
	plan := &Plan{Unresolved: map[string]int{}}

	containers := map[string]*PlannedContainer{}
	usedSlugs := map[string]struct{}{}

	rootName := p.manifest.Defaults.Root
	rootSlug := mark.Slugify(rootName)
	containers[rootSlug] = &PlannedContainer{Path: []string{rootName}, Slug: rootSlug, Name: rootName}
	usedSlugs[rootSlug] = struct{}{}

	for _, entry := range entries {
		rule, captures, ok := p.manifest.Resolve(entry.Path)
		if !ok {
			return nil, fault.New("no manifest rule matched " + entry.Path)
		}

		if rule.Match == "**" {
			plan.CatchAll++
		}

		captures["filename"] = filepath.Base(entry.Path)
		captures["stem"] = strings.TrimSuffix(captures["filename"], filepath.Ext(captures["filename"]))

		parent, axisTags := p.ensureContainers(plan, containers, usedSlugs, rule, captures)

		tags := append(axisTags, p.resolveTags(plan, rule.Tags, captures)...)

		name := captures["stem"]
		if name == "" {
			name = captures["filename"]
		}

		visibility := rule.Visibility
		if visibility == "" {
			visibility = p.manifest.Defaults.Visibility
		}

		plan.Files = append(plan.Files, PlannedFile{
			Entry:      entry,
			ParentSlug: parent.Slug,
			Name:       name,
			Slug:       uniqueSlug(usedSlugs, parent.Slug+"-"+name, entry.SHA256),
			Tags:       dedupeStrings(tags),
			Properties: expandProperties(rule.Properties, captures),
			Visibility: visibility,
			Rule:       rule.Match,
		})

		parent.FileCount++
	}

	plan.Containers = sortedContainers(containers)

	return plan, nil
}

// ensureContainers walks the expanded target path, creating any missing
// container on the way and collecting the axis tags each segment contributes.
func (p *Planner) ensureContainers(
	plan *Plan,
	containers map[string]*PlannedContainer,
	usedSlugs map[string]struct{},
	rule *Rule,
	captures map[string]string,
) (*PlannedContainer, []string) {
	rootName := p.manifest.Defaults.Root
	path := []string{rootName}
	slug := mark.Slugify(rootName)
	current := containers[slug]

	axisTags := []string{}

	for _, spec := range splitPath(Expand(rule.Target, captures)) {
		display, tag := p.resolveSegment(plan, spec)
		if tag != "" {
			axisTags = append(axisTags, tag)
		}

		parentSlug := slug
		path = append(path, display)
		slug = mark.Slugify(strings.Join(path[1:], "-"))

		existing, ok := containers[slug]
		if !ok {
			schema := rule.ChildSchema
			if len(schema) == 0 {
				schema = p.manifest.Defaults.ChildSchema
			}

			existing = &PlannedContainer{
				Path:        append([]string{}, path...),
				Slug:        slug,
				Name:        display,
				ParentSlug:  parentSlug,
				Tags:        dedupeStrings(append([]string{}, axisTags...)),
				ChildSchema: schema,
			}
			containers[slug] = existing
			usedSlugs[slug] = struct{}{}
		}

		current = existing
	}

	return current, axisTags
}

// resolveSegment turns a target segment into a display name and its tag. A
// segment may name its axis explicitly, as in sub:MKG; without a prefix it is
// taken as a literal folder name that contributes no tag.
func (p *Planner) resolveSegment(plan *Plan, spec string) (string, string) {
	kind, raw := splitAxis(spec)

	if kind == "" {
		return raw, ""
	}

	if match, ok := p.vocab.LookupAny(kind, raw); ok {
		display := match.Term.Display
		if display == "" {
			display = match.Term.Tag
		}
		return display, match.Term.Tag
	}

	// An unknown segment still becomes a folder, but it must not mint a tag:
	// the whole point of the closed vocabulary is that it stays closed.
	plan.Unresolved[string(kind)+":"+raw]++

	return raw, ""
}

func splitAxis(spec string) (TermKind, string) {
	prefix, rest, found := strings.Cut(spec, ":")
	if !found {
		return "", spec
	}

	switch prefix {
	case "sec":
		return KindSection, rest
	case "sub":
		return KindSubject, rest
	case "typ":
		return KindType, rest
	}

	return "", spec
}

func (p *Planner) resolveTags(plan *Plan, specs []string, captures map[string]string) []string {
	out := []string{}
	for _, spec := range specs {
		expanded := Expand(spec, captures)
		if expanded == "" {
			continue
		}

		if kind, raw := splitAxis(expanded); kind != "" {
			if match, ok := p.vocab.LookupAny(kind, raw); ok {
				out = append(out, match.Term.Tag)
				continue
			}
			plan.Unresolved[string(kind)+":"+raw]++
			continue
		}

		out = append(out, mark.Slugify(expanded))
	}
	return out
}

func expandProperties(props map[string]string, captures map[string]string) map[string]string {
	if len(props) == 0 {
		return nil
	}

	out := make(map[string]string, len(props))
	for k, v := range props {
		out[k] = Expand(v, captures)
	}
	return out
}

// uniqueSlug keeps node slugs globally distinct. Names repeat constantly across
// the source tree and the slug column is unique, so a deterministic content
// suffix is appended whenever the readable form is already taken.
func uniqueSlug(used map[string]struct{}, base, hash string) string {
	slug := mark.Slugify(base)
	if slug == "" {
		slug = "datei"
	}

	if _, taken := used[slug]; !taken {
		used[slug] = struct{}{}
		return slug
	}

	for n := 6; n <= 32 && n <= len(hash); n += 4 {
		candidate := slug + "-" + hash[:n]
		if _, taken := used[candidate]; !taken {
			used[candidate] = struct{}{}
			return candidate
		}
	}

	// A no-hash inventory has nothing content-derived to fall back on, so the
	// slug is disambiguated by ordinal instead.
	for n := 2; ; n++ {
		candidate := slug + "-" + strconv.Itoa(n)
		if _, taken := used[candidate]; !taken {
			used[candidate] = struct{}{}
			return candidate
		}
	}
}

func dedupeStrings(in []string) []string {
	seen := map[string]struct{}{}
	out := []string{}
	for _, s := range in {
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

// sortedContainers orders parents before children so ingest can create the
// tree in a single forward pass.
func sortedContainers(in map[string]*PlannedContainer) []PlannedContainer {
	out := make([]PlannedContainer, 0, len(in))
	for _, c := range in {
		out = append(out, *c)
	}

	sort.Slice(out, func(i, j int) bool {
		if len(out[i].Path) != len(out[j].Path) {
			return len(out[i].Path) < len(out[j].Path)
		}
		return strings.Join(out[i].Path, "/") < strings.Join(out[j].Path, "/")
	})

	return out
}

func (p *Plan) WriteJSONL(w io.Writer) error {
	enc := json.NewEncoder(w)
	for _, f := range p.Files {
		if err := enc.Encode(f); err != nil {
			return fault.Wrap(err)
		}
	}
	return nil
}
