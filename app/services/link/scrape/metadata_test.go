package scrape

import (
	"context"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_webScraper_postprocess_metadata(t *testing.T) {
	ctx := context.Background()
	w := webScraper{}
	base, _ := url.Parse("https://example.com/forum/")

	cases := []struct {
		name        string
		html        string
		title       string
		description string
		image       string
		favicon     string
	}{
		{
			name: "opengraph",
			html: `<html><head>
				<title>Document title</title>
				<meta property="og:title" content="OG title">
				<meta property="og:description" content="OG description">
				<meta property="og:image" content="/og.png">
				<link rel="icon" href="/icon.png">
			</head><body></body></html>`,
			title:       "OG title",
			description: "OG description",
			image:       "https://example.com/og.png",
			favicon:     "https://example.com/icon.png",
		},
		{
			name: "title element and twitter card fallback",
			html: `<html><head>
				<title> unimedics.de - Index page </title>
				<meta name="twitter:description" content="Twitter description">
				<meta name="twitter:image" content="https://cdn.example.com/card.jpg">
				<link rel="shortcut icon" href="assets/Logo ohne Text.png">
			</head><body></body></html>`,
			title:       "unimedics.de - Index page",
			description: "Twitter description",
			image:       "https://cdn.example.com/card.jpg",
			favicon:     "https://example.com/forum/assets/Logo%20ohne%20Text.png",
		},
		{
			name: "image_src link and apple touch icon preferred",
			html: `<html><head>
				<link rel="image_src" href="//img.example.com/share.png">
				<link rel="icon" href="/favicon-16.png">
				<link rel="apple-touch-icon" href="/apple-touch-icon.png">
			</head><body></body></html>`,
			image:   "https://img.example.com/share.png",
			favicon: "https://example.com/apple-touch-icon.png",
		},
		{
			name:    "nothing available",
			html:    `<html><head></head><body><p>hello</p></body></html>`,
			favicon: "https://example.com/favicon.ico",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			wc, err := w.postprocess(ctx, *base, strings.NewReader(c.html))
			require.NoError(t, err)

			assert.Equal(t, c.title, wc.Title)
			assert.Equal(t, c.description, wc.Description)
			assert.Equal(t, c.image, wc.Image)
			assert.Equal(t, c.favicon, wc.Favicon)
		})
	}
}
