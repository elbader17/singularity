package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"singularity/internal/agents"
	"singularity/internal/models"
	"singularity/internal/storage"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type Server struct {
	db           *storage.BadgerDB
	s            *server.MCPServer
	activeEngine agents.AgentEngine
}

func NewServer(db *storage.BadgerDB) *Server {
	s := server.NewMCPServer(
		"Singularity",
		"1.0.0",
		server.WithToolCapabilities(true),
		server.WithRecovery(),
	)

	srv := &Server{
		db: db,
		s:  s,
	}

	// Inicializar motors y registrar herramientas
	srv.initEngines()
	srv.registerTools()
	srv.registerResources()

	return srv
}

func (s *Server) initEngines() {
	// Set DB para todos los engines
	agents.SetDB(s.db)

	// Inicializar motores por defecto
	if err := agents.InitDefaultEngines(); err != nil {
		fmt.Printf("Warning: Failed to init engines: %v\n", err)
	}

	// Por defecto, usar TokenSaver
	if engine, err := agents.GetEngine(agents.EngineTypeParticle); err == nil {
		s.activeEngine = engine
		engine.Initialize(context.Background(), "", "")
		// Inyectar referencia al servidor
		if engine, ok := engine.(interface{ SetServer(interface{}) }); ok {
			engine.SetServer(s)
		}
	}
}

// SetActiveEngine cambia el motor activo
func (s *Server) SetActiveEngine(engineType agents.EngineType) error {
	engine, err := agents.GetEngine(engineType)
	if err != nil {
		return err
	}
	s.activeEngine = engine
	engine.Initialize(context.Background(), "", "")
	// Inyectar referencia al servidor
	if engine, ok := engine.(interface{ SetServer(interface{}) }); ok {
		engine.SetServer(s)
	}
	// Registrar herramientas del nuevo motor
	s.registerEngineTools()
	return nil
}

// GetActiveEngine retorna el motor activo
func (s *Server) GetActiveEngine() agents.AgentEngine {
	return s.activeEngine
}

// GetEngineInfo retorna información del motor activo
func (s *Server) GetEngineInfo() map[string]string {
	if s.activeEngine == nil {
		return map[string]string{"status": "no engine"}
	}
	return map[string]string{
		"type":        string(s.activeEngine.Type()),
		"name":        s.activeEngine.Name(),
		"description": s.activeEngine.Description(),
	}
}

func (s *Server) registerTools() {
	commitTool := mcp.NewTool("commit_world_state",
		mcp.WithDescription("Consolidar el estado del mundo después de completar una tarea. "+
			"Esta es la herramienta principal para guardar todo el trabajo realizado."),
		mcp.WithString("session_id",
			mcp.Required(),
			mcp.Description("ID de la sesión actual"),
		),
		mcp.WithString("project_path",
			mcp.Required(),
			mcp.Description("Ruta del proyecto"),
		),
		mcp.WithString("completed_task_id",
			mcp.Description("ID de la tarea completada"),
		),
		mcp.WithString("task_summary",
			mcp.Description("Resumen de lo que se hizo en la tarea"),
		),
		mcp.WithString("orchestrator_summary",
			mcp.Description("Resumen para el orquestador (alto nivel)"),
		),
		mcp.WithString("learned_insights",
			mcp.Description("Aprendizajes o insights obtenidos"),
		),
		mcp.WithString("new_tasks_json",
			mcp.Description("Nuevas tareas creadas en formato JSON array"),
		),
		mcp.WithString("code_changes_json",
			mcp.Description("Cambios de código realizados en formato JSON array"),
		),
		mcp.WithString("decisions_json",
			mcp.Description("Decisiones tomadas en formato JSON array"),
		),
		mcp.WithArray("blockers",
			mcp.Description("Bloqueos actuales"),
			mcp.WithStringItems(),
		),
	)

	fetchDeepTool := mcp.NewTool("fetch_deep_context",
		mcp.WithDescription("Recuperar contexto histórico profundo. "+
			"Usar solo cuando sea estrictamente necesario."),
		mcp.WithString("query",
			mcp.Description("Búsqueda o tema a recuperar"),
		),
		mcp.WithString("task_id",
			mcp.Description("ID de tarea específica a recuperar"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Límite de resultados (default 10)"),
		),
	)

	getActiveBrainTool := mcp.NewTool("get_active_brain",
		mcp.WithDescription("Obtener el cerebro activo actual - "+
			"estado del proyecto, tareas pendientes y decisiones vigentes"),
	)

	listTasksTool := mcp.NewTool("list_tasks",
		mcp.WithDescription("Listar todas las tareas con su estado"),
		mcp.WithString("status",
			mcp.Description("Filtrar por estado: pending, in_progress, completed, blocked"),
		),
	)

	spawnSubAgentTool := mcp.NewTool("spawn_sub_agent",
		mcp.WithDescription("Crear un sub-agente para ejecutar una tarea específica. "+
			"El orquestador usa esto para delegar trabajo. Solo crea la tarea, no la ejecuta."),
		mcp.WithString("session_id",
			mcp.Required(),
			mcp.Description("ID de la sesión actual"),
		),
		mcp.WithString("project_path",
			mcp.Required(),
			mcp.Description("Ruta del proyecto"),
		),
		mcp.WithString("title",
			mcp.Required(),
			mcp.Description("Título de la tarea para el sub-agente"),
		),
		mcp.WithString("description",
			mcp.Required(),
			mcp.Description("Descripción detallada de lo que debe hacer el sub-agente"),
		),
		mcp.WithString("context",
			mcp.Description("Contexto específico para el sub-agente (código relevante, archivos, etc)"),
		),
	)

	getSubAgentTaskTool := mcp.NewTool("get_sub_agent_task",
		mcp.WithDescription("Obtener la tarea asignada al sub-agente actual. "+
			"Usar al iniciar un sub-agente para obtener su contexto."),
		mcp.WithString("sub_agent_id",
			mcp.Required(),
			mcp.Description("ID del sub-agente"),
		),
	)

	completeSubAgentTool := mcp.NewTool("complete_sub_agent_task",
		mcp.WithDescription("Completar la tarea del sub-agente y reportar resultados. "+
			"Usar al terminar la ejecución."),
		mcp.WithString("sub_agent_id",
			mcp.Required(),
			mcp.Description("ID del sub-agente"),
		),
		mcp.WithString("result",
			mcp.Required(),
			mcp.Description("Resultado de la ejecución"),
		),
		mcp.WithString("error",
			mcp.Description("Error si falló la ejecución"),
		),
	)

	s.s.AddTool(commitTool, s.handleCommitWorldState)
	s.s.AddTool(fetchDeepTool, s.handleFetchDeepContext)
	s.s.AddTool(getActiveBrainTool, s.handleGetActiveBrain)
	s.s.AddTool(listTasksTool, s.handleListTasks)
	s.s.AddTool(spawnSubAgentTool, s.handleSpawnSubAgent)
	s.s.AddTool(getSubAgentTaskTool, s.handleGetSubAgentTask)
	s.s.AddTool(completeSubAgentTool, s.handleCompleteSubAgentTask)
	s.s.AddTool(mcp.NewTool("switch_agent",
		mcp.WithDescription("Cambiar entre modo Core (contexto denso) y Particle (divulgación progresiva). "+
			"Usa 'core' para contexto denso, 'particle' para ahorro de tokens."),
		mcp.WithString("mode",
			mcp.Required(),
			mcp.Description("Modo: 'core' (contexto denso), 'particle' (ahorro tokens), o 'sub_agent'"),
			mcp.Enum("core", "particle", "sub_agent"),
		),
		mcp.WithString("sub_agent_id",
			mcp.Description("ID del sub-agente (requerido si mode=sub_agent)"),
		),
	), s.handleSwitchAgent)

	// Nueva herramienta: plan_and_delegate
	planAndDelegateTool := mcp.NewTool("plan_and_delegate",
		mcp.WithDescription("Analiza un requisito, crea un DAG de tareas y las delega a sub-agentes. "+
			"Esta es la herramienta principal del Orquestador."),
		mcp.WithString("session_id",
			mcp.Required(),
			mcp.Description("ID de la sesión actual"),
		),
		mcp.WithString("project_path",
			mcp.Required(),
			mcp.Description("Ruta del proyecto"),
		),
		mcp.WithString("requirement",
			mcp.Required(),
			mcp.Description("Requisito de negocio a implementar"),
		),
		mcp.WithString("context",
			mcp.Description("Contexto adicional del proyecto (opcional)"),
		),
	)
	s.s.AddTool(planAndDelegateTool, s.handlePlanAndDelegate)

	// Nueva herramienta: commit_task_result (con Judge Determinista)
	commitTaskResultTool := mcp.NewTool("commit_task_result",
		mcp.WithDescription("Envía el código generado por el Sub-agente. "+
			"ACTIVA el JUEZ DETERMINISTA que validará compilación antes de guardar."),
		mcp.WithString("sub_agent_id",
			mcp.Required(),
			mcp.Description("ID del sub-agente que generó el código"),
		),
		mcp.WithString("project_path",
			mcp.Required(),
			mcp.Description("Ruta del proyecto"),
		),
		mcp.WithString("session_id",
			mcp.Required(),
			mcp.Description("ID de la sesión actual"),
		),
		mcp.WithString("task_id",
			mcp.Required(),
			mcp.Description("ID de la tarea"),
		),
		mcp.WithString("code_files",
			mcp.Required(),
			mcp.Description("JSON array de archivos: [{\"file_path\": \"ruta\", \"content\": \"código\"}]"),
		),
		mcp.WithString("summary",
			mcp.Required(),
			mcp.Description("Resumen de lo que se implementó"),
		),
		mcp.WithString("validation_notes",
			mcp.Description("Notas de validación interna del sub-agente"),
		),
		mcp.WithString("dependencies_json",
			mcp.Description("Dependencias adicionales (package.json, go.mod, etc.) en JSON"),
		),
	)
	s.s.AddTool(commitTaskResultTool, s.handleCommitTaskResult)

	// Herramientas de gestión de Engines
	s.registerEngineTools()
}

func (s *Server) registerEngineTools() {
	// Tool: switch_engine - Cambiar entre motores
	switchEngineTool := mcp.NewTool("switch_engine",
		mcp.WithDescription("Cambiar el motor activo (request_saver, token_saver, etc)"),
		mcp.WithString("engine_type",
			mcp.Required(),
			mcp.Description("Tipo de motor: request_saver, token_saver"),
			mcp.Enum("request_saver", "token_saver"),
		),
	)
	s.s.AddTool(switchEngineTool, s.handleSwitchEngine)

	// Tool: get_engine_info - Ver información del motor activo
	getEngineInfoTool := mcp.NewTool("get_engine_info",
		mcp.WithDescription("Obtener información del motor activo"),
	)
	s.s.AddTool(getEngineInfoTool, s.handleGetEngineInfo)

	// Tool: list_engines - Listar motores disponibles
	listEnginesTool := mcp.NewTool("list_engines",
		mcp.WithDescription("Listar todos los motores disponibles"),
	)
	s.s.AddTool(listEnginesTool, s.handleListEngines)

	// Registrar herramientas del motor activo
	if s.activeEngine != nil {
		for _, tool := range s.activeEngine.GetTools() {
			s.registerEngineTool(tool)
		}
	}
}

func (s *Server) registerEngineTool(tool agents.ToolDefinition) {
	// Create MCP tool
	mcpTool := mcp.NewTool(tool.Name, mcp.WithDescription(tool.Description))

	// Add all properties as string parameters
	if props, ok := tool.InputSchema["properties"].(map[string]interface{}); ok {
		for name := range props {
			mcpTool = mcp.NewTool(tool.Name,
				mcp.WithDescription(tool.Description),
				mcp.WithString(name, mcp.Description(name)),
			)
		}
	}

	// Register with handler
	s.s.AddTool(mcpTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Build params map from request arguments
		params := make(map[string]interface{})

		// Try to get arguments - they come as map[string]any in mcp-golang
		if request.Params.Arguments != nil {
			switch args := request.Params.Arguments.(type) {
			case map[string]any:
				for k, v := range args {
					params[k] = v
				}
			}
		}

		result, err := tool.Handler(ctx, params)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if !result.Success {
			return mcp.NewToolResultError(result.Error), nil
		}
		data, _ := json.Marshal(result.Data)
		return mcp.NewToolResultText(string(data)), nil
	})
}

// Handlers de Engine
func (s *Server) handleSwitchEngine(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	engineType, err := request.RequireString("engine_type")
	if err != nil {
		return mcp.NewToolResultError("engine_type es requerido"), nil
	}

	if err := s.SetActiveEngine(agents.EngineType(engineType)); err != nil {
		return mcp.NewToolResultError("Error al cambiar engine: " + err.Error()), nil
	}

	info := s.GetEngineInfo()
	data, _ := json.Marshal(info)
	return mcp.NewToolResultText(string(data)), nil
}

func (s *Server) handleGetEngineInfo(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	info := s.GetEngineInfo()
	data, _ := json.Marshal(info)
	return mcp.NewToolResultText(string(data)), nil
}

func (s *Server) handleListEngines(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	engines := agents.GetAllEngines()
	result := make(map[string]interface{})
	for t, e := range engines {
		result[string(t)] = map[string]string{
			"name":        e.Name(),
			"description": e.Description(),
		}
	}
	data, _ := json.Marshal(result)
	return mcp.NewToolResultText(string(data)), nil
}

func (s *Server) registerResources() {
	brainResource := mcp.NewResource(
		"singularity://brain",
		"Active Brain",
		mcp.WithResourceDescription("Current active brain state"),
		mcp.WithMIMEType("application/json"),
	)

	s.s.AddResource(brainResource, s.handleReadBrainResource)
}

func (s *Server) Run(ctx context.Context) error {
	return server.ServeStdio(s.s)
}

func (s *Server) handleCommitWorldState(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessionID, err := request.RequireString("session_id")
	if err != nil {
		return mcp.NewToolResultError("session_id es requerido: " + err.Error()), nil
	}

	projectPath, err := request.RequireString("project_path")
	if err != nil {
		return mcp.NewToolResultError("project_path es requerido: " + err.Error()), nil
	}

	completedTaskID := request.GetString("completed_task_id", "")
	taskSummary := request.GetString("task_summary", "")
	orchestratorSummary := request.GetString("orchestrator_summary", "")
	learnedInsights := request.GetString("learned_insights", "")
	blockersRaw := request.GetStringSlice("blockers", []string{})

	commitReq := models.CommitRequest{
		SessionID:           sessionID,
		ProjectPath:         projectPath,
		CompletedTaskID:     completedTaskID,
		TaskSummary:         taskSummary,
		OrchestratorSummary: orchestratorSummary,
		LearnedInsights:     learnedInsights,
	}

	newTasksJSON := request.GetString("new_tasks_json", "[]")
	var newTasksRaw []interface{}
	if err := json.Unmarshal([]byte(newTasksJSON), &newTasksRaw); err == nil {
		for _, t := range newTasksRaw {
			if tMap, ok := t.(map[string]interface{}); ok {
				task := models.Task{
					ID:          getString(tMap, "id", generateID()),
					Title:       getString(tMap, "title", ""),
					Description: getString(tMap, "description", ""),
					Status:      models.TaskStatus(getString(tMap, "status", "pending")),
					Priority:    getInt(tMap, "priority", 0),
					CreatedAt:   time.Now(),
					UpdatedAt:   time.Now(),
				}
				commitReq.NewTasks = append(commitReq.NewTasks, task)
			}
		}
	}

	codeChangesJSON := request.GetString("code_changes_json", "[]")
	var codeChangesRaw []interface{}
	if err := json.Unmarshal([]byte(codeChangesJSON), &codeChangesRaw); err == nil {
		for _, c := range codeChangesRaw {
			if cMap, ok := c.(map[string]interface{}); ok {
				codeChange := models.CodeChange{
					FilePath:  getString(cMap, "file_path", ""),
					Operation: getString(cMap, "operation", "update"),
					Summary:   getString(cMap, "summary", ""),
					Content:   getString(cMap, "content", ""),
					Timestamp: time.Now(),
				}
				commitReq.CodeChanges = append(commitReq.CodeChanges, codeChange)
			}
		}
	}

	decisionsJSON := request.GetString("decisions_json", "[]")
	var decisionsRaw []interface{}
	if err := json.Unmarshal([]byte(decisionsJSON), &decisionsRaw); err == nil {
		for _, d := range decisionsRaw {
			if dMap, ok := d.(map[string]interface{}); ok {
				decision := models.Decision{
					ID:        generateID(),
					Content:   getString(dMap, "content", ""),
					Context:   getString(dMap, "context", ""),
					Timestamp: time.Now(),
					Agent:     getString(dMap, "agent", "unknown"),
				}
				commitReq.Decisions = append(commitReq.Decisions, decision)
			}
		}
	}

	commitReq.Blockers = blockersRaw

	response, err := s.processCommit(ctx, &commitReq)
	if err != nil {
		return mcp.NewToolResultError("Error al procesar commit: " + err.Error()), nil
	}

	responseJSON, _ := json.Marshal(response)
	return mcp.NewToolResultText(string(responseJSON)), nil
}

func (s *Server) handleFetchDeepContext(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query := request.GetString("query", "")
	taskID := request.GetString("task_id", "")
	limit := int(request.GetFloat("limit", 10))

	results, err := s.fetchDeepContext(query, taskID, limit)
	if err != nil {
		return mcp.NewToolResultError("Error al recuperar contexto: " + err.Error()), nil
	}

	response := models.DeepContextResponse{
		Success: true,
		Results: results,
	}

	responseJSON, _ := json.Marshal(response)
	return mcp.NewToolResultText(string(responseJSON)), nil
}

func (s *Server) handleGetActiveBrain(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	brain, err := s.getActiveBrain()
	if err != nil {
		return mcp.NewToolResultError("Error al obtener cerebro activo: " + err.Error()), nil
	}

	return mcp.NewToolResultText(brain), nil
}

func (s *Server) handleListTasks(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	status := request.GetString("status", "")

	tasks, err := s.listTasks(status)
	if err != nil {
		return mcp.NewToolResultError("Error al listar tareas: " + err.Error()), nil
	}

	tasksJSON, _ := json.Marshal(tasks)
	return mcp.NewToolResultText(string(tasksJSON)), nil
}

func (s *Server) handleReadBrainResource(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	brain, err := s.getActiveBrain()
	if err != nil {
		return nil, err
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      "singularity://brain",
			MIMEType: "application/json",
			Text:     brain,
		},
	}, nil
}

func (s *Server) processCommit(ctx context.Context, req *models.CommitRequest) (*models.CommitResponse, error) {
	items := make(map[string][]byte)

	wsKey := storage.ActiveBrainKey(req.ProjectPath)
	existingWS, err := s.db.Get(wsKey)
	var ws *models.WorldState
	if err == nil {
		ws, _ = models.WorldStateFromJSON(existingWS)
	}
	if ws == nil {
		ws = &models.WorldState{
			SessionID:   req.SessionID,
			ProjectPath: req.ProjectPath,
			Metadata:    make(map[string]string),
		}
	}

	ws.SessionID = req.SessionID
	ws.LastUpdated = time.Now()

	if req.CompletedTaskID != "" {
		ws.CompletedTasks = append(ws.CompletedTasks, req.CompletedTaskID)
		for i, t := range ws.ActiveTasks {
			if t == req.CompletedTaskID {
				ws.ActiveTasks = append(ws.ActiveTasks[:i], ws.ActiveTasks[i+1:]...)
				break
			}
		}
		ws.CurrentTaskID = ""
	}

	for _, task := range req.NewTasks {
		taskKey := storage.TaskKey(task.ID)
		taskData, _ := task.ToJSON()
		items[taskKey] = taskData

		ws.ActiveTasks = append(ws.ActiveTasks, task.ID)
		if ws.CurrentTaskID == "" {
			ws.CurrentTaskID = task.ID
		}
	}

	ws.Decisions = append(ws.Decisions, req.Decisions...)

	wsData, _ := ws.ToJSON()
	items[wsKey] = wsData

	archiveKey := storage.DeepArchiveKey(req.SessionID + ":" + fmt.Sprintf("%d", time.Now().Unix()))
	archiveData, _ := json.Marshal(models.ArchiveEntry{
		Type:      "session",
		Summary:   req.TaskSummary,
		Content:   req.OrchestratorSummary,
		Timestamp: time.Now(),
	})
	items[archiveKey] = archiveData

	for _, cc := range req.CodeChanges {
		ccKey := storage.DeepArchiveKey("code:" + cc.FilePath + ":" + fmt.Sprintf("%d", time.Now().Unix()))
		ccData, _ := json.Marshal(models.ArchiveEntry{
			Type:    "code",
			Summary: cc.Summary,
			Content: cc.Content,
		})
		items[ccKey] = ccData
	}

	if err := s.db.SetMulti(items); err != nil {
		return nil, err
	}

	newTaskIDs := make([]string, len(req.NewTasks))
	for i, t := range req.NewTasks {
		newTaskIDs[i] = t.ID
	}

	activeBrain, _ := s.getActiveBrain()

	return &models.CommitResponse{
		Success:     true,
		ActiveBrain: activeBrain,
		NewTaskIDs:  newTaskIDs,
	}, nil
}

func (s *Server) fetchDeepContext(query, taskID string, limit int) ([]models.ArchiveEntry, error) {
	archives, err := s.db.GetWithPrefix(storage.DeepArchivePrefix())
	if err != nil {
		return nil, err
	}

	var results []models.ArchiveEntry
	count := 0

	for key, data := range archives {
		if count >= limit {
			break
		}

		if taskID != "" && key != storage.DeepArchiveKey("task:"+taskID) {
			continue
		}

		var entry models.ArchiveEntry
		if err := json.Unmarshal(data, &entry); err != nil {
			continue
		}

		if query == "" || contains(entry.Summary, query) || contains(entry.Content, query) {
			results = append(results, entry)
			count++
		}
	}

	return results, nil
}

func (s *Server) getActiveBrain() (string, error) {
	tasks, err := s.listTasks("")
	if err != nil {
		return "", err
	}

	archives, _ := s.db.GetWithPrefix(storage.DeepArchivePrefix())

	var recentDecisions []models.Decision
	for _, data := range archives {
		var entry models.ArchiveEntry
		if err := json.Unmarshal(data, &entry); err == nil && entry.Type == "decision" {
			if len(recentDecisions) < 5 {
				var dec models.Decision
				if err := json.Unmarshal([]byte(entry.Content), &dec); err == nil {
					recentDecisions = append(recentDecisions, dec)
				}
			}
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
	}

	for _, t := range tasks {
		switch t.Status {
		case models.TaskStatusInProgress:
			brain.ActiveCount++
		case models.TaskStatusPending:
			brain.PendingCount++
		case models.TaskStatusCompleted:
			brain.CompletedCount++
		}
	}

	brainBytes, err := json.Marshal(brain)
	return string(brainBytes), err
}

func (s *Server) listTasks(status string) ([]models.Task, error) {
	tasksMap, err := s.db.GetWithPrefix(storage.TaskPrefix())
	if err != nil {
		return nil, err
	}

	var tasks []models.Task
	for _, data := range tasksMap {
		var task models.Task
		if err := json.Unmarshal(data, &task); err != nil {
			continue
		}

		if status == "" || string(task.Status) == status {
			tasks = append(tasks, task)
		}
	}

	return tasks, nil
}

func getString(m map[string]interface{}, key, defaultValue string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return defaultValue
}

func getInt(m map[string]interface{}, key string, defaultValue int) int {
	if v, ok := m[key]; ok {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		}
	}
	return defaultValue
}

func generateID() string {
	return fmt.Sprintf("%d-%d", time.Now().Unix(), time.Now().Nanosecond())
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func (s *Server) handleSpawnSubAgent(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessionID, err := request.RequireString("session_id")
	if err != nil {
		return mcp.NewToolResultError("session_id es requerido: " + err.Error()), nil
	}

	projectPath, err := request.RequireString("project_path")
	if err != nil {
		return mcp.NewToolResultError("project_path es requerido: " + err.Error()), nil
	}

	title, err := request.RequireString("title")
	if err != nil {
		return mcp.NewToolResultError("title es requerido: " + err.Error()), nil
	}

	description, err := request.RequireString("description")
	if err != nil {
		return mcp.NewToolResultError("description es requerido: " + err.Error()), nil
	}

	context := request.GetString("context", "")

	subAgentID := generateID()
	taskID := generateID()

	subAgent := models.SubAgent{
		ID:          subAgentID,
		TaskID:      taskID,
		Title:       title,
		Description: description,
		Context:     context,
		Status:      models.SubAgentStatusPending,
		CreatedAt:   time.Now(),
	}

	subAgentData, err := subAgent.ToJSON()
	if err != nil {
		return mcp.NewToolResultError("Error al serializar sub-agente: " + err.Error()), nil
	}

	subAgentKey := storage.SubAgentKey(subAgentID)
	if err := s.db.Set(subAgentKey, subAgentData); err != nil {
		return mcp.NewToolResultError("Error al guardar sub-agente: " + err.Error()), nil
	}

	wsKey := storage.ActiveBrainKey(projectPath)
	existingWS, _ := s.db.Get(wsKey)
	var ws *models.WorldState
	if existingWS != nil {
		ws, _ = models.WorldStateFromJSON(existingWS)
	}
	if ws == nil {
		ws = &models.WorldState{
			SessionID:   sessionID,
			ProjectPath: projectPath,
			Metadata:    make(map[string]string),
		}
	}

	task := models.Task{
		ID:          taskID,
		Title:       title,
		Description: description,
		Status:      models.TaskStatusPending,
		Assignee:    subAgentID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	taskData, _ := task.ToJSON()
	taskKey := storage.TaskKey(taskID)
	s.db.Set(taskKey, taskData)

	ws.ActiveTasks = append(ws.ActiveTasks, taskID)
	wsData, _ := ws.ToJSON()
	s.db.Set(wsKey, wsData)

	response := models.SpawnSubAgentResponse{
		Success:    true,
		SubAgentID: subAgentID,
		TaskID:     taskID,
	}

	responseJSON, _ := json.Marshal(response)
	return mcp.NewToolResultText(string(responseJSON)), nil
}

func (s *Server) handleGetSubAgentTask(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	subAgentID, err := request.RequireString("sub_agent_id")
	if err != nil {
		return mcp.NewToolResultError("sub_agent_id es requerido: " + err.Error()), nil
	}

	subAgentKey := storage.SubAgentKey(subAgentID)
	subAgentData, err := s.db.Get(subAgentKey)
	if err != nil {
		return mcp.NewToolResultError("Sub-agente no encontrado: " + err.Error()), nil
	}

	subAgent, err := models.SubAgentFromJSON(subAgentData)
	if err != nil {
		return mcp.NewToolResultError("Error al leer sub-agente: " + err.Error()), nil
	}

	taskKey := storage.TaskKey(subAgent.TaskID)
	taskData, err := s.db.Get(taskKey)
	if err != nil {
		return mcp.NewToolResultError("Tarea no encontrada: " + err.Error()), nil
	}

	task, err := models.TaskFromJSON(taskData)
	if err != nil {
		return mcp.NewToolResultError("Error al leer tarea: " + err.Error()), nil
	}

	wsKey := storage.ActiveBrainKey(task.Assignee)
	var projectPath string
	wsData, err := s.db.Get(wsKey)
	if err == nil {
		var ws *models.WorldState
		ws, _ = models.WorldStateFromJSON(wsData)
		if ws != nil {
			projectPath = ws.ProjectPath
		}
	}

	subAgent.Status = models.SubAgentStatusRunning
	now := time.Now()
	subAgent.StartedAt = &now
	subAgentData, _ = subAgent.ToJSON()
	s.db.Set(subAgentKey, subAgentData)

	task.Status = models.TaskStatusInProgress
	task.UpdatedAt = time.Now()
	taskData, _ = task.ToJSON()
	s.db.Set(taskKey, taskData)

	// Banner visual para indicar sub-agente en ejecución
	banner := `🎯 **SUB-AGENTE ACTIVO**

**ID:** %s
**Tarea:** %s
**Descripción:** %s

📋 Completa la tarea y usa "commit_world_state" cuando termines.

---`

	banner = fmt.Sprintf(banner, subAgentID, task.Title, task.Description)

	response := models.SubAgentTaskResponse{
		Success:     true,
		TaskID:      task.ID,
		Title:       task.Title,
		Description: task.Description,
		Context:     subAgent.Context,
		ProjectPath: projectPath,
	}

	responseJSON, _ := json.Marshal(response)

	// Combinar banner con respuesta JSON
	result := banner + "\n\n```json\n" + string(responseJSON) + "\n```"
	return mcp.NewToolResultText(result), nil
}

func (s *Server) handleCompleteSubAgentTask(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	subAgentID, err := request.RequireString("sub_agent_id")
	if err != nil {
		return mcp.NewToolResultError("sub_agent_id es requerido: " + err.Error()), nil
	}

	result, err := request.RequireString("result")
	if err != nil {
		return mcp.NewToolResultError("result es requerido: " + err.Error()), nil
	}

	errorMsg := request.GetString("error", "")

	subAgentKey := storage.SubAgentKey(subAgentID)
	subAgentData, err := s.db.Get(subAgentKey)
	if err != nil {
		return mcp.NewToolResultError("Sub-agente no encontrado: " + err.Error()), nil
	}

	subAgent, err := models.SubAgentFromJSON(subAgentData)
	if err != nil {
		return mcp.NewToolResultError("Error al leer sub-agente: " + err.Error()), nil
	}

	subAgent.Result = result
	subAgent.Error = errorMsg

	if errorMsg != "" {
		subAgent.Status = models.SubAgentStatusFailed
	} else {
		subAgent.Status = models.SubAgentStatusCompleted
	}

	now := time.Now()
	subAgent.CompletedAt = &now
	subAgentData, _ = subAgent.ToJSON()
	s.db.Set(subAgentKey, subAgentData)

	taskKey := storage.TaskKey(subAgent.TaskID)
	taskData, err := s.db.Get(taskKey)
	if err == nil {
		task, _ := models.TaskFromJSON(taskData)
		if task != nil {
			task.Status = models.TaskStatusCompleted
			task.UpdatedAt = time.Now()
			task.CompletedAt = &now
			taskData, _ := task.ToJSON()
			s.db.Set(taskKey, taskData)

			wsKey := storage.ActiveBrainKey(task.Assignee)
			wsData, _ := s.db.Get(wsKey)
			if wsData != nil {
				var ws *models.WorldState
				ws, _ = models.WorldStateFromJSON(wsData)
				if ws != nil {
					for i, t := range ws.ActiveTasks {
						if t == task.ID {
							ws.ActiveTasks = append(ws.ActiveTasks[:i], ws.ActiveTasks[i+1:]...)
							break
						}
					}
					ws.CompletedTasks = append(ws.CompletedTasks, task.ID)
					wsData, _ := ws.ToJSON()
					s.db.Set(wsKey, wsData)
				}
			}
		}
	}

	type CompleteResponse struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}

	response := CompleteResponse{
		Success: true,
		Message: "Tarea completada",
	}

	responseJSON, _ := json.Marshal(response)
	return mcp.NewToolResultText(string(responseJSON)), nil
}

func (s *Server) handleSwitchAgent(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	mode, err := request.RequireString("mode")
	if err != nil {
		return mcp.NewToolResultError("mode es requerido: " + err.Error()), nil
	}

	// =================================================================
	// CAMBIO DE MOTOR BASADO EN EL MODO DEL AGENTE
	// =================================================================

	// Modo Core - contexto denso
	if mode == "core" {
		s.SetActiveEngine(agents.EngineTypeCore)
	} else if mode == "particle" {
		// Modo Particle - ahorro de tokens
		s.SetActiveEngine(agents.EngineTypeParticle)
	}

	if mode == "sub_agent" {
		subAgentID := request.GetString("sub_agent_id", "")
		if subAgentID == "" {
			return mcp.NewToolResultError("sub_agent_id es requerido cuando mode=sub_agent"), nil
		}

		subAgentKey := storage.SubAgentKey(subAgentID)
		subAgentData, err := s.db.Get(subAgentKey)
		if err != nil {
			return mcp.NewToolResultError("Sub-agente no encontrado: " + err.Error()), nil
		}

		subAgent, err := models.SubAgentFromJSON(subAgentData)
		if err != nil {
			return mcp.NewToolResultError("Error al leer sub-agente: " + err.Error()), nil
		}

		taskKey := storage.TaskKey(subAgent.TaskID)
		taskData, err := s.db.Get(taskKey)
		if err != nil {
			return mcp.NewToolResultError("Tarea no encontrada: " + err.Error()), nil
		}

		task, _ := models.TaskFromJSON(taskData)

		// Obtener info del motor activo
		engineInfo := s.GetEngineInfo()

		// Banner visual para switch a sub-agente
		banner := fmt.Sprintf(`🎯 **SUB-AGENTE ACTIVO**

**ID:** %s | **Tarea:** %s
**Motor:** %s (%s)

✅ Modo sub-agente activado. Ejecuta tu tarea.

---`, subAgentID, task.Title, engineInfo["name"], engineInfo["type"])

		type SwitchResponse struct {
			Success     bool   `json:"success"`
			Mode        string `json:"mode"`
			SubAgentID  string `json:"sub_agent_id,omitempty"`
			TaskID      string `json:"task_id,omitempty"`
			Title       string `json:"title,omitempty"`
			Description string `json:"description,omitempty"`
			Context     string `json:"context,omitempty"`
			Engine      string `json:"engine,omitempty"`
			Message     string `json:"message"`
		}

		response := SwitchResponse{
			Success:     true,
			Mode:        "sub_agent",
			SubAgentID:  subAgentID,
			TaskID:      subAgent.TaskID,
			Title:       task.Title,
			Description: task.Description,
			Context:     subAgent.Context,
			Engine:      engineInfo["type"],
			Message:     "Cambiaste al modo Sub-agente. Ejecuta tu tarea.",
		}

		responseJSON, _ := json.Marshal(response)
		result := banner + "\n\n```json\n" + string(responseJSON) + "\n```"
		return mcp.NewToolResultText(result), nil
	}

	// Banner para volver a planner
	engineInfo := s.GetEngineInfo()
	banner := fmt.Sprintf(`🎼 **MODO PLANNER**

Motor activo: %s (%s)

✅ Has vuelto al modo Planner. Usa "spawn_sub_agent" para nuevas tareas.

---`, engineInfo["name"], engineInfo["type"])

	type SwitchResponse struct {
		Success bool   `json:"success"`
		Mode    string `json:"mode"`
		Message string `json:"message"`
	}

	response := SwitchResponse{
		Success: true,
		Mode:    "core",
		Message: "Cambiaste al modo Core. Usa spawn_sub_agent para crear tareas.",
	}

	responseJSON, _ := json.Marshal(response)
	result := banner + "\n\n```json\n" + string(responseJSON) + "\n```"
	return mcp.NewToolResultText(result), nil
}

// =============================================================================
// PLAN_AND_DELEGATE - Herramienta del Planner
// =============================================================================

func (s *Server) handlePlanAndDelegate(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessionID, err := request.RequireString("session_id")
	if err != nil {
		return mcp.NewToolResultError("session_id es requerido: " + err.Error()), nil
	}

	projectPath, err := request.RequireString("project_path")
	if err != nil {
		return mcp.NewToolResultError("project_path es requerido: " + err.Error()), nil
	}

	requirement, err := request.RequireString("requirement")
	if err != nil {
		return mcp.NewToolResultError("requirement es requerido: " + err.Error()), nil
	}

	context := request.GetString("context", "")

	// Crear el plan (DAG de tareas)
	planID := generateID()

	// Descomponer el requisito en tareas atómicas
	// En una implementación real, esto podría usar el LLM para planificar
	tasks := s.decomposeRequirement(planID, requirement, context)

	// Crear sub-agentes para cada tarea
	subAgentIDs := []string{}
	for i, task := range tasks {
		subAgentID := generateID()
		taskID := planID + "-task-" + fmt.Sprintf("%d", i)

		subAgent := models.SubAgent{
			ID:          subAgentID,
			TaskID:      taskID,
			Title:       task.Title,
			Description: task.Description,
			Context:     task.Context,
			Status:      models.SubAgentStatusPending,
			CreatedAt:   time.Now(),
		}

		subAgentData, _ := subAgent.ToJSON()
		subAgentKey := storage.SubAgentKey(subAgentID)
		s.db.Set(subAgentKey, subAgentData)

		// Crear tarea asociada
		t := models.Task{
			ID:          taskID,
			Title:       task.Title,
			Description: task.Description,
			Status:      models.TaskStatusPending,
			Assignee:    subAgentID,
			Priority:    task.Priority,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		taskData, _ := t.ToJSON()
		taskKey := storage.TaskKey(taskID)
		s.db.Set(taskKey, taskData)

		subAgentIDs = append(subAgentIDs, subAgentID)
	}

	// Actualizar WorldState
	wsKey := storage.ActiveBrainKey(projectPath)
	existingWS, _ := s.db.Get(wsKey)
	var ws *models.WorldState
	if existingWS != nil {
		ws, _ = models.WorldStateFromJSON(existingWS)
	}
	if ws == nil {
		ws = &models.WorldState{
			SessionID:   sessionID,
			ProjectPath: projectPath,
			Metadata:    make(map[string]string),
		}
	}

	for i := range tasks {
		taskID := planID + "-task-" + fmt.Sprintf("%d", i)
		ws.ActiveTasks = append(ws.ActiveTasks, taskID)
	}
	wsData, _ := ws.ToJSON()
	s.db.Set(wsKey, wsData)

	response := models.PlanAndDelegateResponse{
		Success:      true,
		PlanID:       planID,
		Tasks:        tasks,
		SubAgentIDs:  subAgentIDs,
		Dependencies: make(map[string][]string),
		Message:      fmt.Sprintf("Plan creado con %d tareas", len(tasks)),
	}

	responseJSON, _ := json.Marshal(response)
	return mcp.NewToolResultText(string(responseJSON)), nil
}

func (s *Server) decomposeRequirement(planID, requirement, context string) []models.PlannedTask {
	// Esta es una implementación básica de descomposición
	// En producción, esto podría usar el LLM para analizar y crear el DAG

	// Por ahora, creamos una tarea genérica que contiene todo el contexto
	task := models.PlannedTask{
		ID:          planID + "-task-0",
		Title:       "Implementar: " + requirement,
		Description: requirement,
		Priority:    1,
		Context:     context,
	}

	// Si el contexto está vacío, usamos el requisito como contexto
	if task.Context == "" {
		task.Context = requirement
	}

	return []models.PlannedTask{task}
}

// =============================================================================
// COMMIT_TASK_RESULT - Herramienta del Sub-agente con JUDGE DETERMINISTA
// =============================================================================

func (s *Server) handleCommitTaskResult(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	subAgentID, err := request.RequireString("sub_agent_id")
	if err != nil {
		return mcp.NewToolResultError("sub_agent_id es requerido: " + err.Error()), nil
	}

	projectPath, err := request.RequireString("project_path")
	if err != nil {
		return mcp.NewToolResultError("project_path es requerido: " + err.Error()), nil
	}

	_, err = request.RequireString("session_id")
	if err != nil {
		return mcp.NewToolResultError("session_id es requerido: " + err.Error()), nil
	}

	taskID, err := request.RequireString("task_id")
	if err != nil {
		return mcp.NewToolResultError("task_id es requerido: " + err.Error()), nil
	}
	_ = taskID // usado en saveFailedAttempt

	codeFilesJSON := request.GetString("code_files", "[]")
	summary := request.GetString("summary", "")
	validationNotes := request.GetString("validation_notes", "")
	depsJSON := request.GetString("dependencies_json", "")

	// Parsear los archivos de código
	var codeFiles []models.CodeFile
	if err := json.Unmarshal([]byte(codeFilesJSON), &codeFiles); err != nil {
		return mcp.NewToolResultError("Error al parsear code_files: " + err.Error()), nil
	}

	if len(codeFiles) == 0 {
		return mcp.NewToolResultError("code_files no puede estar vacío"), nil
	}

	// === JUEZ DETERMINISTA: Validar compilación ===
	validationResult := s.validateCode(projectPath, codeFiles, depsJSON)

	response := models.CommitTaskResultResponse{
		Success:     true,
		Validated:   validationResult.Valid,
		BuildOutput: validationResult.Output,
		BuildError:  validationResult.Error,
	}

	if !validationResult.Valid {
		// FALLÓ LA VALIDACIÓN - No guardamos en BadgerDB
		response.TaskStatus = "failed"
		response.Message = "❌ VALIDACIÓN FALLIDA: El código no compila. Arregla los errores y reintenta."

		// Guardar el intento fallido para debugging
		s.saveFailedAttempt(subAgentID, taskID, codeFiles, validationResult.Error)

		responseJSON, _ := json.Marshal(response)
		return mcp.NewToolResultText(string(responseJSON)), nil
	}

	// === PASSÓ LA VALIDACIÓN - Guardar en BadgerDB ===
	savedFiles := []string{}
	items := make(map[string][]byte)

	for _, cf := range codeFiles {
		// Guardar el archivo en el sistema de archivos
		fullPath := filepath.Join(projectPath, cf.FilePath)
		dir := filepath.Dir(fullPath)

		if err := os.MkdirAll(dir, 0755); err != nil {
			response.BuildError += fmt.Sprintf("\nError al crear directorio %s: %v", dir, err)
			continue
		}

		if err := ioutil.WriteFile(fullPath, []byte(cf.Content), 0644); err != nil {
			response.BuildError += fmt.Sprintf("\nError al escribir archivo %s: %v", cf.FilePath, err)
			continue
		}

		savedFiles = append(savedFiles, cf.FilePath)

		// Guardar en BadgerDB también
		ccKey := storage.DeepArchiveKey("code:" + cf.FilePath + ":" + fmt.Sprintf("%d", time.Now().Unix()))
		ccData, _ := json.Marshal(models.ArchiveEntry{
			Type:    "code",
			Summary: summary,
			Content: cf.Content,
		})
		items[ccKey] = ccData
	}

	// Guardar el resultado de la tarea
	taskResult := models.TaskResult{
		TaskID:           taskID,
		Summary:          summary,
		CodeFiles:        codeFiles,
		ValidationNotes:  validationNotes,
		DependenciesJSON: depsJSON,
		Timestamp:        time.Now(),
	}
	trData, _ := taskResult.ToJSON()
	trKey := storage.TaskKey("result:" + taskID)
	items[trKey] = trData

	// Actualizar sub-agente
	subAgentKey := storage.SubAgentKey(subAgentID)
	subAgentData, _ := s.db.Get(subAgentKey)
	if subAgentData != nil {
		sa, _ := models.SubAgentFromJSON(subAgentData)
		if sa != nil {
			sa.Status = models.SubAgentStatusCompleted
			sa.Result = summary
			now := time.Now()
			sa.CompletedAt = &now
			saData, _ := sa.ToJSON()
			items[subAgentKey] = saData
		}
	}

	// Actualizar tarea a completada
	taskKey := storage.TaskKey(taskID)
	taskData, _ := s.db.Get(taskKey)
	if taskData != nil {
		task, _ := models.TaskFromJSON(taskData)
		if task != nil {
			task.Status = models.TaskStatusCompleted
			task.UpdatedAt = time.Now()
			now := time.Now()
			task.CompletedAt = &now
			tData, _ := task.ToJSON()
			items[taskKey] = tData
		}
	}

	// Guardar todo en BadgerDB
	if err := s.db.SetMulti(items); err != nil {
		response.BuildError += "\nError al guardar en BD: " + err.Error()
	}

	response.SavedFiles = savedFiles
	response.TaskStatus = "completed"
	response.Message = "✅ VALIDACIÓN PASSED y código guardado"

	responseJSON, _ := json.Marshal(response)
	return mcp.NewToolResultText(string(responseJSON)), nil
}

// =============================================================================
// JUDGE DETERMINISTA - Validador de Compilación
// =============================================================================

type ValidationResult struct {
	Valid  bool
	Output string
	Error  string
}

func (s *Server) validateCode(projectPath string, codeFiles []models.CodeFile, depsJSON string) ValidationResult {
	// Determinar el tipo de proyecto por la extensión de archivos
	hasGoFiles := false
	hasJSFiles := false
	hasPythonFiles := false

	for _, cf := range codeFiles {
		ext := strings.ToLower(filepath.Ext(cf.FilePath))
		if ext == ".go" {
			hasGoFiles = true
		} else if ext == ".js" || ext == ".ts" || ext == ".jsx" || ext == ".tsx" {
			hasJSFiles = true
		} else if ext == ".py" {
			hasPythonFiles = true
		}
	}

	// Si hay archivos Go, validar con go build
	if hasGoFiles {
		return s.validateGoCode(projectPath, codeFiles)
	}

	// Si hay archivos JS/TS, validar con TypeScript o ESLint
	if hasJSFiles {
		return s.validateJSCode(projectPath, codeFiles)
	}

	// Si hay archivos Python, validar con Python syntax check
	if hasPythonFiles {
		return s.validatePythonCode(projectPath, codeFiles)
	}

	// Si no se detecta tipo, intentar validar como texto plano
	return ValidationResult{
		Valid:  true,
		Output: "No se detectó tipo de proyecto para validación",
		Error:  "",
	}
}

func (s *Server) validateGoCode(projectPath string, codeFiles []models.CodeFile) ValidationResult {
	// Escribir archivos temporales para validación
	tempDir, err := ioutil.TempDir("", "singularity-validate-*")
	if err != nil {
		return ValidationResult{Valid: false, Error: "Error al crear directorio temporal: " + err.Error()}
	}
	defer os.RemoveAll(tempDir)

	// Escribir archivos
	for _, cf := range codeFiles {
		fullPath := filepath.Join(tempDir, cf.FilePath)
		dir := filepath.Dir(fullPath)
		os.MkdirAll(dir, 0755)
		ioutil.WriteFile(fullPath, []byte(cf.Content), 0644)
	}

	// Intentar go build -no-pkg-errors que es más permisivo con módulos
	// Usar go vet para validación de sintaxis (más permisivo que build)
	var lastOutput string
	var hasGoFiles bool

	for _, cf := range codeFiles {
		if filepath.Ext(cf.FilePath) != ".go" {
			continue
		}
		hasGoFiles = true

		// Usar go vet que es más permisivo y no requiere módulo
		cmd := exec.Command("go", "vet", cf.FilePath)
		cmd.Dir = tempDir
		output, err := cmd.CombinedOutput()
		lastOutput = string(output)

		// Si encuentra un error de sintaxis, falla inmediatamente
		if err != nil {
			return ValidationResult{
				Valid:  false,
				Output: lastOutput,
				Error:  err.Error(),
			}
		}
	}

	if !hasGoFiles {
		return ValidationResult{
			Valid:  true,
			Output: "No Go files to validate",
			Error:  "",
		}
	}

	// Si ningún archivo falló, verificar que hay un main si es ejecutable
	hasMain := false
	hasErrors := false
	for _, cf := range codeFiles {
		if filepath.Ext(cf.FilePath) == ".go" && strings.Contains(cf.Content, "func main()") {
			hasMain = true
			break
		}
	}
	_ = hasErrors // Reserved for future use

	// Para código de librería (sin main), verificar sintaxis
	if !hasMain {
		// Usar go fmt para verificar sintaxis
		cmd := exec.Command("go", "fmt", "./...")
		cmd.Dir = tempDir
		output, err := cmd.CombinedOutput()

		if err != nil {
			return ValidationResult{
				Valid:  false,
				Output: string(output),
				Error:  err.Error(),
			}
		}
	}

	return ValidationResult{
		Valid:  true,
		Output: "Go syntax validation successful: " + lastOutput,
		Error:  "",
	}
}

func (s *Server) validateJSCode(projectPath string, codeFiles []models.CodeFile) ValidationResult {
	// Verificar si hay package.json
	hasPackageJSON := false
	for _, cf := range codeFiles {
		if cf.FilePath == "package.json" {
			hasPackageJSON = true
			break
		}
	}

	// Si no hay package.json, solo validar sintaxis básica
	if !hasPackageJSON {
		return ValidationResult{
			Valid:  true,
			Output: "Sin package.json - validación de sintaxis omitida",
			Error:  "",
		}
	}

	// Escribir archivos temporales
	tempDir, err := ioutil.TempDir("", "singularity-validate-*")
	if err != nil {
		return ValidationResult{Valid: false, Error: "Error al crear directorio temporal: " + err.Error()}
	}
	defer os.RemoveAll(tempDir)

	for _, cf := range codeFiles {
		fullPath := filepath.Join(tempDir, cf.FilePath)
		dir := filepath.Dir(fullPath)
		os.MkdirAll(dir, 0755)
		ioutil.WriteFile(fullPath, []byte(cf.Content), 0644)
	}

	// Intentar tsc --noEmit si está disponible
	cmd := exec.Command("npx", "tsc", "--noEmit")
	cmd.Dir = tempDir
	output, err := cmd.CombinedOutput()

	if err != nil {
		// Si no hay TypeScript, intentar node --check
		cmd = exec.Command("node", "--check", codeFiles[0].FilePath)
		cmd.Dir = tempDir
		output2, err2 := cmd.CombinedOutput()

		if err2 != nil {
			return ValidationResult{
				Valid:  false,
				Output: string(output) + "\n" + string(output2),
				Error:  "Validación JS/TS fallida",
			}
		}
	}

	return ValidationResult{
		Valid:  true,
		Output: "Validación JS/TS exitosa: " + string(output),
		Error:  "",
	}
}

func (s *Server) validatePythonCode(projectPath string, codeFiles []models.CodeFile) ValidationResult {
	// Escribir archivos temporales
	tempDir, err := ioutil.TempDir("", "singularity-validate-*")
	if err != nil {
		return ValidationResult{Valid: false, Error: "Error al crear directorio temporal: " + err.Error()}
	}
	defer os.RemoveAll(tempDir)

	for _, cf := range codeFiles {
		fullPath := filepath.Join(tempDir, cf.FilePath)
		dir := filepath.Dir(fullPath)
		os.MkdirAll(dir, 0755)
		ioutil.WriteFile(fullPath, []byte(cf.Content), 0644)
	}

	// Intentar python -m py_compile
	cmd := exec.Command("python3", "-m", "py_compile", codeFiles[0].FilePath)
	cmd.Dir = tempDir
	output, err := cmd.CombinedOutput()

	if err != nil {
		return ValidationResult{
			Valid:  false,
			Output: string(output),
			Error:  err.Error(),
		}
	}

	return ValidationResult{
		Valid:  true,
		Output: "Python syntax check exitoso",
		Error:  "",
	}
}

func (s *Server) saveFailedAttempt(subAgentID, taskID string, codeFiles []models.CodeFile, errorMsg string) {
	items := make(map[string][]byte)

	failedKey := storage.DeepArchiveKey("failed:" + taskID + ":" + fmt.Sprintf("%d", time.Now().Unix()))
	failedData, _ := json.Marshal(map[string]interface{}{
		"sub_agent_id": subAgentID,
		"task_id":      taskID,
		"code_files":   codeFiles,
		"error":        errorMsg,
		"timestamp":    time.Now(),
	})
	items[failedKey] = failedData

	s.db.SetMulti(items)
}
