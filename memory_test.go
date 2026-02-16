package agent

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestMemory_Session(t *testing.T) {
	sessionID := uuid.New().String()
	ctx := context.Background()

	err := testMemory.EnsureSession(ctx, sessionID)
	if err != nil {
		t.Fatalf("EnsureSession failed: %v", err)
	}

	// Verify session exists (indirectly via FK constraint or just another EnsureSession)
	err = testMemory.EnsureSession(ctx, sessionID)
	if err != nil {
		t.Fatalf("EnsureSession again failed: %v", err)
	}
}

func TestMemory_Messages(t *testing.T) {
	sessionID := uuid.New().String()
	ctx := context.Background()
	testMemory.EnsureSession(ctx, sessionID)

	msg1 := Message{
		Role:       "user",
		Content:    "Hello",
		CreatedAt:  time.Now().Unix(),
		TokenCount: 10,
	}
	if err := testMemory.AppendMessage(ctx, sessionID, msg1); err != nil {
		t.Fatalf("AppendMessage failed: %v", err)
	}

	msgs, err := testMemory.GetMessages(ctx, sessionID, 10)
	if err != nil {
		t.Fatalf("GetMessages failed: %v", err)
	}
	if len(msgs) != 1 {
		t.Errorf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0].Content != "Hello" {
		t.Errorf("expected content 'Hello', got '%s'", msgs[0].Content)
	}

	msg2 := Message{
		Role:       "assistant",
		Content:    "Hi",
		CreatedAt:  time.Now().Unix() + 1,
		TokenCount: 5,
	}
	if err := testMemory.AppendMessage(ctx, sessionID, msg2); err != nil {
		t.Fatalf("AppendMessage failed: %v", err)
	}

	msgs, err = testMemory.GetMessages(ctx, sessionID, 10)
	if err != nil {
		t.Fatalf("GetMessages failed: %v", err)
	}
	if len(msgs) != 2 {
		t.Errorf("expected 2 messages, got %d", len(msgs))
	}

	// Test DeleteMessages
	if err := testMemory.DeleteMessages(ctx, sessionID, []string{msgs[0].ID}); err != nil {
		t.Fatalf("DeleteMessages failed: %v", err)
	}

	msgs, err = testMemory.GetMessages(ctx, sessionID, 10)
	if err != nil {
		t.Fatalf("GetMessages failed: %v", err)
	}
	if len(msgs) != 1 {
		t.Errorf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0].Content != "Hi" {
		t.Errorf("expected content 'Hi', got '%s'", msgs[0].Content)
	}
}

func TestMemory_Episodes(t *testing.T) {
	sessionID := uuid.New().String()
	ctx := context.Background()
	testMemory.EnsureSession(ctx, sessionID)

	err := testMemory.SaveEpisode(ctx, sessionID, "Summary of convo", 100, "msg1", "msg2")
	if err != nil {
		t.Fatalf("SaveEpisode failed: %v", err)
	}

	eps, err := testMemory.GetEpisodes(ctx, sessionID, 10)
	if err != nil {
		t.Fatalf("GetEpisodes failed: %v", err)
	}
	if len(eps) != 1 {
		t.Errorf("expected 1 episode, got %d", len(eps))
	}
	if eps[0].Summary != "Summary of convo" {
		t.Errorf("expected summary 'Summary of convo', got '%s'", eps[0].Summary)
	}
}

func TestMemory_Knowledge(t *testing.T) {
	sessionID := uuid.New().String()
	ctx := context.Background()
	testMemory.EnsureSession(ctx, sessionID)

	// Save global knowledge
	if err := testMemory.SaveKnowledge(ctx, "", "Go is a programming language", "manual"); err != nil {
		t.Fatalf("SaveKnowledge failed: %v", err)
	}

	// Save session knowledge
	if err := testMemory.SaveKnowledge(ctx, sessionID, "My favorite color is blue", "manual"); err != nil {
		t.Fatalf("SaveKnowledge failed: %v", err)
	}

	// Search global
	kns, err := testMemory.SearchKnowledge(ctx, "programming", "", 10)
	if err != nil {
		t.Fatalf("SearchKnowledge failed: %v", err)
	}
	if len(kns) == 0 {
		t.Errorf("expected results for 'programming'")
	}

	// Search session (should include global)
	kns, err = testMemory.SearchKnowledge(ctx, "programming", sessionID, 10)
	if err != nil {
		t.Fatalf("SearchKnowledge failed: %v", err)
	}
	if len(kns) == 0 {
		t.Errorf("expected results for 'programming' in session")
	}

	// Search session specific
	kns, err = testMemory.SearchKnowledge(ctx, "blue", sessionID, 10)
	if err != nil {
		t.Fatalf("SearchKnowledge failed: %v", err)
	}
	if len(kns) == 0 {
		t.Errorf("expected results for 'blue' in session")
	}

	// Search session specific from another session (should be empty)
	kns, err = testMemory.SearchKnowledge(ctx, "blue", "other_session", 10)
	if err != nil {
		t.Fatalf("SearchKnowledge failed: %v", err)
	}
	if len(kns) != 0 {
		t.Errorf("expected no results for 'blue' in other session")
	}
}

func TestMemory_ToolLogs(t *testing.T) {
	sessionID := uuid.New().String()
	ctx := context.Background()
	testMemory.EnsureSession(ctx, sessionID)

	err := testMemory.LogToolCall(ctx, sessionID, "calculator", `{"a": 1, "b": 2}`, "3", "", 10)
	if err != nil {
		t.Fatalf("LogToolCall failed: %v", err)
	}

	logs, err := testMemory.GetToolLogs(ctx, sessionID, "calculator", 10)
	if err != nil {
		t.Fatalf("GetToolLogs failed: %v", err)
	}
	if len(logs) != 1 {
		t.Errorf("expected 1 log, got %d", len(logs))
	}
	if logs[0].OutputText != "3" {
		t.Errorf("expected output '3', got '%s'", logs[0].OutputText)
	}
}
