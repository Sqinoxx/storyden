package library_import

import (
	"sort"
	"strings"
)

type DuplicateGroup struct {
	SHA256     string
	Size       int64
	Canonical  string
	Duplicates []string
}

type DedupeResult struct {
	Canonical  []Entry
	Groups     []DuplicateGroup
	SavedBytes int64
}

// Dedupe collapses entries that share content. Within a group the canonical
// copy is the one whose path is not under a demoted prefix, then the shallowest
// path, then the shortest, then lexicographic order so runs are reproducible.
func Dedupe(entries []Entry, demote []string) DedupeResult {
	byHash := map[string][]Entry{}
	order := []string{}
	for _, e := range entries {
		if _, seen := byHash[e.SHA256]; !seen {
			order = append(order, e.SHA256)
		}
		byHash[e.SHA256] = append(byHash[e.SHA256], e)
	}

	result := DedupeResult{}

	for _, hash := range order {
		group := byHash[hash]
		sort.Slice(group, func(i, j int) bool { return lessCanonical(group[i].Path, group[j].Path, demote) })

		result.Canonical = append(result.Canonical, group[0])

		if len(group) == 1 {
			continue
		}

		dupes := make([]string, 0, len(group)-1)
		for _, e := range group[1:] {
			dupes = append(dupes, e.Path)
			result.SavedBytes += e.Size
		}

		result.Groups = append(result.Groups, DuplicateGroup{
			SHA256:     hash,
			Size:       group[0].Size,
			Canonical:  group[0].Path,
			Duplicates: dupes,
		})
	}

	sort.Slice(result.Canonical, func(i, j int) bool { return result.Canonical[i].Path < result.Canonical[j].Path })

	return result
}

func lessCanonical(a, b string, demote []string) bool {
	da, db := hasAnyPrefix(a, demote), hasAnyPrefix(b, demote)
	if da != db {
		return db
	}

	depthA, depthB := strings.Count(a, "/"), strings.Count(b, "/")
	if depthA != depthB {
		return depthA < depthB
	}

	if len(a) != len(b) {
		return len(a) < len(b)
	}

	return a < b
}
