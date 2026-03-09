# Singularity

> State memory and orchestration engine for AI agents

Singularity is a persistent memory system designed to maximize the autonomy of AI agents like OpenCode. It implements the **Blackboard Pattern** where agents read and write state without communicating directly with each other.

## Features

- **Active Brain**: Current project state always available
- **Deep Archive**: Historical context when needed
- **Task Orchestration**: DAG of tasks with dependencies
- **ISOLATED Sub-Agents**: Each agent receives only its specific context
- **Zero Ping-Pong**: One request contains all necessary information

## Installation

### Prerequisites

- Go 1.23+
- OpenCode installed

### Build

```bash
git clone <repo>
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

### commit_world_state

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

### get_active_brain

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

### list_tasks

Lists all tasks with their status.

```json
{
  "status": "pending"  // optional: pending, in_progress, completed
}
```

### fetch_deep_context

Retrieves historical context. Use only when necessary.

```json
{
  "query": "authentication",
  "task_id": "task-123",  // optional
  "limit": 10
}
```

### spawn_sub_agent

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

### get_sub_agent_task

Gets the assigned task (for sub-agents).

```json
{
  "sub_agent_id": "sub-abc123"
}
```

### complete_sub_agent_task

Completes the sub-agent task.

```json
{
  "sub_agent_id": "sub-abc123",
  "result": "Logout implemented with JWT blacklist"
}
```

### switch_agent

Switches between Orchestrator and Sub-agent mode in the same session.

```json
{
  "mode": "sub_agent",  // or "orchestrator"
  "sub_agent_id": "sub-abc123"  // required if mode=sub_agent
}
```

## Architecture

```
┌──────────────────────────────────────────────────────────┐
│                         Singularity                       │
│  ┌───────────────┐  ┌───────────────┐  ┌────────────┐  │
│  │    Brain      │  │    Archive    │  │   Tasks    │  │
│  │   (Active)    │  │    (Deep)     │  │   (DAG)    │  │
│  └───────────────┘  └───────────────┘  └────────────┘  │
└──────────────────────────────────────────────────────────┘
          ▲                     ▲                     ▲
          │                     │                     │
    ┌─────┴─────┐        ┌─────┴─────┐        ┌─────┴─────┐
    │Orchestrator│        │ Sub-agent │        │ Sub-agent │
    └───────────┘        └───────────┘        └───────────┘
```

### Components

| Component | Description |
|-----------|-------------|
| `internal/storage` | BadgerDB for persistence |
| `internal/mcp` | MCP server + tools |
| `internal/models` | Data structures |
| `internal/protocol` | Context injection |

## Workflow Pattern

### Orchestrator Flow

```
1. get_active_brain()     → View current state
2. list_tasks()           → View pending tasks
3. spawn_sub_agent()     → Create task for sub-agent
4. [WAIT]
5. commit_world_state()  → Consolidate results
```

### Sub-agent Flow

```
1. get_sub_agent_task()   → Get my task and context
2. [WORK]
3. commit_world_state()   → Consolidate my work
4. complete_sub_agent_task() → Report to orchestrator
```

## Core Rules

1. **One request = all information**
   - Don't ask "where did we leave off?" - use `get_active_brain`
   - Don't do exploratory ping-pong - deduce and act

2. **Strict isolation**
   - Orchestrator only sees high-level summaries
   - Sub-agent only sees its specific context

3. **Mandatory consolidation**
   - Always use `commit_world_state` when done
   - Include code, decisions, learnings

## Session Example

```bash
# Session 1 - Orchestrator
$ opencode .

> What's the project state?
[Uses get_active_brain]

> Create a task to implement login
[Uses spawn_sub_agent]

# Session 2 - Sub-agent (new terminal)
$ opencode .

> My task is...
[Uses get_sub_agent_task]

> Implementing login...
[Works]

> Done
[Uses commit_world_state + complete_sub_agent_task]

# Session 1 - Orchestrator
> How did it go?
[Uses get_active_brain]
```

## Development

```bash
go mod tidy
go build ./...
go test ./...
```

## License

MIT
