package agent

import (
	"context"
	"encoding/json"
)

// MockMCPClient implements mcpCaller with an injectable function field.
// Use it for unit tests that require deterministic MCP responses or error injection
// without spinning up a real HTTP server.
// Pattern: same as MockLLMClient.GenerateFunc.
var _ mcpCaller = (*MockMCPClient)(nil)

type MockMCPClient struct {
	CallFunc func(ctx context.Context, method string, params any) (json.RawMessage, error)
}

func (m *MockMCPClient) Call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	if m.CallFunc != nil {
		return m.CallFunc(ctx, method, params)
	}
	return nil, nil
}
