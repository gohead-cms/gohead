package models

import (
	"time"
)

// Run statuses.
const (
	RunRunning   = "running"
	RunSucceeded = "succeeded"
	RunFailed    = "failed"
	// RunLimited: stopped, or not started, because a rate or cost limit was
	// reached.
	RunLimited = "limited"
	// RunQueued: requested (run now, test run) and waiting for a worker.
	RunQueued = "queued"
)

// Step types.
const (
	StepLLM  = "llm"  // a call to the agent's model
	StepTool = "tool" // a function call requested by the model
)

// AgentRun records one execution of an agent: what started it, what it
// did, what it cost and how it ended.
type AgentRun struct {
	ID uint `json:"id" gorm:"primaryKey"`
	// DryRun: a test run; content functions only simulated their writes.
	DryRun       bool       `json:"dryRun" gorm:"not null;default:false"`
	AgentID      uint       `json:"agentId" gorm:"index:idx_agent_runs_agent_started,priority:1"`
	AgentName    string     `json:"agentName"`
	Status       string     `json:"status" gorm:"type:varchar(20);index"`
	TriggerType  string     `json:"triggerType" gorm:"type:varchar(30)"` // collection_event, webhook, schedule, manual
	Trigger      string     `json:"trigger" gorm:"type:text"`            // the trigger payload (JSON, truncated)
	Input        string     `json:"input" gorm:"type:text"`              // first message sent to the model
	Output       string     `json:"output" gorm:"type:text"`             // final answer of the model
	Error        string     `json:"error" gorm:"type:text"`
	Attempt      int        `json:"attempt"` // 1, or more when the queue retried the job
	Turns        int        `json:"turns"`
	ToolCalls    int        `json:"toolCalls"`
	Provider     string     `json:"provider"`
	Model        string     `json:"model"`
	InputTokens  int        `json:"inputTokens"`
	OutputTokens int        `json:"outputTokens"`
	CostUSD      float64    `json:"costUsd"`
	StartedAt    time.Time  `json:"startedAt" gorm:"index:idx_agent_runs_agent_started,priority:2"`
	FinishedAt   *time.Time `json:"finishedAt"`
	DurationMs   int64      `json:"durationMs"`

	Steps []AgentRunStep `json:"steps,omitempty" gorm:"foreignKey:RunID"`
}

// AgentRunStep is one LLM call or one function call of a run.
type AgentRunStep struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	RunID        uint      `json:"runId" gorm:"index"`
	Seq          int       `json:"seq"`
	Type         string    `json:"type" gorm:"type:varchar(10)"` // llm | tool
	Name         string    `json:"name"`                         // model, or function name
	ImplKey      string    `json:"implKey,omitempty"`            // tool steps
	Arguments    string    `json:"arguments" gorm:"type:text"`   // tool arguments (JSON, truncated)
	Result       string    `json:"result" gorm:"type:text"`      // model text / tool result (truncated)
	Error        string    `json:"error" gorm:"type:text"`
	InputTokens  int       `json:"inputTokens"`
	OutputTokens int       `json:"outputTokens"`
	CostUSD      float64   `json:"costUsd"`
	DurationMs   int64     `json:"durationMs"`
	CreatedAt    time.Time `json:"createdAt"`
}

// Limits bound how much an agent may run and spend. Zero means no limit.
// Windows are rolling: "per hour" is the last 60 minutes, "per day" the
// last 24 hours.
type Limits struct {
	MaxRunsPerHour  int     `json:"max_runs_per_hour,omitempty"`
	MaxRunsPerDay   int     `json:"max_runs_per_day,omitempty"`
	MaxTokensPerRun int     `json:"max_tokens_per_run,omitempty"`
	MaxTokensPerDay int     `json:"max_tokens_per_day,omitempty"`
	MaxCostPerDay   float64 `json:"max_cost_per_day,omitempty"` // USD; needs prices in llm_config
}

// Any reports whether a limit is set.
func (l Limits) Any() bool {
	return l.MaxRunsPerHour > 0 || l.MaxRunsPerDay > 0 || l.MaxTokensPerRun > 0 || l.MaxTokensPerDay > 0 || l.MaxCostPerDay > 0
}
