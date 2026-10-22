package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/anthropics/anthropic-sdk-go/v2"
	"github.com/anthropics/anthropic-sdk-go/v2/option"
)

// sseEvent formats one server-sent event.
func sseEvent(event, data string) string {
	return fmt.Sprintf("event: %s\ndata: %s\n\n", event, data)
}

// TestAnthropicProviderQuery exercises the real AnthropicProvider against a fake
// streaming endpoint to guard the request shape and stream parsing across SDK upgrades.
func TestAnthropicProviderQuery(t *testing.T) {
	var gotBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(body, &gotBody); err != nil {
			t.Errorf("request body is not JSON: %v", err)
		}

		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, sseEvent("message_start",
			`{"type":"message_start","message":{"id":"msg_1","type":"message","role":"assistant","model":"m","content":[],"usage":{"input_tokens":1,"output_tokens":1}}}`))
		_, _ = io.WriteString(w, sseEvent("content_block_start",
			`{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`))
		_, _ = io.WriteString(w, sseEvent("content_block_delta",
			`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"ls -la\n"}}`))
		_, _ = io.WriteString(w, sseEvent("content_block_delta",
			`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"List files"}}`))
		_, _ = io.WriteString(w, sseEvent("content_block_stop", `{"type":"content_block_stop","index":0}`))
		_, _ = io.WriteString(w, sseEvent("message_stop", `{"type":"message_stop"}`))
	}))
	defer srv.Close()

	p := &AnthropicProvider{
		client: anthropic.NewClient(
			option.WithAPIKey("test-key"),
			option.WithBaseURL(srv.URL),
			option.WithMaxRetries(0),
		),
	}

	got, err := p.Query(context.Background(), "system prompt", "list files")
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}
	if want := "ls -la\nList files"; got != want {
		t.Errorf("Query() = %q, want %q", got, want)
	}

	// Request shape: one user message carrying the query, system prompt with caching.
	msgs, _ := gotBody["messages"].([]any)
	if len(msgs) != 1 {
		t.Fatalf("messages = %v, want exactly 1", gotBody["messages"])
	}
	msg, _ := msgs[0].(map[string]any)
	if msg["role"] != "user" {
		t.Errorf("message role = %v, want user", msg["role"])
	}
	content, _ := msg["content"].([]any)
	if len(content) != 1 {
		t.Fatalf("content = %v, want exactly 1 block", msg["content"])
	}
	block, _ := content[0].(map[string]any)
	if block["type"] != "text" || block["text"] != "list files" {
		t.Errorf("content block = %v, want text block with query", block)
	}

	system, _ := gotBody["system"].([]any)
	if len(system) != 1 {
		t.Fatalf("system = %v, want exactly 1 block", gotBody["system"])
	}
	sys, _ := system[0].(map[string]any)
	if sys["text"] != "system prompt" {
		t.Errorf("system text = %v, want system prompt", sys["text"])
	}
	if _, ok := sys["cache_control"]; !ok {
		t.Error("system block missing cache_control (prompt caching disabled)")
	}
	if gotBody["stream"] != true {
		t.Errorf("stream = %v, want true", gotBody["stream"])
	}
}

// TestAnthropicProviderQueryAPIError ensures HTTP errors surface instead of returning an empty answer.
func TestAnthropicProviderQueryAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `{"type":"error","error":{"type":"authentication_error","message":"invalid x-api-key"}}`)
	}))
	defer srv.Close()

	p := &AnthropicProvider{
		client: anthropic.NewClient(
			option.WithAPIKey("bad"),
			option.WithBaseURL(srv.URL),
			option.WithMaxRetries(0),
		),
	}

	if _, err := p.Query(context.Background(), "sys", "q"); err == nil {
		t.Fatal("Query() error = nil, want auth error")
	}
}
