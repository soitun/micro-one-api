package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestOpenAIProvider_ChatCompletionsStream(t *testing.T) {
	chunkCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST request, got %s", r.Method)
		}

		if r.Header.Get("Content-Type") != "application/json" {
			t.Fatalf("expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer test-api-key" {
			t.Fatalf("expected Authorization Bearer test-api-key, got %s", authHeader)
		}

		var req ChatCompletionsRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		if !req.Stream {
			t.Fatalf("expected stream=true, got %v", req.Stream)
		}

		// Send SSE response
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")

		chunks := []string{
			`data: {"id":"chunk1","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4o-mini","choices":[{"index":0,"delta":{"content":"Hello"},"finish_reason":null}]}`,
			`data: {"id":"chunk2","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4o-mini","choices":[{"index":0,"delta":{"content":" world"},"finish_reason":null}]}`,
			`data: {"id":"chunk3","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4o-mini","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`,
			`data: [DONE]`,
		}

		for _, chunk := range chunks {
			w.Write([]byte(chunk + "\n\n"))
			w.(http.Flusher).Flush()
		}
	}))
	defer server.Close()

	provider := NewOpenAIProvider(server.URL, "test-api-key", 30*time.Second)

	req := &ChatCompletionsRequest{
		Model: "gpt-4o-mini",
		Messages: []Message{
			{Role: "user", Content: "Hello"},
		},
		Stream: true,
	}

	chunkChan, err := provider.ChatCompletionsStream(context.Background(), req)
	if err != nil {
		t.Fatalf("ChatCompletionsStream() error = %v", err)
	}

	var fullContent strings.Builder
	for chunk := range chunkChan {
		chunkCount++
		if len(chunk.Choices) > 0 {
			fullContent.WriteString(chunk.Choices[0].Delta.Content)
		}
	}

	if chunkCount != 3 {
		t.Fatalf("expected 3 chunks, got %d", chunkCount)
	}

	if fullContent.String() != "Hello world" {
		t.Fatalf("expected content 'Hello world', got '%s'", fullContent.String())
	}
}

func TestOpenAIProvider_ChatCompletionsStream_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": {"message": "Invalid API key"}}`))
	}))
	defer server.Close()

	provider := NewOpenAIProvider(server.URL, "test-api-key", 30*time.Second)

	req := &ChatCompletionsRequest{
		Model:    "gpt-4o-mini",
		Messages: []Message{{Role: "user", Content: "Hello"}},
		Stream:   true,
	}

	_, err := provider.ChatCompletionsStream(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for unauthorized request")
	}
}