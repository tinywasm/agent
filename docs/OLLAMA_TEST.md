# Plan: Local Model Setup for Integration Tests — `tinywasm/agent`

## Context

The agent needs real integration tests that simulate a clinic receptionist scenario in Spanish (session isolation: 2 concurrent clients must not mix messages). Cloud models (Anthropic) cannot be used in test CI without API keys. The developer has Debian 12 + Docker; production is Windows Server 2012 (Xeon E-2224, 16GB DDR4, no Docker). The inference tool must work **natively on both platforms without Docker**.

---

## Decision: Ollama

| Criterion | Ollama | LM Studio | LocalAI |
|-----------|--------|-----------|---------|
| Native Linux binary | ✓ | ✓ | Docker-first |
| Native Windows binary (no Docker) | ✓ | GUI only | ✗ |
| REST API (OpenAI-compatible) | ✓ | ✓ | ✓ |
| Programmable health check | ✓ (`GET /api/tags`) | ✗ | ✓ |
| Tool calling support | ✓ (model-dependent) | ✓ | ✓ |
| Headless / service mode | ✓ | ✗ | ✓ |
| Future WASM frontend compatibility | REST API → tinywasm/fetch | REST | REST |

**Ollama** is the only tool that satisfies all constraints: native on both platforms, headless, REST API, programmable detection.

---

## Decision: Model

### Hardware Constraints (Production)
- Xeon E-2224: 4 cores @ 3.4GHz, DDR4 ECC 2666 (~42 GB/s bandwidth)
- 16GB RAM → OS 1.5GB + Go server 300MB = **~14GB available**
- Expected CPU inference: ~30-50 tokens/s (DDR4 bottleneck vs DDR5 benchmarks in docs)

### Model Selection

| Model | Ollama Tag | RAM (Q4_K_M) | Spanish | Tokens/s (DDR4 est.) | Tool Calling |
|-------|-----------|-------------|---------|----------------------|-------------|
| **qwen2.5:7b** ← **SELECTED** | `qwen2.5:7b` | ~4.7 GB | Excellent | ~50-70 | ✓ |
| llama3.1:8b | `llama3.1:8b` | ~5.2 GB | Good (89.5) | ~40-60 | ✓ |
| qwen3:8b | `qwen3:8b` | ~5.5 GB | Excellent | ~35-50 (thinking overhead) | ✓ |
| phi4-mini | `phi4-mini` | ~2.6 GB | Competent | ~80-100 | ✓ |

**Selected: `qwen2.5:7b`**
- Best RAM/Spanish ratio for this hardware
- Stable (not newest, well-tested in production scenarios)
- Supports tool calling via Ollama OpenAI-compatible API
- 128K context window — more than enough for clinic session tests

**Fallback**: `llama3.1:8b` if qwen2.5 has issues.

---

## New File: `llm_ollama.go`

Implements `LLMClient` using Ollama's OpenAI-compatible endpoint (`/v1/chat/completions`):

```go
package agent

// OllamaClient implements LLMClient using Ollama's OpenAI-compatible API.
// It uses the same wire format as llm_anthropic.go but targets localhost:11434.
type OllamaClient struct {
    baseURL string // default: "http://localhost:11434"
    model   string // e.g. "qwen2.5:7b"
    timeout time.Duration
}

func NewOllamaClient(model string) *OllamaClient {
    return &OllamaClient{
        baseURL: "http://localhost:11434",
        model:   model,
        timeout: 120 * time.Second, // local inference can be slow
    }
}

// Generate implements LLMClient.
// Endpoint: POST /v1/chat/completions (OpenAI-compatible)
func (c *OllamaClient) Generate(ctx context.Context, req LLMRequest) (LLMResponse, error) { ... }
```

**Wire format** (OpenAI-compatible):
```json
POST http://localhost:11434/v1/chat/completions
{
  "model": "qwen2.5:7b",
  "messages": [{"role": "system", "content": "..."}, {"role": "user", "content": "..."}],
  "tools": [...],
  "stream": false
}
```

---

## New File: `integration_test.go`

Build tag `//go:build integration` separates these from standard unit tests.

```go
//go:build integration

package agent_test

// TestMain checks if Ollama is available — skips all integration tests if not.
// This means standard `gotest` (no -tags) never blocks on Ollama.
func TestMain(m *testing.M) {
    if !ollamaAvailable() {
        fmt.Println("SKIP: Ollama not running at localhost:11434")
        os.Exit(0)
    }
    os.Exit(m.Run())
}

func ollamaAvailable() bool {
    resp, err := http.Get("http://localhost:11434/api/tags")
    return err == nil && resp.StatusCode == 200
}
```

**Clinic Scenarios (Spanish):**

```go
// TestIntegration_ClinicHours — single session
// System prompt: "Eres la recepcionista de Clínica San Miguel. Horario: L-V 8h-20h, Sáb 9h-14h."
// Input: "¿A qué hora abren los lunes?"
// Assert: response contains "8" or "lunes" or "horario"

// TestIntegration_SessionIsolation — 2 concurrent goroutines with DIFFERENT sessionIDs
// Session A: asks "¿Cuál es el horario del lunes?"
// Session B: asks "¿Tienen servicio de urgencias?"
// Run both concurrently with sync.WaitGroup
// Assert:
//   - Messages of session A do NOT appear in session B memory
//   - Messages of session B do NOT appear in session A memory
//   - Both receive coherent responses
```

---

## Installation Steps

### Phase 1: Developer Machine (Debian 12)

```bash
# 1. Install Ollama
curl -fsSL https://ollama.com/install.sh | sh

# 2. Pull model
ollama pull qwen2.5:7b

# 3. Smoke test (Spanish clinic query)
ollama run qwen2.5:7b "Eres la recepcionista de una clínica. ¿A qué hora abren los lunes?"
# Expected: response in Spanish mentioning hours

# 4. Verify API endpoint
curl http://localhost:11434/api/tags
# Expected: JSON with qwen2.5:7b in models list
```

### Phase 2: Run Integration Tests

```bash
go test -tags integration -run TestIntegration -v -timeout 300s ./...
```

Standard `gotest` (no `-tags integration`) skips these entirely — no breaking change.

### Phase 3: Production (Windows Server 2012 — future reference)

```powershell
# Download installer from https://ollama.com (Windows binary)
# Install and register as Windows Service:
ollama serve  # runs on port 11434
ollama pull qwen2.5:7b
```

---

## Cambios en `docs/IMPLEMENTATION.md`

Todo el contenido de este plan aterriza en secciones específicas de IMPLEMENTATION.md:

### §1 Project Structure — añadir archivos nuevos
```
├── llm_ollama.go               # OllamaClient — OpenAI-compatible local inference
├── integration_test.go         # //go:build integration — clinic real LLM tests
```
Y en **Test-Only Dependencies** añadir:
```
- Ollama (external process, not a Go import) — local inference for integration tests
```

### §6 Testing Strategy — nueva subsección "Real LLM Integration Tests"

```markdown
### Real LLM Integration Tests (`//go:build integration`)

Require Ollama running locally with `qwen2.5:7b`. Skipped automatically if Ollama
is not available. Run with:

    go test -tags integration -run TestIntegration -v -timeout 300s ./...

**Model:** `qwen2.5:7b` (Q4_K_M, ~4.7 GB RAM)
- Spanish benchmark: excellent (100+ languages)
- Tool calling: ✓
- Context: 128K tokens
- Fallback: `llama3.1:8b`

**Guard pattern (`integration_test.go`):**
```go
//go:build integration

func ollamaAvailable() bool {
    resp, err := http.Get("http://localhost:11434/api/tags")
    return err == nil && resp.StatusCode == 200
}
```

**Clinic Scenarios:**

| Test | Description | Assert |
|------|-------------|--------|
| `TestIntegration_ClinicHours` | Single session, asks "¿A qué hora abren los lunes?" | Response in Spanish mentions hours |
| `TestIntegration_SessionIsolation` | 2 concurrent goroutines, different sessionIDs | Messages of session A absent from session B memory |
```

### §10 Constructor Pattern — añadir `OllamaClient` en examples

```go
// Integration test usage:
agent.New(agent.Config{
    LLMs: agent.LLMConfig{
        Primary: agent.NewOllamaClient("qwen2.5:7b"),
    },
    ...
})
```

### §11 Implementation Sequence — añadir paso

```
| 9 | llm_ollama.go | types.go, interfaces.go |
```

---

## Cambios en `docs/DEFAULT_LLM_SKILL.md`

### §2 Testing — añadir en "Test-Only Dependencies"
```
**Local inference (external process, not Go import):**
- Ollama (v0.4+) with `qwen2.5:7b` — required for `//go:build integration` tests
- Install: curl -fsSL https://ollama.com/install.sh | sh && ollama pull qwen2.5:7b
- Detection: GET http://localhost:11434/api/tags
```

---

## Nuevo archivo: `llm_ollama.go`

```go
package agent

type OllamaClient struct {
    baseURL string        // default: "http://localhost:11434"
    model   string        // e.g. "qwen2.5:7b"
    timeout time.Duration // default: 120s
}

func NewOllamaClient(model string) *OllamaClient

// Generate implements LLMClient via POST /v1/chat/completions
func (c *OllamaClient) Generate(ctx context.Context, req LLMRequest) (LLMResponse, error)
```

---

## Nuevo archivo: `integration_test.go`

```go
//go:build integration

package agent_test

// Skips all tests if Ollama not running
// TestIntegration_ClinicHours
// TestIntegration_SessionIsolation — concurrent, verifies no session mixing
```

---

## Verificación

```bash
# 1. Instalar Ollama + modelo
curl -fsSL https://ollama.com/install.sh | sh
ollama pull qwen2.5:7b
curl http://localhost:11434/api/tags  # debe retornar JSON con qwen2.5:7b

# 2. Smoke test en español
ollama run qwen2.5:7b "Eres recepcionista de una clínica. ¿A qué hora abren los lunes?"

# 3. Tests
go test -tags integration -run TestIntegration -v -timeout 300s ./...

# 4. Verificar que tests normales no se rompen
gotest
```
