package triggers

import (
	"github.com/hibiken/asynq"
)

// asynqClient is a package-level variable that holds the shared Asynq client.
// It is used by both cron and webhook triggers to enqueue jobs.
var asynqClient *asynq.Client

// InitAsynqClient initializes the triggers package with a shared Asynq client.
// This function must be called once when your server starts, as you've done
// in `cmd/start.go`.
func InitAsynqClient(client *asynq.Client) {
	asynqClient = client
}
