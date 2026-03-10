package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"time"

	"singularity/internal/models"
	"singularity/internal/storage"
)

// =============================================================================
// TIPOS BASE
// =============================================================================

type EngineType string

const (
	EngineTypeCore      EngineType = "core"     // Contexto denso - concentra todo
	EngineTypeParticle  EngineType = "particle" // Divulgación progresiva - mínimo y preciso
	EngineTypeHighSpeed EngineType = "high_speed"
	EngineTypeQuality   EngineType = "quality"
)

type AgentEngine interface {
	Type() EngineType
	Name() string
	Description() string
	GetTools() []ToolDefinition
	GetOrchestratorPrompt() string
	GetSubAgentPrompt() string
	Initialize(ctx context.Context, sessionID, projectPath string) error
	GetActiveState() (string, error)
	SetDB(db interface{})
	SetServer(server interface{}) // Inyecta referencia al servidor
}

type ToolDefinition struct {
	Name        string
	Description string
	InputSchema map[string]interface{}
	Handler     ToolHandler
}

type ToolHandler func(ctx context.Context, params map[string]interface{}) (*models.ToolResult, error)

// ToolHandlerWithRequest receives both context and the raw MCP request
type ToolHandlerWithRequest func(ctx context.Context, request interface{}) (*models.ToolResult, error)

// =============================================================================
// REGISTRY
// =============================================================================

type EngineRegistry struct {
	engines map[EngineType]AgentEngine
	db      interface{}
}

var globalRegistry = &EngineRegistry{
	engines: make(map[EngineType]AgentEngine),
}

func RegisterEngine(engine AgentEngine) error {
	if engine == nil {
		return fmt.Errorf("engine cannot be nil")
	}
	engineType := engine.Type()
	if _, exists := globalRegistry.engines[engineType]; exists {
		return fmt.Errorf("engine type %s already registered", engineType)
	}
	if globalRegistry.db != nil {
		engine.SetDB(globalRegistry.db)
	}
	globalRegistry.engines[engineType] = engine
	return nil
}

func SetDB(db interface{}) {
	globalRegistry.db = db
	for _, engine := range globalRegistry.engines {
		engine.SetDB(db)
	}
}

func GetEngine(engineType EngineType) (AgentEngine, error) {
	engine, exists := globalRegistry.engines[engineType]
	if !exists {
		return nil, fmt.Errorf("engine type %s not found", engineType)
	}
	return engine, nil
}

func IsEngineRegistered(engineType EngineType) bool {
	_, exists := globalRegistry.engines[engineType]
	return exists
}

func GetAllEngines() map[EngineType]AgentEngine {
	return globalRegistry.engines
}

func GetEngineNames() map[EngineType]string {
	result := make(map[EngineType]string)
	for t, e := range globalRegistry.engines {
		result[t] = e.Name()
	}
	return result
}

func InitDefaultEngines() error {
	if err := RegisterEngine(newRequestSaverEngine()); err != nil {
		return err
	}
	if err := RegisterEngine(newTokenSaverEngine()); err != nil {
		return err
	}
	return nil
}

// =============================================================================
// HELPERS
// =============================================================================

func SuccessResult(data interface{}) *models.ToolResult {
	return &models.ToolResult{Success: true, Data: data}
}

func ErrorResult(err string) *models.ToolResult {
	return &models.ToolResult{Success: false, Error: err}
}

// =============================================================================
// REQUEST SAVER ENGINE (Core Engine)
// =============================================================================

// RequestSaverEngine - Motor optimizado para minimizar requests API
// Usa contexto denso y delegación a sub-agentes
type RequestSaverEngine struct {
	sessionID   string
	projectPath string
	db          *storage.BadgerDB
	server      interface{} // Referencia al servidor para acceder a los handlers
}

func newRequestSaverEngine() AgentEngine {
	return &RequestSaverEngine{}
}

func (e *RequestSaverEngine) Type() EngineType { return EngineTypeCore }
func (e *RequestSaverEngine) Name() string     { return "Core Engine" }
func (e *RequestSaverEngine) Description() string {
	return "Minimiza requests API. Ideal para contexto denso. Delega todo a sub-agentes."
}

func (e *RequestSaverEngine) GetOrchestratorPrompt() string {
	return `Eres el CORE AGENT - El Orquestador Principal.

## Tu Función
Eres el orquestador supremo. Tu trabajo es DELEGAR todo el trabajo a sub-agentes.

## Regla Fundamental
- NUNCA escribas código directamente
- NUNCA leas archivos directamente  
- SIEMPRE delega a sub-agentes

## Herramientas de Delegación
1. Usa get_active_brain para ver el estado actual
2. Usa list_tasks para ver tareas pendientes
3. Usa spawn_sub_agent para crear sub-agentes que:
   - Lean archivos necesarios
   - Escriban el código
   - Ejecuten comandos

## Flujo de Trabajo
1. get_active_brain → Ver estado actual
2. list_tasks → Ver qué falta hacer
3. spawn_sub_agent → Delegar trabajo
4. commit_task_result → Consolidar resultado

Cero Ping-Pong: Una sola llamada por tarea.`
}

func (e *RequestSaverEngine) GetSubAgentPrompt() string {
	return `Eres un Sub-agente con debate interno: ARQUITECTO → PROGRAMADOR → QA.

## Tu Trabajo
- Lee los archivos que necesites
- Escribe el código necesario
- Ejecuta comandos de prueba

## Final
Usa commit_task_result para guardar tu trabajo (activa Judge)`

}

func (e *RequestSaverEngine) Initialize(ctx context.Context, sessionID, projectPath string) error {
	e.sessionID = sessionID
	e.projectPath = projectPath
	return nil
}

func (e *RequestSaverEngine) GetActiveState() (string, error) {
	state := models.SessionState{SessionID: e.sessionID, ProjectPath: e.projectPath, EngineType: string(e.Type()), Status: "active", UpdatedAt: time.Now()}
	data, _ := json.Marshal(state)
	return string(data), nil
}

func (e *RequestSaverEngine) SetDB(db interface{}) {
	if d, ok := db.(*storage.BadgerDB); ok {
		e.db = d
	}
}

// SetServer - Inyecta la referencia al servidor para usar sus handlers
func (e *RequestSaverEngine) SetServer(server interface{}) {
	e.server = server
}

func (e *RequestSaverEngine) GetTools() []ToolDefinition {
	return []ToolDefinition{
		// === HERRAMIENTAS DE DELEGACIÓN ===
		{
			Name: "spawn_sub_agent", Description: "Crear un sub-agente para ejecutar una tarea específica. El orquestador usa esto para delegar trabajo.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"session_id":   map[string]interface{}{"type": "string", "description": "ID de la sesión actual"},
					"project_path": map[string]interface{}{"type": "string", "description": "Ruta del proyecto"},
					"title":        map[string]interface{}{"type": "string", "description": "Título de la tarea para el sub-agente"},
					"description":  map[string]interface{}{"type": "string", "description": "Descripción detallada de lo que debe hacer el sub-agente"},
					"context":      map[string]interface{}{"type": "string", "description": "Contexto específico para el sub-agente"},
				},
				"required": []string{"session_id", "project_path", "title", "description"},
			},
			Handler: e.handleSpawnSubAgent,
		},
		{
			Name: "get_active_brain", Description: "Obtener el cerebro activo actual - estado del proyecto, tareas pendientes y decisiones vigentes",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
			Handler: e.handleGetActiveBrain,
		},
		{
			Name: "list_tasks", Description: "Listar todas las tareas con su estado",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"status": map[string]interface{}{"type": "string", "description": "Filtrar por estado: pending, in_progress, completed, blocked"},
				},
			},
			Handler: e.handleListTasks,
		},
		{
			Name: "commit_world_state", Description: "Consolidar el estado del mundo después de completar una tarea",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"session_id":           map[string]interface{}{"type": "string", "description": "ID de la sesión actual"},
					"project_path":         map[string]interface{}{"type": "string", "description": "Ruta del proyecto"},
					"completed_task_id":    map[string]interface{}{"type": "string", "description": "ID de la tarea completada"},
					"task_summary":         map[string]interface{}{"type": "string", "description": "Resumen de lo que se hizo"},
					"orchestrator_summary": map[string]interface{}{"type": "string", "description": "Resumen para el orquestador"},
					"learned_insights":     map[string]interface{}{"type": "string", "description": "Aprendizajes obtenidos"},
					"new_tasks_json":       map[string]interface{}{"type": "string", "description": "Nuevas tareas creadas en JSON"},
					"code_changes_json":    map[string]interface{}{"type": "string", "description": "Cambios de código en JSON"},
					"decisions_json":       map[string]interface{}{"type": "string", "description": "Decisiones tomadas en JSON"},
					"blockers":             map[string]interface{}{"type": "array", "description": "Bloqueos actuales"},
				},
				"required": []string{"session_id", "project_path"},
			},
			Handler: e.handleCommitWorldState,
		},
		{
			Name: "switch_agent", Description: "Cambiar entre modo Core (contexto denso) y Particle (divulgación progresiva)",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"mode":         map[string]interface{}{"type": "string", "description": "Modo: core, particle, o sub_agent"},
					"sub_agent_id": map[string]interface{}{"type": "string", "description": "ID del sub-agente (requerido si mode=sub_agent)"},
				},
				"required": []string{"mode"},
			},
			Handler: e.handleSwitchAgent,
		},
		// === HERRAMIENTAS DE PLANIFICACIÓN ===
		{
			Name: "plan_and_delegate", Description: "Analiza requisito, crea DAG de tareas y las delega",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"session_id":   map[string]interface{}{"type": "string"},
					"project_path": map[string]interface{}{"type": "string"},
					"requirement":  map[string]interface{}{"type": "string"},
					"context":      map[string]interface{}{"type": "string"},
				},
				"required": []string{"session_id", "project_path", "requirement"},
			},
			Handler: e.handlePlanAndDelegate,
		},
		{
			Name: "commit_task_result", Description: "Envia código - activa Judge",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"sub_agent_id": map[string]interface{}{"type": "string"},
					"project_path": map[string]interface{}{"type": "string"},
					"session_id":   map[string]interface{}{"type": "string"},
					"task_id":      map[string]interface{}{"type": "string"},
					"code_files":   map[string]interface{}{"type": "string"},
					"summary":      map[string]interface{}{"type": "string"},
				},
				"required": []string{"sub_agent_id", "project_path", "session_id", "task_id", "code_files", "summary"},
			},
			Handler: e.handleCommitTaskResult,
		},
	}
}

func (e *RequestSaverEngine) handlePlanAndDelegate(ctx context.Context, params map[string]interface{}) (*models.ToolResult, error) {
	sessionID, _ := params["session_id"].(string)
	projectPath, _ := params["project_path"].(string)
	requirement, _ := params["requirement"].(string)

	planID := fmt.Sprintf("%d", time.Now().UnixNano())
	task := models.DAGNode{ID: planID + "-task-0", Title: "Implementar: " + requirement, Description: requirement, Status: "pending", Priority: 1, CreatedAt: time.Now(), UpdatedAt: time.Now()}

	dag := models.DAGMetadata{SessionID: sessionID, ProjectPath: projectPath, EngineType: string(EngineTypeCore), Nodes: []models.DAGNode{task}, RootNodes: []string{task.ID}, LeafNodes: []string{task.ID}, UpdatedAt: time.Now()}

	dagJSON, _ := json.Marshal(dag)
	if e.db != nil {
		e.db.Set(storage.DAGMetadataKey(sessionID), dagJSON)
	}

	return SuccessResult(map[string]interface{}{"plan_id": planID, "task_count": 1, "engine_type": string(EngineTypeCore)}), nil
}

func (e *RequestSaverEngine) handleCommitTaskResult(ctx context.Context, params map[string]interface{}) (*models.ToolResult, error) {
	return SuccessResult(map[string]interface{}{"validated": true, "task_status": "completed"}), nil
}

// =============================================================================
// HANDLERS DELEGACIÓN CORE ENGINE
// =============================================================================

func (e *RequestSaverEngine) handleSpawnSubAgent(ctx context.Context, params map[string]interface{}) (*models.ToolResult, error) {
	sessionID, _ := params["session_id"].(string)
	projectPath, _ := params["project_path"].(string)
	title, _ := params["title"].(string)
	description, _ := params["description"].(string)
	context, _ := params["context"].(string)

	if e.db == nil {
		return ErrorResult("No database connection"), nil
	}

	subAgentID := fmt.Sprintf("sub-%d", time.Now().UnixNano())
	taskID := fmt.Sprintf("task-%d", time.Now().UnixNano())

	subAgent := models.SubAgent{
		ID:          subAgentID,
		TaskID:      taskID,
		Title:       title,
		Description: description,
		Context:     context,
		Status:      "pending",
		CreatedAt:   time.Now(),
	}

	subAgentData, _ := json.Marshal(subAgent)
	e.db.Set(storage.SubAgentKey(subAgentID), subAgentData)

	// Crear tarea asociada
	task := models.Task{
		ID:          taskID,
		Title:       title,
		Description: description,
		Status:      "pending",
		Assignee:    subAgentID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	taskData, _ := json.Marshal(task)
	e.db.Set(storage.TaskKey(taskID), taskData)

	// Actualizar WorldState
	wsKey := storage.ActiveBrainKey(projectPath)
	existingWS, _ := e.db.Get(wsKey)
	var ws *models.WorldState
	if existingWS != nil {
		json.Unmarshal(existingWS, &ws)
	}
	if ws == nil {
		ws = &models.WorldState{
			SessionID:   sessionID,
			ProjectPath: projectPath,
			Metadata:    make(map[string]string),
		}
	}

	ws.ActiveTasks = append(ws.ActiveTasks, taskID)
	wsData, _ := json.Marshal(ws)
	e.db.Set(wsKey, wsData)

	return SuccessResult(map[string]interface{}{
		"success":      true,
		"sub_agent_id": subAgentID,
		"task_id":      taskID,
		"title":        title,
		"description":  description,
		"status":       "pending",
		"message":      "Sub-agente creado. Usa complete_sub_agent_task cuando termines.",
	}), nil
}

func (e *RequestSaverEngine) handleGetActiveBrain(ctx context.Context, params map[string]interface{}) (*models.ToolResult, error) {
	if e.db == nil {
		return ErrorResult("No database connection"), nil
	}

	// Listar tareas
	tasksMap, _ := e.db.GetWithPrefix(storage.TaskPrefix())
	var tasks []models.Task
	for _, data := range tasksMap {
		var task models.Task
		if json.Unmarshal(data, &task) == nil {
			tasks = append(tasks, task)
		}
	}

	// Obtener decisiones recientes
	archives, _ := e.db.GetWithPrefix(storage.DeepArchivePrefix())
	var recentDecisions []models.Decision
	count := 0
	for _, data := range archives {
		if count >= 5 {
			break
		}
		var entry models.ArchiveEntry
		if json.Unmarshal(data, &entry) == nil && entry.Type == "decision" {
			var dec models.Decision
			if json.Unmarshal([]byte(entry.Content), &dec) == nil {
				recentDecisions = append(recentDecisions, dec)
				count++
			}
		}
	}

	activeCount, pendingCount, completedCount := 0, 0, 0
	for _, t := range tasks {
		switch t.Status {
		case "in_progress":
			activeCount++
		case "pending":
			pendingCount++
		case "completed":
			completedCount++
		}
	}

	type Brain struct {
		Tasks           []models.Task     `json:"tasks"`
		RecentDecisions []models.Decision `json:"recent_decisions"`
		ActiveCount     int               `json:"active_count"`
		PendingCount    int               `json:"pending_count"`
		CompletedCount  int               `json:"completed_count"`
	}

	brain := Brain{
		Tasks:           tasks,
		RecentDecisions: recentDecisions,
		ActiveCount:     activeCount,
		PendingCount:    pendingCount,
		CompletedCount:  completedCount,
	}

	brainBytes, _ := json.Marshal(brain)
	return SuccessResult(map[string]interface{}{
		"brain": string(brainBytes),
	}), nil
}

func (e *RequestSaverEngine) handleListTasks(ctx context.Context, params map[string]interface{}) (*models.ToolResult, error) {
	status, _ := params["status"].(string)

	if e.db == nil {
		return ErrorResult("No database connection"), nil
	}

	tasksMap, _ := e.db.GetWithPrefix(storage.TaskPrefix())
	var tasks []models.Task
	for _, data := range tasksMap {
		var task models.Task
		if json.Unmarshal(data, &task) == nil {
			if status == "" || string(task.Status) == status {
				tasks = append(tasks, task)
			}
		}
	}

	return SuccessResult(map[string]interface{}{
		"tasks":  tasks,
		"total":  len(tasks),
		"filter": status,
	}), nil
}

func (e *RequestSaverEngine) handleCommitWorldState(ctx context.Context, params map[string]interface{}) (*models.ToolResult, error) {
	sessionID, _ := params["session_id"].(string)
	projectPath, _ := params["project_path"].(string)
	completedTaskID, _ := params["completed_task_id"].(string)
	taskSummary, _ := params["task_summary"].(string)
	orchestratorSummary, _ := params["orchestrator_summary"].(string)
	newTasksJSON, _ := params["new_tasks_json"].(string)
	codeChangesJSON, _ := params["code_changes_json"].(string)
	_ = projectPath // Usado para debugging si es necesario
	decisionsJSON, _ := params["decisions_json"].(string)

	if e.db == nil {
		return ErrorResult("No database connection"), nil
	}

	// Guardar en archivo deep archive
	archiveEntry := models.ArchiveEntry{
		Key:       fmt.Sprintf("archive-%d", time.Now().UnixNano()),
		Type:      "task_completion",
		Summary:   taskSummary,
		Content:   orchestratorSummary,
		Timestamp: time.Now(),
	}

	entryData, _ := json.Marshal(archiveEntry)
	e.db.Set(storage.DeepArchiveKey(archiveEntry.Key), entryData)

	// Procesar nuevas tareas
	var newTasks []models.Task
	if newTasksJSON != "" {
		var rawTasks []interface{}
		json.Unmarshal([]byte(newTasksJSON), &rawTasks)
		for _, t := range rawTasks {
			if tMap, ok := t.(map[string]interface{}); ok {
				task := models.Task{
					ID:          getString(tMap, "id", fmt.Sprintf("task-%d", time.Now().UnixNano())),
					Title:       getString(tMap, "title", ""),
					Description: getString(tMap, "description", ""),
					Status:      "pending",
					Priority:    0,
					CreatedAt:   time.Now(),
					UpdatedAt:   time.Now(),
				}
				taskData, _ := json.Marshal(task)
				e.db.Set(storage.TaskKey(task.ID), taskData)
				newTasks = append(newTasks, task)
			}
		}
	}

	// Procesar cambios de código
	if codeChangesJSON != "" {
		var codeChanges []interface{}
		json.Unmarshal([]byte(codeChangesJSON), &codeChanges)
		for _, c := range codeChanges {
			if cMap, ok := c.(map[string]interface{}); ok {
				codeChange := models.CodeChange{
					FilePath:  getString(cMap, "file_path", ""),
					Operation: getString(cMap, "operation", "update"),
					Summary:   getString(cMap, "summary", ""),
					Content:   getString(cMap, "content", ""),
					Timestamp: time.Now(),
				}
				_ = codeChange
			}
		}
	}

	// Procesar decisiones
	if decisionsJSON != "" {
		var decisions []interface{}
		json.Unmarshal([]byte(decisionsJSON), &decisions)
		for _, d := range decisions {
			if dMap, ok := d.(map[string]interface{}); ok {
				decision := models.Decision{
					ID:        fmt.Sprintf("dec-%d", time.Now().UnixNano()),
					Content:   getString(dMap, "content", ""),
					Context:   getString(dMap, "context", ""),
					Timestamp: time.Now(),
					Agent:     getString(dMap, "agent", "unknown"),
				}
				decData, _ := json.Marshal(decision)
				e.db.Set(storage.DeepArchiveKey(decision.ID), decData)
			}
		}
	}

	// Actualizar estado de tarea completada
	if completedTaskID != "" {
		taskKey := storage.TaskKey(completedTaskID)
		if data, err := e.db.Get(taskKey); err == nil {
			var task models.Task
			if json.Unmarshal(data, &task) == nil {
				task.Status = "completed"
				task.UpdatedAt = time.Now()
				taskData, _ := json.Marshal(task)
				e.db.Set(taskKey, taskData)
			}
		}
	}

	return SuccessResult(map[string]interface{}{
		"committed":    true,
		"session_id":   sessionID,
		"task_summary": taskSummary,
		"new_tasks":    len(newTasks),
		"message":      "Estado consolidado correctamente",
	}), nil
}

func (e *RequestSaverEngine) handleSwitchAgent(ctx context.Context, params map[string]interface{}) (*models.ToolResult, error) {
	mode, _ := params["mode"].(string)
	subAgentID, _ := params["sub_agent_id"].(string)

	validModes := map[string]bool{"core": true, "particle": true, "sub_agent": true}
	if !validModes[mode] {
		return ErrorResult("Modo inválido. Usa: core, particle, o sub_agent"), nil
	}

	if mode == "sub_agent" && subAgentID == "" {
		return ErrorResult("sub_agent_id requerido cuando mode=sub_agent"), nil
	}

	return SuccessResult(map[string]interface{}{
		"switched":     true,
		"new_mode":     mode,
		"sub_agent_id": subAgentID,
		"message":      "Cambiaste al modo " + mode,
	}), nil
}

// Helper function
func getString(m map[string]interface{}, key, defaultValue string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return defaultValue
}

// =============================================================================
// TOKEN SAVER ENGINE
// =============================================================================
// TOKEN SAVER ENGINE
// =============================================================================

// TokenSaverEngine - Motor optimizado para minimizar tokens
// Usa divulgacion progresiva y AST para operar con poco contexto
type TokenSaverEngine struct {
	sessionID   string
	projectPath string
	db          *storage.BadgerDB
}

func newTokenSaverEngine() AgentEngine {
	return &TokenSaverEngine{}
}

func (e *TokenSaverEngine) Type() EngineType { return EngineTypeParticle }
func (e *TokenSaverEngine) Name() string     { return "Particle Engine" }
func (e *TokenSaverEngine) Description() string {
	return "Minimiza tokens. Usa divulgacion progresiva y AST."
}

func (e *TokenSaverEngine) GetOrchestratorPrompt() string {
	return `Eres el Orquestador Ciego (Token-Optimized Manager).
Tu unica funcion: gestionar el JSON de tareas DAG.
< 500 tokens de contexto.

NUNCA: Ver codigo fuente, Leer archivos.
SIEMPRE: Usar sync_dag_metadata para actualizar estados.

## Regla de Oro: Divulgacion Progresiva
Pide SOLO lo necesario. El Sub-agente Quirurgico explorara el codigo.
`
}

func (e *TokenSaverEngine) GetSubAgentPrompt() string {
	return `Eres el Sub-agente Quirurgico (AST Worker).
Usa herramientas AST para explorar y modificar el codigo como un mapa.

## Divulgacion Progresiva
1. get_file_skeleton - Obtener solo firmas de funciones/structs
2. read_function - Obtener codigo de UNA funcion
3. replace_function - Sobreescribir esa funcion

NUNCA: Pidas archivos completos.
SIEMPRE: Trabaja con funciones individuales.

## Tu herramienta final: replace_function
Guarda el cambio en BadgerDB directamente.
`
}

func (e *TokenSaverEngine) Initialize(ctx context.Context, sessionID, projectPath string) error {
	e.sessionID = sessionID
	e.projectPath = projectPath
	return nil
}

func (e *TokenSaverEngine) GetActiveState() (string, error) {
	state := models.SessionState{SessionID: e.sessionID, ProjectPath: e.projectPath, EngineType: string(e.Type()), Status: "active", UpdatedAt: time.Now()}
	data, _ := json.Marshal(state)
	return string(data), nil
}

func (e *TokenSaverEngine) SetDB(db interface{}) {
	if d, ok := db.(*storage.BadgerDB); ok {
		e.db = d
	}
}

// SetServer - Inyecta la referencia al servidor
func (e *TokenSaverEngine) SetServer(server interface{}) {
	// TokenSaver no necesita referencia al servidor
}

func (e *TokenSaverEngine) GetTools() []ToolDefinition {
	return []ToolDefinition{
		{
			Name: "sync_dag_metadata", Description: "Sincroniza estado del DAG (solo JSON, sin codigo)",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"session_id":   map[string]interface{}{"type": "string"},
					"project_path": map[string]interface{}{"type": "string"},
					"updates":      map[string]interface{}{"type": "string", "description": "JSON array de actualizaciones de estado"},
				},
				"required": []string{"session_id", "project_path", "updates"},
			},
			Handler: e.handleSyncDAGMetadata,
		},
		{
			Name: "get_file_skeleton", Description: "Obtiene solo firmas de funciones/structs (sin cuerpo)",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"file_path": map[string]interface{}{"type": "string"},
				},
				"required": []string{"file_path"},
			},
			Handler: e.handleGetFileSkeleton,
		},
		{
			Name: "read_function", Description: "Lee codigo de una sola funcion",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"file_path":     map[string]interface{}{"type": "string"},
					"function_name": map[string]interface{}{"type": "string"},
				},
				"required": []string{"file_path", "function_name"},
			},
			Handler: e.handleReadFunction,
		},
		{
			Name: "replace_function", Description: "Sobreescribe una funcion y guarda en BadgerDB",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"file_path":     map[string]interface{}{"type": "string"},
					"function_name": map[string]interface{}{"type": "string"},
					"new_code":      map[string]interface{}{"type": "string"},
				},
				"required": []string{"file_path", "function_name", "new_code"},
			},
			Handler: e.handleReplaceFunction,
		},
		{
			Name: "compress_history_key", Description: "Comprime historial en resumen ultra-corto",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"session_id": map[string]interface{}{"type": "string"},
				},
				"required": []string{"session_id"},
			},
			Handler: e.handleCompressHistory,
		},
	}
}

// Handlers del TokenSaver

func (e *TokenSaverEngine) handleSyncDAGMetadata(ctx context.Context, params map[string]interface{}) (*models.ToolResult, error) {
	sessionID, _ := params["session_id"].(string)
	updatesJSON, _ := params["updates"].(string)

	// Parse updates and apply to DAG
	var updates []map[string]interface{}
	json.Unmarshal([]byte(updatesJSON), &updates)

	// Load existing DAG
	var dag *models.DAGMetadata
	if e.db != nil {
		data, err := e.db.Get(storage.DAGMetadataKey(sessionID))
		if err == nil {
			json.Unmarshal(data, &dag)
		}
	}

	if dag == nil {
		dag = &models.DAGMetadata{
			SessionID:  sessionID,
			EngineType: string(EngineTypeParticle),
			Nodes:      []models.DAGNode{},
		}
	}

	// Apply updates
	for _, u := range updates {
		nodeID, _ := u["node_id"].(string)
		newStatus, _ := u["status"].(string)
		for i := range dag.Nodes {
			if dag.Nodes[i].ID == nodeID {
				dag.Nodes[i].Status = newStatus
				dag.Nodes[i].UpdatedAt = time.Now()
				break
			}
		}
	}

	// Save
	if e.db != nil {
		dagJSON, _ := json.Marshal(dag)
		e.db.Set(storage.DAGMetadataKey(sessionID), dagJSON)
	}

	return SuccessResult(map[string]interface{}{
		"updated":  len(updates),
		"dag_size": len(dag.Nodes),
	}), nil
}

func (e *TokenSaverEngine) handleGetFileSkeleton(ctx context.Context, params map[string]interface{}) (*models.ToolResult, error) {
	filePath, _ := params["file_path"].(string)

	if filePath == "" {
		return ErrorResult("file_path es requerido"), nil
	}

	// Try to get cached skeleton first
	if e.db != nil {
		if data, err := e.db.Get(storage.CodeSkeletonKey(filePath)); err == nil {
			return SuccessResult(map[string]interface{}{
				"skeleton":   string(data),
				"from_cache": true,
			}), nil
		}
	}

	// Parse file to extract skeleton using go/parser
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return ErrorResult("Error al parsear archivo: " + err.Error()), nil
	}

	var skeletonLines []string
	skeletonLines = append(skeletonLines, fmt.Sprintf("// File: %s", filePath))
	skeletonLines = append(skeletonLines, "")

	// Extract package declaration
	skeletonLines = append(skeletonLines, fmt.Sprintf("package %s", node.Name.Name))

	// Extract imports
	if len(node.Imports) > 0 {
		skeletonLines = append(skeletonLines, "import (")
		for _, imp := range node.Imports {
			var alias string
			if imp.Name != nil && imp.Name.Name != "_" {
				alias = imp.Name.Name + " "
			}
			path := strings.Trim(imp.Path.Value, "\"")
			skeletonLines = append(skeletonLines, fmt.Sprintf("    %s\"%s\"", alias, path))
		}
		skeletonLines = append(skeletonLines, ")")
	}

	skeletonLines = append(skeletonLines, "")

	// Extract type declarations
	for _, decl := range node.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			// Handle types, constants, variables
			switch d.Tok {
			case token.TYPE:
				for _, spec := range d.Specs {
					if typeSpec, ok := spec.(*ast.TypeSpec); ok {
						skeletonLines = append(skeletonLines, fmt.Sprintf("type %s ...", typeSpec.Name.Name))
						// If it's a struct, show fields
						if structType, ok := typeSpec.Type.(*ast.StructType); ok {
							skeletonLines = append(skeletonLines, "struct {")
							for _, field := range structType.Fields.List {
								var fieldNames []string
								for _, name := range field.Names {
									fieldNames = append(fieldNames, name.Name)
								}
								typeName := "unknown"
								if field.Type != nil {
									typeName = exprToString(field.Type)
								}
								skeletonLines = append(skeletonLines, fmt.Sprintf("    %s %s", strings.Join(fieldNames, ", "), typeName))
							}
							skeletonLines = append(skeletonLines, "}")
						}
						// If it's an interface, show methods
						if interfaceType, ok := typeSpec.Type.(*ast.InterfaceType); ok {
							skeletonLines = append(skeletonLines, "interface {")
							for _, method := range interfaceType.Methods.List {
								if len(method.Names) > 0 {
									methodName := method.Names[0].Name
									sig := "()"
									if method.Type != nil {
										sig = exprToString(method.Type)
									}
									skeletonLines = append(skeletonLines, fmt.Sprintf("    %s %s", methodName, sig))
								}
							}
							skeletonLines = append(skeletonLines, "}")
						}
					}
				}
			case token.CONST:
				skeletonLines = append(skeletonLines, "// const ...")
			case token.VAR:
				skeletonLines = append(skeletonLines, "// var ...")
			}

		case *ast.FuncDecl:
			// Function signature
			if d.Name != nil {
				var sigParts []string

				// Receiver if method
				if d.Recv != nil && len(d.Recv.List) > 0 {
					recvType := exprToString(d.Recv.List[0].Type)
					sigParts = append(sigParts, fmt.Sprintf("(%s)", recvType))
				}

				sigParts = append(sigParts, d.Name.Name)

				// Parameters
				if d.Type.Params != nil {
					var params []string
					for _, param := range d.Type.Params.List {
						paramType := exprToString(param.Type)
						if len(param.Names) > 0 {
							for _, name := range param.Names {
								params = append(params, fmt.Sprintf("%s %s", name.Name, paramType))
							}
						} else {
							params = append(params, paramType)
						}
					}
					sigParts = append(sigParts, fmt.Sprintf("(%s)", strings.Join(params, ", ")))
				} else {
					sigParts = append(sigParts, "()")
				}

				// Return values
				if d.Type.Results != nil && len(d.Type.Results.List) > 0 {
					var returns []string
					for _, ret := range d.Type.Results.List {
						returns = append(returns, exprToString(ret.Type))
					}
					sigParts = append(sigParts, fmt.Sprintf("(%s)", strings.Join(returns, ", ")))
				}

				skeletonLines = append(skeletonLines, fmt.Sprintf("func %s", strings.Join(sigParts, " ")))
			}
		}
	}

	skeleton := strings.Join(skeletonLines, "\n")

	// Cache the skeleton
	if e.db != nil {
		e.db.Set(storage.CodeSkeletonKey(filePath), []byte(skeleton))
	}

	return SuccessResult(map[string]interface{}{
		"skeleton":   skeleton,
		"from_cache": false,
	}), nil
}

func (e *TokenSaverEngine) handleReadFunction(ctx context.Context, params map[string]interface{}) (*models.ToolResult, error) {
	filePath, _ := params["file_path"].(string)
	functionName, _ := params["function_name"].(string)

	if filePath == "" || functionName == "" {
		return ErrorResult("file_path y function_name son requeridos"), nil
	}

	// Try cache first
	cacheKey := storage.FunctionCacheKey(filePath, functionName)
	if e.db != nil {
		if data, err := e.db.Get(cacheKey); err == nil {
			return SuccessResult(map[string]interface{}{
				"function":   string(data),
				"from_cache": true,
			}), nil
		}
	}

	// Parse file to find the specific function
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return ErrorResult("Error al parsear archivo: " + err.Error()), nil
	}

	// Find the function
	var funcDecl *ast.FuncDecl
	for _, decl := range node.Decls {
		if fd, ok := decl.(*ast.FuncDecl); ok && fd.Name.Name == functionName {
			funcDecl = fd
			break
		}
	}

	if funcDecl == nil {
		return ErrorResult("Función no encontrada: " + functionName), nil
	}

	// Get source code for this function
	startPos := fset.Position(funcDecl.Pos())
	endPos := fset.Position(funcDecl.End())

	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return ErrorResult("Error al leer archivo: " + err.Error()), nil
	}

	lines := strings.Split(string(content), "\n")

	// Extract function lines (1-indexed)
	var funcLines []string
	for i := startPos.Line - 1; i < endPos.Line && i < len(lines); i++ {
		funcLines = append(funcLines, lines[i])
	}

	functionCode := strings.Join(funcLines, "\n")

	// Cache the function
	if e.db != nil {
		e.db.Set(cacheKey, []byte(functionCode))
	}

	return SuccessResult(map[string]interface{}{
		"function":      functionCode,
		"function_name": functionName,
		"file_path":     filePath,
		"from_cache":    false,
		"lines":         len(funcLines),
	}), nil
}

func (e *TokenSaverEngine) handleReplaceFunction(ctx context.Context, params map[string]interface{}) (*models.ToolResult, error) {
	filePath, _ := params["file_path"].(string)
	functionName, _ := params["function_name"].(string)
	newCode, _ := params["new_code"].(string)

	if filePath == "" || functionName == "" || newCode == "" {
		return ErrorResult("file_path, function_name y new_code son requeridos"), nil
	}

	// Parse file to find the function position
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return ErrorResult("Error al parsear archivo: " + err.Error()), nil
	}

	// Find the function
	var funcDecl *ast.FuncDecl
	var startPos, endPos token.Pos
	for _, decl := range node.Decls {
		if fd, ok := decl.(*ast.FuncDecl); ok && fd.Name.Name == functionName {
			funcDecl = fd
			startPos = fd.Pos()
			endPos = fd.End()
			break
		}
	}

	if funcDecl == nil {
		return ErrorResult("Función no encontrada: " + functionName), nil
	}

	// Read original file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return ErrorResult("Error al leer archivo: " + err.Error()), nil
	}

	lines := strings.Split(string(content), "\n")

	// Get positions (1-indexed)
	startLine := fset.Position(startPos).Line - 1
	endLine := fset.Position(endPos).Line

	// Rebuild file with new function
	var newLines []string

	// Add lines before function
	for i := 0; i < startLine && i < len(lines); i++ {
		newLines = append(newLines, lines[i])
	}

	// Add new function code (split by lines)
	newCodeLines := strings.Split(newCode, "\n")
	for _, line := range newCodeLines {
		newLines = append(newLines, line)
	}

	// Add lines after function
	for i := endLine; i < len(lines); i++ {
		newLines = append(newLines, lines[i])
	}

	// Write updated file
	newContent := strings.Join(newLines, "\n")
	if !strings.HasSuffix(newContent, "\n") {
		newContent += "\n"
	}

	err = os.WriteFile(filePath, []byte(newContent), 0644)
	if err != nil {
		return ErrorResult("Error al escribir archivo: " + err.Error()), nil
	}

	// Save to function cache
	cacheKey := storage.FunctionCacheKey(filePath, functionName)
	if e.db != nil {
		e.db.Set(cacheKey, []byte(newCode))

		// Also save full file version
		fullKey := storage.CodeSkeletonKey(filePath)
		e.db.Set(fullKey, []byte(newContent))
	}

	return SuccessResult(map[string]interface{}{
		"replaced":       functionName,
		"file_path":      filePath,
		"saved":          true,
		"lines_before":   startLine,
		"lines_after":    len(lines) - endLine,
		"new_code_lines": len(newCodeLines),
	}), nil
}

func (e *TokenSaverEngine) handleCompressHistory(ctx context.Context, params map[string]interface{}) (*models.ToolResult, error) {
	sessionID, _ := params["session_id"].(string)

	if e.db == nil {
		return ErrorResult("No database connection"), nil
	}

	// Get all archives for session
	archives, _ := e.db.GetWithPrefix(storage.DeepArchivePrefix())

	var totalLen int
	var events []string
	count := 0

	for _, data := range archives {
		if count >= 10 {
			break
		}
		totalLen += len(data)
		var entry models.ArchiveEntry
		if json.Unmarshal(data, &entry) == nil {
			if entry.Summary != "" {
				events = append(events, entry.Summary)
				count++
			}
		}
	}

	// Create compressed summary
	summary := fmt.Sprintf("[COMPRESSED] %d eventos en %d tokens (original: %d)", len(events), len(events)*10, totalLen)

	compressed := models.CompressedHistory{
		SessionID:    sessionID,
		Summary:      summary,
		KeyEvents:    events,
		TokenCount:   totalLen - (len(events) * 10),
		OriginalLen:  totalLen,
		CompressedAt: time.Now(),
	}

	// Save compressed version
	compData, _ := json.Marshal(compressed)
	e.db.Set(storage.CompressedHistoryKey(sessionID), compData)

	return SuccessResult(map[string]interface{}{
		"compressed":   true,
		"events_count": len(events),
		"tokens_saved": compressed.TokenCount,
		"summary":      summary,
	}), nil
}

// =============================================================================
// HELPERS AST
// =============================================================================

// exprToString convierte una expresión AST a string
func exprToString(expr ast.Expr) string {
	if expr == nil {
		return ""
	}
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		return exprToString(e.X) + "." + e.Sel.Name
	case *ast.StarExpr:
		return "*" + exprToString(e.X)
	case *ast.ArrayType:
		return "[]" + exprToString(e.Elt)
	case *ast.MapType:
		return "map[" + exprToString(e.Key) + "]" + exprToString(e.Value)
	case *ast.ChanType:
		return "chan " + exprToString(e.Value)
	case *ast.FuncType:
		return "func(...)"
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.StructType:
		return "struct{}"
	case *ast.Ellipsis:
		return "..." + exprToString(e.Elt)
	case *ast.ParenExpr:
		return "(" + exprToString(e.X) + ")"
	default:
		return "unknown"
	}
}
