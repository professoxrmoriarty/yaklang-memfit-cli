package aiclientv2

import "time"

type RunStatus string

const (
	RunStatusPending   RunStatus = "pending"
	RunStatusRunning   RunStatus = "running"
	RunStatusCompleted RunStatus = "completed"
	RunStatusCancelled RunStatus = "cancelled"
	RunStatusFailed    RunStatus = "failed"
)

type AIParams struct {
	ForgeName      string   `json:"forge_name,omitempty"`
	ReviewPolicy   string   `json:"review_policy,omitempty"`
	AIService      string   `json:"ai_service,omitempty"`
	AIModelName    string   `json:"ai_model_name,omitempty"`
	MaxIteration   int32    `json:"max_iteration,omitempty"`
	DisableToolUse bool     `json:"disable_tool_use,omitempty"`
	UseDefaultAI   bool     `json:"use_default_ai,omitempty"`
	AttachedFiles  []string `json:"attached_files,omitempty"`
}

type CreateSessionRequest struct {
	RunID  string   `json:"run_id,omitempty"`
	TaskID string   `json:"task_id,omitempty"`
	Params AIParams `json:"params,omitempty"`
}

type CreateSessionResponse struct {
	RunID  string    `json:"run_id"`
	Status RunStatus `json:"status"`
}

type RunEvent struct {
	ID            string `json:"id"`
	Type          string `json:"type"`
	CoordinatorID string `json:"coordinator_id,omitempty"`
	NodeID        string `json:"node_id,omitempty"`
	IsSystem      bool   `json:"is_system,omitempty"`
	IsStream      bool   `json:"is_stream,omitempty"`
	IsReason      bool   `json:"is_reason,omitempty"`
	StreamDelta   string `json:"stream_delta,omitempty"`
	Content       string `json:"content,omitempty"`
	Timestamp     int64  `json:"timestamp"`
	TaskIndex     string `json:"task_index,omitempty"`
	EventUUID     string `json:"event_uuid,omitempty"`
	TaskUUID      string `json:"task_uuid,omitempty"`
}

type RunResult struct {
	RunID         string     `json:"run_id"`
	TaskID        string     `json:"task_id"`
	Status        RunStatus  `json:"status"`
	CoordinatorID string     `json:"coordinator_id,omitempty"`
	Events        []RunEvent `json:"events"`
	CreatedAt     time.Time  `json:"created_at"`
	FinishedAt    *time.Time `json:"finished_at,omitempty"`
	Error         string     `json:"error,omitempty"`
}

type PushEventRequest struct {
	Type    string `json:"type"`
	Content string `json:"content,omitempty"`

	IsStart          bool      `json:"is_start,omitempty"`
	IsConfigHotpatch bool      `json:"is_config_hotpatch,omitempty"`
	HotpatchType     string    `json:"hotpatch_type,omitempty"`
	Params           *AIParams `json:"params,omitempty"`

	IsInteractiveMessage bool   `json:"is_interactive_message,omitempty"`
	InteractiveID        string `json:"interactive_id,omitempty"`
	InteractiveJSONInput string `json:"interactive_json_input,omitempty"`

	IsSyncMessage bool   `json:"is_sync_message,omitempty"`
	SyncType      string `json:"sync_type,omitempty"`
	SyncJSONInput string `json:"sync_json_input,omitempty"`
	SyncID        string `json:"sync_id,omitempty"`

	IsFreeInput   bool     `json:"is_free_input,omitempty"`
	FreeInput     string   `json:"free_input,omitempty"`
	AttachedFiles []string `json:"attached_files,omitempty"`
	FocusModeLoop string   `json:"focus_mode_loop,omitempty"`
}

type SettingRequest struct {
	UseDefaultAIConfig bool     `json:"use_default_ai_config"`
	AIService          string   `json:"ai_service,omitempty"`
	AIModelName        string   `json:"ai_model_name,omitempty"`
	ForgeName          string   `json:"forge_name,omitempty"`
	ReviewPolicy       string   `json:"review_policy,omitempty"`
	MaxIteration       int32    `json:"max_iteration,omitempty"`
	DisableToolUse     bool     `json:"disable_tool_use,omitempty"`
	AttachedFiles      []string `json:"attached_files,omitempty"`
	SelectedProviderID int64    `json:"selected_provider_id,omitempty"`
	SelectedModelName  string   `json:"selected_model_name,omitempty"`
	SelectedModelTier  string   `json:"selected_model_tier,omitempty"`
	FocusModeLoop      string   `json:"focus_mode_loop,omitempty"`
}

type SettingResponse struct {
	UseDefaultAIConfig bool     `json:"use_default_ai_config"`
	AIService          string   `json:"ai_service,omitempty"`
	AIModelName        string   `json:"ai_model_name,omitempty"`
	ForgeName          string   `json:"forge_name,omitempty"`
	ReviewPolicy       string   `json:"review_policy,omitempty"`
	MaxIteration       int32    `json:"max_iteration,omitempty"`
	DisableToolUse     bool     `json:"disable_tool_use,omitempty"`
	AttachedFiles      []string `json:"attached_files,omitempty"`
	SelectedProviderID int64    `json:"selected_provider_id,omitempty"`
	SelectedModelName  string   `json:"selected_model_name,omitempty"`
	SelectedModelTier  string   `json:"selected_model_tier,omitempty"`
	FocusModeLoop      string   `json:"focus_mode_loop,omitempty"`
}

type ProviderOption struct {
	ID         int64
	Type       string
	Domain     string
	Disabled   bool
	Identifier string
}

type FocusModeOption struct {
	Name          string
	VerboseName   string
	VerboseNameZh string
	Description   string
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Code    int    `json:"code"`
	Message string `json:"message,omitempty"`
}
