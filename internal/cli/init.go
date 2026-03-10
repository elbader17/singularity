package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("86")).
			Padding(0, 1)

	subtleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("86")).
			Bold(true)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(1, 2)

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("82")).
			Bold(true)
)

func RunInit() {
	fmt.Println()
	fmt.Println(headerStyle.Render("╔══════════════════════════════════════════╗"))
	fmt.Println(headerStyle.Render("║         Singularity - Setup              ║"))
	fmt.Println(headerStyle.Render("╚══════════════════════════════════════════╝"))
	fmt.Println()

	fmt.Println("Selecciona una opción:")
	fmt.Println()
	fmt.Println("  1. Instalar en OpenCode")
	fmt.Println("  2. Mostrar configuración manual")
	fmt.Println("  3. Salir")
	fmt.Println()

	fmt.Print("> ")

	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return
	}

	choice := strings.TrimSpace(scanner.Text())

	switch choice {
	case "1", "1\n":
		installOpenCode()
	case "2":
		printManualConfig()
	case "3", "3\n":
		fmt.Println("¡Hasta luego!")
		return
	default:
		fmt.Println("Opción inválida")
		return
	}
}

func installOpenCode() {
	binPath, err := os.Executable()
	if err != nil {
		fmt.Printf("Error: no se pudo obtener la ruta del binario: %v\n", err)
		return
	}

	home, _ := os.UserHomeDir()
	configDir := filepath.Join(home, ".config", "opencode")

	if err := os.MkdirAll(configDir, 0755); err != nil {
		fmt.Printf("Error: no se pudo crear el directorio de config: %v\n", err)
		return
	}

	configFile := filepath.Join(configDir, "opencode.jsonc")

	existingContent := []byte{}
	if _, err := os.Stat(configFile); err == nil {
		existingContent, err = os.ReadFile(configFile)
		if err != nil {
			fmt.Printf("Error: no se pudo leer config existente: %v\n", err)
			return
		}
	}

	existingStr := strings.TrimSpace(string(existingContent))

	// Configuración del servidor MCP
	singularityConfig := fmt.Sprintf(`{
      "type": "local",
      "command": ["%s"],
      "enabled": true
    }`, binPath)

	var finalConfig string

	if strings.Contains(existingStr, `"singularity"`) {
		finalConfig = updateSingularityConfig(existingStr, binPath)
	} else if strings.Contains(existingStr, `"mcp"`) {
		finalConfig = addSingularityToMcp(existingStr, singularityConfig)
	} else if len(existingStr) > 0 && strings.Contains(existingStr, "{") {
		finalConfig = addMcpSection(existingStr, binPath)
	} else {
		finalConfig = fmt.Sprintf(`{
  "mcp": {
    "singularity": {
      "type": "local",
      "command": ["%s"],
      "enabled": true
    }
  }
}`, binPath)
	}

	// Agregar/actualizar configuración de los agentes core y particle
	finalConfig = ensureCoreAndParticleAgents(finalConfig, binPath)

	if err := os.WriteFile(configFile, []byte(finalConfig), 0644); err != nil {
		fmt.Printf("Error: no se pudo escribir config: %v\n", err)
		return
	}

	fmt.Println()
	fmt.Println(successStyle.Render("✓ Instalación completada!"))
	fmt.Println()
	fmt.Println(boxStyle.Render(
		fmt.Sprintf("Configuración creada en:\n%s", configFile),
	))
	fmt.Println()
	fmt.Println("Reinicia OpenCode y las herramientas de Singularity estarán disponibles.")
}

func updateSingularityConfig(content, binPath string) string {
	lines := strings.Split(content, "\n")
	var result []string

	for _, line := range lines {
		if strings.Contains(line, `"command"`) && strings.Contains(line, "singularity") {
			parts := strings.Split(line, `"command"`)
			if len(parts) > 1 {
				result = append(result, parts[0]+`"command": ["`+binPath+`"],`)
				continue
			}
		}
		result = append(result, line)
	}

	return strings.Join(result, "\n")
}

func addSingularityToMcp(content, singularityConfig string) string {
	lines := strings.Split(content, "\n")
	var result []string
	inMcp := false
	added := false

	for _, line := range lines {
		if strings.Contains(line, `"mcp"`) && !strings.Contains(line, "singularity") {
			inMcp = true
		}

		if inMcp && strings.Contains(line, "}") && !added {
			indent := getIndent(line)
			result = append(result, fmt.Sprintf(`%s"singularity": {%s`, indent, singularityConfig[1:len(singularityConfig)-1]))
			added = true
			inMcp = false
		}

		result = append(result, line)
	}

	if !added {
		return content
	}

	return strings.Join(result, "\n")
}

func addMcpSection(content, binPath string) string {
	lines := strings.Split(content, "\n")
	var result []string

	for _, line := range lines {
		result = append(result, line)
		if strings.Contains(line, "}") && !strings.Contains(line, "mcp") {
			result = append(result, `  "mcp": {`)
			result = append(result, `    "singularity": {`)
			result = append(result, fmt.Sprintf(`      "type": "local",`))
			result = append(result, fmt.Sprintf(`      "command": ["%s"],`, binPath))
			result = append(result, `      "enabled": true`)
			result = append(result, `    }`)
			result = append(result, `  }`)
			break
		}
	}

	return strings.Join(result, "\n")
}

func getIndent(line string) string {
	for i, c := range line {
		if c != ' ' && c != '\t' {
			return line[:i]
		}
	}
	return ""
}

func printManualConfig() {
	binPath, _ := os.Executable()
	home, _ := os.UserHomeDir()
	configFile := filepath.Join(home, ".config", "opencode", "opencode.jsonc")

	fmt.Println()
	fmt.Println("Agrega esto a tu opencode.jsonc:")
	fmt.Println()
	fmt.Println(boxStyle.Render(fmt.Sprintf(`{
  "mcp": {
    "singularity": {
      "type": "local",
      "command": ["%s"],
      "enabled": true
    }
  }
}`, binPath)))
	fmt.Println()
	fmt.Printf("Ruta: %s\n", configFile)
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// ensureCoreAndParticleAgents ensures core and particle agents have all required tools
func ensureCoreAndParticleAgents(content, binPath string) string {
	// Required tools for sub-agent support
	requiredTools := []string{
		"singularity_spawn_sub_agent",
		"singularity_get_sub_agent_task",
		"singularity_complete_sub_agent_task",
		"singularity_switch_agent",
		"singularity_list_tasks",
		"singularity_get_active_brain",
		"singularity_commit_world_state",
		"singularity_fetch_deep_context",
	}

	// Check if core agent already exists
	if strings.Contains(content, `"core"`) {
		// Agent exists, check if it has all required tools
		missingTools := []string{}
		for _, tool := range requiredTools {
			if !strings.Contains(content, `"`+tool+`"`) {
				missingTools = append(missingTools, tool)
			}
		}

		if len(missingTools) > 0 {
			content = addMissingToolsToAgent(content, missingTools)
		}
		return content
	}

	// Migration: if old "orchestrator" exists, add core and particle
	if strings.Contains(content, `"orchestrator"`) {
		content = addCoreAndParticleAgents(content, binPath, requiredTools)
		return content
	}

	// Agent doesn't exist, add both core and particle
	return addCoreAndParticleAgents(content, binPath, requiredTools)
}

func addCoreAndParticleAgents(content, binPath string, tools []string) string {
	// Build tools section
	toolsStr := `"read": true,
        "write": true,
        "edit": true,
        "bash": true,
        "glob": true,
        "grep": true,
        "webfetch": true,`

	for _, tool := range tools {
		toolsStr += fmt.Sprintf("\n        \"%s\": true,", tool)
	}
	// Remove trailing comma
	toolsStr = strings.TrimSuffix(toolsStr, ",")

	// Create TWO agents: core and particle with DIFFERENT prompts
	agentConfig := fmt.Sprintf(`  "agent": {
    "core": {
      "description": "Agente con contexto denso. Minimiza requests.",
      "mode": "primary",
      "prompt": "{file:/home/eduardo/project/singularity/prompts/core.md}",
      "temperature": 0.2,
      "tools": {
        %s
      }
    },
    "particle": {
      "description": "Agente con divulgacion progresiva. Minimiza tokens.",
      "mode": "primary",
      "prompt": "{file:/home/eduardo/project/singularity/prompts/particle.md}",
      "temperature": 0.2,
      "tools": {
        %s
      }
    }
  }`, toolsStr, toolsStr)

	lines := strings.Split(content, "\n")
	var result []string

	// Find the line that closes the mcp object (second "}" from the end)
	// The last "}" closes the root object, the second-to-last closes "mcp"
	// We need to add comma after the mcp closing brace
	mcpClosingBraceIdx := -1
	closeBraceCount := 0
	for i := len(lines) - 1; i >= 0; i-- {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "}" {
			closeBraceCount++
			if closeBraceCount == 2 {
				// Second closing brace from the end closes "mcp"
				mcpClosingBraceIdx = i
				break
			}
		}
	}

	if mcpClosingBraceIdx >= 0 {
		// Add comma after the mcp closing brace
		if !strings.HasSuffix(lines[mcpClosingBraceIdx], ",") {
			lines[mcpClosingBraceIdx] = lines[mcpClosingBraceIdx] + ","
		}
	}

	// Insert agent section before the closing brace
	for i := 0; i < len(lines)-1; i++ {
		result = append(result, lines[i])
	}
	result = append(result, agentConfig)
	result = append(result, "}")
	return strings.Join(result, "\n")
}

func addMissingToolsToAgent(content string, missingTools []string) string {
	lines := strings.Split(content, "\n")
	var result []string
	inTools := false
	toolsCloseIdx := -1

	for i, line := range lines {
		result = append(result, line)

		if strings.Contains(line, `"tools"`) {
			inTools = true
		}

		if inTools && strings.TrimSpace(line) == "}" {
			toolsCloseIdx = i
			break
		}
	}

	if toolsCloseIdx > 0 {
		// Remove the closing brace
		result = result[:len(result)-1]

		// Add comma to previous line if needed
		prevIdx := len(result) - 1
		prevLine := strings.TrimSpace(result[prevIdx])
		if !strings.HasSuffix(prevLine, ",") {
			result[prevIdx] = result[prevIdx] + ","
		}

		// Add missing tools
		indent := "        "
		for _, tool := range missingTools {
			result = append(result, fmt.Sprintf(`%s"%s": true,`, indent, tool))
		}

		// Add closing brace
		result = append(result, "}")
	}

	return strings.Join(result, "\n")
}
