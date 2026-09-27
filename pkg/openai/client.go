package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"
)

const maxErrorBody = 8 << 10

type Client struct {
	apiKey         string
	baseURL        string
	chatModel      string
	embeddingModel string
	httpClient     *http.Client
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func NewClient(apiKey, baseURL, chatModel, embeddingModel string, timeout time.Duration) *Client {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	return &Client{
		apiKey:         strings.TrimSpace(apiKey),
		baseURL:        baseURL,
		chatModel:      strings.TrimSpace(chatModel),
		embeddingModel: strings.TrimSpace(embeddingModel),
		httpClient:     &http.Client{Timeout: timeout},
	}
}

func (c *Client) Enabled() bool { return c != nil && c.apiKey != "" }

func (c *Client) Embed(ctx context.Context, inputs []string) ([][]float64, error) {
	if !c.Enabled() {
		return nil, errors.New("OpenAI API key is not configured")
	}
	if len(inputs) == 0 {
		return [][]float64{}, nil
	}
	payload := struct {
		Model      string   `json:"model"`
		Input      []string `json:"input"`
		Dimensions int      `json:"dimensions"`
	}{Model: c.embeddingModel, Input: inputs, Dimensions: 1536}

	var response struct {
		Data []struct {
			Index     int       `json:"index"`
			Embedding []float64 `json:"embedding"`
		} `json:"data"`
	}
	if err := c.postJSON(ctx, "/embeddings", payload, &response); err != nil {
		return nil, err
	}
	if len(response.Data) != len(inputs) {
		return nil, fmt.Errorf("embedding response count mismatch: got %d want %d", len(response.Data), len(inputs))
	}
	result := make([][]float64, len(inputs))
	seen := make([]bool, len(inputs))
	for _, item := range response.Data {
		if item.Index < 0 || item.Index >= len(result) || seen[item.Index] || len(item.Embedding) != 1536 {
			return nil, fmt.Errorf("invalid embedding response")
		}
		for _, value := range item.Embedding {
			if math.IsNaN(value) || math.IsInf(value, 0) {
				return nil, fmt.Errorf("invalid embedding response")
			}
		}
		seen[item.Index] = true
		result[item.Index] = item.Embedding
	}
	for _, ok := range seen {
		if !ok {
			return nil, fmt.Errorf("invalid embedding response")
		}
	}
	return result, nil
}

func (c *Client) Respond(ctx context.Context, instructions string, messages []Message, safetyIdentifier string) (string, error) {
	if !c.Enabled() {
		return "", errors.New("OpenAI API key is not configured")
	}
	payload := struct {
		Model            string    `json:"model"`
		Instructions     string    `json:"instructions"`
		Input            []Message `json:"input"`
		MaxOutputTokens  int       `json:"max_output_tokens"`
		Store            bool      `json:"store"`
		SafetyIdentifier string    `json:"safety_identifier,omitempty"`
	}{
		Model:            c.chatModel,
		Instructions:     instructions,
		Input:            messages,
		MaxOutputTokens:  900,
		Store:            false,
		SafetyIdentifier: strings.TrimSpace(safetyIdentifier),
	}

	var response struct {
		Output []struct {
			Type    string `json:"type"`
			Content []struct {
				Type    string `json:"type"`
				Text    string `json:"text"`
				Refusal string `json:"refusal"`
			} `json:"content"`
		} `json:"output"`
	}
	if err := c.postJSON(ctx, "/responses", payload, &response); err != nil {
		return "", err
	}
	var parts []string
	for _, out := range response.Output {
		for _, content := range out.Content {
			switch content.Type {
			case "output_text":
				if text := strings.TrimSpace(content.Text); text != "" {
					parts = append(parts, text)
				}
			case "refusal":
				if refusal := strings.TrimSpace(content.Refusal); refusal != "" {
					parts = append(parts, refusal)
				}
			}
		}
	}
	text := strings.TrimSpace(strings.Join(parts, "\n"))
	if text == "" {
		return "", errors.New("OpenAI returned an empty response")
	}
	return text, nil
}

func (c *Client) postJSON(ctx context.Context, path string, payload, dst any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode OpenAI request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create OpenAI request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("OpenAI request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		limited := io.LimitReader(resp.Body, maxErrorBody)
		data, _ := io.ReadAll(limited)
		var apiErr struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		message := strings.TrimSpace(string(data))
		if json.Unmarshal(data, &apiErr) == nil && strings.TrimSpace(apiErr.Error.Message) != "" {
			message = apiErr.Error.Message
		}
		return fmt.Errorf("OpenAI API returned %d: %s", resp.StatusCode, message)
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 16<<20)).Decode(dst); err != nil {
		return fmt.Errorf("decode OpenAI response: %w", err)
	}
	return nil
}
