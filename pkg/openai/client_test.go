package openai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestEmbed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/embeddings" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("unexpected authorization: %q", got)
		}
		var body struct {
			Model      string   `json:"model"`
			Input      []string `json:"input"`
			Dimensions int      `json:"dimensions"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Model != "text-embedding-3-small" || body.Dimensions != 1536 || len(body.Input) != 1 {
			t.Fatalf("unexpected request: %+v", body)
		}
		embedding := make([]float64, 1536)
		embedding[0] = 0.5
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []any{map[string]any{"index": 0, "embedding": embedding}},
		})
	}))
	defer server.Close()

	client := NewClient("test-key", server.URL, "gpt-5.6", "text-embedding-3-small", time.Second)
	result, err := client.Embed(context.Background(), []string{"hello"})
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 || len(result[0]) != 1536 || result[0][0] != 0.5 {
		t.Fatalf("unexpected embeddings")
	}
}

func TestRespond(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/responses" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		var body struct {
			Model            string    `json:"model"`
			Instructions     string    `json:"instructions"`
			Input            []Message `json:"input"`
			Store            bool      `json:"store"`
			SafetyIdentifier string    `json:"safety_identifier"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Model != "gpt-5.6" || body.Instructions == "" || len(body.Input) != 1 || body.Store || body.SafetyIdentifier != "hashed-user" {
			t.Fatalf("unexpected request: %+v", body)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"output": []any{map[string]any{
				"type":    "message",
				"content": []any{map[string]any{"type": "output_text", "text": "  answer  "}},
			}},
		})
	}))
	defer server.Close()

	client := NewClient("test-key", server.URL, "gpt-5.6", "text-embedding-3-small", time.Second)
	answer, err := client.Respond(context.Background(), "instructions", []Message{{Role: "user", Content: "question"}}, "hashed-user")
	if err != nil {
		t.Fatal(err)
	}
	if answer != "answer" {
		t.Fatalf("unexpected answer: %q", answer)
	}
}

func TestDisabledWithoutKey(t *testing.T) {
	client := NewClient("", "", "gpt-5.6", "text-embedding-3-small", time.Second)
	if client.Enabled() {
		t.Fatal("client must be disabled without API key")
	}
}

func TestRespondReturnsRefusalText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"output": []any{map[string]any{
				"type":    "message",
				"content": []any{map[string]any{"type": "refusal", "refusal": "I can't help with that."}},
			}},
		})
	}))
	defer server.Close()

	client := NewClient("test-key", server.URL, "gpt-5.6", "text-embedding-3-small", time.Second)
	answer, err := client.Respond(context.Background(), "instructions", []Message{{Role: "user", Content: "question"}}, "hashed-user")
	if err != nil {
		t.Fatal(err)
	}
	if answer != "I can't help with that." {
		t.Fatalf("unexpected refusal: %q", answer)
	}
}
