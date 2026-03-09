package protocol

import (
	"fmt"
	"os"
	"strings"
	"time"

	"singularity/internal/models"
)

type Injector struct {
	systemPrompt string
}

func NewInjector() *Injector {
	prompt, err := os.ReadFile("prompts/subagent.md")
	if err != nil {
		// Fallback to minimal prompt if file not found
		prompt = []byte("Eres Singularity. Usa commit_world_state al terminar.")
	}

	return &Injector{
		systemPrompt: string(prompt),
	}
}

func (i *Injector) GetSystemPrompt() string {
	return i.systemPrompt
}

func (i *Injector) BuildInitialContext(brain *models.WorldState) string {
	var sb strings.Builder

	sb.WriteString(i.systemPrompt)
	sb.WriteString("\n\n")

	if brain != nil {
		sb.WriteString("## Estado Actual del Proyecto\n\n")
		sb.WriteString(fmt.Sprintf("- **Sesión**: %s\n", brain.SessionID))
		sb.WriteString(fmt.Sprintf("- **Proyecto**: %s\n", brain.ProjectPath))
		sb.WriteString(fmt.Sprintf("- **Última actualización**: %s\n\n", brain.LastUpdated.Format(time.RFC3339)))

		if len(brain.ActiveTasks) > 0 {
			sb.WriteString("### Tareas Activas\n")
			for _, taskID := range brain.ActiveTasks {
				sb.WriteString(fmt.Sprintf("- `%s`\n", taskID))
			}
			sb.WriteString("\n")
		}

		if len(brain.BlockedTasks) > 0 {
			sb.WriteString("### Tareas Bloqueadas\n")
			for _, taskID := range brain.BlockedTasks {
				sb.WriteString(fmt.Sprintf("- `%s`\n", taskID))
			}
			sb.WriteString("\n")
		}

		if len(brain.CompletedTasks) > 0 {
			sb.WriteString(fmt.Sprintf("### Tareas Completadas (%d total)\n", len(brain.CompletedTasks)))
			sb.WriteString("- ")
			sb.WriteString(strings.Join(brain.CompletedTasks[len(brain.CompletedTasks)-5:], ", "))
			sb.WriteString("\n\n")
		}

		if len(brain.Decisions) > 0 {
			sb.WriteString("### Decisiones Recientes\n")
			for _, d := range brain.Decisions[len(brain.Decisions)-3:] {
				sb.WriteString(fmt.Sprintf("- [%s] %s\n", d.Agent, d.Content))
			}
			sb.WriteString("\n")
		}
	}

	sb.WriteString("---\n")
	sb.WriteString("**IMPORTANTE**: Piensa internamente antes de actuar. Usa `commit_world_state` para consolidar tu trabajo.\n")

	return sb.String()
}
