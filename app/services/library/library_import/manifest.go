package library_import

import (
	"strconv"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fmsg"
	"github.com/goccy/go-yaml"
)

type Manifest struct {
	Defaults Defaults `yaml:"defaults"`
	Rules    []Rule   `yaml:"rules"`
}

type Defaults struct {
	Root        string   `yaml:"root"`
	Visibility  string   `yaml:"visibility"`
	ChildSchema []string `yaml:"child_schema"`
	Demote      []string `yaml:"demote"`
	Skip        []string `yaml:"skip"`
}

type Rule struct {
	Match       string            `yaml:"match"`
	Target      string            `yaml:"target"`
	Tags        []string          `yaml:"tags"`
	Properties  map[string]string `yaml:"properties"`
	Visibility  string            `yaml:"visibility"`
	ChildSchema []string          `yaml:"child_schema"`

	pattern Pattern
}

func ParseManifest(data []byte) (*Manifest, error) {
	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fault.Wrap(err, fmsg.With("failed to parse manifest file"))
	}

	if len(m.Rules) == 0 {
		return nil, fault.New("manifest has no rules", fmsg.With("The import manifest must define at least one rule."))
	}

	for i := range m.Rules {
		r := &m.Rules[i]
		if r.Match == "" {
			return nil, fault.New("manifest rule without a match", fmsg.With("Rule "+strconv.Itoa(i)+" is missing its match pattern."))
		}
		if r.Target == "" {
			return nil, fault.New("manifest rule without a target", fmsg.With("Rule "+strconv.Itoa(i)+" is missing its target."))
		}
		r.pattern = NewPattern(r.Match)
	}

	if m.Defaults.Visibility == "" {
		m.Defaults.Visibility = "unlisted"
	}
	if m.Defaults.Root == "" {
		m.Defaults.Root = "Bibliothek"
	}

	return &m, nil
}

// Resolve returns the first rule whose pattern matches, along with its
// captures. Rule order is significant: the last rule is expected to be a
// catch-all so nothing is silently dropped.
func (m *Manifest) Resolve(path string) (*Rule, map[string]string, bool) {
	for i := range m.Rules {
		if captures, ok := m.Rules[i].pattern.Match(path); ok {
			return &m.Rules[i], captures, true
		}
	}
	return nil, nil, false
}
