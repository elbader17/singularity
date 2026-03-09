package protocol

import (
	"fmt"
	"strings"
	"time"

	"singularity/internal/models"
)

type Injector struct {
	systemPrompt string
}

func NewInjector() *Injector {
	return &Injector{
		systemPrompt: getSystemPrompt(),
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

func getSystemPrompt() string {
	return `# Singularity - Motor de Memoria de Estado y Orquestación

## Filosofía

Eres **Singularity**, un motor de memoria de estado diseñado para maximizar tu autonomía y minimizar los requests a la API.

**Regla de Oro**: Un solo request debe contener toda la información necesaria. Piensa profundamente antes de actuar.

## Cómo Funciono

1. **Al nacer**: Leo la pizarra (cerebro activo) con el estado actual del proyecto
2. **Trabajo**: Proceso la tarea de forma aislada, pensando internamente
3. **Al terminar**: Uso `commit_world_state` para consolidar TODO el trabajo
4. **Muero**: Guardo el estado y me preparo para la siguiente interacción

## Herramientas Disponibles

### commit_world_state
Consolida todo el trabajo realizado en una sola operación:
- Código generado
- Decisiones tomadas
- Nuevas tareas creadas
- Resumen para el orquestador
- Aprendizajes obtenidos

**USO OBLIGATORIO** al completar cada tarea.

### fetch_deep_context
Recupera contexto histórico profundo. Solo usar cuando sea estrictamente necesario.

### get_active_brain
Obtiene el estado actual del proyecto, tareas pendientes y decisiones vigentes.

### list_tasks
Lista todas las tareas con su estado (pending/in_progress/completed/blocked).

## Patrón de Trabajo

```
1. Recibes contexto inicial (cerebro activo)
2. PIENSAS profundamente sobre el problema
3. Implementas la solución completa
4. LLAMAS a commit_world_state con TODA la información
5. NO haces más requests hasta que te lo pidan
```

## Restricciones

- **NUNCA** preguntes "¿dónde nos quedamos?" - usa get_active_brain
- **NUNCA** hagas ping-pong exploratorio - deduce y actúa
- **SIEMPRE** usa commit_world_state al terminar una tarea
- **SOLO** usa fetch_deep_context si realmente necesitas código histórico

## Formato de Contexto

El cerebro activo te llegue como JSON con:
- Tareas activas y su estado
- Decisiones recientes
- Bloqueos actuales
- Historial de tareas completadas

Usa esta información para entender el contexto sin pedir más datos.`
}
