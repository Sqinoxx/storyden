package scrape

import (
	"bytes"
	"context"
	"io"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/Southclaws/dt"
	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"golang.org/x/net/html"

	"github.com/Southclaws/storyden/app/resources/datagraph"
)

func (s *webScraper) postprocess(ctx context.Context, addr url.URL, r io.Reader) (*WebContent, error) {
	buf, err := io.ReadAll(r)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(buf))
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	t := metatable(doc)
	rc, err := getArticleContent(bytes.NewReader(buf))
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	text := rc.Short()

	withBaseURL := func(urlOrPath string) string {
		if urlOrPath == "" {
			return ""
		}

		u, err := url.Parse(urlOrPath)
		if err != nil {
			return ""
		}

		if u.IsAbs() {
			return u.String()
		}

		return addr.ResolveReference(u).String()
	}

	wc := &WebContent{
		Title:       title(t, doc),
		Description: description(t),
		Text:        text,
		Favicon:     withBaseURL(favicon(doc)),
		Image:       withBaseURL(image(t, doc)),
		Content:     rc,
	}

	return wc, nil
}

func getArticleContent(r io.Reader) (datagraph.Content, error) {
	rc, err := datagraph.NewRichTextFromReader(r)
	if err != nil {
		return datagraph.Content{}, nil
	}

	return rc, nil
}

func metatable(doc *goquery.Document) map[string]string {
	return dt.Reduce(doc.Find("head > meta").Nodes, func(wc map[string]string, n *html.Node) map[string]string {
		k, v := ogtable(n.Attr)
		if k != "" && v != "" {
			wc[k] = v
		}

		return wc
	}, map[string]string{})
}

func ogtable(attrs []html.Attribute) (k string, v string) {
	for _, a := range attrs {
		switch a.Key {
		case "property":
			k = a.Val
		case "name":
			k = a.Val
		case "content":
			v = a.Val
		}
	}

	return
}

func title(t map[string]string, doc *goquery.Document) string {
	if v := first(t, "og:title", "twitter:title", "title"); v != "" {
		return v
	}
	if v := strings.TrimSpace(doc.Find("title").First().Text()); v != "" {
		return v
	}

	return first(t, "og:site_name", "og:url")
}

func description(t map[string]string) string {
	return first(t, "og:description", "twitter:description", "description")
}

func image(t map[string]string, doc *goquery.Document) string {
	if v := first(t, "og:image", "og:image:secure_url", "og:image:url", "twitter:image", "twitter:image:src"); v != "" {
		return v
	}
	if href, ok := doc.Find("link[rel='image_src']").Attr("href"); ok {
		return strings.TrimSpace(href)
	}

	return ""
}

func first(t map[string]string, keys ...string) string {
	for _, k := range keys {
		if v := strings.TrimSpace(t[k]); v != "" {
			return v
		}
	}

	return ""
}

func favicon(doc *goquery.Document) string {
	for _, sel := range []string{
		"link[rel='apple-touch-icon']",
		"link[rel='apple-touch-icon-precomposed']",
		"link[rel~='icon']",
	} {
		if href, ok := doc.Find(sel).First().Attr("href"); ok && strings.TrimSpace(href) != "" {
			return strings.TrimSpace(href)
		}
	}

	return "/favicon.ico"
}
