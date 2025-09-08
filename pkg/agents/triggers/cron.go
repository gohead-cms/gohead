package triggers

import (
	"log"
	"time"

	"github.com/go-co-op/gocron"

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

// StartJob is the core function that an agent's trigger will call.
// This is where you would implement the agent's flow logic.
func StartJob(agentID uint) {
	logger.Log.WithField("agent_id", agentID).Info("Starting agent job")

	// Placeholder for the actual workflow execution.
	// You would typically load the agent config, execute the steps, etc.
	agent, err := storage.GetAgentByID(agentID)
	if err != nil {
		logger.Log.WithError(err).WithField("agent_id", agentID).Error("Failed to retrieve agent for job")
		return
	}

	// For the MVP, let's just log a message.
	log.Printf("Agent job for '%s' completed successfully.", agent.Name)
}
