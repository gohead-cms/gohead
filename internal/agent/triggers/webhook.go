package triggers

import (
	"github.com/gohead-cms/gohead/pkg/queue"
)

// queueClient queues the jobs of cron triggers.
var queueClient queue.Client

// InitQueue sets the queue client of the triggers package.
func InitQueue(client queue.Client) {
	queueClient = client
}
