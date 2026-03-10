package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"singularity/internal/cli"
	"singularity/internal/mcp"
	"singularity/internal/storage"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "init":
			cli.RunInit()
			return
		case "help", "--help", "-h":
			printHelp()
			return
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		cancel()
	}()

	db, err := storage.NewBadgerDB(storage.DefaultDir())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	server := mcp.NewServer(db)

	if err := server.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Print(`Singularity - Motor de Memoria de Estado

Uso:
  singularity           Iniciar servidor MCP
  singularity init      Instalar en OpenCode (TUI interactiva)
  singularity help      Mostrar esta ayuda

Integración con OpenCode:
  Agrega a tu opencode.jsonc:
  {
    "mcp": {
      "singularity": {
        "type": "local",
        "command": ["./singularity"],
        "enabled": true
      }
    }
  }

Herramientas disponibles:
  - commit_world_state    Consolidar estado
  - fetch_deep_context   Recuperar contexto histórico
  - get_active_brain     Estado actual
  - list_tasks           Listar tareas
  - spawn_sub_agent      Crear sub-agente
  - switch_agent         Cambiar entre orquestador/sub-agente
`)
}
