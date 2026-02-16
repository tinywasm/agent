# Agent Memory Architecture Diagram

This diagram visualizes the unified SQLite-based memory system for the `tinywasm/agent`, highlighting the isomorphic consistency between Backend and Frontend. Both environments use the same Go client, same SQL schema, and the same `sqlite-vec` extension — only the compilation target differs.

```mermaid
graph TD
    subgraph "Agent Runtime"
        Orchestrator["Agent Orchestrator (FSM)"]
        MemInterface["Memory Interface (Go)"]

        Orchestrator -->|Read/Write| MemInterface
    end

    subgraph "Data Storage Layer (SQLite)"
        SQLite["SQLite Engine"]

        MemInterface -->|SQL Queries| SQLite

        subgraph "Tables / Schema"
            T_Short["Short-Term Memory<br/>(Messages Table)"]
            T_Episodic["Episodic Memory<br/>(Episodes Table)"]
            T_Action["Action Memory<br/>(Tool Logs Table)"]
            T_Semantic["Semantic Memory<br/>(knowledge + knowledge_vec + FTS5)"]
        end

        SQLite --> T_Short
        SQLite --> T_Episodic
        SQLite --> T_Action
        SQLite --> T_Semantic
    end

    subgraph "Extensions & Search"
        VecExt["sqlite-vec<br/>(Vector Search)"]
        FTS["FTS5<br/>(Keyword Search)"]
        RRF["RRF Fusion Logic"]

        T_Semantic -.-> VecExt
        T_Semantic -.-> FTS
        VecExt --> RRF
        FTS --> RRF
        RRF -->|Ranked Results| MemInterface
    end

    subgraph "Compilation Targets"
        Backend["Backend<br/>Go + modernc.org/sqlite<br/>+ sqlite-vec native"]
        Frontend["Frontend (future)<br/>TinyGo WASM<br/>+ sqlite-vec WASM"]

        Backend -.->|same Go client| SQLite
        Frontend -.->|same Go client| SQLite

        Disk["Server Disk"]

        Backend -->|Persist| Disk
    end

    classDef storage fill:#f9f,stroke:#333,stroke-width:2px;
    classDef component fill:#dfd,stroke:#333,stroke-width:2px;
    classDef logic fill:#bbf,stroke:#333,stroke-width:2px;
    classDef target fill:#ffe,stroke:#333,stroke-width:2px;

    class SQLite,Disk storage;
    class Orchestrator,MemInterface component;
    class VecExt,FTS,RRF logic;
    class Backend,Frontend target;
```
