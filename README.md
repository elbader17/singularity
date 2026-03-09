# Singularity

> Motor de memoria de estado y orquestación para agentes de IA

Singularity es un sistema de memoria persistente diseñado para maximizar la autonomía de agentes de IA como OpenCode. Implementa el patrón **Blackboard (Pizarra Centralizada)** donde los agentes leen y escriben estado sin comunicarse directamente entre sí.

## Características

- **Cerebro Activo**: Estado actual del proyecto siempre disponible
- **Archivo Profundo**: Contexto histórico cuando es necesario
- **Orquestación de Tareas**: DAG de tareas con dependencias
- **Sub-Agentes AISLADOS**: Cada agente recibe solo su contexto específico
- **Zero Ping-Pong**: Un request contiene toda la información necesaria

## Instalación

### Prerrequisitos

- Go 1.23+
- OpenCode instalado

### Compilar

```bash
git clone <repo>
cd singularity
go build -o singularity ./cmd/singularity
```

### Instalar en OpenCode

```bash
./singularity init
```

Esto agregará automáticamente la configuración a `~/.config/opencode/opencode.jsonc`

### Configuración manual

Agrega a tu `opencode.jsonc`:

```jsonc
{
  "mcp": {
    "singularity": {
      "type": "local",
      "command": ["./singularity"],
      "enabled": true
    }
  }
}
```

## Uso

### Iniciar servidor MCP

```bash
./singularity
```

### Instalar en OpenCode

```bash
./singularity init
```

### Ver ayuda

```bash
./singularity help
```

## Herramientas MCP

### commit_world_state

Consolida el estado después de completar una tarea. **Uso obligatorio** al terminar.

```json
{
  "session_id": "ses-123",
  "project_path": "/path/to/project",
  "completed_task_id": "task-456",
  "task_summary": "Implementado sistema de login",
  "orchestrator_summary": "Autenticación lista para integrar con API",
  "learned_insights": "El módulo auth tiene código reusable"
}
```

### get_active_brain

Obtiene el estado actual del proyecto sin costo de API.

```json
{
  // Respuesta
  {
    "tasks": [...],
    "recent_decisions": [...],
    "active_count": 2,
    "pending_count": 5,
    "completed_count": 10
  }
}
```

### list_tasks

Lista todas las tareas con su estado.

```json
{
  "status": "pending"  // opcional: pending, in_progress, completed
}
```

### fetch_deep_context

Recupera contexto histórico. Usar solo cuando sea necesario.

```json
{
  "query": "autenticación",
  "task_id": "task-123",  // opcional
  "limit": 10
}
```

### spawn_sub_agent

Crea un sub-agente para ejecutar una tarea específica.

```json
{
  "session_id": "ses-123",
  "project_path": "/path/to/project",
  "title": "Implementar logout",
  "description": "Crear endpoint de cierre de sesión",
  "context": "Usa el código de auth/ existente"
}
```

**Respuesta:**
```json
{
  "success": true,
  "sub_agent_id": "sub-abc123",
  "task_id": "task-789"
}
```

### get_sub_agent_task

Obtiene la tarea asignada (para sub-agentes).

```json
{
  "sub_agent_id": "sub-abc123"
}
```

### complete_sub_agent_task

Completa la tarea del sub-agente.

```json
{
  "sub_agent_id": "sub-abc123",
  "result": "Logout implementado con JWT blacklist"
}
```

### switch_agent

Cambia entre modo Orquestador y Sub-agente en la misma sesión.

```json
{
  "mode": "sub_agent",  // o "orchestrator"
  "sub_agent_id": "sub-abc123"  // requerido si mode=sub_agent
}
```

## Arquitectura

```
┌─────────────────────────────────────────────────────┐
│                    Singularity                      │
│  ┌─────────────┐  ┌─────────────┐  ┌────────────┐ │
│  │ Cerebro    │  │ Archivo     │  │ Tareas     │ │
│  │ Activo     │  │ Profundo    │  │ (DAG)      │ │
│  └─────────────┘  └─────────────┘  └────────────┘ │
└─────────────────────────────────────────────────────┘
         ▲                  ▲                  ▲
         │                  │                  │
    ┌────┴────┐       ┌────┴────┐        ┌────┴────┐
    │Orquestador│       │Sub-agente│       │Sub-agente│
    └──────────┘       └──────────┘        └──────────┘
```

### Componentes

| Componente | Descripción |
|------------|-------------|
| `internal/storage` | BadgerDB para persistencia |
| `internal/mcp` | Servidor MCP + herramientas |
| `internal/models` | Estructuras de datos |
| `internal/protocol` | Inyección de contexto |

## Patrón de Trabajo

### Flujo Orquestador

```
1. get_active_brain()     → Ver estado actual
2. list_tasks()           → Ver tareas pendientes  
3. spawn_sub_agent()      → Crear tarea para sub-agente
4. [ESPERAR]
5. commit_world_state()   → Consolidar resultados
```

### Flujo Sub-agente

```
1. get_sub_agent_task()    → Obtener mi tarea y contexto
2. [TRABAJAR]
3. commit_world_state()   → Consolidar mi trabajo
4. complete_sub_agent_task() → Reportar al orquestador
```

## Reglas Fundamentales

1. **Un request = toda la información**
   - No preguntes "¿dónde nos quedamos?" - usa `get_active_brain`
   - No hagas ping-pong exploratorio - deduce y actúa

2. **Aislamiento estricto**
   - Orquestador solo ve resúmenes de alto nivel
   - Sub-agente solo ve su contexto específico

3. **Consolidación obligatoria**
   - Siempre usa `commit_world_state` al terminar
   - Incluye código, decisiones, aprendizajes

## Ejemplo de Sesión

```bash
# Sesión 1 - Orquestador
$ opencode .

> ¿Cuál es el estado del proyecto?
[Usa get_active_brain]

> Crea una tarea para implementar el login
[Usa spawn_sub_agent]

# Sesión 2 - Sub-agente (nueva terminal)
$ opencode .

> Mi tarea es...
[Usa get_sub_agent_task]

> Implementando login...
[Trabaja]

> Listo
[Usa commit_world_state + complete_sub_agent_task]

# Sesión 1 - Orquestador
> ¿cómo quedó?
[Usa get_active_brain]
```

## Desarrollo

```bash
go mod tidy
go build ./...
go test ./...
```

## Licencia

MIT
