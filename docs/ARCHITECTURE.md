# Arquitectura del Sistema de Agente Personalizado

Este documento define la arquitectura técnica para la implementación de un sistema de Agente de IA personalizado, basado en los principios de diseño "bare metal" descritos en `docs/CUSTOM_AGENT.md` y siguiendo las directrices de ingeniería de `docs/DEFAULT_LLM_SKILL.md`.

## 1. Visión General y Filosofía

El objetivo es construir un sistema de agente orquestado manualmente, evitando la complejidad y abstracción de frameworks "caja negra" (como LangChain o AutoGen). La arquitectura prioriza:

*   **Control Determinista:** Uso de Máquinas de Estados Finitos (FSM) para gobernar el flujo de ejecución.
*   **Inyección de Dependencias:** Todos los componentes externos (LLM, Base de Datos, Herramientas) se definen mediante interfaces.
*   **Aislamiento de Memoria:** Gestión estricta del contexto por `tenant_id` y `session_id`.
*   **Observabilidad:** Trazabilidad completa de cada paso de razonamiento y ejecución de herramientas.

## 2. Componentes Principales del Sistema

La arquitectura se divide en capas de responsabilidad única, desacopladas mediante interfaces Go.

### 2.1. El Núcleo (Core Engine)

El corazón del sistema es el `AgentNode`, que actúa como el orquestador principal. No contiene lógica de negocio específica, sino la lógica de control del bucle de razonamiento.

*   **Responsabilidad:** Gestionar el ciclo *Percibir -> Razonar -> Actuar -> Observar*.
*   **Implementación:** Un bucle de control basado en FSM (Máquina de Estados Finitos).

### 2.2. Capa de Memoria (Memory Layer)

Siguiendo la jerarquía definida en `CUSTOM_AGENT.md`, la memoria se divide en:

1.  **Memoria de Trabajo (Short-Term):** Almacena el historial inmediato de la conversación y el estado actual de la FSM.
2.  **Memoria Episódica (Long-Term):** Almacén vectorial para recuperación semántica de interacciones pasadas.

### 2.3. Registro de Herramientas (Tool Registry)

Un componente agnóstico al protocolo que gestiona las capacidades del agente.

*   **Abstracción:** Define una interfaz común para todas las herramientas, independientemente de si son funciones locales o llamadas API externas.
*   **Seguridad:** Valida los argumentos contra esquemas JSON antes de la ejecución.

### 2.4. Cliente LLM (Model Gateway)

Una interfaz que abstrae al proveedor del modelo (OpenAI, Anthropic, Llama), permitiendo cambiar el "cerebro" sin alterar la lógica del agente.

## 3. Diagramas de Arquitectura

### 3.1. Diagrama de Contexto del Sistema

[Ver Diagrama de Contexto](diagrams/SYSTEM_CONTEXT.md)

### 3.2. Flujo de Ejecución (Patrón ReAct)

Este diagrama de secuencia ilustra cómo el Orquestador gestiona el ciclo de vida de una solicitud utilizando el patrón ReAct.

[Ver Diagrama de Flujo ReAct](diagrams/REACT_FLOW.md)

### 3.3. Máquina de Estados Finitos (FSM)

El comportamiento del agente no es libre; está restringido por estados para asegurar fiabilidad.

[Ver Diagrama de Estados FSM](diagrams/FSM_STATE.md)

## 4. Definición de Interfaces (Contrato Go)

Siguiendo `DEFAULT_LLM_SKILL.md`, definimos las interfaces clave que permiten la inyección de dependencias y el testing.

```go
package agent

import "context"

// LLMClient abstrae la interacción con el proveedor del modelo.
type LLMClient interface {
    Generate(ctx context.Context, prompt []Message, tools []ToolDefinition) (LLMResponse, error)
}

// MemoryStore gestiona la persistencia del estado y el historial.
type MemoryStore interface {
    GetSession(ctx context.Context, sessionID string) (*Session, error)
    SaveSession(ctx context.Context, session *Session) error
    AppendMessage(ctx context.Context, sessionID string, msg Message) error
}

// Tool define una capacidad ejecutable por el agente.
type Tool interface {
    Name() string
    Description() string
    Schema() string // JSON Schema de los argumentos
    Execute(ctx context.Context, argsJSON string) (string, error)
}

// Orchestrator coordina el flujo.
type Orchestrator interface {
    Run(ctx context.Context, input string, sessionID string) (string, error)
}
```

## 5. Estrategia de Implementación

1.  **Sin Frameworks:** Se utilizará la librería estándar de Go (`net/http`, `encoding/json`, `context`) para la lógica central.
2.  **Testing:**
    *   Mocks para `LLMClient` y `MemoryStore` para pruebas unitarias deterministas.
    *   Tests de integración para verificar el flujo completo de la FSM.
3.  **Gestión de Errores:** Las fallas en las herramientas no deben detener el agente; deben reportarse como observaciones de error al LLM para que intente corregirlas (Auto-corrección).

---
*Este documento sirve como referencia arquitectónica y debe actualizarse si cambian los patrones fundamentales del sistema.*
