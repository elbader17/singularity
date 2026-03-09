package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"singularity/internal/models"
	"singularity/internal/storage"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type Server struct {
	db *storage.BadgerDB
	s  *server.MCPServer
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

	srv.registerTools()
	srv.registerResources()

	return srv
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
		mcp.WithDescription("Cambiar entre modo Orquestador y Sub-agente. "+
			"Útil para cambiar de rol sin cerrar OpenCode."),
		mcp.WithString("mode",
			mcp.Required(),
			mcp.Description("Modo: 'orchestrator' o 'sub_agent'"),
			mcp.Enum("orchestrator", "sub_agent"),
		),
		mcp.WithString("sub_agent_id",
			mcp.Description("ID del sub-agente (requerido si mode=sub_agent)"),
		),
	), s.handleSwitchAgent)
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

	response := models.SubAgentTaskResponse{
		Success:     true,
		TaskID:      task.ID,
		Title:       task.Title,
		Description: task.Description,
		Context:     subAgent.Context,
		ProjectPath: projectPath,
	}

	responseJSON, _ := json.Marshal(response)
	return mcp.NewToolResultText(string(responseJSON)), nil
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

		type SwitchResponse struct {
			Success     bool   `json:"success"`
			Mode        string `json:"mode"`
			SubAgentID  string `json:"sub_agent_id,omitempty"`
			TaskID      string `json:"task_id,omitempty"`
			Title       string `json:"title,omitempty"`
			Description string `json:"description,omitempty"`
			Context     string `json:"context,omitempty"`
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
			Message:     "Cambiaste al modo Sub-agente. Ejecuta tu tarea.",
		}

		responseJSON, _ := json.Marshal(response)
		return mcp.NewToolResultText(string(responseJSON)), nil
	}

	type SwitchResponse struct {
		Success bool   `json:"success"`
		Mode    string `json:"mode"`
		Message string `json:"message"`
	}

	response := SwitchResponse{
		Success: true,
		Mode:    "orchestrator",
		Message: "Cambiaste al modo Orquestador. Usa spawn_sub_agent para crear tareas.",
	}

	responseJSON, _ := json.Marshal(response)
	return mcp.NewToolResultText(string(responseJSON)), nil
}
