// Package asset_ref extracts references to locally-hosted assets from rich
// text content, so that a post/reply/node body can be linked to its
// attachments without relying on the client to always send explicit IDs.
package asset_ref

import (
	"net/url"
	"strings"

	"golang.org/x/net/html"

	"github.com/Southclaws/storyden/app/resources/asset"
	"github.com/Southclaws/storyden/app/resources/datagraph"
)

// ExtractAssetIDs scans the rich text content's HTML tree for links/media
// pointing at locally hosted assets (as emitted by the upload UI, e.g.
// `<a href="{API}/api/assets/{id}-{slug}">` or `<img src="...">`) and
// returns their asset IDs, deduplicated, in document order.
//
// This walks the tree directly rather than using Content.Links()/Media():
// Links() only records <a href> when the URL scheme is http/https, silently
// dropping a relative href, and Media() resolves relative URLs against a
// placeholder base that mangles the path. Both matter here because asset
// hrefs are sometimes relative.
func ExtractAssetIDs(c datagraph.Content) []asset.AssetID {
	root := c.HTMLTree()
	if root == nil {
		return nil
	}

	return ExtractAssetIDsFromRefs(collectRefs(root))
}

// ExtractAssetIDsFromRefs parses a list of href/src strings and returns the
// asset IDs found among them, deduplicated, in input order.
func ExtractAssetIDsFromRefs(refs []string) []asset.AssetID {
	seen := make(map[asset.AssetID]bool)
	var ids []asset.AssetID

	for _, ref := range refs {
		id, ok := assetIDFromURL(ref)
		if !ok || seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
	}

	return ids
}

func collectRefs(n *html.Node) []string {
	var refs []string

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			var attrName string
			switch n.Data {
			case "a":
				attrName = "href"
			case "img", "source", "video":
				attrName = "src"
			}

			if attrName != "" {
				for _, a := range n.Attr {
					if a.Key == attrName && a.Val != "" {
						refs = append(refs, a.Val)
					}
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)

	return refs
}

// assetIDFromURL parses a URL or path and, if it points at a locally hosted
// asset (i.e. it has an "assets" path segment followed by a valid
// "{xid}-{slug}" filename), returns that asset's ID.
func assetIDFromURL(raw string) (asset.AssetID, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return asset.AssetID{}, false
	}

	p := raw
	if u, err := url.Parse(raw); err == nil && u.Path != "" {
		p = u.Path
	}

	segments := strings.Split(strings.Trim(p, "/"), "/")

	assetsIdx := -1
	for i, seg := range segments {
		if seg == "assets" {
			assetsIdx = i
		}
	}
	if assetsIdx == -1 || assetsIdx+1 >= len(segments) {
		return asset.AssetID{}, false
	}

	filename := segments[assetsIdx+1]

	parsed, err := asset.ParseAssetFilename(filename)
	if err != nil {
		return asset.AssetID{}, false
	}

	return asset.AssetID(parsed.GetID()), true
}
