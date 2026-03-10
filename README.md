# Singularity

> State memory and orchestration engine for AI agents

Singularity is a persistent memory system designed to maximize the autonomy of AI agents like OpenCode. It implements the **Blackboard Pattern** where agents read and write state without communicating directly with each other.

## Features

- **Active Brain**: Current project state always available
- **Deep Archive**: Historical context when needed
- **Task Orchestration**: DAG of tasks with dependencies
- **ISOLATED Sub-Agents**: Each agent receives only its specific context
- **Zero Ping-Pong**: One request contains all necessary information
- **Dual Engine System**: Core (dense context) and Particle (progressive disclosure)

## Dual Engine Architecture

Singularity provides two engines optimized for different LLM capabilities:

### Core Engine (Contexto Denso)

- **Context**: Maximum (full context window)
- **Goal**: Minimize API requests
- **Best for**: LLMs with excellent Chain of Thought
- **Rule**: Never writes code - always delegates to sub-agents
- **Tools**: spawn_sub_agent, get_active_brain, commit_world_state, plan_and_delegate

### Particle Engine (Divulgación Progresiva)

- **Context**: Minimum (<500 tokens)
- **Goal**: Minimize input tokens
- **Best for**: LLMs with "context amnesia"
- **Rule**: Work function by function using AST tools
- **Tools**: get_file_skeleton, read_function, replace_function, sync_dag_metadata

### Switching Engines

Use `switch_engine` to change between engines:

```json
{
  "engine_type": "core"  // or "particle"
}
```

## Installation

### Prerequisites

- Go 1.23+
- OpenCode installed

### Build

```bash
git clone https://github.com/elbader17/singularity
cd singularity
go build -o singularity ./cmd/singularity
```

### Install in OpenCode

```bash
./singularity init
```

This will automatically add the configuration to `~/.config/opencode/opencode.jsonc`

### Manual Configuration

Add to your `opencode.jsonc`:

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

## Usage

### Start MCP Server

```bash
./singularity
```

### Install in OpenCode

```bash
./singularity init
```

### View Help

```bash
./singularity help
```

## MCP Tools

### Engine Management

#### switch_engine

Switches between Core and Particle engines.

```json
{
  "engine_type": "core"
}
```

#### get_engine_info

Gets current engine information.

```json
{}
```

**Response:**
```json
{
  "type": "core",
  "name": "Core Engine",
  "description": "Minimiza requests API. Ideal para contexto denso."
}
```

### State Management

#### commit_world_state

Consolidates state after completing a task. **Mandatory use** when done.

```json
{
  "session_id": "ses-123",
  "project_path": "/path/to/project",
  "completed_task_id": "task-456",
  "task_summary": "Implemented login system",
  "orchestrator_summary": "Authentication ready for API integration",
  "learned_insights": "Auth module has reusable code"
}
```

#### get_active_brain

Gets the current project state at no API cost.

```json
{
  // Response
  {
    "tasks": [...],
    "recent_decisions": [...],
    "active_count": 2,
    "pending_count": 5,
    "completed_count": 10
  }
}
```

#### list_tasks

Lists all tasks with their status.

```json
{
  "status": "pending"  // optional: pending, in_progress, completed
}
```

#### fetch_deep_context

Retrieves historical context. Use only when necessary.

```json
{
  "query": "authentication",
  "task_id": "task-123",  // optional
  "limit": 10
}
```

### Sub-Agent Management

#### spawn_sub_agent

Creates a sub-agent to execute a specific task.

```json
{
  "session_id": "ses-123",
  "project_path": "/path/to/project",
  "title": "Implement logout",
  "description": "Create logout endpoint",
  "context": "Use existing auth/ code"
}
```

**Response:**
```json
{
  "success": true,
  "sub_agent_id": "sub-abc123",
  "task_id": "task-789"
}
```

#### get_sub_agent_task

Gets the assigned task (for sub-agents).

```json
{
  "sub_agent_id": "sub-abc123"
}
```

#### switch_agent

Switches between Core and Sub-agent mode.

```json
{
  "mode": "sub_agent",  // or "core"
  "sub_agent_id": "sub-abc123"  // required if mode=sub_agent
}
```

### Particle Engine Tools (AST-Based)

#### get_file_skeleton

Gets only function/struct signatures (no body).

```json
{
  "file_path": "/path/to/file.go"
}
```

#### read_function

Reads code of a single function.

```json
{
  "file_path": "/path/to/file.go",
  "function_name": "MyFunction"
}
```

#### replace_function

Overwrites a function and saves to BadgerDB.

```json
{
  "file_path": "/path/to/file.go",
  "function_name": "MyFunction",
  "new_code": "func MyFunction() { ... }"
}
```

#### sync_dag_metadata

Synchronizes DAG state JSON.

```json
{
  "updates": "{\"task-1\": \"completed\"}"
}
```

#### compress_history_key

Compresses history into summary.

```json
{
  "session_id": "ses-123"
}
```

## Architecture

```
┌──────────────────────────────────────────────────────────┐
│                         Singularity                      │
│  ┌───────────────┐  ┌───────────────┐  ┌────────────┐    │
│  │    Brain      │  │    Archive    │  │   Tasks    │    │
│  │   (Active)    │  │    (Deep)     │  │   (DAG)    │    │
│  └───────────────┘  └───────────────┘  └────────────┘    │
│         │                   │                   │          │
│  ┌──────┴──────┐   ┌──────┴──────┐    ┌──────┴──────┐    │
│  │Core Engine   │   │Particle Eng │    │  Sub-agents │    │
│  │(Dense Context)│  │(AST Tools)  │    │             │    │
│  └─────────────┘    └─────────────┘    └─────────────┘    │
└──────────────────────────────────────────────────────────┘
```

### Components

| Component | Description |
|-----------|-------------|
| `internal/storage` | BadgerDB for persistence |
| `internal/mcp` | MCP server + tools |
| `internal/models` | Data structures |
| `internal/agents` | Engine implementations (Core/Particle) |
| `internal/protocol` | Context injection |

## Workflow Pattern

### Core Agent Flow (Dense Context)

```
1. switch_engine("core")       → Initialize Core engine
2. get_active_brain()          → View current state
3. spawn_sub_agent()           → Create task for sub-agent
4. switch_agent(mode="sub_agent", sub_agent_id="...")
5. [WAIT - Sub-agent works autonomously]
6. switch_agent(mode="core")   → Core reactivated when done
7. commit_world_state()        → Consolidate results
```

### Sub-agent Flow

```
1. get_sub_agent_task()        → Get my task and context
2. [WORK]
3. commit_world_state()        → Consolidate my work
4. switch_agent(mode="core")   → Return to Core
```

### Particle Agent Flow (Progressive Disclosure)

```
1. switch_engine("particle")   → Initialize Particle engine
2. get_file_skeleton("file.go")→ Get only signatures
3. read_function("file.go", "FuncName") → Get specific function
4. replace_function(...)       → Modify function
5. [Repeat for each function]
```

## Core Rules

1. **One request = all information**
   - Don't ask "where did we leave off?" - use `get_active_brain`
   - Don't do exploratory ping-pong - deduce and act

2. **Strict isolation**
   - Core only sees high-level summaries
   - Sub-agent only sees its specific context
   - Particle works function-by-function

3. **Mandatory consolidation**
   - Always use `commit_world_state` when done
   - Include code, decisions, learnings

4. **No polling for sub-agents**
   - After `switch_agent` to sub-agent mode, the system automatically reactivates Core when done
   - Don't use `get_sub_agent_task` repeatedly

## Session Example

```bash
# Session 1 - Core Agent
$ opencode .

> What's the project state?
[Uses get_active_brain]

> Create a task to implement login
[Uses spawn_sub_agent + switch_agent]

# Session 2 - Sub-agent (automatically activated)
> My task is...
[Uses get_sub_agent_task]

> Implementing login...
[Works]

> Done
[Uses commit_world_state + switch_agent(mode="core")]

# Session 1 - Core Agent (automatically reactivated)
> Task completed successfully
[Reports to user]
```

## Development

```bash
go mod tidy
go build ./...
go test ./...
```

## License

MIT
