# AGENTS.md - Singularity

## Project Structure

- `cmd/singularity/main.go` - Entry point
- `internal/storage/badger.go` - BadgerDB persistence layer
- `internal/mcp/server.go` - MCP server and tools
- `internal/models/state.go` - Data models
- `internal/protocol/injector.go` - Protocol injection
- `prompts/orchestrator.md` - Orchestrator agent prompt
- `orchestrator.md` - OpenCode agent configuration

## Conventions

- **Go**: gofmt standard, no CGO
- **Dependencies**: Pure Go libraries only
- **Pattern**: Light MVC with storage/mcp/models separation

## Orchestrator Agent

This project includes an **Orchestrator Agent** that enforces SOLID principles with a rigorous 6-phase workflow.

### Setup

The orchestrator is already configured in your OpenCode config at `~/.config/opencode/opencode.jsonc`.

### Usage

Press **Tab** to switch to the Orchestrator agent in OpenCode, or use:

```
@orchestrator implement a login feature
```

### Workflow (6 Phases)

1. **Planning & Research** - Explore codebase, design solution
2. **Test Creation** - Create failing tests first
3. **Implementation** - Write code following SOLID
4. **Testing** - Run tests, fix failures
5. **Verification** - Self-review against requirements
6. **Final Verification** - Calculate completion percentage

### SOLID Principles Enforced

- **SRP**: Single Responsibility
- **OCP**: Open/Closed
- **LSP**: Liskov Substitution
- **ISP**: Interface Segregation
- **DIP**: Dependency Inversion

## Useful Commands

```bash
go build -o singularity ./cmd/singularity
go run ./cmd/singularity
go test ./...
```
