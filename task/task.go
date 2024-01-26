package task

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
)

// A list of task types
const (
	TypeWelcomeEmail  = "email:welcome"
	TypeReminderEmail = "email:reminder"
)

// Task Payload for any email related task
type emailTaskPayload struct {
	UserId int
}

func NewWelcomeEmailTask(id int) (*asynq.Task, error) {
	payload, err := json.Marshal(emailTaskPayload{UserId: id})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeWelcomeEmail, payload), nil
}

func NewReminderEmailTask(id int) (*asynq.Task, error) {
	payload, err := json.Marshal(emailTaskPayload{UserId: id})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeReminderEmail, payload), nil
}

func SendWelcomeEmail(ctx context.Context, t *asynq.Task) error {
	var p emailTaskPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return err
	}
	fmt.Printf(" ========================> [*] Send Welcome Email to User %d", p.UserId)
	return nil
}

func SendReminderEmail(ctx context.Context, t *asynq.Task) error {
	var p emailTaskPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return err
	}
	fmt.Printf(" =======================> [*] Send Reminder Email to User %d", p.UserId)
	return nil
}
