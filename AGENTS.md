# AGENTS.md - Singularity

## Estructura del Proyecto

- `cmd/singularity/main.go` - Punto de entrada
- `internal/storage/badger.go` - Capa de persistencia BadgerDB
- `internal/mcp/server.go` - Servidor MCP y herramientas
- `internal/models/state.go` - Modelos de datos
- `internal/protocol/injector.go` - Inyección de protocolo
- `prompts/system.md` - System prompts

## Convenciones

- **Go**: Estándar gofmt, sin CGO
- **Dependencias**: Solo librerías puras Go
- **Patrón**: MVC ligero con separación storage/mcp/models

## Comandos Útiles

```bash
go build -o singularity ./cmd/singularity
go run ./cmd/singularity
go test ./...
```
