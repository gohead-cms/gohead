package memory

import (
	"context"
	"time"

	"github.com/gohead-cms/gohead/pkg/database"
	"github.com/gohead-cms/gohead/pkg/logger"

	"github.com/gohead-cms/gohead/internal/agents"
	"gorm.io/gorm"
)



func (AgentMessage) TableName() string { return "agent_messages" }

type GormMemory struct{}

func NewGorm(_ agents.MemoryConfig) (*GormMemory, error) {
	if err := database.DB.AutoMigrate(&AgentMessage{}); err != nil {
		return nil, err
	}
	return &GormMemory{}, nil
}

func (m *GormMemory) Append(ctx context.Context, agentID, sessionID string, msg agents.Message) error {
	row := &AgentMessage{
		AgentID:   agentID,
		SessionID: sessionID,
		Role:      msg.Role,
		Name:      msg.Name,
		Content:   msg.Content,
	}
	if err := database.DB.WithContext(ctx).Create(row).Error; err != nil {
		logger.Log.WithError(err).WithField("agent_id", agentID).Error("failed to append agent message")
		return err
	}
	return nil
}

func (m *GormMemory) Load(ctx context.Context, agentID, sessionID string, limit int) ([]agents.Message, error) {
	var rows []AgentMessage
	q := database.DB.WithContext(ctx).
		Where("agent_id = ? AND session_id = ?", agentID, sessionID).
		Order("id ASC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	if err := q.Find(&rows).Error; err != nil {
		logger.Log.WithError(err).WithField("agent_id", agentID).Error("failed to load agent messages")
		return nil, err
	}
	out := make([]agents.Message, 0, len(rows))
	for _, r := range rows {
		out = append(out, agents.Message{
			Role:    r.Role,
			Name:    r.Name,
			Content: r.Content,
		})
	}
	return out, nil
}


// ============================================================================
// file: internal/agents/storage/agent_store.go
// ============================================================================
package storage

import (
	"context"
	"encoding/json"
	"time"

	"github.com/gohead-cms/gohead/pkg/database"
	"github.com/gohead-cms/gohead/pkg/logger"

	"github.com/gohead-cms/gohead/internal/agents"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// AgentRecord: flat GORM model that stores the Agent config as JSON blobs.
// Keeps schema stable while Agent struct evolves.
type AgentRecord struct {
	ID        string         `gorm:"primaryKey;size:128"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`

	Name         string `gorm:"size:256;not null"`
	Description  string `gorm:"size:1024"`
	Enabled      bool   `gorm:"index"`
	SystemPrompt string `gorm:"type:text"`

	ProviderJSON  datatypes.JSON `gorm:"type:jsonb"`
	MemoryJSON    datatypes.JSON `gorm:"type:jsonb"`
	FunctionsJSON datatypes.JSON `gorm:"type:jsonb"`
	TriggersJSON  datatypes.JSON `gorm:"type:jsonb"`

	MaxTurns int
}

func (AgentRecord) TableName() string { return "agents" }

func AutoMigrateAgentStore() error {
	return database.DB.AutoMigrate(&AgentRecord{})
}

// ToRecord converts runtime Agent to DB record.
func ToRecord(a *agents.Agent) (*AgentRecord, error) {
	pj, err := json.Marshal(a.Provider)
	if err != nil {
		return nil, err
	}
	mj, err := json.Marshal(a.Memory)
	if err != nil {
		return nil, err
	}
	fj, err := json.Marshal(a.Functions)
	if err != nil {
		return nil, err
	}
	tj, err := json.Marshal(a.Triggers)
	if err != nil {
		return nil, err
	}
	return &AgentRecord{
		ID:           a.ID,
		Name:         a.Name,
		Description:  a.Description,
		Enabled:      a.Enabled,
		SystemPrompt: a.SystemPrompt,
		ProviderJSON: pj,
		MemoryJSON:   mj,
		FunctionsJSON: fj,
		TriggersJSON:  tj,
		MaxTurns:     a.MaxTurns,
	}, nil
}

// FromRecord converts DB record to runtime Agent.
func FromRecord(r *AgentRecord) (*agents.Agent, error) {
	var prov agents.ProviderConfig
	var mem agents.MemoryConfig
	var funcs []agents.FunctionSpec
	var trig agents.TriggerSpec

	if err := json.Unmarshal(r.ProviderJSON, &prov); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(r.MemoryJSON, &mem); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(r.FunctionsJSON, &funcs); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(r.TriggersJSON, &trig); err != nil {
		return nil, err
	}
	return &agents.Agent{
		ID:           r.ID,
		Name:         r.Name,
		Description:  r.Description,
		Enabled:      r.Enabled,
		SystemPrompt: r.SystemPrompt,
		Provider:     prov,
		Memory:       mem,
		Functions:    funcs,
		Triggers:     trig,
		MaxTurns:     r.MaxTurns,
	}, nil
}

// UpsertAgent persists (create/update) an Agent config.
func UpsertAgent(ctx context.Context, a *agents.Agent) error {
	rec, err := ToRecord(a)
	if err != nil {
		return err
	}
	tx := database.DB.WithContext(ctx)
	if err := tx.Clauses(
		// GORM upsert (ON CONFLICT DO UPDATE) — works on PG/MySQL/SQLite appropriately
	).Save(rec).Error; err != nil {
		logger.Log.WithError(err).WithField("agent_id", a.ID).Error("failed to upsert agent")
		return err
	}
	return nil
}

// LoadAllAgents returns all Agent configs from DB.
func LoadAllAgents(ctx context.Context) ([]*agents.Agent, error) {
	var recs []AgentRecord
	if err := database.DB.WithContext(ctx).Find(&recs).Error; err != nil {
		return nil, err
	}
	out := make([]*agents.Agent, 0, len(recs))
	for _, r := range recs {
		a, err := FromRecord(&r)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, nil
}


// ============================================================================
// file: internal/agents/engine.go (EDITED: default memory backend set to GORM)
// ============================================================================
package agents

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	memgorm "github.com/gohead-cms/gohead/internal/agents/memory"
	"github.com/gohead-cms/gohead/internal/agents/memory"
	"github.com/gohead-cms/gohead/internal/agents/providers"
	"github.com/gohead-cms/gohead/internal/agents/triggers"
)

type Engine struct {
	mu sync.RWMutex

	agents map[string]*Agent

	memoryFactory func(cfg MemoryConfig) (Memory, error)
	providerCtor  func(cfg ProviderConfig) (Provider, error)

	funcReg   FunctionRegistry
	triggerMgr *triggers.Manager
	sessionFor func(a *Agent, ev Event) string

	logger *log.Logger
}

type EngineDeps struct {
	FuncRegistry FunctionRegistry
	Logger       *log.Logger
	Cron         *triggers.CronScheduler
}

func NewEngine(deps EngineDeps) *Engine {
	tm := triggers.NewManager(deps.Cron)
	e := &Engine{
		agents:     map[string]*Agent{},
		funcReg:    deps.FuncRegistry,
		triggerMgr: tm,
		logger:     deps.Logger,
	}
	// ---- Memory: prefer GORM by default, fallback to Bolt for explicit "bbolt" ----
	e.memoryFactory = func(cfg MemoryConfig) (Memory, error) {
		switch cfg.Backend {
		case "", "gorm":
			return memgorm.NewGorm(cfg)
		case "bbolt":
			return memory.NewBolt(cfg)
		default:
			return nil, fmt.Errorf("unknown memory backend %q", cfg.Backend)
		}
	}
	// ---- Provider ctor ----
	e.providerCtor = func(cfg ProviderConfig) (Provider, error) {
		switch cfg.Type {
		case "openai":
			if cfg.APIKeyRef == "" {
				return nil, errors.New("missing API key for openai")
			}
			return providers.NewOpenAI(cfg.APIKeyRef), nil
		default:
			return nil, fmt.Errorf("unknown provider %q", cfg.Type)
		}
	}
	e.sessionFor = func(a *Agent, _ Event) string { return a.ID }
	return e
}

func (e *Engine) RegisterAgent(ctx context.Context, a *Agent) error {
	if a.ID == "" {
		return errors.New("agent ID required")
	}
	if a.MaxTurns <= 0 {
		a.MaxTurns = 4
	}
	e.mu.Lock()
	e.agents[a.ID] = a
	e.mu.Unlock()

	e.triggerMgr.Remove(a.ID)
	if a.Enabled && a.Triggers.Cron != "" {
		e.triggerMgr.AddCron(a.ID, a.Triggers.Cron, func(when time.Time) {
			_ = e.handleEvent(context.Background(), Event{AgentID: a.ID, Kind: "cron", When: when})
		})
	}
	return nil
}

func (e *Engine) ListAgents() []*Agent {
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := make([]*Agent, 0, len(e.agents))
	for _, a := range e.agents {
		out = append(out, a)
	}
	return out
}

func (e *Engine) HandleWebhook(ctx context.Context, agentID string, payload json.RawMessage) error {
	return e.handleEvent(ctx, Event{
		AgentID: agentID,
		Kind:    "webhook",
		When:    time.Now(),
		Payload: payload,
	})
}

func (e *Engine) handleEvent(ctx context.Context, ev Event) error {
	e.mu.RLock()
	a := e.agents[ev.AgentID]
	e.mu.RUnlock()
	if a == nil || !a.Enabled {
		return fmt.Errorf("agent %s not found or disabled", ev.AgentID)
	}

	mem, err := e.memoryFactory(a.Memory)
	if err != nil {
		return err
	}
	prov, err := e.providerCtor(a.Provider)
	if err != nil {
		return err
	}
	tools, impls, err := e.funcReg.ToolsFromSpecs(a.Functions)
	if err != nil {
		return err
	}

	sessionID := e.sessionFor(a, ev)
	history, _ := mem.Load(ctx, a.ID, sessionID, 100)

	var msgs []Message
	if a.SystemPrompt != "" {
		msgs = append(msgs, Message{Role: "system", Content: a.SystemPrompt})
	}
	msgs = append(msgs, history...)
	if len(ev.Payload) > 0 {
		msgs = append(msgs, Message{Role: "user", Name: "event", Content: string(ev.Payload)})
	} else {
		msgs = append(msgs, Message{Role: "user", Content: fmt.Sprintf("Event: %s at %s", ev.Kind, ev.When)})
	}
	if err := mem.Append(ctx, a.ID, sessionID, msgs[len(msgs)-1]); err != nil {
		return err
	}

	respContent, _, err := RunTurn(ctx, RunnerDeps{
		Provider:      prov,
		Memory:        mem,
		FunctionImpls: impls,
	}, RunnerInput{
		AgentID:     a.ID,
		SessionID:   sessionID,
		Model:       a.Provider.Model,
		Messages:    msgs,
		Tools:       tools,
		ToolChoice:  a.Provider.ToolChoice,
		MaxTurns:    a.MaxTurns,
		Temperature: a.Provider.Temperature,
		TopP:        a.Provider.TopP,
	})
	if err != nil {
		e.logger.Printf("agent %s error: %v", a.ID, err)
	}
	if respContent != "" {
		_ = mem.Append(ctx, a.ID, sessionID, Message{Role: "assistant", Content: respContent})
	}
	return nil
}


// ============================================================================
// file: cmd/agents-demo/main.go (EDITED to load/save agents via GORM store)
// ============================================================================
package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"github.com/gohead-cms/gohead/pkg/database"
	"github.com/gohead-cms/gohead/pkg/logger"

	"github.com/gohead-cms/gohead/internal/agents"
	"github.com/gohead-cms/gohead/internal/agents/functions"
	astorage "github.com/gohead-cms/gohead/internal/agents/storage"
	"github.com/gohead-cms/gohead/internal/agents/triggers"
)

// ---- Dummy collection service (replace with your real services) ----
type CollectionServiceStub struct{}

func (s *CollectionServiceStub) Upsert(ctx context.Context, collection string, id string, doc map[string]any) error {
	_ = collection; _ = id; _ = doc
	// TODO: wire to your content storage using GORM.
	return nil
}
func (s *CollectionServiceStub) Get(ctx context.Context, collection string, id string) (map[string]any, error) {
	_ = collection; _ = id
	return map[string]any{}, nil
}
func (s *CollectionServiceStub) Query(ctx context.Context, collection string, filter map[string]any, limit int) ([]map[string]any, error) {
	_ = collection; _ = filter; _ = limit
	return []map[string]any{}, nil
}

// ---- HTTP webhook bridge to engine ----
type HTTPServer struct {
	engine *agents.Engine
}

func (h *HTTPServer) HandleWebhook(r *http.Request, agentID string, payload json.RawMessage) error {
	return h.engine.HandleWebhook(r.Context(), agentID, payload)
}

func main() {
	// Ensure your global database.DB is initialized before this (as in your app).
	_ = database.DB

	lg := log.New(os.Stdout, "[agents] ", log.LstdFlags|log.Lshortfile)
	logger.Log = logger.Log // keep your logger wired

	// DB migrations for agents & memory
	if err := astorage.AutoMigrateAgentStore(); err != nil {
		panic(err)
	}

	cron := triggers.NewCron()
	cron.Start()
	defer cron.Stop()

	// Function registry: plug in tools you allow.
	reg := agents.NewStaticRegistry()
	reg.Register("collections.upsert", functions.NewCollectionsUpsert(&CollectionServiceStub{}))

	engine := agents.NewEngine(agents.EngineDeps{
		FuncRegistry: reg,
		Logger:       lg,
		Cron:         cron,
	})

	// Load agents from DB; if none, seed an example
	ctx := context.Background()
	loaded, err := astorage.LoadAllAgents(ctx)
	if err != nil {
		panic(err)
	}
	if len(loaded) == 0 {
		openaiKey := os.Getenv("OPENAI_API_KEY")
		ex := &agents.Agent{
			ID:           "agent-" + uuid.NewString(),
			Name:         "Demo Agent (DB)",
			Description:  "Writes notes on schedule or via webhook",
			Enabled:      true,
			SystemPrompt: "You are a helpful automation. Be concise.",
			Provider: agents.ProviderConfig{
				Type:        "openai",
				Model:       "gpt-4o-mini",
				APIKeyRef:   openaiKey,
				Temperature: 0.2,
				TopP:        1.0,
				ToolChoice:  "auto",
			},
			Memory: agents.MemoryConfig{
				Backend:      "gorm",       // <<< use GORM-backed memory
				Namespace:    "gohead",
				SessionScope: "per-agent",
			},
			Functions: []agents.FunctionSpec{{
				Name:        "collections_upsert",
				Description: "Create or update a document inside a collection.",
				ImplKey:     "collections.upsert",
			}},
			Triggers: agents.TriggerSpec{
				Cron: "0 0 * * * *", // hourly (with seconds)
			},
			MaxTurns: 4,
		}
		if err := astorage.UpsertAgent(ctx, ex); err != nil {
			panic(err)
		}
		loaded = append(loaded, ex)
	}

	// Register all loaded agents into the runtime engine
	for _, a := range loaded {
		if err := engine.RegisterAgent(ctx, a); err != nil {
			panic(err)
		}
	}

	// HTTP
	s := &HTTPServer{engine: engine}
	r := mux.NewRouter()

	// Admin: list/add agents (persist via DB)
	r.HandleFunc("/agents", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			json.NewEncoder(w).Encode(engine.ListAgents())
			return
		case http.MethodPost:
			var a agents.Agent
			if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			// Persist to DB
			if err := astorage.UpsertAgent(r.Context(), &a); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			// Register live
			if err := engine.RegisterAgent(r.Context(), &a); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusCreated)
			return
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}).Methods("GET", "POST")

	// Webhook trigger
	triggers.MountWebhook(r, s)

	addr := ":8080"
	lg.Println("listening on", addr)
	_ = http.ListenAndServe(addr, r)
}
