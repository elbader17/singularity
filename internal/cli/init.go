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
