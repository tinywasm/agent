package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMCPClient_Discovery(t *testing.T) {
	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
							"name": "test_tool",
							"description": "A test tool",
							"inputSchema": {"type": "object"}
						}
					]
				}`),
			}
			json.NewEncoder(w).Encode(resp)
			return
		}
		http.Error(w, "unknown method", http.StatusNotFound)
	}))
	defer server.Close()

	client := NewHTTPMCPClient(server.URL)
	res, err := client.Call(context.Background(), "tools/list", nil)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	var list listToolsResult
	if err := json.Unmarshal(res, &list); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if len(list.Tools) != 1 {
		t.Errorf("expected 1 tool, got %d", len(list.Tools))
	}
	if list.Tools[0].Name != "test_tool" {
		t.Errorf("expected tool name 'test_tool', got '%s'", list.Tools[0].Name)
	}
}

func TestMCPClient_CallTool(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req jsonRPCRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		if req.Method == "tools/call" {
			// Check params
			// We expect params to be map with name and arguments
			// But here we just mock response

			resp := jsonRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result: json.RawMessage(`{
					"content": [
						{
							"type": "text",
							"text": "tool output"
						}
					],
					"isError": false
				}`),
			}
			json.NewEncoder(w).Encode(resp)
			return
		}
		http.Error(w, "unknown method", http.StatusNotFound)
	}))
	defer server.Close()

	client := NewHTTPMCPClient(server.URL)
	res, err := client.Call(context.Background(), "tools/call", map[string]any{
		"name": "test_tool",
		"arguments": map[string]any{},
	})
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	var result callToolResult
	if err := json.Unmarshal(res, &result); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if len(result.Content) != 1 {
		t.Errorf("expected 1 content block, got %d", len(result.Content))
	}
	if result.Content[0].Text != "tool output" {
		t.Errorf("expected output 'tool output', got '%s'", result.Content[0].Text)
	}
}
