package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"singularity/internal/cli"
	"singularity/internal/mcp"
	"singularity/internal/storage"
)

func main() {
	// Parse flags first
	dataDir := flag.String("data", "", "Custom data directory path")
	flag.Parse()

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "init":
			cli.RunInit()
			return
		case "restart":
			runRestart()
			return
		case "update":
			runUpdate()
			return
		case "help", "--help", "-h":
			printHelp()
			return
		}
	}

	startServer(*dataDir)
}

func startServer(customDataDir string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Determine data directory
	dataDir := determineDataDir(customDataDir)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGUSR1)

	go func() {
		sig := <-sigChan
		if sig == syscall.SIGUSR1 {
			fmt.Println("\n🔄 Señal de reinicio recibida (SIGUSR1)")
		}
		cancel()
	}()

	db, err := storage.NewBadgerDB(dataDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	server := mcp.NewServer(db)

	projectName := filepath.Base(dataDir)
	fmt.Println("🚀 Singularity MCP Server iniciado")
	fmt.Printf("   Proyecto: %s\n", projectName)
	fmt.Printf("   Datos: %s\n", dataDir)
	fmt.Println("   Presiona Ctrl+C para detener, o.envía SIGUSR1 para reiniciar")

	if err := server.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}

// determineDataDir determines the data directory based on priority:
// 1. Custom -data flag
// 2. SINGULARITY_DATA environment variable
// 3. Auto-detect from current project (cwd)
func determineDataDir(customDir string) string {
	// Priority 1: Custom flag
	if customDir != "" {
		return customDir
	}

	// Priority 2: Environment variable
	if envDir := os.Getenv("SINGULARITY_DATA"); envDir != "" {
		return envDir
	}

	// Priority 3: Auto-detect from current project
	projectName := detectProjectName()
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join("./singularity-data", projectName)
	}
	return filepath.Join(home, ".singularity", projectName)
}

// detectProjectName detects the project name from the current directory
func detectProjectName() string {
	cwd, err := os.Getwd()
	if err != nil {
		return "default"
	}

	// Get base name of current directory
	projectName := filepath.Base(cwd)

	// Sanitize: replace characters that are invalid in directory names
	projectName = strings.ReplaceAll(projectName, "/", "-")
	projectName = strings.ReplaceAll(projectName, " ", "_")

	// If empty or starts with dot, use default
	if projectName == "" || strings.HasPrefix(projectName, ".") {
		return "default"
	}

	return projectName
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

func runUpdate() {
	fmt.Println("⬆️  Buscando actualizaciones...")

	// 1. Check if update script exists and run it
	updateScript := "./update.sh"
	if _, err := os.Stat(updateScript); err == nil {
		fmt.Println("📜 Ejecutando script de actualización...")
		cmd := exec.Command("bash", updateScript)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error al ejecutar update.sh: %v\n", err)
		}
	}

	// 2. Pull latest RELEASE commit
	fmt.Println("📥 Pulling último commit RELEASE...")
	pullCmd := exec.Command("git", "pull", "origin", "main")
	pullCmd.Stdout = os.Stdout
	pullCmd.Stderr = os.Stderr
	if err := pullCmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error en git pull: %v\n", err)
	}

	// 3. Fetch and checkout latest RELEASE commit
	fmt.Println("📦 Fetching y cambiando al último RELEASE...")
	fetchCmd := exec.Command("git", "fetch", "origin")
	fetchCmd.Stdout = os.Stdout
	fetchCmd.Stderr = os.Stderr
	fetchCmd.Run()

	// Find latest RELEASE commit
	logCmd := exec.Command("git", "log", "--format=%H %s", "-n", "1", "--grep=RELEASE", "origin/main")
	logCmd.Stdout = os.Stdout
	logCmd.Stderr = os.Stderr
	logOutput, err := logCmd.Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error buscando commit RELEASE: %v\n", err)
		fmt.Println("⚠️  Continuando con build actual...")
	} else {
		parts := strings.Fields(string(logOutput))
		if len(parts) >= 1 {
			releaseCommit := parts[0]
			fmt.Printf("📌 Cambiando a commit: %s\n", releaseCommit[:8])
			checkoutCmd := exec.Command("git", "checkout", releaseCommit)
			checkoutCmd.Stdout = os.Stdout
			checkoutCmd.Stderr = os.Stderr
			if err := checkoutCmd.Run(); err != nil {
				fmt.Fprintf(os.Stderr, "Error en checkout: %v\n", err)
			}
		}
	}

	// 4. Build
	fmt.Println("🔨 Building...")
	buildCmd := exec.Command("go", "build", "-o", "singularity", "./cmd/singularity")
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr
	if err := buildCmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error en build: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ Build completado")

	// 5. Restart
	fmt.Println("🔄 Reiniciando servidor...")
	runRestart()
}

func printHelp() {
	fmt.Print(`Singularity - Motor de Memoria de Estado

Uso:
  singularity           Iniciar servidor MCP
  singularity -data     Iniciar con directorio de datos personalizado
  singularity init      Instalar en OpenCode (TUI interactiva)
  singularity restart   Reiniciar servidor
  singularity update    Actualizar a la última versión RELEASE
  singularity help      Mostrar esta ayuda

Opciones:
  -data <path>  Directorio de datos personalizado (override)
                También puedes usar la variable de entorno SINGULARITY_DATA

Variables de entorno:
  SINGULARITY_DATA  Directorio de datos (ej: ~/.singularity/myproject)
  SINGULARITY_PROJECT Nombre del proyecto (para organizar datos)

Auto-detección de proyecto:
  Por defecto, usa el nombre del directorio actual como nombre de proyecto.
  Los datos se almacenan en: ~/.singularity/<project_name>

Ejemplos:
  singularity                           # Auto-detectar proyecto
  singularity -data ~/data/miproyecto  # Directorio personalizado
  SINGULARITY_DATA=~/data/proyecto singularity  # Con variable de entorno

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
