package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

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
		case "restart":
			runRestart()
			return
		case "help", "--help", "-h":
			printHelp()
			return
		}
	}

	startServer()
}

func startServer() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGUSR1)

	go func() {
		sig := <-sigChan
		if sig == syscall.SIGUSR1 {
			fmt.Println("\n🔄 Señal de reinicio recibida (SIGUSR1)")
		}
		cancel()
	}()

	db, err := storage.NewBadgerDB(storage.DefaultDir())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	server := mcp.NewServer(db)

	fmt.Println("🚀 Singularity MCP Server iniciado")
	fmt.Println("   Presiona Ctrl+C para detener, o.envía SIGUSR1 para reiniciar")

	if err := server.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}

func runRestart() {
	fmt.Println("🔄 Reiniciando Singularity MCP Server...")

	// Buscar y matar proceso existente
	cmd := exec.Command("pkill", "-f", "singularity")
	cmd.Run()

	time.Sleep(500 * time.Millisecond)

	// Iniciar nuevo proceso en background
	execPath, err := os.Executable()
	if err != nil {
		execPath = "./singularity"
	}

	newCmd := exec.Command(execPath)
	newCmd.Stdout = os.Stdout
	newCmd.Stderr = os.Stderr
	newCmd.Stdin = os.Stdin

	if err := newCmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Error al iniciar servidor: %v\n", err)
		os.Exit(1)
	}

	if newCmd.Process != nil {
		fmt.Printf("✅ Servidor reiniciado (PID: %d)\n", newCmd.Process.Pid)
	} else {
		fmt.Println("✅ Servidor reiniciado")
	}
}

func printHelp() {
	fmt.Print(`Singularity - Motor de Memoria de Estado

Uso:
  singularity           Iniciar servidor MCP
  singularity init      Instalar en OpenCode (TUI interactiva)
  singularity restart   Reiniciar servidor
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

Señales:
  SIGINT/SIGTERM  - Detener servidor
  SIGUSR1         - Reiniciar servidor

Herramientas disponibles:
  - commit_world_state    Consolidar estado
  - commit_task_result   Commit con validación (Judge)
  - fetch_deep_context   Recuperar contexto histórico
  - get_active_brain     Estado actual
  - list_tasks           Listar tareas
  - plan_and_delegate    Planificar y delegar
  - spawn_sub_agent      Crear sub-agente
  - switch_agent         Cambiar entre orquestador/sub-agente
  - get_sub_agent_task   Obtener tarea de sub-agente
  - complete_sub_agent   Completar tarea de sub-agente
`)
}
