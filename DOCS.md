# Singularity - Technical Documentation

## Table of Contents

1. [Philosophy](#philosophy)
2. [Architecture](#architecture)
3. [Storage](#storage)
4. [MCP Protocol](#mcp-protocol)
5. [Data Model](#data-model)
6. [Execution Flows](#execution-flows)

---

## Philosophy

### The Request is King

The main cost in modern AI APIs is not token count but latency and number of requests. Singularity optimizes for:

- **Minimize requests**: Each request must contain all necessary information
- **Think before acting**: Agent must reason internally before emitting output
- **Consolidate on finish**: `commit_world_state` atomizes all work

### Blackboard Pattern

Agents don't communicate directly. Instead:

1. **Write to the blackboard** (BadgerDB)
2. **Read from the blackboard** (initial context)
3. **Die** (session ends)

This eliminates the need to maintain state in memory between interactions.

---

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                     OpenCode (Client)                   │
└────────────────────────┬────────────────────────────────┘
                         │ stdio
┌────────────────────────▼────────────────────────────────┐
│                      MCP Server (mcp-go)                │
└────────────────────────┬────────────────────────────────┘
                         │
        ┌────────────────┼────────────────┐
        │                │                │
┌───────▼───────┐ ┌────▼────┐ ┌────────▼────────┐
│    Storage    │ │ Models  │ │    Protocol    │
│  (BadgerDB)   │ │ (State) │ │   (Injector)   │
└───────────────┘ └─────────┘ └─────────────────┘
```

### Components

#### `internal/storage/badger.go`

Persistence layer with key prefixes:

| Prefix | Usage |
|--------|-------|
| `brain:` | Active brain (current state) |
| `archive:` | Deep archive (history) |
| `task:` | Individual tasks |
| `subagent:` | Sub-agents |

#### `internal/mcp/server.go`

MCP server with registered tools:

- JSON-RPC request handling
- Parameter validation
- Automatic persistence

#### `internal/models/state.go`

Immutable data structures with serialization methods.

---

## Storage

### BadgerDB

Key-value database optimized for SSD:

- **Write-optimized**: LSM tree
- **Transactions**: ACID
- **No external dependencies**: Pure Go

### Data Hierarchy

```
Active Brain (brain:project-path)
├── session_id
├── current_task_id
├── active_tasks[]
├── completed_tasks[]
├── decisions[]
└── last_updated

Task (task:task-id)
├── id
├── title
├── description
├── status
├── assignee (sub-agent ID)
├── depends_on[]
└── timestamps

Sub-Agent (subagent:sub-id)
├── id
├── task_id
├── title
├── description
├── context
├── status
└── timestamps
```

---

## MCP Protocol

### Initialization

```
Client → Server: initialize
Server → Client: capabilities
Client → Server: initialized
```

### Tool Call

```
Client → Server: tools/call
         ├─ name: "commit_world_state"
         └─ arguments: {...}
         
Server → Client: tools/call result
         └─ content: [{type: "text", text: "..."}]
```

### Resources

```
singularity://brain  → Current state in JSON
```

---

## Data Model

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
    Context     string    // Isolation: only this sees the sub-agent
    Status      SubAgentStatus  // pending, running, completed, failed
    Result      string
    Error       string
    CreatedAt   time.Time
    StartedAt   *time.Time
    CompletedAt *time.Time
}
```

---

## Execution Flows

### Flow 1: Normal Session

```
1. OpenCode starts
2. MCP Server starts (stdio)
3. OpenCode requests tools
4. [User works]
5. user → commit_world_state()
6. Server updates BadgerDB
7. OpenCode closes
8. MCP Server closes
```

### Flow 2: Orchestrator → Sub-agent

```
ORCHESTRATOR                         SUB-AGENT
    │                                      │
    ├─ spawn_sub_agent()                  │
    │   → Creates task in DB               │
    │   → Creates sub-agent in DB          │
    │   → Returns sub_agent_id             │
    │                                      │
    │                              ├─ get_sub_agent_task(sub_agent_id)
    │                              │   → Reads sub-agent from DB
    │                              │   → Updates status to "running"
    │                              │   → Returns context
    │                              │
    │                              ├─ [WORKS]
    │                              │
    │                              ├─ complete_sub_agent_task()
    │                              │   → Updates status to "completed"
    │                              │   → Saves result
    │                              │
    ├─ get_active_brain()                 │
    │   → Sees result from sub-agent      │
```

### Flow 3: Agent Switch (switch_agent)

```
# In same OpenCode session

> Switch to sub-agent mode
switch_agent(mode="sub_agent", sub_agent_id="abc")

→ Returns sub-agent context
→ Internal state changes to "sub_agent mode"

> Work on the task...

> Return to orchestrator mode
switch_agent(mode="orchestrator")

→ Internal state changes to "orchestrator mode"
```

---

## Security Considerations

1. **No authentication**: Local MCP assumes trusted access
2. **Data in ~/.singularity**: Configurable path
3. **No encryption**: Use FileVault/EncFS if needed

---

## Known Limitations

1. **Single process**: No simultaneous connections supported
2. **stdio only**: No HTTP server
3. **No synchronization**: Each project is isolated

---

## Future Improvements

- [ ] HTTP/SSE server for remote MCP
- [ ] OAuth authentication
- [ ] Cross-device synchronization
- [ ] Metrics and analytics
- [ ] Editor integrations
