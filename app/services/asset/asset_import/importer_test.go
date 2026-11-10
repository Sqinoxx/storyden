package asset_import

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const fileID = "1wLKOp83T2tUPw5neR2qhioik8UQLX5so"

func TestResolve(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"drive file view", "https://drive.google.com/file/d/" + fileID + "/view?usp=sharing", "https://drive.usercontent.google.com/download?export=download&id=" + fileID},
		{"drive open", "https://drive.google.com/open?id=" + fileID, "https://drive.usercontent.google.com/download?export=download&id=" + fileID},
		{"drive uc", "https://drive.google.com/uc?id=" + fileID + "&export=download", "https://drive.usercontent.google.com/download?export=download&id=" + fileID},
		{"google doc", "https://docs.google.com/document/d/" + fileID + "/edit?usp=sharing", "https://docs.google.com/document/d/" + fileID + "/export?format=pdf"},
		{"google slides", "https://docs.google.com/presentation/d/" + fileID + "/edit#slide=id.p", "https://docs.google.com/presentation/d/" + fileID + "/export?format=pdf"},
		{"google sheet", "https://docs.google.com/spreadsheets/d/" + fileID + "/edit", "https://docs.google.com/spreadsheets/d/" + fileID + "/export?format=pdf"},
		{"drive folder untouched", "https://drive.google.com/drive/folders/" + fileID, "https://drive.google.com/drive/folders/" + fileID},
		{"direct pdf untouched", "https://register.awmf.org/assets/guidelines/083-001l_S3_Kariesprophylaxe.pdf", "https://register.awmf.org/assets/guidelines/083-001l_S3_Kariesprophylaxe.pdf"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := Resolve(c.in)
			require.NoError(t, err)
			assert.Equal(t, c.want, got.String())
		})
	}
}

func TestResolveRejectsNonHTTP(t *testing.T) {
	for _, in := range []string{"file:///etc/passwd", "ftp://example.com/a.pdf", "javascript:alert(1)", "/relative/path.pdf"} {
		_, err := Resolve(in)
		assert.Error(t, err, in)
	}
}

func TestFilename(t *testing.T) {
	u, _ := url.Parse("https://example.com/files/Leitlinie%20Karies.pdf?x=1")

	assert.Equal(t, "Aspirin.pdf", filename(`attachment; filename="Aspirin.pdf"`, u, "pdf"))
	assert.Equal(t, "Übersicht.pdf", filename(`attachment; filename*=UTF-8''%C3%9Cbersicht.pdf`, u, "pdf"))
	assert.Equal(t, "passwd", filename(`attachment; filename="../../etc/passwd"`, u, "pdf"))
	assert.Equal(t, "Leitlinie Karies.pdf", filename("", u, "pdf"))

	bare, _ := url.Parse("https://drive.usercontent.google.com/download?id=" + fileID)
	assert.Equal(t, "dokument.docx", filename("", bare, "docx"))
}
