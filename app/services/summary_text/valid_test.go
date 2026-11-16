package summary_text

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	german  = "Die Klausur ist nicht schwer und wird mit einer Aufgabe zur Endodontie eröffnet, die sich auf das Skript bezieht."
	english = "The exam is not hard and it opens with a question on endodontics that is taken from the script of this course."
)

func TestLanguage(t *testing.T) {
	assert.Equal(t, "de", Language(german))
	assert.Equal(t, "en", Language(english))
	assert.Equal(t, "", Language("Endodontie Parodontologie"))
}

func TestValid(t *testing.T) {
	assert.True(t, Valid("<ul><li>Die Klausur ist nicht schwer und wird mit der Endodontie eröffnet, die sich auf das Skript bezieht.</li></ul>", german))
	assert.True(t, Valid("<ul><li>Endodontie</li></ul>", german))
	assert.False(t, Valid("<p>Die Klausur ist nicht schwer.</p>", german))
	assert.False(t, Valid("<ul><li>"+english+"</li></ul>", german))
	assert.False(t, Valid("", german))
}
