package models

import (
	"encoding/json"
	"time"
)

type TaskStatus string

const (
	TaskStatusPending    TaskStatus = "pending"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusCompleted  TaskStatus = "completed"
	TaskStatusBlocked    TaskStatus = "pending_sub_agent"
)

type SubAgentStatus string

const (
	SubAgentStatusPending   SubAgentStatus = "pending"
	SubAgentStatusRunning   SubAgentStatus = "running"
	SubAgentStatusCompleted SubAgentStatus = "completed"
	SubAgentStatusFailed    SubAgentStatus = "failed"
)

type SubAgent struct {
	ID          string         `json:"id"`
	TaskID      string         `json:"task_id"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Status      SubAgentStatus `json:"status"`
	Context     string         `json:"context"`
	Result      string         `json:"result,omitempty"`
	Error       string         `json:"error,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	StartedAt   *time.Time     `json:"started_at,omitempty"`
	CompletedAt *time.Time     `json:"completed_at,omitempty"`
}

type SpawnSubAgentRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Context     string `json:"context"`
	DependsOn   string `json:"depends_on,omitempty"`
}

type SpawnSubAgentResponse struct {
	Success    bool   `json:"success"`
	SubAgentID string `json:"sub_agent_id"`
	TaskID     string `json:"task_id"`
	Error      string `json:"error,omitempty"`
}

type SubAgentTaskResponse struct {
	Success     bool   `json:"success"`
	TaskID      string `json:"task_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Context     string `json:"context"`
	ProjectPath string `json:"project_path"`
	Error       string `json:"error,omitempty"`
}

type Task struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      TaskStatus `json:"status"`
	Priority    int        `json:"priority"`
	Assignee    string     `json:"assignee,omitempty"`
	DependsOn   []string   `json:"depends_on,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

type WorldState struct {
	SessionID      string            `json:"session_id"`
	ProjectPath    string            `json:"project_path"`
	CurrentTaskID  string            `json:"current_task_id"`
	ActiveTasks    []string          `json:"active_tasks"`
	BlockedTasks   []string          `json:"blocked_tasks"`
	CompletedTasks []string          `json:"completed_tasks"`
	Decisions      []Decision        `json:"decisions"`
	LastUpdated    time.Time         `json:"last_updated"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

type Decision struct {
	ID        string    `json:"id"`
	Content   string    `json:"content"`
	Context   string    `json:"context"`
	Timestamp time.Time `json:"timestamp"`
	Agent     string    `json:"agent"`
}

type CodeChange struct {
	FilePath  string    `json:"file_path"`
	Operation string    `json:"operation"` // create, update, delete
	Summary   string    `json:"summary"`
	Content   string    `json:"content,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

type CommitRequest struct {
	SessionID           string       `json:"session_id"`
	ProjectPath         string       `json:"project_path"`
	CompletedTaskID     string       `json:"completed_task_id,omitempty"`
	TaskSummary         string       `json:"task_summary,omitempty"`
	NewTasks            []Task       `json:"new_tasks,omitempty"`
	CodeChanges         []CodeChange `json:"code_changes,omitempty"`
	Decisions           []Decision   `json:"decisions,omitempty"`
	OrchestratorSummary string       `json:"orchestrator_summary,omitempty"`
	LearnedInsights     string       `json:"learned_insights,omitempty"`
	Blockers            []string     `json:"blockers,omitempty"`
}

type CommitResponse struct {
	Success     bool     `json:"success"`
	ActiveBrain string   `json:"active_brain"`
	NewTaskIDs  []string `json:"new_task_ids,omitempty"`
	Error       string   `json:"error,omitempty"`
}

type DeepContextRequest struct {
	Query  string `json:"query"`
	TaskID string `json:"task_id,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

type DeepContextResponse struct {
	Success bool           `json:"success"`
	Results []ArchiveEntry `json:"results"`
	Error   string         `json:"error,omitempty"`
}

type ArchiveEntry struct {
	Key       string    `json:"key"`
	Type      string    `json:"type"` // task, code, decision, session
	Summary   string    `json:"summary"`
	Content   string    `json:"content,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

func (t *Task) ToJSON() ([]byte, error) {
	return json.Marshal(t)
}

func TaskFromJSON(data []byte) (*Task, error) {
	var task Task
	err := json.Unmarshal(data, &task)
	return &task, err
}

func (w *WorldState) ToJSON() ([]byte, error) {
	return json.Marshal(w)
}

func WorldStateFromJSON(data []byte) (*WorldState, error) {
	var ws WorldState
	err := json.Unmarshal(data, &ws)
	return &ws, err
}

func (c *CommitRequest) ToJSON() ([]byte, error) {
	return json.Marshal(c)
}

func CommitRequestFromJSON(data []byte) (*CommitRequest, error) {
	var cr CommitRequest
	err := json.Unmarshal(data, &cr)
	return &cr, err
}

func (s *SubAgent) ToJSON() ([]byte, error) {
	return json.Marshal(s)
}

func SubAgentFromJSON(data []byte) (*SubAgent, error) {
	var sa SubAgent
	err := json.Unmarshal(data, &sa)
	return &sa, err
}
