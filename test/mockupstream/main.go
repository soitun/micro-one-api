package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	relayprovider "micro-one-api/internal/relay/provider"
)

func main() {
	port := "9999"
	if p := ""; p != "" {
		port = p
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/chat/completions", handleChatCompletions)
	mux.HandleFunc("/health", handleHealth)

	log.Printf("Mock Upstream server listening on port %s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		panic(fmt.Sprintf("failed to start server: %v", err))
	}
}

func handleChatCompletions(w http.ResponseWriter, r *http.Request) {
	var req relayprovider.ChatCompletionsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid request: %v", err), http.StatusBadRequest)
		return
	}

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, "missing authorization", http.StatusUnauthorized)
		return
	}

	log.Printf("Received chat completions request: model=%s, messages=%d, auth=%s", req.Model, len(req.Messages), authHeader)

	content := fmt.Sprintf("Response to: %s", req.Messages[0].Content)
	if len(req.Messages) > 0 {
		content = fmt.Sprintf("Mock response for %s from model %s", req.Messages[0].Content, req.Model)
	}

	resp := relayprovider.ChatCompletionsResponse{
		ID:      "mock-" + fmt.Sprintf("%d", time.Now().Unix()),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   req.Model,
		Choices: []relayprovider.Choice{
			{
				Index: 0,
				Message: relayprovider.Message{
					Role:    "assistant",
					Content: content,
				},
				FinishReason: "stop",
			},
		},
		Usage: relayprovider.Usage{
			PromptTokens:     len(req.Messages[0].Content) / 4,
			CompletionTokens: len(content) / 4,
			TotalTokens:      (len(req.Messages[0].Content) + len(content)) / 4,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
