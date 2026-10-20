package triggers

import (
	"context"
	"fmt"
	"time"

	"github.com/go-co-op/gocron"

	"github.com/gohead-cms/gohead/internal/agent/jobs"
	"github.com/gohead-cms/gohead/pkg/logger"
	"github.com/gohead-cms/gohead/pkg/storage"
)

// Scheduler manages all cron-based agent jobs.
var Scheduler *gocron.Scheduler

// StartScheduler initializes the gocron scheduler and adds jobs for all agents with a cron field.
func StartScheduler() {
	if Scheduler != nil {
		// Stop any running scheduler to prevent duplicates.
		Scheduler.Stop()
	}

	logger.Log.Info("Starting cron scheduler for agents")

	// Create a new scheduler instance that runs jobs concurrently.
	Scheduler = gocron.NewScheduler(time.UTC)
	Scheduler.SetMaxConcurrentJobs(10, gocron.WaitMode)

	// Retrieve all agents from storage.
	agents, _, err := storage.GetAllAgents(nil, nil)
	if err != nil {
		logger.Log.WithError(err).Error("Failed to retrieve agents for scheduling")
		return
	}

	for _, agent := range agents {
		// Only schedule agents with a defined Cron field.
		if agent.Trigger.Cron != "" {
			// Schedule a new job.
			job, err := Scheduler.Cron(agent.Trigger.Cron).Do(StartJob, agent.ID)

			if err != nil {
				logger.Log.WithError(err).WithField("agent_name", agent.Name).Error("Failed to schedule cron job")
				continue
			}

			logger.Log.WithField("agent_name", agent.Name).WithField("next_run", job.NextRun()).Info("Scheduled cron job successfully")
		}
	}

	// Start the scheduler asynchronously.
	Scheduler.StartAsync()
}

// StartJob enqueues a run of the agent on the worker queue. It is called by
// the cron scheduler at each tick of the agent's schedule.
func StartJob(agentID uint) {
	agent, err := storage.GetAgentByID(agentID)
	if err != nil {
		logger.Log.WithError(err).WithField("agent_id", agentID).Error("Failed to retrieve agent for scheduled run")
		return
	}

	now := time.Now().UTC()
	payload := jobs.AgentJobPayload{
		AgentID: agent.ID,
		InitialInput: fmt.Sprintf(
			"Scheduled run at %s (cron: %s). Execute your task according to your system prompt.",
			now.Format(time.RFC3339), agent.Trigger.Cron),
		TriggerEvent: &jobs.TriggerEvent{
			Type:         "schedule",
			ScheduleData: map[string]any{"cron": agent.Trigger.Cron, "fired_at": now.Format(time.RFC3339)},
		},
		CreatedAt: now,
	}
	if err := jobs.EnqueueAgentJob(context.Background(), queueClient, payload); err != nil {
		logger.Log.WithError(err).WithField("agent_name", agent.Name).Error("Failed to enqueue scheduled agent run")
		return
	}
	logger.Log.WithField("agent_name", agent.Name).Info("Scheduled agent run enqueued")
}
