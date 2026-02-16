package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestReAct_ToolCallThenAnswer(t *testing.T) {
	sessionID := t.Name()
	ctx := context.Background()
	testMemory.EnsureSession(ctx, sessionID)

	// Mock MCP Server
	mcpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req jsonRPCRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		if req.Method == "tools/list" {
			resp := jsonRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result: json.RawMessage(`{
					"tools": [
						{
							"name": "calculator",
							"description": "Calculates sum",
							"inputSchema": {
								"type": "object",
								"properties": {
									"a": {"type": "integer"},
									"b": {"type": "integer"}
								}
							}
						}
					]
				}`),
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		if req.Method == "tools/call" {
			// Parse params
			// We expect "name": "calculator", "arguments": {a: 1, b: 2}
			// Just return "3"
			resp := jsonRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result: json.RawMessage(`{
					"content": [{"type": "text", "text": "3"}],
					"isError": false
				}`),
			}
			json.NewEncoder(w).Encode(resp)
			return
		}
	}))
	defer mcpServer.Close()

	// Mock LLM
	// Call 1: Reasoning -> Tool Call
	// Call 2: Reasoning (after tool) -> Answer
	// Call 3: Reflection -> SUFFICIENT

	step := 0
	mockLLM := &MockLLMClient{
		GenerateFunc: func(ctx context.Context, req LLMRequest) (LLMResponse, error) {
			step++
			// t.Logf("LLM called step %d, prompt: %s", step, req.SystemPrompt)

			// Check if reflection
			if strings.Contains(req.SystemPrompt, "critic that evaluates") {
				return LLMResponse{
					Text:       "SUFFICIENT",
					TokensUsed: 10,
					StopReason: "end_turn",
				}, nil
			}

			// Reasoning steps
			// If we see tool result in messages, we answer.
			hasToolResult := false
			for _, m := range req.Messages {
				if m.Role == "tool" {
					hasToolResult = true
					break
				}
			}

			if !hasToolResult {
				// First turn: Call tool
				return LLMResponse{
					Text:       "I will calculate 1+2.",
					StopReason: "tool_use",
					ToolCalls: []ToolCall{
						{
							ID:    "call_1",
							Name:  "calculator",
							Input: `{"a": 1, "b": 2}`,
						},
					},
					TokensUsed: 20,
				}, nil
			} else {
				// Second turn: Answer
				return LLMResponse{
					Text:       "The answer is 3.",
					StopReason: "end_turn",
					TokensUsed: 15,
				}, nil
			}
		},
	}

	cfg := Config{
		Identity: IdentityConfig{Name: "Bot"},
		LLMs: LLMConfig{
			Primary: mockLLM,
		},
		Memory:     testMemory,
		MCPServers: []string{mcpServer.URL},
	}

	agent, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	answer, err := agent.Run(ctx, sessionID, "What is 1+2?")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if answer != "The answer is 3." {
		t.Errorf("expected answer 'The answer is 3.', got '%s'", answer)
	}

	// Verify state
	if agent.fsm.current != StateIdle {
		t.Errorf("expected agent to be Idle, got %s", agent.fsm.current)
	}
}
