package agents

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/gohead-cms/gohead/internal/agents/memory"
	memgorm "github.com/gohead-cms/gohead/internal/agents/memory"
	"github.com/gohead-cms/gohead/internal/agents/providers"
	"github.com/gohead-cms/gohead/internal/agents/triggers"
)

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
