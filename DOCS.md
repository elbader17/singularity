# Documentación Técnica - Singularity

## Índice

1. [Filosofía](#filosofía)
2. [Arquitectura](#arquitectura)
3. [Almacenamiento](#almacenamiento)
4. [Protocolo MCP](#protocolo-mcp)
5. [Modelo de Datos](#modelo-de-datos)
6. [Flujos de Ejecución](#flujos-de-ejecución)

---

## Filosofía

### El Request es Rey

El costo principal en APIs modernas de IA no es el número de tokens sino la latencia y el número de requests. Singularity optimiza para:

- **Minimizar requests**: Cada request debe contener toda la información necesaria
- **Pensar antes de actuar**: El agente debe razonar internamente antes de emitir salida
- **Consolidar al terminar**: `commit_world_state` atomiciza todo el trabajo

### Patrón Blackboard

Los agentes no se comunican directamente. En su lugar:

1. **Escriben en la pizarra** (BadgerDB)
2. **Leen de la pizarra** (contexto inicial)
3. **Mueren** (sesión termina)

Esto elimina la necesidad de mantener estado en memoria entre interacciones.

---

## Arquitectura

```
                    ┌─────────────────┐
                    │   OpenCode      │
                    │   (Cliente)     │
                    └────────┬────────┘
                             │ stdio
                    ┌────────▼────────┐
                    │  MCP Server     │
                    │  (mcp-go)       │
                    └────────┬────────┘
                             │
         ┌───────────────────┼───────────────────┐
         │                   │                   │
┌────────▼────────┐  ┌──────▼──────┐  ┌────────▼────────┐
│   Storage       │  │   Models    │  │   Protocol     │
│  (BadgerDB)     │  │  (State)    │  │  (Injector)    │
└─────────────────┘  └─────────────┘  └────────────────┘
```

### Componentes

#### `internal/storage/badger.go`

Capa de persistencia con prefijos de clave:

| Prefijo | Uso |
|---------|-----|
| `brain:` | Cerebro activo (estado actual) |
| `archive:` | Archivo profundo (historial) |
| `task:` | Tareas individuales |
| `subagent:` | Sub-agentes |

#### `internal/mcp/server.go`

Servidor MCP con herramientas registradas:

- Manejo de requests JSON-RPC
- Validación de parámetros
- Persistencia automática

#### `internal/models/state.go`

Estructuras de datos inmutables con métodos de serialización.

---

## Almacenamiento

### BadgerDB

Base de datos clave-valor optimizada para SSD:

- **Write-optimized**: LSM tree
- **Transacciones**: ACID
- **Sin внешних dependencias**: Solo Go

### Jerarquía de Datos

```
Cerebro Activo (brain:project-path)
├── session_id
├── current_task_id
├── active_tasks[]
├── completed_tasks[]
├── decisions[]
└── last_updated

Tarea (task:task-id)
├── id
├── title
├── description
├── status
├── assignee (sub-agent ID)
├── depends_on[]
└── timestamps

Sub-Agente (subagent:sub-id)
├── id
├── task_id
├── title
├── description
├── context
├── status
└── timestamps
```

---

## Protocolo MCP

### Inicialización

```
Cliente → Server: initialize
Server → Cliente: capabilities
Cliente → Server: initialized
```

### Llamada a Herramienta

```
Cliente → Server: tools/call
         ├─ name: "commit_world_state"
         └─ arguments: {...}
         
Server → Cliente: tools/call result
         └─ content: [{type: "text", text: "..."}]
```

### Recursos

```
singularity://brain  → Estado actual en JSON
```

---

## Modelo de Datos

### WorldState

```go
type WorldState struct {
    SessionID      string
    ProjectPath    string
    CurrentTaskID  string
    ActiveTasks    []string
    CompletedTasks []string
    Decisions      []Decision
    LastUpdated    time.Time
    Metadata       map[string]string
}
```

### Task

```go
type Task struct {
    ID          string
    Title       string
    Description string
    Status      TaskStatus  // pending, in_progress, completed
    Priority    int
    Assignee    string      // sub-agent ID
    DependsOn   []string
    CreatedAt   time.Time
    UpdatedAt   time.Time
    CompletedAt *time.Time
}
```

### SubAgent

```go
type SubAgent struct {
    ID          string
    TaskID      string
    Title       string
    Description string
    Context     string    // Aislamiento: solo esto ve el sub-agente
    Status      SubAgentStatus  // pending, running, completed, failed
    Result      string
    Error       string
    CreatedAt   time.Time
    StartedAt   *time.Time
    CompletedAt *time.Time
}
```

---

## Flujos de Ejecución

### Flujo 1: Sesión Normal

```
1. OpenCode inicia
2. MCP Server arranca (stdio)
3. OpenCode запроса herramientas
4. [Usuario trabaja]
5. usuario → commit_world_state()
6. Server actualiza BadgerDB
7. OpenCode cierra
8. MCP Server cierra
```

### Flujo 2: Orquestador → Sub-agente

```
ORQUESTADOR                          SUB-AGENTE
    │                                    │
    ├─ spawn_sub_agent()                │
    │   → Crea task en BD                │
    │   → Crea sub-agent en BD           │
    │   → Retorna sub_agent_id           │
    │                                    │
    │                            ├─ get_sub_agent_task(sub_agent_id)
    │                            │   → Lee sub-agent de BD
    │                            │   → Actualiza status a "running"
    │                            │   → Retorna contexto
    │                            │
    │                            ├─ [TRABAJA]
    │                            │
    │                            ├─ complete_sub_agent_task()
    │                            │   → Actualiza status a "completed"
    │                            │   → Guarda resultado
    │                            │
    ├─ get_active_brain()               │
    │   → Ve resultado del sub-agente    │
```

### Flujo 3: Cambio de Agente (switch_agent)

```
# En misma sesión de OpenCode

> Cambia a modo sub-agente
switch_agent(mode="sub_agent", sub_agent_id="abc")

→ Retorna contexto del sub-agent
→ Estado interno cambia a "sub_agent mode"

> Trabajas en la tarea...

> Regresa a modo orquestador  
switch_agent(mode="orchestrator")

→ Estado interno cambia a "orchestrator mode"
```

---

## Consideraciones de Seguridad

1. **Sin autenticación**: MCP local asume acceso confiable
2. **Datos en ~/.singularity**:路径 configurable
3. **Sin cifrado**: Usar FileVault/EncFS si es necesario

---

## Limitaciones Conocidas

1. **Un solo proceso**: No soport múltiples conexiones simultáneas
2. **stdio only**: No hay servidor HTTP
3. **Sin sincronización**: Cada proyecto es aislado

---

## Futuras Mejoras

- [ ] Servidor HTTP/SSE para remote MCP
- [ ] Autenticación OAuth
- [ ] Sincronización entre dispositivos
- [ ] Métricas y analytics
- [ ] Integración con más editores
