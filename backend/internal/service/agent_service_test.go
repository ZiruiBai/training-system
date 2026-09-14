package service

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseNDJSON_ExtractsAnswerAndFiltersThought(t *testing.T) {
	stream := `{"session_id":"sess-123"}
{"content":{"parts":[{"text":"I should ","thought":true}],"role":"model"},"partial":true}
{"content":{"parts":[{"text":"think about this","thought":true}],"role":"model"},"partial":true}
{"content":{"parts":[{"text":"Hello","thought":false}],"role":"model"},"partial":true}
{"content":{"parts":[{"text":"! I am the assistant","thought":false}],"role":"model"},"partial":true}
`
	res, err := parseNDJSON(strings.NewReader(stream))
	if err != nil {
		t.Fatalf("parseNDJSON error: %v", err)
	}
	if res.SessionID != "sess-123" {
		t.Fatalf("session_id = %q, want sess-123", res.SessionID)
	}
	// Only non-thought text parts are retained, joined.
	want := "Hello! I am the assistant"
	if res.Answer != want {
		t.Fatalf("answer = %q, want %q", res.Answer, want)
	}
}

func TestParseNDJSON_EmptyAnswer(t *testing.T) {
	stream := `{"session_id":"s"}`
	res, err := parseNDJSON(strings.NewReader(stream))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Answer == "" {
		t.Fatal("expected a graceful fallback answer, got empty")
	}
}

func TestAgentService_Chat_EndToEnd(t *testing.T) {
	t.Setenv("GAIOS_AGENT_API_KEY", "test-key")

	var gotBody, gotAuth, gotAccept string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("X-Agent-API-Key")
		gotAccept = r.Header.Get("Accept")
		buf := make([]byte, 4096)
		n, _ := r.Body.Read(buf)
		gotBody = string(buf[:n])
		w.Header().Set("Content-Type", "application/x-ndjson")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"session_id":"abc"}
{"content":{"parts":[{"text":"reasoning","thought":true}],"role":"model"},"partial":true}
{"content":{"parts":[{"text":"Final ","thought":false}],"role":"model"},"partial":true}
{"content":{"parts":[{"text":"answer","thought":false}],"role":"model"},"partial":true}
`))
	}))
	defer srv.Close()

	svc := NewAgentService(srv.URL)
	res, err := svc.Chat(context.Background(), "employee:1", ChatRequest{Message: "hi"})
	if err != nil {
		t.Fatalf("Chat error: %v", err)
	}
	if gotAuth != "test-key" {
		t.Fatalf("X-Agent-API-Key = %q, want test-key", gotAuth)
	}
	if gotAccept != "application/x-ndjson" {
		t.Fatalf("Accept = %q, want application/x-ndjson", gotAccept)
	}
	if !strings.Contains(gotBody, `"agent_id":"huNN49qHNBkmXY29"`) {
		t.Fatalf("request body missing agent_id: %s", gotBody)
	}
	if !strings.Contains(gotBody, `"user_id":"employee:1"`) {
		t.Fatalf("request body missing user_id: %s", gotBody)
	}
	if res.SessionID != "abc" {
		t.Fatalf("session_id = %q, want abc", res.SessionID)
	}
	if res.Answer != "Final answer" {
		t.Fatalf("answer = %q, want %q", res.Answer, "Final answer")
	}
}

func TestAgentService_Chat_MissingKey(t *testing.T) {
	t.Setenv("GAIOS_AGENT_API_KEY", "")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()

	svc := NewAgentService(srv.URL)
	_, err := svc.Chat(context.Background(), "employee:1", ChatRequest{Message: "hi"})
	if !errors.Is(err, ErrAgentKeyMissing) {
		t.Fatalf("expected ErrAgentKeyMissing, got %v", err)
	}
}

func TestAgentService_Chat_Non200(t *testing.T) {
	t.Setenv("GAIOS_AGENT_API_KEY", "test-key")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "forbidden", http.StatusForbidden)
	}))
	defer srv.Close()

	svc := NewAgentService(srv.URL)
	_, err := svc.Chat(context.Background(), "employee:1", ChatRequest{Message: "hi"})
	if !errors.Is(err, ErrAgentRequest) {
		t.Fatalf("expected ErrAgentRequest, got %v", err)
	}
}
