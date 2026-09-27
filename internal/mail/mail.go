// Package mail sends transactional email. Resend in production, a console
// printer in development.
package mail

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// Message is one outbound email.
type Message struct {
	To      string
	Subject string
	Text    string
	HTML    string
}

// Mailer sends messages.
type Mailer interface {
	Send(ctx context.Context, m Message) error
}

// Console logs the message instead of sending it. Used with -dev.
type Console struct{}

// Send prints the message to the log.
func (Console) Send(_ context.Context, m Message) error {
	log.Printf("mail (not sent) to=%s subject=%q\n%s", m.To, m.Subject, m.Text)
	return nil
}

// Resend sends through https://resend.com.
type Resend struct {
	apiKey string
	from   string
	client *http.Client
}

// NewResend builds a client. The HTTP client never follows redirects, so the
// bearer token cannot be replayed to another host.
func NewResend(apiKey, from string) *Resend {
	return &Resend{
		apiKey: apiKey,
		from:   from,
		client: &http.Client{
			Timeout: 10 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

// Send posts to the Resend API.
func (r *Resend) Send(ctx context.Context, m Message) error {
	body, err := json.Marshal(map[string]any{
		"from": r.from, "to": []string{m.To}, "subject": m.Subject, "text": m.Text, "html": m.HTML,
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.resend.com/emails", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+r.apiKey)
	req.Header.Set("Content-Type", "application/json")
	res, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("mail: resend: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		snippet, _ := io.ReadAll(io.LimitReader(res.Body, 512))
		return fmt.Errorf("mail: resend returned %d: %s", res.StatusCode, snippet)
	}
	return nil
}

// ErrNotConfigured is returned by New when production settings are missing.
var ErrNotConfigured = errors.New("mail: RESEND_API_KEY and MAIL_FROM are required outside -dev")

// New picks the mailer. In dev mode with no key, mail goes to the console.
func New(dev bool, apiKey, from string) (Mailer, error) {
	if apiKey == "" || from == "" {
		if dev {
			return Console{}, nil
		}
		return nil, ErrNotConfigured
	}
	return NewResend(apiKey, from), nil
}
