package agent

import "context"

type LLMClient interface {
	Generate(ctx context.Context, req LLMRequest) (LLMResponse, error)
}

type MemoryStore interface {
	EnsureSession(ctx context.Context, sessionID string) error
	AppendMessage(ctx context.Context, sessionID string, msg Message) error
	GetMessages(ctx context.Context, sessionID string, limit int) ([]Message, error)
	DeleteMessages(ctx context.Context, sessionID string, ids []string) error
	SaveEpisode(ctx context.Context, sessionID, summary string, tokenCount int, fromID, toID string) error
	GetEpisodes(ctx context.Context, sessionID string, limit int) ([]Episode, error)
	SaveKnowledge(ctx context.Context, sessionID, content, source string) error
	SearchKnowledge(ctx context.Context, query, sessionID string, limit int) ([]Knowledge, error)
	LogToolCall(ctx context.Context, sessionID, toolName, inputJSON, outputText, errText string, durationMS int64) error
	GetToolLogs(ctx context.Context, sessionID, toolName string, limit int) ([]ToolLog, error)
}

type MCPServer interface {
	URL() string
}

type Tool interface {
	Name() string
	Description() string
	InputSchema() string
	Execute(ctx context.Context, argsJSON string) (string, error)
}
