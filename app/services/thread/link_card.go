package thread

import (
	"net/url"

	"github.com/Southclaws/opt"
	"github.com/samber/lo"

	"github.com/Southclaws/storyden/app/resources/datagraph"
	"github.com/Southclaws/storyden/app/resources/post/thread"
)

func firstCardURL(c datagraph.Content) opt.Optional[url.URL] {
	for _, raw := range c.LinkCards() {
		if u, err := url.Parse(raw); err == nil {
			return opt.New(*u)
		}
	}
	return opt.NewEmpty[url.URL]()
}

func isStaleCardLink(thr *thread.Thread, next datagraph.Content) bool {
	ln, ok := thr.WebLink.Get()
	if !ok {
		return false
	}
	if !lo.Contains(thr.Content.LinkCards(), ln.URL) {
		return false
	}
	return !lo.Contains(next.LinkCards(), ln.URL) && !lo.Contains(next.Links(), ln.URL)
}
