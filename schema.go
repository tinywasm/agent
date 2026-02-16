package agent

const schema = `
CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY, created_at INTEGER NOT NULL DEFAULT (unixepoch()),
    updated_at INTEGER NOT NULL DEFAULT (unixepoch())
);
CREATE TABLE IF NOT EXISTS messages (
    id TEXT PRIMARY KEY, session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    role TEXT NOT NULL CHECK(role IN ('user','assistant','system','tool')),
    content TEXT NOT NULL, tool_name TEXT, tool_call_id TEXT,
    token_count INTEGER NOT NULL DEFAULT 0, created_at INTEGER NOT NULL DEFAULT (unixepoch())
);
CREATE INDEX IF NOT EXISTS idx_messages_session ON messages(session_id, created_at);

CREATE TABLE IF NOT EXISTS episodes (
    id TEXT PRIMARY KEY, session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    summary TEXT NOT NULL, token_count INTEGER NOT NULL DEFAULT 0,
    from_msg_id TEXT NOT NULL, to_msg_id TEXT NOT NULL,
    created_at INTEGER NOT NULL DEFAULT (unixepoch())
);
CREATE TABLE IF NOT EXISTS knowledge (
    id TEXT PRIMARY KEY, session_id TEXT REFERENCES sessions(id) ON DELETE CASCADE,
    content TEXT NOT NULL, source TEXT NOT NULL DEFAULT 'agent',
    created_at INTEGER NOT NULL DEFAULT (unixepoch())
);
CREATE VIRTUAL TABLE IF NOT EXISTS knowledge_fts USING fts5(
    content, content='knowledge', content_rowid='rowid'
);

-- Triggers to keep knowledge_fts in sync with knowledge
CREATE TRIGGER IF NOT EXISTS knowledge_ai AFTER INSERT ON knowledge BEGIN
  INSERT INTO knowledge_fts(rowid, content) VALUES (new.rowid, new.content);
END;
CREATE TRIGGER IF NOT EXISTS knowledge_ad AFTER DELETE ON knowledge BEGIN
  INSERT INTO knowledge_fts(knowledge_fts, rowid, content) VALUES('delete', old.rowid, old.content);
END;
CREATE TRIGGER IF NOT EXISTS knowledge_au AFTER UPDATE ON knowledge BEGIN
  INSERT INTO knowledge_fts(knowledge_fts, rowid, content) VALUES('delete', old.rowid, old.content);
  INSERT INTO knowledge_fts(rowid, content) VALUES (new.rowid, new.content);
END;

-- v2 (future): vector search via sqlite-vec extension.
-- CREATE VIRTUAL TABLE IF NOT EXISTS knowledge_vec USING vec0(embedding float[768]);
-- Each rowid matches the corresponding rowid in the knowledge table.
CREATE TABLE IF NOT EXISTS tool_logs (
    id TEXT PRIMARY KEY, session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    tool_name TEXT NOT NULL, input_json TEXT NOT NULL,
    output_text TEXT, error_text TEXT,
    duration_ms INTEGER NOT NULL DEFAULT 0, created_at INTEGER NOT NULL DEFAULT (unixepoch())
);
CREATE INDEX IF NOT EXISTS idx_tool_logs_session ON tool_logs(session_id, created_at);
`
