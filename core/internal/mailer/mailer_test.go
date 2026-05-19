package mailer

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
)

func TestLogMailer_Send(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	m := &LogMailer{Logger: logger}

	err := m.Send(context.Background(), Message{
		To:      "alice@example.com",
		Subject: "Подтверждение",
		Body:    "token=abc",
	})
	if err != nil {
		t.Fatalf("Send err: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "alice@example.com") {
		t.Errorf("log missing recipient: %s", out)
	}
	if !strings.Contains(out, "token=abc") {
		t.Errorf("log missing body: %s", out)
	}
	if !strings.Contains(out, "provider=log") {
		t.Errorf("log missing provider tag: %s", out)
	}
}
