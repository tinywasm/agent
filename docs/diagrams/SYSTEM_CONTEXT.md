```mermaid
graph TD
    User[Usuario Final] -->|Consulta HTTP| API[API Gateway / Router]
    API -->|Enruta| Orchestrator[Orquestador (AgentNode)]

    subgraph "Core Agent Architecture"
        Orchestrator -->|Consulta| LLM[Interfaz LLM]
        Orchestrator -->|Lee/Escribe| Memory[Gestor de Memoria]
        Orchestrator -->|Ejecuta| Tools[Registro de Herramientas]
    end

    subgraph "Infraestructura"
        LLM -.->|HTTP| OpenAI[Proveedor Modelo (e.g. GPT-4)]
        Memory -.->|Persistencia| DB[(Redis / Vector DB)]
        Tools -.->|Acción| ExternalAPIs[APIs Externas]
    end
```
