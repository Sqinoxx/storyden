package summary_stream

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func collect(s Stream) string {
	var out strings.Builder
	for ev := range s {
		if ev.Done {
			if ev.Err != nil {
				out.WriteString("|err")
			} else {
				out.WriteString("|" + ev.HTML)
			}
			break
		}
		out.WriteString(ev.Text)
	}
	return out.String()
}

func TestBroadcastLateSubscriber(t *testing.T) {
	b := New()
	b.Push("Hallo ")
	b.Push("Welt")

	got := make(chan string, 1)
	go func() { got <- collect(b.Subscribe(context.Background())) }()

	b.Push("!")
	b.Finish("<p>Hallo Welt!</p>", nil)
	assert.Equal(t, "Hallo Welt!|<p>Hallo Welt!</p>", <-got)
}

func TestBroadcastError(t *testing.T) {
	b := New()
	b.Push("x")
	b.Finish("", errors.New("boom"))
	assert.Equal(t, "x|err", collect(b.Subscribe(context.Background())))
}

func TestReady(t *testing.T) {
	assert.Equal(t, "|<p>a</p>", collect(Ready("<p>a</p>")))
}

func TestGroupSharesRunningBroadcast(t *testing.T) {
	g := NewGroup[string]()

	a, started, done := g.Join("k")
	assert.True(t, started)

	b, startedAgain, _ := g.Join("k")
	assert.False(t, startedAgain)
	assert.Same(t, a, b)

	done()
	c, startedNew, _ := g.Join("k")
	assert.True(t, startedNew)
	assert.NotSame(t, a, c)
}
