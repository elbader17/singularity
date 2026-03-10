package models

import (
	"encoding/json"
	"testing"
	"time"
)

// =============================================================================
// TESTS PARA CodeFile
// =============================================================================

func TestCodeFile_JSONMarshaling(t *testing.T) {
	cf := CodeFile{
		FilePath: "main.go",
		Content:  "package main",
		Language: "go",
	}

	data, err := json.Marshal(cf)
	if err != nil {
		t.Fatalf("Failed to marshal CodeFile: %v", err)
	}

	var cf2 CodeFile
	if err := json.Unmarshal(data, &cf2); err != nil {
		t.Fatalf("Failed to unmarshal CodeFile: %v", err)
	}

	if cf2.FilePath != cf.FilePath {
		t.Errorf("Expected FilePath %s, got %s", cf.FilePath, cf2.FilePath)
	}
	if cf2.Content != cf.Content {
		t.Errorf("Expected Content %s, got %s", cf.Content, cf2.Content)
	}
	if cf2.Language != cf.Language {
		t.Errorf("Expected Language %s, got %s", cf.Language, cf2.Language)
	}
}

func TestCodeFile_EmptyContent(t *testing.T) {
	cf := CodeFile{
		FilePath: "empty.go",
		Content:  "",
		Language: "go",
	}

	data, err := json.Marshal(cf)
	if err != nil {
		t.Fatalf("Failed to marshal empty CodeFile: %v", err)
	}

	if len(data) == 0 {
		t.Errorf("Expected non-empty JSON for empty content CodeFile")
	}
}

// =============================================================================
// TESTS PARA TaskResult
// =============================================================================

func TestTaskResult_ToJSON(t *testing.T) {
	tr := TaskResult{
		TaskID:  "task-123",
		Summary: "Implemented login feature",
		CodeFiles: []CodeFile{
			{FilePath: "auth.go", Content: "package main", Language: "go"},
		},
		ValidationNotes: "Tested locally",
		Timestamp:       time.Now(),
	}

	data, err := tr.ToJSON()
	if err != nil {
		t.Fatalf("Failed to marshal TaskResult: %v", err)
	}

	if len(data) == 0 {
		t.Errorf("Expected non-empty JSON")
	}
}

func TestTaskResult_FromJSON(t *testing.T) {
	jsonData := `{
		"task_id": "task-456",
		"summary": "Added validation",
		"code_files": [
			{"file_path": "validate.go", "content": "func Validate()", "language": "go"}
		],
		"validation_notes": "Edge cases covered",
		"timestamp": "2024-01-15T10:30:00Z"
	}`

	tr, err := TaskResultFromJSON([]byte(jsonData))
	if err != nil {
		t.Fatalf("Failed to unmarshal TaskResult: %v", err)
	}

	if tr.TaskID != "task-456" {
		t.Errorf("Expected TaskID 'task-456', got '%s'", tr.TaskID)
	}
	if tr.Summary != "Added validation" {
		t.Errorf("Expected Summary 'Added validation', got '%s'", tr.Summary)
	}
	if len(tr.CodeFiles) != 1 {
		t.Errorf("Expected 1 CodeFile, got %d", len(tr.CodeFiles))
	}
}

func TestTaskResult_EmptyCodeFiles(t *testing.T) {
	tr := TaskResult{
		TaskID:    "task-789",
		Summary:   "Empty task",
		CodeFiles: []CodeFile{},
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(tr)
	if err != nil {
		t.Fatalf("Failed to marshal TaskResult with empty code files: %v", err)
	}

	tr2, err := TaskResultFromJSON(data)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if len(tr2.CodeFiles) != 0 {
		t.Errorf("Expected 0 CodeFiles, got %d", len(tr2.CodeFiles))
	}
}

// =============================================================================
// TESTS PARA CommitTaskResultRequest
// =============================================================================

func TestCommitTaskResultRequest_Complete(t *testing.T) {
	req := CommitTaskResultRequest{
		SubAgentID:  "agent-123",
		ProjectPath: "/project/app",
		SessionID:   "session-456",
		TaskID:      "task-789",
		CodeFiles: []CodeFile{
			{FilePath: "main.go", Content: "package main"},
		},
		Summary:          "Added new feature",
		ValidationNotes:  "Unit tests passed",
		DependenciesJSON: `{"go.mod": "module app"}`,
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var req2 CommitTaskResultRequest
	if err := json.Unmarshal(data, &req2); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if req2.SubAgentID != req.SubAgentID {
		t.Errorf("SubAgentID mismatch")
	}
	if req2.ProjectPath != req.ProjectPath {
		t.Errorf("ProjectPath mismatch")
	}
	if len(req2.CodeFiles) != 1 {
		t.Errorf("CodeFiles count mismatch")
	}
}

func TestCommitTaskResultRequest_Minimal(t *testing.T) {
	req := CommitTaskResultRequest{
		SubAgentID:  "agent-1",
		ProjectPath: "/test",
		SessionID:   "session-1",
		TaskID:      "task-1",
		CodeFiles: []CodeFile{
			{FilePath: "test.go", Content: "package test"},
		},
		Summary: "Minimal commit",
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal minimal request: %v", err)
	}

	// Should work with empty optional fields
	if string(data) == "" {
		t.Errorf("Expected non-empty JSON")
	}
}

// =============================================================================
// TESTS PARA CommitTaskResultResponse
// =============================================================================

func TestCommitTaskResultResponse_JSON(t *testing.T) {
	resp := CommitTaskResultResponse{
		Success:     true,
		Validated:   true,
		BuildOutput: "Build successful",
		BuildError:  "",
		SavedFiles:  []string{"main.go", "utils.go"},
		TaskStatus:  "completed",
		Message:     "All good",
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal response: %v", err)
	}

	var resp2 CommitTaskResultResponse
	if err := json.Unmarshal(data, &resp2); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if !resp2.Success {
		t.Errorf("Expected Success=true")
	}
	if !resp2.Validated {
		t.Errorf("Expected Validated=true")
	}
	if resp2.TaskStatus != "completed" {
		t.Errorf("Expected TaskStatus='completed', got '%s'", resp2.TaskStatus)
	}
	if len(resp2.SavedFiles) != 2 {
		t.Errorf("Expected 2 SavedFiles, got %d", len(resp2.SavedFiles))
	}
}

func TestCommitTaskResultResponse_Failure(t *testing.T) {
	resp := CommitTaskResultResponse{
		Success:    true,  // Request was processed
		Validated:  false, // But validation failed
		BuildError: "syntax error: unexpected token",
		TaskStatus: "failed",
		Message:    "Validation failed",
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal failure response: %v", err)
	}

	var resp2 CommitTaskResultResponse
	json.Unmarshal(data, &resp2)

	if resp2.Validated {
		t.Errorf("Expected Validated=false for failed validation")
	}
	if resp2.TaskStatus != "failed" {
		t.Errorf("Expected TaskStatus='failed', got '%s'", resp2.TaskStatus)
	}
}

// =============================================================================
// TESTS PARA PlanAndDelegateRequest
// =============================================================================

func TestPlanAndDelegateRequest_Full(t *testing.T) {
	req := PlanAndDelegateRequest{
		SessionID:   "session-abc",
		ProjectPath: "/project/backend",
		Requirement: "Implementar sistema de caché",
		Context:     "Proyecto Go con BadgerDB",
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var req2 PlanAndDelegateRequest
	if err := json.Unmarshal(data, &req2); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if req2.Requirement != "Implementar sistema de caché" {
		t.Errorf("Requirement mismatch")
	}
	if req2.Context != "Proyecto Go con BadgerDB" {
		t.Errorf("Context mismatch")
	}
}

func TestPlanAndDelegateRequest_WithoutContext(t *testing.T) {
	req := PlanAndDelegateRequest{
		SessionID:   "session-xyz",
		ProjectPath: "/test",
		Requirement: "Add feature",
		// Context is empty
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var req2 PlanAndDelegateRequest
	if err := json.Unmarshal(data, &req2); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if req2.Context != "" {
		t.Errorf("Expected empty Context, got '%s'", req2.Context)
	}
}

// =============================================================================
// TESTS PARA PlanAndDelegateResponse
// =============================================================================

func TestPlanAndDelegateResponse_WithTasks(t *testing.T) {
	resp := PlanAndDelegateResponse{
		Success: true,
		PlanID:  "plan-123",
		Tasks: []PlannedTask{
			{
				ID:          "task-1",
				Title:       "Diseño de API",
				Description: "Crear endpoints REST",
				Priority:    1,
				Context:     "Usar Gin framework",
			},
			{
				ID:          "task-2",
				Title:       "Implementar lógica",
				Description: "Business logic para caché",
				Priority:    2,
				Context:     "Usar BadgerDB",
			},
		},
		SubAgentIDs: []string{"agent-1", "agent-2"},
		Dependencies: map[string][]string{
			"task-2": {"task-1"}, // task-2 depends on task-1
		},
		Message: "Plan created with 2 tasks",
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var resp2 PlanAndDelegateResponse
	if err := json.Unmarshal(data, &resp2); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if len(resp2.Tasks) != 2 {
		t.Errorf("Expected 2 tasks, got %d", len(resp2.Tasks))
	}
	if len(resp2.SubAgentIDs) != 2 {
		t.Errorf("Expected 2 SubAgentIDs, got %d", len(resp2.SubAgentIDs))
	}
	if resp2.Dependencies["task-2"][0] != "task-1" {
		t.Errorf("Expected dependency task-1 for task-2")
	}
}

func TestPlanAndDelegateResponse_Empty(t *testing.T) {
	resp := PlanAndDelegateResponse{
		Success: false,
		Message: "No tasks created",
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var resp2 PlanAndDelegateResponse
	json.Unmarshal(data, &resp2)

	if resp2.Success {
		t.Errorf("Expected Success=false for empty response")
	}
}

// =============================================================================
// TESTS PARA PlannedTask
// =============================================================================

func TestPlannedTask_Full(t *testing.T) {
	task := PlannedTask{
		ID:          "planned-task-123",
		Title:       "Implement OAuth",
		Description: "Agregar autenticación OAuth2 con Google",
		Priority:    1,
		Context:     "Código existente usa struct User{}",
	}

	data, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var task2 PlannedTask
	if err := json.Unmarshal(data, &task2); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if task2.Priority != 1 {
		t.Errorf("Expected Priority=1, got %d", task2.Priority)
	}
	if task2.Context == "" {
		t.Errorf("Expected non-empty Context")
	}
}

func TestPlannedTask_PriorityOrdering(t *testing.T) {
	tasks := []PlannedTask{
		{ID: "low", Priority: 3},
		{ID: "high", Priority: 1},
		{ID: "medium", Priority: 2},
	}

	// Verify priorities are preserved after marshal
	for i, task := range tasks {
		data, _ := json.Marshal(task)
		var task2 PlannedTask
		json.Unmarshal(data, &task2)

		if task2.Priority != tasks[i].Priority {
			t.Errorf("Priority mismatch at index %d", i)
		}
	}
}

// =============================================================================
// TESTS DE SERIALIZACIÓN COMPLETA
// =============================================================================

func TestFullSerialization_CommitTaskResultFlow(t *testing.T) {
	// Simulate a complete flow from request to response

	// 1. Create request
	req := CommitTaskResultRequest{
		SubAgentID:  "agent-001",
		ProjectPath: "/workspace/myapp",
		SessionID:   "session-xyz",
		TaskID:      "task-123",
		CodeFiles: []CodeFile{
			{FilePath: "main.go", Content: "package main\n\nfunc main() {}", Language: "go"},
			{FilePath: "utils.go", Content: "package main\n\nfunc Add(a,b int) int { return a+b }", Language: "go"},
		},
		Summary:         "Added calculator feature",
		ValidationNotes: "Tested with unit tests",
	}

	// 2. Serialize request
	reqBytes, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	// 3. Deserialize request
	var req2 CommitTaskResultRequest
	if err := json.Unmarshal(reqBytes, &req2); err != nil {
		t.Fatalf("Failed to unmarshal request: %v", err)
	}

	// 4. Create response (simulating successful validation)
	resp := CommitTaskResultResponse{
		Success:     true,
		Validated:   true,
		BuildOutput: "Build successful",
		SavedFiles:  []string{req.CodeFiles[0].FilePath, req.CodeFiles[1].FilePath},
		TaskStatus:  "completed",
		Message:     "Code saved successfully",
	}

	// 5. Serialize response
	respBytes, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal response: %v", err)
	}

	// 6. Deserialize response
	var resp2 CommitTaskResultResponse
	if err := json.Unmarshal(respBytes, &resp2); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// 7. Verify all data integrity
	if resp2.TaskStatus != "completed" {
		t.Errorf("Final status should be 'completed'")
	}
	if len(resp2.SavedFiles) != 2 {
		t.Errorf("Should have saved 2 files")
	}
}

func TestFullSerialization_PlanFlow(t *testing.T) {
	// Simulate complete planning flow

	req := PlanAndDelegateRequest{
		SessionID:   "session-plan",
		ProjectPath: "/project/new",
		Requirement: "Create user management system",
		Context:     "Tech stack: Go, PostgreSQL, REST API",
	}

	reqBytes, _ := json.Marshal(req)
	var req2 PlanAndDelegateRequest
	json.Unmarshal(reqBytes, &req2)

	// Simulate planning response
	planResp := PlanAndDelegateResponse{
		Success:     true,
		PlanID:      "plan-001",
		Tasks:       []PlannedTask{{ID: "t1", Title: "DB Schema", Priority: 1}},
		SubAgentIDs: []string{"sub-1"},
		Message:     "Plan created",
	}

	planBytes, _ := json.Marshal(planResp)
	var planResp2 PlanAndDelegateResponse
	json.Unmarshal(planBytes, &planResp2)

	if !planResp2.Success {
		t.Errorf("Plan should succeed")
	}
}
