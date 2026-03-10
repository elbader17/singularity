package models

import "time"

// ToolResult representa el resultado de una herramienta MCP
type ToolResult struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// SessionState representa el estado de una sesión
type SessionState struct {
	SessionID   string            `json:"session_id"`
	EngineType  string            `json:"engine_type"`
	ProjectPath string            `json:"project_path"`
	Status      string            `json:"status"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// DAGNode representa un nodo en el DAG de tareas
type DAGNode struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	Priority    int       `json:"priority"`
	DependsOn   []string  `json:"depends_on,omitempty"`
	Assignee    string    `json:"assignee,omitempty"`
	Metadata    string    `json:"metadata,omitempty"` // JSON string for engine-specific data
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// DAGMetadata representa metadatos del DAG completo
type DAGMetadata struct {
	SessionID   string    `json:"session_id"`
	ProjectPath string    `json:"project_path"`
	EngineType  string    `json:"engine_type"`
	Nodes       []DAGNode `json:"nodes"`
	RootNodes   []string  `json:"root_nodes"`
	LeafNodes   []string  `json:"leaf_nodes"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CompressedHistory representa historial comprimido
type CompressedHistory struct {
	SessionID    string    `json:"session_id"`
	Summary      string    `json:"summary"`     // Resumen ultra-corto
	KeyEvents    []string  `json:"key_events"`  // Eventos clave
	TokenCount   int       `json:"token_count"` // Estimación de tokens ahorrados
	OriginalLen  int       `json:"original_len"`
	CompressedAt time.Time `json:"compressed_at"`
}
