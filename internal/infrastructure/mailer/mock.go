package mailer

import (
	"context"
	"fmt"
	"net/mail"
	"strings"
	"sync"
)

type MockEmail struct {
	Address mail.Address
	Name    string
	Subject string
	Html    string
	Plain   string
}

type Mock struct {
	mu   sync.Mutex
	sent []MockEmail
}

func (m *Mock) Send(
	ctx context.Context,
	msg Message,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	fmt.Printf(`Mock email sent to: %s <%s> '%s'
%s
`,
		msg.Name,
		msg.Address.String(),
		msg.Subject,
		msg.Content.Plain,
	)

	m.sent = append(m.sent, MockEmail{
		Address: msg.Address,
		Name:    msg.Name,
		Subject: msg.Subject,
		Html:    msg.Content.HTML,
		Plain:   msg.Content.Plain,
	})

	return nil
}

func (m *Mock) GetLast() MockEmail {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.sent[len(m.sent)-1]
}

func (m *Mock) GetLastTo(address string) (MockEmail, bool) {
	return m.GetLastWhere(func(email MockEmail) bool {
		return strings.EqualFold(email.Address.Address, address)
	})
}

func (m *Mock) GetLastWhere(match func(MockEmail) bool) (MockEmail, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i := len(m.sent) - 1; i >= 0; i-- {
		if match(m.sent[i]) {
			return m.sent[i], true
		}
	}

	return MockEmail{}, false
}
