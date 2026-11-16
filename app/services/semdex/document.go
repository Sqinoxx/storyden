package semdex

import "github.com/Southclaws/storyden/app/resources/datagraph"

// Document is an item indexed with more text than its own content, such as a
// thread together with its replies and attachments.
type Document struct {
	datagraph.Item
	Text string
}

func (d *Document) EmbeddingText() string { return d.Text }

type EmbeddingTexter interface {
	EmbeddingText() string
}
