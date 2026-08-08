package drive

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseFolderID(t *testing.T) {
	t.Parallel()

	const id = "1a2B3c4D5e6F7g8H9i0JkLmNoPqRsTuV"

	for _, tt := range []struct {
		name string
		in   string
		want string
	}{
		{"bare id", id, id},
		{"share link", "https://drive.google.com/drive/folders/" + id, id},
		{"share link with query", "https://drive.google.com/drive/folders/" + id + "?usp=sharing", id},
		{"account scoped link", "https://drive.google.com/drive/u/0/folders/" + id, id},
		{"open link", "https://drive.google.com/open?id=" + id, id},
		{"trailing slash", "https://drive.google.com/drive/folders/" + id + "/", id},
		{"surrounding whitespace", "  " + id + "  ", id},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseFolderID(tt.in)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestParseFolderIDRejects(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name string
		in   string
	}{
		{"empty", ""},
		{"whitespace", "   "},
		{"too short", "abc"},
		{"no id anywhere", "https://drive.google.com/drive/my-drive"},
		{"account scope only", "https://drive.google.com/drive/u/0"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := ParseFolderID(tt.in)
			require.Error(t, err)
		})
	}
}
