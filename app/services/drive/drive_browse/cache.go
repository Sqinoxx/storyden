package drive_browse

import (
	"context"
	"encoding/json"
	"time"

	"github.com/Southclaws/storyden/internal/infrastructure/cache"
	"github.com/Southclaws/storyden/internal/infrastructure/gdrive"
)

// listCache keeps Drive responses out of the request path. Every navigation
// step otherwise costs one listing plus one Get per ancestry level, and Drive
// enforces a per-project quota that a busy forum would exhaust quickly.
//
// Cache misses are the normal, correct state: failures here are swallowed so a
// cache outage degrades to slower browsing rather than a broken page.
type listCache struct {
	store cache.Store
	ttl   time.Duration
}

func newListCache(store cache.Store, ttl time.Duration) *listCache {
	return &listCache{store: store, ttl: ttl}
}

func (c *listCache) getListing(ctx context.Context, folderID string, pageToken string) (*gdrive.Listing, bool) {
	var listing gdrive.Listing
	if !c.get(ctx, listingKey(folderID, pageToken), &listing) {
		return nil, false
	}

	return &listing, true
}

func (c *listCache) setListing(ctx context.Context, folderID string, pageToken string, listing *gdrive.Listing) {
	c.set(ctx, listingKey(folderID, pageToken), listing)
}

func (c *listCache) getAncestry(ctx context.Context, rootDriveID string, targetID string) ([]Crumb, bool) {
	var crumbs []Crumb
	if !c.get(ctx, ancestryKey(rootDriveID, targetID), &crumbs) {
		return nil, false
	}

	return crumbs, true
}

func (c *listCache) setAncestry(ctx context.Context, rootDriveID string, targetID string, crumbs []Crumb) {
	c.set(ctx, ancestryKey(rootDriveID, targetID), crumbs)
}

func (c *listCache) get(ctx context.Context, key string, into any) bool {
	if c.ttl <= 0 {
		return false
	}

	raw, err := c.store.Get(ctx, key)
	if err != nil || raw == "" {
		return false
	}

	return json.Unmarshal([]byte(raw), into) == nil
}

func (c *listCache) set(ctx context.Context, key string, value any) {
	if c.ttl <= 0 {
		return
	}

	encoded, err := json.Marshal(value)
	if err != nil {
		return
	}

	_ = c.store.Set(ctx, key, string(encoded), c.ttl)
}

func listingKey(folderID string, pageToken string) string {
	return "drive:list:" + folderID + ":" + pageToken
}

func ancestryKey(rootDriveID string, targetID string) string {
	return "drive:anc:" + rootDriveID + ":" + targetID
}
