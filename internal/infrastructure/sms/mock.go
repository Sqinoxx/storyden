package sms

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

type MockMessage struct {
	Phone   string
	Message string
}

// Mock records every "sent" message in memory instead of calling a real SMS
// provider, so tests can read back a one-time code the same way they read a
// mock-sent email.
type Mock struct {
	mu   sync.Mutex
	sent []MockMessage
}

func newMock(l *slog.Logger) (Sender, error) {
	l.Debug("using mock sms sender - check the console for outgoing messages")
	return &Mock{}, nil
}

func (m *Mock) Send(ctx context.Context, phone string, message string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	fmt.Printf(`
[MOCK SMS] to: "%s" message:

%s

`, phone, message)

	m.sent = append(m.sent, MockMessage{Phone: phone, Message: message})

	return nil
}

func (m *Mock) Count() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return len(m.sent)
}

func (m *Mock) GetLast() MockMessage {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.sent[len(m.sent)-1]
}
