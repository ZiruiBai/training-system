package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// GAiosEndpoint is the G.AIOS runtime streaming endpoint.
const GAiosEndpoint = "https://ai.gazellio.com/adk/run_stream"

// gAIOSAgentID is the configured agent to invoke.
const gAIOSAgentID = "huNN49qHNBkmXY29"

var (
	// ErrAgentKeyMissing is returned when GAIOS_AGENT_API_KEY is not configured.
	ErrAgentKeyMissing = errors.New("GAIOS_AGENT_API_KEY is not configured on the server")
	// ErrAgentRequest is returned when the agent stream fails or returns an error.
	ErrAgentRequest = errors.New("AI assistant request failed")
)

// ChatRequest is the inbound training chat request.
type ChatRequest struct {
	EmployeeCode string `json:"employee_id"`
	Message      string `json:"message"`
	SessionID    string `json:"session_id"`
	Context      string `json:"-"` // backend-injected employee business data
}

// ChatResult is the normalised agent reply returned to the frontend.
type ChatResult struct {
	SessionID string `json:"session_id"`
	Answer    string `json:"answer"`
}

// AgentService proxies chat to the G.AIOS runtime agent and normalises the
// NDJSON stream into a single final answer.
type AgentService struct {
	endpoint string
	client   *http.Client
}

// NewAgentService constructs an AgentService. The endpoint can be overridden
// (used by tests); it defaults to the live G.AIOS runtime.
func NewAgentService(endpoint string) *AgentService {
	if endpoint == "" {
		endpoint = GAiosEndpoint
	}
	return &AgentService{
		endpoint: endpoint,
		client:   &http.Client{Timeout: 120 * time.Second},
	}
}

type streamMessage struct {
	SessionID string `json:"session_id"`
	Content   *struct {
		Parts []struct {
			Text    string `json:"text"`
			Thought *bool  `json:"thought"`
		} `json:"parts"`
	} `json:"content"`
}

// Chat sends a message to the agent and returns the assembled final answer.
func (s *AgentService) Chat(ctx context.Context, userID string, req ChatRequest) (*ChatResult, error) {
	apiKey := os.Getenv("GAIOS_AGENT_API_KEY")
	if apiKey == "" {
		return nil, ErrAgentKeyMissing
	}

	// Compose the message the agent sees: the user's question plus the
	// authoritative employee context injected by the backend.
	agentMessage := req.Message
	if strings.TrimSpace(req.Context) != "" {
		agentMessage = agentMessage + "\n\n【当前员工真实业务数据（由后端提供，请仅基于这些数据回答，不要自行查询或猜测）】\n" + req.Context
	}

	payload := map[string]any{
		"agent_id":   gAIOSAgentID,
		"user_id":    userID,
		"session_id": req.SessionID,
		"message":    agentMessage,
		"images":     []any{},
		"audios":     []any{},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAgentRequest, err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, s.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAgentRequest, err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/x-ndjson")
	httpReq.Header.Set("X-Agent-API-Key", apiKey)

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAgentRequest, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Read a bounded amount for diagnostics (never log secrets).
		buf, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		msg := strings.TrimSpace(string(buf))
		if len(msg) > 400 {
			msg = msg[:400]
		}
		return nil, fmt.Errorf("%w: status %d: %s", ErrAgentRequest, resp.StatusCode, msg)
	}

	return parseNDJSON(resp.Body)
}

// parseNDJSON reads the agent's newline-delimited JSON stream and assembles
// the final answer from all non-thought text parts, also capturing session_id.
func parseNDJSON(r io.Reader) (*ChatResult, error) {
	result := &ChatResult{}
	scanner := bufio.NewScanner(r)
	// Streamed parts can be large; allow a generous buffer.
	scanner.Buffer(make([]byte, 0, 256*1024), 4*1024*1024)
	var builder strings.Builder
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var msg streamMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			// Tolerate a non-JSON line but do not crash the stream.
			continue
		}
		if msg.SessionID != "" {
			result.SessionID = msg.SessionID
			continue
		}
		if msg.Content == nil {
			continue
		}
		for _, p := range msg.Content.Parts {
			// Skip internal reasoning; keep only user-visible text.
			if p.Thought != nil && *p.Thought {
				continue
			}
			builder.WriteString(p.Text)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAgentRequest, err)
	}
	result.Answer = strings.TrimSpace(builder.String())
	if result.Answer == "" {
		// The agent produced no user-visible text this turn (e.g. it only
		// reasoned internally or a tool call failed). Surface a graceful
		// fallback instead of failing the whole request.
		result.Answer = "抱歉，AI 助手本次未能生成有效回答，请稍后再试。"
	}
	return result, nil
}
