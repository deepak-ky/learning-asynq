package main

import (
	"fmt"
	"learning-asynq/task"
	"log"
	"time"

	"github.com/hibiken/asynq"
)

func main() {
	client := asynq.NewClient(asynq.RedisClientOpt{Addr: "localhost:6379"})
	fmt.Println(client)
	/* //create a task with a typename and a payload
	payloadBytes, _ := json.Marshal(map[string]interface{}{"user_id": 42})
	fmt.Println(payloadBytes)
	fmt.Println(string(payloadBytes)) */

	/* t1 := asynq.NewTask("email:welcome", payloadBytes)
	fmt.Println(t1)
	t2 := asynq.NewTask("email:reminder", payloadBytes)
	fmt.Println(t2) */

	t1, err := task.NewWelcomeEmailTask(42)
	if err != nil {
		log.Fatal(err)
	}
	//Process the welcome email task immediately
	info, err := client.Enqueue(t1)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf(" [*] Successfully enqueued task: %+v", info)

	t2, err := task.NewReminderEmailTask(42)
	if err != nil {
		log.Fatal(err)
	}
	//Process the reminder email task 24 hrs later
	info, err = client.Enqueue(t2, asynq.ProcessIn(24*time.Hour))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf(" [*] Successfully enqueued task: %+v", info)
}
