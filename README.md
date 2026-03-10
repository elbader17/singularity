# Singularity

> State memory and orchestration engine for AI agents

Singularity is a persistent memory system designed to maximize the autonomy of AI agents like OpenCode. It implements the **Blackboard Pattern** where agents read and write state without communicating directly with each other.

---

## Features

- **Active Brain**: Current project state always available
- **Deep Archive**: Historical context when needed
- **Task Orchestration**: DAG of tasks with dependencies
- **ISOLATED Sub-Agents**: Each agent receives only its specific context
- **Zero Ping-Pong**: One request contains all necessary information
- **Dual Engine System**: Core (dense context) and Particle (progressive disclosure)

---

## Quick Start

### Installation

```bash
git clone https://github.com/elbader17/singularity
cd singularity
go build -o singularity ./cmd/singularity
./singularity init
```

### Two Pre-configured Agents

Singularity comes with **two default agents** configured:

| Agent | Engine | Purpose | Best For |
|-------|--------|---------|----------|
| `@core` | Core | Context-dense planning & delegation | Complex multi-step tasks |
| `@particle` | Particle | Surgical code editing | Quick fixes & refactoring |

### Usage

```bash
# Start the server
./singularity

# In OpenCode, use agents with their prefixes:
@core implement a login system
@particle fix the bug in auth.go
```

---

## Dual Engine Architecture (Deep Dive)

Singularity provides **two specialized engines** optimized for different LLM capabilities and use cases.

### When to Use Each Engine

```
┌─────────────────────────────────────────────────────────────────┐
│                    DECISION TREE                                │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Is the task complex or multi-step?                             │
│      │                                                          │
│      ├── YES → Use CORE ENGINE                                  │
│      │         - Full context available                         │
│      │         - Delegates to sub-agents                        │
│      │ planning                                                 │
│      |  - Strategic                                             │
│      └── NO → Is it a quick fix or small change?                │
│                │                                                │
│                ├── YES → Use PARTICLE ENGINE                    │
│                │         - Work function by function            │
│                │         - Minimal context                      │
│                │         - AST-based editing                    │
│                │                                                │
│                └── NO → Use CORE ENGINE                         │
│                          - General purpose                      │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## Core Engine (Contexto Denso)

The **Core Engine** is designed for LLMs with excellent reasoning capabilities that benefit from full context.

### Characteristics

| Aspect | Description |
|--------|-------------|
| **Context** | Maximum (full context window) |
| **API Requests** | Minimized (1-2 requests per task) |
| **Strategy** | Delegation-based |
| **Best LLM** | GPT-4, Claude 3, Gemini Ultra |

### Philosophy

> "Think deeply, act once."

Core agents analyze the entire codebase, plan the approach, and delegate execution to specialized sub-agents. This reduces API costs while maintaining high quality.

### Tools

| Tool | Purpose |
|------|---------|
| `spawn_sub_agent` | Create sub-agents for specific tasks |
| `get_active_brain` | View current project state |
| `list_tasks` | See all pending/in-progress tasks |
| `commit_world_state` | Save work and consolidate |
| `switch_agent` | Activate a sub-agent |
| `plan_and_delegate` | Create DAG plan and execute |

### Core Agent Rules

1. **NEVER write code directly** - Always delegate to sub-agents
2. **Analyze before acting** - Use full context to plan
3. **Consolidate always** - Use `commit_world_state` when done
4. **No polling** - Wait for sub-agents to complete

### Example Flow

```
User: "Implement user authentication"

Core Agent (thinking):
  1. get_active_brain() → See current state
  2. spawn_sub_agent("Create User model") 
  3. spawn_sub_agent("Create Auth service")
  4. spawn_sub_agent("Create login endpoint")
  5. switch_agent(sub_agent_id_1)
  
[Sub-agents work independently]

Core Agent (reactivated):
  6. commit_world_state()
  7. Report to user: "Authentication implemented"
```

---

## Particle Engine (Divulgación Progresiva)

The **Particle Engine** is designed for LLMs with limited context windows or "amnesia" between requests. It uses **AST-based surgical tools** to work with precise code sections.

### Characteristics

| Aspect | Description |
|--------|-------------|
| **Context** | Minimum (<500 tokens per request) |
| **Strategy** | Function-by-function editing |
| **Tool Type** | AST-based surgical operations |
| **Best LLM** | GPT-3.5, Claude Instant, Gemini Flash |

### Philosophy

> "Precision over volume. Work function by function."

Instead of reading entire files, Particle agents:
1. Get **file skeleton** (just function signatures)
2. Read **one function** at a time
3. Modify **one function** at a time
4. Save directly to persistent storage

### The AST Tool Chain

```
┌─────────────────────────────────────────────────────────────────┐
│                 PARTICLE TOOL CHAIN                             │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌──────────────────┐                                           │
│  │ get_file_skeleton│ → Returns only:                           │
│  │                  │   - Package name                          │
│  │ "path/to/file.go │   - Imports                               │
│  └────────┬─────────┘   - Function signatures                   │
│           │             - Struct definitions                    │
│           ▼                                                     │
│  ┌──────────────────┐                                           │
│  │  read_function   │ → Returns ONLY the requested function:    │
│  │                  │                                           │
│  │ func Login(...)  │   func Login(user, pass) error {          │
│  └────────┬─────────┘     // just this function                 │
│           │             }                                       │
│           ▼                                                     │
│  ┌──────────────────┐                                           │
│  │ replace_function │ → Saves to BadgerDB:                      │
│  │                  │   - Only modifies one function            │
│  │ new_code: "..."  │   - Preserves rest of file                │
│  └────────┬─────────┘   - Immediate persistence                 │
│           │                                                     │
│           ▼                                                     │
│      [Repeat for next function]                                 │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Why AST Tools?

Traditional code editing:
```
❌ Read entire file (10,000 tokens)
❌ Modify function (500 tokens)  
❌ Write entire file back (10,000 tokens)
Total: ~20,000 tokens per function
```

Particle AST tools:
```
✅ Get skeleton (200 tokens)
✅ Read one function (300 tokens)
✅ Replace function (400 tokens)
Total: ~900 tokens per function
```

**Savings: 95%+ reduction in token usage**

### Tools

| Tool | Purpose | Example |
|------|---------|---------|
| `get_file_skeleton` | Get only function signatures | `{"file_path": "auth.go"}` |
| `read_function` | Read specific function | `{"file_path": "auth.go", "function_name": "Login"}` |
| `replace_function` | Overwrite function | `{"file_path": "auth.go", "function_name": "Login", "new_code": "..."}` |
| `sync_dag_metadata` | Update task state JSON | `{"updates": "{\"task-1\": \"done\"}"}` |
| `compress_history_key` | Compress conversation | `{"session_id": "sess-123"}` |

### Example Flow

```
User: "Add validation to Login function"

Particle Agent:
  1. get_file_skeleton("auth.go")
     → Returns: func Login(), func Logout(), type User struct

  2. read_function("auth.go", "Login")
     → Returns only the Login function code

  3. Replace function with validated version:
     replace_function(
       file_path="auth.go",
       function_name="Login", 
       new_code="func Login(email, pass string) error {\n  if email == \"\" { return ErrEmptyEmail }\n  ..."
     )

  4. Done! Commit with commit_world_state()
```

---

## Creating Custom Agents

You can create agents that use specific engines by configuring OpenCode.

### Configuration Example

Add to your `opencode.jsonc`:

```jsonc
{
  "mcp": {
    "singularity": {
      "type": "local",
      "command": ["./singularity"],
      "enabled": true
    }
  },
  
  // Agent aliases for quick access
  "agents": {
    "@core": {
      "description": "Context-dense planner",
      "system_prompt": "You are the Core Agent. Use dense context and delegate to sub-agents."
    },
    "@particle": {
      "description": "Surgical code editor",
      "system_prompt": "You are the Particle Agent. Work function by function using AST tools."
    },
    "@analyzer": {
      "description": "Code reviewer",
      "engine": "core",
      "system_prompt": "You analyze code and provide reviews. Use get_file_skeleton to understand structure."
    },
    "@fixer": {
      "description": "Quick bug fixer",
      "engine": "particle", 
      "system_prompt": "You fix bugs surgically. Read one function, fix it, move to next."
    }
  }
}
```

### Agent Types

#### 1. Core-Based Agents

Best for: Planning, architecture, complex implementations

```jsonc
{
  "@architect": {
    "engine": "core",
    "system_prompt": "You design system architectures. Analyze requirements, create specs, delegate implementation."
  },
  "@reviewer": {
    "engine": "core", 
    "system_prompt": "You review code thoroughly. Read full files, analyze patterns, provide detailed feedback."
  },
  "@manager": {
    "engine": "core",
    "system_prompt": "You manage project tasks. Track progress, coordinate sub-agents, maintain project state."
  }
}
```

#### 2. Particle-Based Agents

Best for: Quick fixes, refactoring, small changes

```jsonc
{
  "@fixer": {
    "engine": "particle",
    "system_prompt": "Fix bugs quickly. Use get_file_skeleton to find the function, read_function to see it, replace_function to fix it."
  },
  "@refactor": {
    "engine": "particle",
    "system_prompt": "Refactor code methodically. Work one function at a time. Preserve behavior while improving structure."
  },
  "@docs": {
    "engine": "particle",
    "system_prompt": "Add documentation. Read functions and add comments where needed. Use replace_function to update."
  }
}
```

### Hybrid Agents

Some agents can use **both engines** strategically:

```jsonc
{
  "@developer": {
    "engine": "hybrid",
    "system_prompt": "You are a full-stack developer. For complex features, use Core (spawn_sub_agent). For quick fixes, use Particle (get_file_skeleton). Switch engines with switch_engine.",
    "rules": {
      "multi_file": "core",
      "single_function": "particle",
      "debugging": "particle",
      "planning": "core"
    }
  }
}
```

---

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────┐
│                            SINGULARITY SERVER                           │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │                     ACTIVE BRAIN (State)                        │    │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │    │
│  │  │  Tasks DAG  │  │   Decisions │  │   Project Metadata      │  │    │
│  │  └─────────────┘  └─────────────┘  └─────────────────────────┘  │    │
│  └─────────────────────────────────────────────────────────────────┘    │
│                                    │                                    │
│                                    ▼                                    │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │                      ARCHIVE (History)                          │    │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │    │
│  │  │   Events    │  │   Context   │  │   Learnings             │  │    │
│  │  │  (Full Log) │  │  (Compressed│  │   (Insights)            │  │    │
│  │  └─────────────┘  └─────────────┘  └─────────────────────────┘  │    │
│  └─────────────────────────────────────────────────────────────────┘    │
│                                                                         │
├─────────────────────────────────────────────────────────────────────────┤
│                         ENGINE LAYER                                    │
│                                                                         │
│  ┌─────────────────────────┐        ┌─────────────────────────────────┐ │
│  │      CORE ENGINE        │        │        PARTICLE ENGINE          │ │
│  │  ┌───────────────────┐  │        │  ┌───────────────────────────┐  │ │
│  │  │ • Dense Context   │  │        │  │ • Progressive Disclosure  │  │ │
│  │  │ • Delegation      │  │        │  │ • AST-Based Tools         │  │ │
│  │  │ • Strategic Plan  │  │        │  │ • Function-Level Edit     │  │ │
│  │  └───────────────────┘  │        │  └───────────────────────────┘  │ │
│  │                         │        │                                 │ │
│  │  Tools:                 │        │  Tools:                         │ │
│  │  • spawn_sub_agent      │        │  • get_file_skeleton            │ │
│  │  • get_active_brain     │        │  • read_function                │ │
│  │  • commit_world_state   │        │  • replace_function             │ │
│  │  • plan_and_delegate    │        │  • sync_dag_metadata            │ │
│  │                         │        │  • compress_history_key         │ │
│  └─────────────────────────┘        └─────────────────────────────────┘ │
│                    │                              │                     │
│                    └──────────┬───────────────────┘                     │
│                               ▼                                         │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │                      TOOL REGISTRY                              │    │
│  │     switch_engine | get_engine_info | list_engines              │    │
│  └─────────────────────────────────────────────────────────────────┘    │
│                                                                         │
├─────────────────────────────────────────────────────────────────────────┤
│                      PERSISTENCE LAYER                                  │
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │                    BADGER DATABASE                              │    │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────────────┐    │    │
│  │  │  State   │ │  Tasks   │ │  Code    │ │   History        │    │    │
│  │  │ (Brain)  │ │   (DAG)  │ │ (AST)    │ │   (Archive)      │    │    │
│  │  └──────────┘ └──────────┘ └──────────┘ └──────────────────┘    │    │
│  └─────────────────────────────────────────────────────────────────┘    │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## MCP Tools Reference

### Engine Management

#### switch_engine

Switch between Core and Particle engines.

```json
{
  "engine_type": "core"  // or "particle"
}
```

#### get_engine_info

Get current engine information.

```json
{}
```

Response:
```json
{
  "type": "core",
  "name": "Core Engine", 
  "description": "Minimiza requests API. Ideal para contexto denso."
}
```

#### list_engines

List all available engines.

```json
{}
```

### State Management

#### commit_world_state

**Mandatory** - Consolidate state after completing a task.

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

Get current project state.

```json
{}
```

#### list_tasks

List tasks by status.

```json
{
  "status": "pending"  // optional: pending, in_progress, completed
}
```

#### fetch_deep_context

Retrieve historical context.

```json
{
  "query": "authentication",
  "task_id": "task-123",  // optional
  "limit": 10
}
```

### Sub-Agent Management

#### spawn_sub_agent

Create a sub-agent.

```json
{
  "session_id": "ses-123",
  "project_path": "/path/to/project",
  "title": "Implement logout",
  "description": "Create logout endpoint",
  "context": "Use existing auth/ code"
}
```

#### switch_agent

Switch between Core and Sub-agent mode.

```json
{
  "mode": "sub_agent",  // or "core"
  "sub_agent_id": "sub-abc123"  // required if mode=sub_agent
}
```

---

## Configuration

### Environment Variables

```bash
# Database location
SINGULARITY_DB_PATH=~/.singularity

# Log level (debug, info, warn, error)
SINGULARITY_LOG_LEVEL=info

# Default engine (core or particle)
SINGULARITY_DEFAULT_ENGINE=particle
```

### Full OpenCode Configuration

```jsonc
{
  "mcp": {
    "singularity": {
      "type": "local",
      "command": ["./singularity"],
      "enabled": true,
      "env": {
        "SINGULARITY_DB_PATH": "~/.singularity",
        "SINGULARITY_DEFAULT_ENGINE": "core"
      }
    }
  },
  
  "agents": {
    // Pre-configured agents
    "@core": {
      "description": "Context-dense planner and orchestrator",
      "engine": "core",
      "auto_init": true,
      "system_prompt_file": "./prompts/core.md"
    },
    "@particle": {
      "description": "Surgical code editor",
      "engine": "particle", 
      "auto_init": true,
      "system_prompt_file": "./prompts/particle.md"
    },
    
    // Custom agents
    "@architect": {
      "description": "System architect",
      "engine": "core"
    },
    "@fixer": {
      "description": "Bug fixer",
      "engine": "particle"
    }
  }
}
```

---

## Workflow Examples

### Example 1: Complex Feature (Core)

```
User: "Implement a full authentication system"

Core Agent:
  1. get_active_brain() → Check current state
  2. Analyze requirements
  3. spawn_sub_agent("Create User model + migrations")
  4. spawn_sub_agent("Create Auth service with JWT")
  5. spawn_sub_agent("Create login/logout endpoints")
  6. spawn_sub_agent("Create auth middleware")
  7. switch_agent(sub_agent_id_1)
  
[All sub-agents work in parallel]

Core Agent (reactivated):
  8. commit_world_state()
  9. Report: "Authentication system implemented"
```

### Example 2: Quick Fix (Particle)

```
User: "Fix the nil pointer in validateEmail"

Particle Agent:
  1. get_file_skeleton("email.go")
     → Shows: func validateEmail(), func formatEmail()
  
  2. read_function("email.go", "validateEmail")
     → Shows only validateEmail function
  
  3. replace_function(
       file_path="email.go",
       function_name="validateEmail", 
       new_code="func validateEmail(e string) error {\n  if e == \"\" { return ErrEmpty }\n  if !strings.Contains(e, \"@\") { return ErrInvalid }\n  return nil\n}"
     )
  
  4. commit_world_state()
  5. Report: "Fixed nil pointer in validateEmail"
```

### Example 3: Mixed Approach

```
User: "Refactor the entire auth module"

Core Agent (Planning):
  1. get_active_brain()
  2. spawn_sub_agent("Refactor auth models")
  3. spawn_sub_agent("Refactor auth handlers") 
  4. spawn_sub_agent("Refactor auth middleware")
  5. switch_agent(sub_agent_id_1)

Each Sub-agent (using Particle):
  1. get_file_skeleton("file.go")
  2. read_function("file.go", "FuncName")
  3. replace_function(...)
  4. Repeat for each function
  5. commit_world_state()
  6. switch_agent(mode="core")

Core Agent (Finalizing):
  7. commit_world_state()
  8. Report: "Auth module refactored"
```

---

## Core Rules Summary

### For Core Agents

1. **Never write code** - Always delegate
2. **Use full context** - Analyze thoroughly
3. **Consolidate always** - Use `commit_world_state`
4. **No polling** - Wait for sub-agents
5. **Plan strategically** - Break into sub-tasks

### For Particle Agents

1. **Function by function** - Never read entire files
2. **Use AST tools** - Precision over volume
3. **Minimal context** - Stay under 500 tokens
4. **Immediate persistence** - Changes saved automatically
5. **Repeat as needed** - Work through functions systematically

---

## Development

```bash
# Build
go build -o singularity ./cmd/singularity

# Run tests
go test ./...

# Run with debug logging
SINGULARITY_LOG_LEVEL=debug ./singularity
```

---

## License

MIT
