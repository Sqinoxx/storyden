package mime

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	assert.Equal(t, "application/pdf", New("application/pdf").String())
	assert.Equal(t, "text/plain", New("text/plain; charset=utf-8").String())
	assert.Equal(t, DefaultMIME, New("not/a-type").String())
}
