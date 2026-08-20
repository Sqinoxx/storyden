package library_import

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPatternMatch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		pattern  string
		path     string
		want     map[string]string
		wantFail bool
	}{
		{
			name:    "captures a subject segment",
			pattern: "Altklausuren/Vorklinik (1.-4. Semester)/{fach}/**",
			path:    "Altklausuren/Vorklinik (1.-4. Semester)/Physiologie/2019/klausur.pdf",
			want:    map[string]string{"fach": "Physiologie", "path": "2019/klausur.pdf"},
		},
		{
			name:    "double star matches zero segments",
			pattern: "Altklausuren/{fach}/**",
			path:    "Altklausuren/Physiologie",
			want:    map[string]string{"fach": "Physiologie", "path": ""},
		},
		{
			name:    "double star in the middle",
			pattern: "Zusammenfassungen/**/{fach}/*",
			path:    "Zusammenfassungen/Ankis/Vorklinik/Biochemie/deck.apkg",
			want:    map[string]string{"fach": "Biochemie"},
		},
		{
			name:    "literal comparison ignores case spacing and ampersand padding",
			pattern: "Klausuren & Prüfungen/{fach}/**",
			path:    "klausuren&prüfungen/MKG/x.pdf",
			want:    map[string]string{"fach": "MKG", "path": "x.pdf"},
		},
		{
			name:    "single star consumes without capturing",
			pattern: "Vorlesungen&Unterlagen/*/{fach}/**",
			path:    "Vorlesungen&Unterlagen/Klinik/Kons/kurs/a.pdf",
			want:    map[string]string{"fach": "Kons", "path": "kurs/a.pdf"},
		},
		{
			name:    "catch-all captures the whole path",
			pattern: "**",
			path:    "Altprotokolle/Z1/protokoll.pdf",
			want:    map[string]string{"path": "Altprotokolle/Z1/protokoll.pdf"},
		},
		{
			name:     "non-matching literal fails",
			pattern:  "Altklausuren/{fach}/**",
			path:     "Vorlesungen&Unterlagen/Kons/a.pdf",
			wantFail: true,
		},
		{
			name:     "pattern longer than path fails",
			pattern:  "Altklausuren/{fach}/{typ}",
			path:     "Altklausuren/Kons",
			wantFail: true,
		},
		{
			name:    "windows separators are normalised",
			pattern: "Altklausuren/{fach}/**",
			path:    `Altklausuren\Kons\a.pdf`,
			want:    map[string]string{"fach": "Kons", "path": "a.pdf"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, ok := NewPattern(tt.pattern).Match(tt.path)
			if tt.wantFail {
				assert.False(t, ok)
				return
			}

			require.True(t, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExpand(t *testing.T) {
	t.Parallel()

	got := Expand("klinik/{fach}/{typ}", map[string]string{"fach": "MKG", "typ": "Altklausuren"})
	assert.Equal(t, "klinik/MKG/Altklausuren", got)
}
