package agents

import (
	"context"
	"testing"

	"singularity/internal/storage"
)

// =============================================================================
// TESTS EXHAUSTIVOS DEL SISTEMA DE ENGINES
// =============================================================================

func TestEngineRegistry_RegisterAndGet(t *testing.T) {
	// Limpiar registry
	globalRegistry = &EngineRegistry{
		engines: make(map[EngineType]AgentEngine),
	}

	// Registrar engines
	if err := InitDefaultEngines(); err != nil {
		t.Fatalf("Failed to init engines: %v", err)
	}

	// Verificar que se registraron ambos
	if !IsEngineRegistered(EngineTypeCore) {
		t.Error("RequestSaver engine should be registered")
	}

	if !IsEngineRegistered(EngineTypeParticle) {
		t.Error("TokenSaver engine should be registered")
	}
}

func TestEngineRegistry_GetByType(t *testing.T) {
	globalRegistry = &EngineRegistry{
		engines: make(map[EngineType]AgentEngine),
	}

	InitDefaultEngines()

	// Obtener RequestSaver
	rs, err := GetEngine(EngineTypeCore)
	if err != nil {
		t.Fatalf("Failed to get RequestSaver: %v", err)
	}
	if rs.Name() != "Core Engine" {
		t.Errorf("Expected 'Core Engine', got '%s'", rs.Name())
	}

	// Obtener TokenSaver
	ts, err := GetEngine(EngineTypeParticle)
	if err != nil {
		t.Fatalf("Failed to get TokenSaver: %v", err)
	}
	if ts.Name() != "Particle Engine" {
		t.Errorf("Expected 'Particle Engine', got '%s'", ts.Name())
	}
}

func TestEngineRegistry_GetUnknown(t *testing.T) {
	globalRegistry = &EngineRegistry{
		engines: make(map[EngineType]AgentEngine),
	}

	InitDefaultEngines()

	_, err := GetEngine("unknown")
	if err == nil {
		t.Error("Should fail for unknown engine type")
	}
}

func TestRequestSaverEngine_Type(t *testing.T) {
	engine := newRequestSaverEngine()
	if engine.Type() != EngineTypeCore {
		t.Errorf("Expected %s, got %s", EngineTypeCore, engine.Type())
	}
}

func TestRequestSaverEngine_Name(t *testing.T) {
	engine := newRequestSaverEngine()
	if engine.Name() != "Core Engine" {
		t.Errorf("Expected 'Core Engine', got '%s'", engine.Name())
	}
}

func TestRequestSaverEngine_Description(t *testing.T) {
	engine := newRequestSaverEngine()
	desc := engine.Description()
	if desc == "" {
		t.Error("Description should not be empty")
	}
}

func TestRequestSaverEngine_GetTools(t *testing.T) {
	engine := newRequestSaverEngine()
	tools := engine.GetTools()

	if len(tools) == 0 {
		t.Error("Should have at least one tool")
	}

	// Verificar herramientas específicas
	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolNames[tool.Name] = true
	}

	if !toolNames["plan_and_delegate"] {
		t.Error("Should have plan_and_delegate tool")
	}

	if !toolNames["commit_task_result"] {
		t.Error("Should have commit_task_result tool")
	}
}

func TestRequestSaverEngine_GetPrompts(t *testing.T) {
	engine := newRequestSaverEngine()

	orchPrompt := engine.GetOrchestratorPrompt()
	if orchPrompt == "" {
		t.Error("Orchestrator prompt should not be empty")
	}

	subAgentPrompt := engine.GetSubAgentPrompt()
	if subAgentPrompt == "" {
		t.Error("Sub-agent prompt should not be empty")
	}
}

func TestRequestSaverEngine_Initialize(t *testing.T) {
	engine := newRequestSaverEngine()
	ctx := context.Background()

	err := engine.Initialize(ctx, "test-session", "/test/project")
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	state, err := engine.GetActiveState()
	if err != nil {
		t.Fatalf("GetActiveState failed: %v", err)
	}

	if state == "" {
		t.Error("State should not be empty")
	}
}

func TestTokenSaverEngine_Type(t *testing.T) {
	engine := newTokenSaverEngine()
	if engine.Type() != EngineTypeParticle {
		t.Errorf("Expected %s, got %s", EngineTypeParticle, engine.Type())
	}
}

func TestTokenSaverEngine_Name(t *testing.T) {
	engine := newTokenSaverEngine()
	if engine.Name() != "Particle Engine" {
		t.Errorf("Expected 'Particle Engine', got '%s'", engine.Name())
	}
}

func TestTokenSaverEngine_GetTools(t *testing.T) {
	engine := newTokenSaverEngine()
	tools := engine.GetTools()

	if len(tools) == 0 {
		t.Error("Should have tools")
	}

	// Verificar herramientas específicas de TokenSaver
	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolNames[tool.Name] = true
	}

	if !toolNames["sync_dag_metadata"] {
		t.Error("Should have sync_dag_metadata tool")
	}

	if !toolNames["get_file_skeleton"] {
		t.Error("Should have get_file_skeleton tool")
	}

	if !toolNames["read_function"] {
		t.Error("Should have read_function tool")
	}

	if !toolNames["replace_function"] {
		t.Error("Should have replace_function tool")
	}

	if !toolNames["compress_history_key"] {
		t.Error("Should have compress_history_key tool")
	}
}

func TestTokenSaverEngine_GetPrompts(t *testing.T) {
	engine := newTokenSaverEngine()

	orchPrompt := engine.GetOrchestratorPrompt()
	if orchPrompt == "" {
		t.Error("Orchestrator prompt should not be empty")
	}

	subAgentPrompt := engine.GetSubAgentPrompt()
	if subAgentPrompt == "" {
		t.Error("Sub-agent prompt should not be empty")
	}
}

func TestTokenSaverEngine_Initialize(t *testing.T) {
	engine := newTokenSaverEngine()
	ctx := context.Background()

	err := engine.Initialize(ctx, "test-session", "/test/project")
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
}

func TestToolDefinition_Schema(t *testing.T) {
	engine := newTokenSaverEngine()
	tools := engine.GetTools()

	for _, tool := range tools {
		if tool.Name == "" {
			t.Error("Tool name should not be empty")
		}

		if tool.Description == "" {
			t.Error("Tool description should not be empty")
		}

		if tool.InputSchema == nil {
			t.Error("InputSchema should not be nil")
		}

		if tool.Handler == nil {
			t.Error("Handler should not be nil")
		}
	}
}

func TestEngineNames(t *testing.T) {
	globalRegistry = &EngineRegistry{
		engines: make(map[EngineType]AgentEngine),
	}

	InitDefaultEngines()

	names := GetEngineNames()

	if len(names) != 2 {
		t.Errorf("Expected 2 engines, got %d", len(names))
	}

	if names[EngineTypeCore] != "Core Engine" {
		t.Errorf("Unexpected name for RequestSaver")
	}

	if names[EngineTypeParticle] != "Particle Engine" {
		t.Errorf("Unexpected name for TokenSaver")
	}
}

func TestGetAllEngines(t *testing.T) {
	globalRegistry = &EngineRegistry{
		engines: make(map[EngineType]AgentEngine),
	}

	InitDefaultEngines()

	engines := GetAllEngines()

	if len(engines) != 2 {
		t.Errorf("Expected 2 engines, got %d", len(engines))
	}
}

func TestEngineWithDB(t *testing.T) {
	// Crear DB temporal
	tmpDir := t.TempDir()
	db, err := storage.NewBadgerDB(tmpDir)
	if err != nil {
		t.Skipf("Skipping DB test: %v", err)
	}
	defer db.Close()

	// Inicializar engine con DB
	engine := newTokenSaverEngine()
	engine.SetDB(db)

	// Verificar que se inyectó
	tools := engine.GetTools()
	if len(tools) == 0 {
		t.Error("Should have tools")
	}
}

func TestToolResult_Structure(t *testing.T) {
	result := SuccessResult(map[string]interface{}{"test": "value"})
	if !result.Success {
		t.Error("SuccessResult should have Success=true")
	}

	result = ErrorResult("test error")
	if result.Success {
		t.Error("ErrorResult should have Success=false")
	}
	if result.Error != "test error" {
		t.Errorf("Expected error 'test error', got '%s'", result.Error)
	}
}

// =============================================================================
// TESTS DE INTEGRACIÓN
// =============================================================================

func TestFullFlow_RequestSaver(t *testing.T) {
	globalRegistry = &EngineRegistry{
		engines: make(map[EngineType]AgentEngine),
	}

	InitDefaultEngines()

	// Obtener engine
	engine, err := GetEngine(EngineTypeCore)
	if err != nil {
		t.Fatalf("Failed to get engine: %v", err)
	}

	// Inicializar
	ctx := context.Background()
	engine.Initialize(ctx, "session-1", "/project")

	// Obtener tools
	tools := engine.GetTools()
	if len(tools) < 2 {
		t.Errorf("Expected at least 2 tools, got %d", len(tools))
	}

	// Llamar tool
	params := map[string]interface{}{
		"session_id":   "session-1",
		"project_path": "/project",
		"requirement":  "Test requirement",
	}

	result, err := tools[0].Handler(ctx, params)
	if err != nil {
		t.Errorf("Handler error: %v", err)
	}
	if result == nil {
		t.Error("Result should not be nil")
	}
}

func TestFullFlow_TokenSaver(t *testing.T) {
	globalRegistry = &EngineRegistry{
		engines: make(map[EngineType]AgentEngine),
	}

	InitDefaultEngines()

	// Obtener engine
	engine, err := GetEngine(EngineTypeParticle)
	if err != nil {
		t.Fatalf("Failed to get engine: %v", err)
	}

	// Inicializar
	ctx := context.Background()
	engine.Initialize(ctx, "session-2", "/project")

	// Obtener tools
	tools := engine.GetTools()
	if len(tools) < 5 {
		t.Errorf("Expected at least 5 tools, got %d", len(tools))
	}

	// Verificar que todas las herramientas TokenSaver existen
	toolNames := make(map[string]bool)
	for _, t := range tools {
		toolNames[t.Name] = true
	}

	expected := []string{
		"sync_dag_metadata",
		"get_file_skeleton",
		"read_function",
		"replace_function",
		"compress_history_key",
	}

	for _, name := range expected {
		if !toolNames[name] {
			t.Errorf("Missing tool: %s", name)
		}
	}
}
