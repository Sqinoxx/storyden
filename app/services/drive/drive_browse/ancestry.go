package drive_browse

import (
	"context"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/Southclaws/fault/fmsg"
	"github.com/Southclaws/fault/ftag"
)

// maxAncestryNodes bounds the walk from a target up towards a registered root.
// Drive hierarchies are shallow in practice; the budget exists so a malformed
// or adversarial ID cannot turn one request into an unbounded number of Drive
// API calls.
const maxAncestryNodes = 64

// ancestry proves that targetID sits somewhere beneath rootDriveID and returns
// the folder trail from just below the root down to the target.
//
// This is the security boundary of the whole feature. The service account can
// read every folder anyone has shared with it, so without this check any Drive
// ID guessed or leaked by a member would be downloadable through the proxy.
// Every failure path returns NotFound: distinguishing "outside your root" from
// "does not exist" would confirm the existence of files the caller has no
// business knowing about.
func (b *Browser) ancestry(ctx context.Context, rootDriveID string, targetID string) ([]Crumb, error) {
	if targetID == rootDriveID {
		return nil, nil
	}

	if cached, ok := b.cache.getAncestry(ctx, rootDriveID, targetID); ok {
		return cached, nil
	}

	// Breadth-first upwards. Drive is single-parent today but has historically
	// allowed several, and a file reachable through any registered root is
	// legitimately viewable through it.
	queue := []string{targetID}
	seen := map[string]bool{targetID: true}
	cameFrom := map[string]string{}
	names := map[string]string{}

	for budget := 0; len(queue) > 0 && budget < maxAncestryNodes; budget++ {
		id := queue[0]
		queue = queue[1:]

		f, err := b.client.Get(ctx, id)
		if err != nil {
			return nil, fault.Wrap(err, fctx.With(ctx))
		}

		names[id] = f.Name

		for _, parent := range f.Parents {
			if parent == rootDriveID {
				crumbs := trail(id, names, cameFrom)
				b.cache.setAncestry(ctx, rootDriveID, targetID, crumbs)

				return crumbs, nil
			}

			if seen[parent] {
				continue
			}

			seen[parent] = true
			cameFrom[parent] = id
			queue = append(queue, parent)
		}
	}

	return nil, fault.New("target is not within the registered folder",
		fctx.With(ctx),
		ftag.With(ftag.NotFound),
		fmsg.WithDesc("ancestry check failed", "This file or folder is not available."))
}

// trail reconstructs the root-to-target order from the child pointers recorded
// on the way up.
func trail(from string, names map[string]string, cameFrom map[string]string) []Crumb {
	crumbs := []Crumb{}

	for cur := from; ; {
		crumbs = append(crumbs, Crumb{ID: cur, Name: names[cur]})

		next, ok := cameFrom[cur]
		if !ok {
			break
		}

		cur = next
	}

	return crumbs
}
