package main

import (
	"fmt"
	"learning-asynq/task"
	"log"

	"github.com/hibiken/asynq"
)

// Task payload for any email related tasks.
type EmailTaskPayload struct {
	// ID for the email recipient.
	UserID int
}

func main() {
	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: "localhost:6379"},
		asynq.Config{Concurrency: 10},
	)
	fmt.Printf("%+v", srv)

	mux := asynq.NewServeMux()
	mux.HandleFunc("email:welcome", task.SendWelcomeEmail)
	/*
		This also works
		mux.Handle("email:welcome", asynq.HandlerFunc(sendWelcomeEmail))
	*/
	mux.HandleFunc("email:reminder", task.SendReminderEmail)

	if err := srv.Run(mux); err != nil {
		log.Fatal(err)
	}
}
