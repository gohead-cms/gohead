package storage

import "github.com/gohead-cms/gohead/pkg/queue"

var queueClient queue.Client

func InitQueue(client queue.Client) {
	queueClient = client
}
