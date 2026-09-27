package mailer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"testing"
)

func TestLogSenderSignalsNoDelivery(t *testing.T) {
	s := &LogSender{Log: slog.Default()}
	err := s.Send(context.Background(), Message{To: "u@gmail.com", Subject: "test", TextBody: "body"})
	if !errors.Is(err, ErrLogOnly) {
		t.Fatalf("expected ErrLogOnly, got %v", err)
	}
}

func TestBuildMIMESanitizesHeaders(t *testing.T) {
	msg := buildMIME(SMTPConfig{From: "sender@gmail.com", FromName: "SFEDU\r\nBcc: evil@example.com"}, "u@gmail.com", Message{
		Subject:  "Hello\r\nBcc: evil@example.com",
		TextBody: "text",
		HTMLBody: "<p>text</p>",
	})
	if strings.Contains(msg, "\r\nBcc: evil@example.com") {
		t.Fatal("header injection survived sanitization")
	}
	if !strings.Contains(msg, "multipart/alternative") {
		t.Fatal("expected multipart message")
	}
}

func TestActionURLIsLoggedWithoutEscapedMessageSuffix(t *testing.T) {
	var output bytes.Buffer
	sender := &LogSender{Log: slog.New(slog.NewJSONHandler(&output, nil))}
	link := "http://localhost:8080/?action=activate&token=" + strings.Repeat("A", 43)
	err := sender.Send(context.Background(), Message{To: "local@gmail.com", Subject: "Activation", ActionURL: link, TextBody: "Open " + link + "\n\nThe link expires."})
	if !errors.Is(err, ErrLogOnly) {
		t.Fatal("log mode must not report email delivery")
	}
	var entry map[string]any
	if err := json.Unmarshal(output.Bytes(), &entry); err != nil {
		t.Fatal(err)
	}
	if entry["link"] != link {
		t.Fatal("action URL changed")
	}
	if _, exists := entry["body"]; exists {
		t.Fatal("action links must not be embedded in a JSON-escaped email body")
	}
	if strings.Contains(output.String(), `\n`) {
		t.Fatal("escaped newline can contaminate copied URL")
	}
}
