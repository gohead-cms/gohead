package storage

import "github.com/hibiken/asynq"

var asynqClient *asynq.Client

func InitAsynqClient(client *asynq.Client) {
	asynqClient = client
}
