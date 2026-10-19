package account_phone

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalise(t *testing.T) {
	valid := map[string]string{
		"+43 664 1234567":   "+436641234567",
		"0664/123 45 67":    "+436641234567",
		"0043 (664) 123-45": "+4366412345",
		"+49 151 12345678":  "+4915112345678",
	}
	for in, want := range valid {
		got, err := Normalise(in)
		assert.NoError(t, err, in)
		assert.Equal(t, want, got, in)
	}

	for _, in := range []string{"", "abc", "0664", "+0123456789", "12+34567890", "+1234567890123456"} {
		_, err := Normalise(in)
		assert.Error(t, err, in)
	}
}
