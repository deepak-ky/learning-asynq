# Asynq

Basic framework of Asynq: 

![Untitled](https://prod-files-secure.s3.us-west-2.amazonaws.com/9c2cca07-8579-43d0-b578-d16891f5ca43/7dca8858-2229-46b6-ba60-cdcb3d2db065/Untitled.png)

We are going to write two programs

1. client.go → This file will create and schedule tasks to be processed asynchronously by the background workers.
2. workers.go → This file will start multiple concurrent workers to process the task created by the client

Asynq uses client as redis broker. Both client.go and workers.go need to connect with redis to write and read from it respectively.

In Asynq, a unit or work is encapsulated in a type called Task, which conceptually has two fields: Type and Payload

```go
//Type is a string value that indicates the type of the task
func (t *Task) Type() string

//Payload is the data needed for the task execution
func (t *Task) Payload() []byte
```

### Client Program

In `client.go`, we are going to create a few tasks and enqueue them using `asynq.Client`.

To create a task, use `NewTask` function and pass type and payload for the task.

The `[Enqueue](https://godoc.org/github.com/hibiken/asynq#Client.Enqueue)` method takes a task and any number of options.

Use `[ProcessIn](https://godoc.org/github.com/hibiken/asynq#ProcessIn)` or `[ProcessAt](https://godoc.org/github.com/hibiken/asynq#ProcessAt)` option to schedule tasks to be processed in the future.

Client

A Client is responsible for scheduling tasks.

A Client is used to register tasks that should be processed immediately or sometime in the future.

Clients are safe for concurrent use by multiple goroutines.

```go
type Client struct {
	broker base.Broker
}

// Broker is a message broker that supports operations to manage task queues

type Broker interface {
	Ping() error
	Close() error
	***Enqueue***(ctx context.Context, msg *TaskMessage) error
	EnqueueUnique(ctx context.Context, msg *TaskMessage, ttl time.Duration) error
	***Dequeue***(qnames ...string) (*TaskMessage, time.Time, error)
	***Done***(ctx context.Context, msg *TaskMessage) error
	***MarkAsComplete***(ctx context.Context, msg *TaskMessage) error
	***Requeue***(ctx context.Context, msg *TaskMessage) error
	Schedule(ctx context.Context, msg *TaskMessage, processAt time.Time) error
	ScheduleUnique(ctx context.Context, msg *TaskMessage, processAt time.Time, ttl time.Duration) error
	***Retry***(ctx context.Context, msg *TaskMessage, processAt time.Time, errMsg string, isFailure bool) error
	Archive(ctx context.Context, msg *TaskMessage, errMsg string) error
	ForwardIfReady(qnames ...string) error

	// Group aggregation related methods
	AddToGroup(ctx context.Context, msg *TaskMessage, gname string) error
	AddToGroupUnique(ctx context.Context, msg *TaskMessage, groupKey string, ttl time.Duration) error
	ListGroups(qname string) ([]string, error)
	AggregationCheck(qname, gname string, t time.Time, gracePeriod, maxDelay time.Duration, maxSize int) (aggregationSetID string, err error)
	ReadAggregationSet(qname, gname, aggregationSetID string) ([]*TaskMessage, time.Time, error)
	DeleteAggregationSet(ctx context.Context, qname, gname, aggregationSetID string) error
	ReclaimStaleAggregationSets(qname string) error

	// Task retention related method
	DeleteExpiredCompletedTasks(qname string) error

	// Lease related methods
	ListLeaseExpired(cutoff time.Time, qnames ...string) ([]*TaskMessage, error)
	ExtendLease(qname string, ids ...string) (time.Time, error)

	// State snapshot related methods
	WriteServerState(info *ServerInfo, workers []*WorkerInfo, ttl time.Duration) error
	ClearServerState(host string, pid int, serverID string) error

	// Cancelation related methods
	***CancelationPubSub***() (*redis.PubSub, error) // TODO: Need to decouple from redis to support other brokers
	PublishCancelation(id string) error

	WriteResult(qname, id string, data []byte) (n int, err error)
}
```

→ RedisClientOpt is used to create a redis client that connects to a redis server directly.

Task

Task basically represents a unit of work to be performed.

```go
// Task represents a unit of work to be performed.
type Task struct {
	// typename indicates the type of task to be performed.
	typename string

	// payload holds data needed to perform the task.
	payload []byte

	// opts holds options for the task.
	opts []Option

	// w is the ResultWriter for the task.
	w *ResultWriter
}
```

```go
// NewTask returns a new Task given a type name and payload data.
// Options can be passed to configure task processing behavior.
func NewTask(typename string, payload []byte, opts ...Option) *Task {
	return &Task{
		typename: typename,
		payload:  payload,
		opts:     opts,
	}
}

```

By passing the opts …Option, we have made it kind of a optional parameter.

Queues that are formed in Redis

1. **asynq:queues** → 
    1. this is a **set** of all the queues 
    2. ⇒ SMEMBERS asynq:queues 
2. **asynq:{default}:pending** → 
    1. this is a **list** of all the task ids which are **pending** in **default** queue       
    2. ⇒    LRANGE asynq:{default}:pending 0 -1 > "d5b05d66-ff91-49a0-b35f-47962b1996be”
3. **asynq:{default}:t:d5b05d66-ff91-49a0-b35f-47962b1996be →**
    1.  ****this is the hashmap which contains all the details about the task 
    2. ⇒ hgetall "asynq:{default}:t:d5b05d66-ff91-49a0-b35f-47962b1996be” > 
    3. Values in the hashmap
    
    ```go
    "pending_since" : "1706164254626464900"
    "state" : "pending"
    "msg" : "\n\remail:welcome\x12\x0e{\"user_id\":42}\x1a$d5b05d66-ff91-49a0-b35f-47962b1996be\"\adefault(\x19@\x88\x0e"
    ```
    

Queue details after enqueing

```json
{
ID:14818436-3f9e-4be1-97e5-a5f2393b9707 
Queue:default  // gets enqued in default queue
Type:email:welcome  // type of the task
Payload:[123 34 117 115 101 114 95 105 100 34 58 52 50 125] 
State:pending  
MaxRetry:25 
Retried:0 
LastErr: 
LastFailedAt:0001-01-01 00:00:00 +0000 UTC 
Timeout:30m0s 
Deadline:0001-01-01 00:00:00 +0000 UTC 
Group: 
NextProcessAt:2024-01-25 12:57:07.1329176 +0530 IST m=+0.006314401 
IsOrphaned:false 
Retention:0s 
CompletedAt:0001-01-01 00:00:00 +0000 UTC 
Result:[]
}
```

- **Orphaned:** In the context of task queues, an orphaned task is one that has become disconnected from its designated queue. This can happen for various reasons, such as:
    - The queue itself might have been deleted while the task was still waiting to be processed.
    - The broker storing the queue data might have experienced an error or failure, leading to the task being lost or detached.
    - The task might have been manually moved or copied outside the regular queueing flow.

**IsOrphaned:false**  This explicitly states that the specific task under consideration hasn't encountered any of the scenarios mentioned above. **It remains firmly associated with its original queue and is part of the regular processing flow.**

```go
// Enqueue enqueues the given task to a queue.
//
// Enqueue returns TaskInfo and nil error if the task is enqueued successfully, otherwise returns a non-nil error.
//
// The argument opts specifies the behavior of task processing.
// If there are conflicting Option values the last one overrides others.
// Any options provided to NewTask can be overridden by options passed to Enqueue.
// By default, **max retry is set to 25** and **timeout is set to 30 minutes**.
//
// If no ProcessAt or ProcessIn options are provided, the task will be pending immediately.
//
// Enqueue uses context.Background internally; to specify the context, use EnqueueContext.
func (c *Client) Enqueue(task *Task, opts ...Option) (*TaskInfo, error) {
	return c.EnqueueContext(context.Background(), task, opts...)
}
```

- By default, **max retry is set to 25** and **timeout is set to 30 minutes**.
- If no **ProcessAt** or **ProcessIn** options are provided, the task will be **pending immediately.**
- **ProcessIn** returns an option to specify when to process the given task relative to the current time.
- **ProcessAt** returns an option to specify when to process the given task.

```go
**// ProcessIn returns an option to specify when to process the given task relative to the current time.**
//
// If there's a conflicting ProcessAt option, the last option passed to Enqueue overrides the others.
func ProcessIn(d time.Duration) Option {
	return processInOption(d)

```

```go
**// ProcessAt returns an option to specify when to process the given task.**
//
// If there's a conflicting ProcessIn option, the last option passed to Enqueue overrides the others.
func ProcessAt(t time.Time) Option {
	return processAtOption(t)
}
```

What is timeout??? 

Task State 

```go
// TaskState denotes the state of a task.
type TaskState int

const (
	// Indicates that the task is currently being processed by Handler.
	TaskStateActive TaskState = iota + 1

	// Indicates that the task is ready to be processed by Handler.
	TaskStatePending

	// Indicates that the task is scheduled to be processed some time in the future.
	TaskStateScheduled

	// Indicates that the task has previously ***failed*** and scheduled to be processed some time in the future.
	TaskStateRetry

	// Indicates that the task is ***archived*** and stored for inspection purposes.
	TaskStateArchived

	// ***Indicates that the task is processed successfully and retained until the retention TTL expires.***
	TaskStateCompleted

	// Indicates that the task is waiting in a group to be aggregated into one task.
	TaskStateAggregating
)
```

Task Info  (returned by enqueue function)

```go
// A TaskInfo describes a task and its metadata.
type TaskInfo struct {
	// ID is the identifier of the task.
	ID string

	// Queue is the name of the queue in which the task belongs.
	Queue string

	// Type is the type name of the task.
	Type string

	// Payload is the payload data of the task.
	Payload []byte

	// State indicates the task state.
	State TaskState

	// MaxRetry is the maximum number of times the task can be retried.
	MaxRetry int

	// Retried is the number of times the task has retried so far.
	Retried int

	// LastErr is the error message from the last failure.
	LastErr string

	// LastFailedAt is the time time of the last failure if any.
	// If the task has no failures, LastFailedAt is zero time (i.e. time.Time{}).
	LastFailedAt time.Time

	***// Timeout is the duration the task can be processed by Handler before being retried,
	// zero if not specified***
	Timeout time.Duration

	***// Deadline is the deadline for the task, zero value if not specified.***
	Deadline time.Time

	// Group is the name of the group in which the task belongs.
	//
	// Tasks in the same queue can be grouped together by Group name and will be aggregated into one task
	// by a Server processing the queue.
	//
	// Empty string (default) indicates task does not belong to any groups, and no aggregation will be applied to the task.
	Group string

	// NextProcessAt is the time the task is scheduled to be processed,
	// zero if not applicable.
	NextProcessAt time.Time

	***// IsOrphaned describes whether the task is left in active state with no worker processing it.
	// An orphaned task indicates that the worker has crashed or experienced network failures and was not able to
	// extend its lease on the task.***
	//
	***// This task will be recovered by running a server against the queue the task is in.
	// This field is only applicable to tasks with TaskStateActive.***
	IsOrphaned bool

	***// Retention is duration of the retention period after the task is successfully processed.***
	Retention time.Duration

	***// CompletedAt is the time when the task is processed successfully.
	// Zero value (i.e. time.Time{}) indicates no value.***
	CompletedAt time.Time

	// Result holds the result data associated with the task.
	// Use ResultWriter to write result data from the Handler.
	Result []byte
}
```

```go
127.0.0.1:6379> keys *
1) "asynq:{default}:scheduled"
2) "asynq:{default}:pending"
3) "asynq:{default}:t:77005fca-225f-452d-adbe-66aac5561ad5"
4) "asynq:{default}:t:4974258a-d77a-4552-b8fa-16200a70380f"
5) "asynq:queues"
127.0.0.1:6379> zrange asynq:{default}:scheduled 0 -1
1) "77005fca-225f-452d-adbe-66aac5561ad5"
127.0.0.1:6379> hgetall "asynq:{default}:t:77005fca-225f-452d-adbe-66aac5561ad5"
1) "state"
2) "scheduled"
3) "msg"
4) "\n\x0eemail:reminder\x12\x0e{\"user_id\":42}\x1a$77005fca-225f-452d-adbe-66aac5561ad5\"\adefault(\x19@\x88\x0e"
127.0.0.1:6379> hgetall "asynq:{default}:t:4974258a-d77a-4552-b8fa-16200a70380f"
1) "pending_since"
2) "1706169155430582500" // January 25, 2024 1:22:35.430 PM GMT+05:30
3) "state"
4) "pending"
5) "msg"
6) "\n\remail:welcome\x12\x0e{\"user_id\":42}\x1a$4974258a-d77a-4552-b8fa-16200a70380f\"\adefault(\x19@\x88\x0e"
127.0.0.1:6379> smembers asynq:queues
1) "default"
127.0.0.1:6379> lrange asynq:{default}:pending 0 -1
1) "4974258a-d77a-4552-b8fa-16200a70380f"
127.0.0.1:6379>
```

```go
[*] Successfully enqueued task: &{ID:4974258a-d77a-4552-b8fa-16200a70380f 
Queue:default 
Type:email:welcome 
Payload:[123 34 117 115 101 114 95 105 100 34 58 52 50 125] 
State:pending 
MaxRetry:25 
Retried:0 
LastErr: 
LastFailedAt:0001-01-01 00:00:00 +0000 UTC 
Timeout:30m0s 
Deadline:0001-01-01 00:00:00 +0000 UTC 
Group: 
NextProcessAt:2024-01-25 13:22:35.4181074 +0530 IST m=+0.005620601 
IsOrphaned:false 
Retention:0s 
CompletedAt:0001-01-01 00:00:00 +0000 UTC 
Result:[]} 

[*] Successfully enqueued task: &{
ID:77005fca-225f-452d-adbe-66aac5561ad5 
Queue:default 
Type:email:reminder 
Payload:[123 34 117 115 101 114 95 105 100 34 58 52 50 125] 
State:scheduled 
MaxRetry:25 
Retried:0 
LastErr: 
LastFailedAt:0001-01-01 00:00:00 +0000 UTC 
Timeout:30m0s 
Deadline:0001-01-01 00:00:00 +0000 UTC 
Group: 
NextProcessAt:2024-01-26 13:22:35.432787 +0530 IST m=+86400.020300201 // will execute tomorrwo
IsOrphaned:false 
Retention:0s 
CompletedAt:0001-01-01 00:00:00 +0000 UTC 
Result:[]}
```

Client Program 

```go
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/hibiken/asynq"
)

func main() {
	client := asynq.NewClient(asynq.RedisClientOpt{Addr: "localhost:6379"})
	fmt.Println(client)
	//create a task with a typename and a payload
	payloadBytes, _ := json.Marshal(map[string]interface{}{"user_id": 42})
	fmt.Println(payloadBytes)
	fmt.Println(string(payloadBytes))

	t1 := asynq.NewTask("email:welcome", payloadBytes)
	fmt.Println(t1)
	t2 := asynq.NewTask("email:reminder", payloadBytes)
	fmt.Println(t2)

	//Process the welcome email task immediately
	info, err := client.Enqueue(t1)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf(" [*] Successfully enqueued task: %+v", info)

	//Process the reminder email task 24 hrs later
	info, err = client.Enqueue(t2, asynq.ProcessIn(24*time.Hour))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf(" [*] Successfully enqueued task: %+v", info)
}
```

### Workers Program

In `workers.go`, we'll create a `asynq.Server` instance to start the workers.

`NewServer` function takes `RedisConnOpt` and `Config`

- Server is reponsible for task processing and task lifecycle managment.
- Server pulls tasks off queues and processes them.
- If the processing of a task is unsuccessful, server will schedule it for a retry
- A task will be retried until either the task gets processed successfully or until it reaches its max retry count
- If a task exhausts its retries, it will be moved to the archive and will be kept in the archive set
- **Note that the archive size is finite and once it reaches its max size, oldest tasks in the archive will be deleted**

```go
type Server struct {
	logger *log.Logger

	broker base.Broker

	state *serverState

	// wait group to wait for all goroutines to finish.
	wg            sync.WaitGroup
	// A forwarder is responsible for moving scheduled and retry tasks to pending state
	// so that the tasks get processed by the workers.
	forwarder     *forwarder
	
	processor     *processor

	// syncer is responsible for queuing up failed requests to redis and retry
	// those requests to sync state between the background process and redis.
	syncer        *syncer

	// heartbeater is responsible for writing process info to redis periodically to
	// indicate that the background worker process is up.
	heartbeater   *heartbeater

	
	subscriber    *subscriber
	recoverer     *recoverer
	// healthchecker is responsible for pinging broker periodically
	// and call user provided HeathCheckFunc with the ping result.
	healthchecker *healthchecker
	
	// A janitor is responsible for deleting expired completed tasks from the specified
	// queues. It periodically checks for any expired tasks in the completed set, and
	// deletes them.
	janitor       *janitor 
	
	// An aggregator is responsible for checking groups and aggregate into one task
	// if any of the grouping condition is met.
	aggregator    *aggregator
}
```

```go
srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: "localhost:6379"},
		asynq.Config{Concurrency: 10},
	)

// NewServer returns a new Server given a redis connection option
// and server configuration.
```

***Config is used to tune the servers task processing behaviour*** 

Config specifies the server’s background-task processing behaviour

- Concurrency : **Maximum number of concurrent processing of tasks**. If set to a zero or negative value, NewServer will overwrite the value to the number of CPUs usable by the current process.
- RetryDelayFunc : **Function to calculate retry delay for a failed task.** By default it uses exponential backoff algorithm to calculate the delay
- IsFailure func(error) bool : Predicate function to determine whether the error returned from Handler is a failure. **If the function returns false, Server will not increment the retried counter for the task, and Server won’t record the queue stats (processed and failed stats) to avoid skewing the error rate of the queue.** By default, if the given error is non-nil the function returns true.
- Queues map[string]int : **List of queues to process with given priority value. Keys are the names of the queues and values are associated priority value**. If set to nil or not specified, the **server will process only the default queue.** Priority is treated as follows to avoid starving low priority queues. Example
    - Queues: map[string]int {
    - “critical” : 6 ,
    - “default” : 3 ,
    - “low” : 1,
    - }
    - With the above config and given that all queues are not empty, the tasks in “critical”, “default”, “low” should be processed 60%, 30%, 10% of the time respectively.
    - **If a queue has a zero or negative priority value, the queue will be ignored**
- StrictPriority bool : **StrictPriority indicates whether the queue priority should be treated strictly. If set to true, tasks in the queue with the highest priority is processed first. The tasks in lower priority queues are processed only when those queues with higher priorities are empty**
- ErrorHandler ErrorHandler,:  **ErrorHandler handles errors returned by the task handler. HandleError is invoked only if the task handler returns a non-nil error.**
    - ****Example
    
    ```go
    func reportError(ctx context, task *asynq.Task, err error) {
    	retried, _ := asynq.GetRetryCount(ctx)
      maxRetry, _ := asynq.GetMaxRetry(ctx)
      if retried >= maxRetry {
    		err = fmt.Errorf("retry exhausted for task %s: %w", task.Type, err)
    	}
    	errorReportingService.Notify(err)
    }
    
    ErrorHandler: asynq.ErrorHandlerFunc(reportError)
    ```
    
- Logger Logger
- LogLevel LogLevel
- ShutDownTimeout time.Duration : **ShutdownTimeout specifies the duration to wait to let workers finish their tasks before forcing them to abort when stopping the server.** If unset or zero, default timeout of 8 seconds is used.
- HealthCheckFunc func(error) : HealthCheckFunc is called periodically with any errors encountered during ping to the connected redis server
- HealthCheckInterval time.Duration : HealthCheckInterval specifies the interval between healthchecks. If unset or zero, the interval is set to 15 seconds.
- DelayedTaskCheckInterval time.Duration : **DelayedTaskCheckInterval specifies the interval between checks run on ‘scheduled’ and ‘retry’ tasks, and forwarding them to ‘pending’ state if they are ready to be processed. If unset or zero, the interval is set to 5 seconds.**
- GroupGracePeriod time.Duration
- GroupMaxDelay time.Duration
- GroupMaxSize int
- GroupAggregator

Exponential Backoff Stradey 

```go
// DefaultRetryDelayFunc is the default RetryDelayFunc used if one is not specified in Config.
// It uses exponential back-off strategy to calculate the retry delay.
func DefaultRetryDelayFunc(n int, e error, t *Task) time.Duration {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	// Formula taken from https://github.com/mperham/sidekiq.
	s := int(math.Pow(float64(n), 4)) + 15 + (r.Intn(30) * (n + 1))
	return time.Duration(s) * time.Second
}
```

```go
// DefaultQueueName is the queue name used if none are specified by user.
const DefaultQueueName = "default"

var defaultQueueConfig = map[string]int{
	base.DefaultQueueName: 1,
}
```

```go
defaultShutdownTimeout = 8 * time.Second

	defaultHealthCheckInterval = 15 * time.Second

	defaultDelayedTaskCheckInterval = 5 * time.Second

	defaultGroupGracePeriod = 1 * time.Minute
```

The argument to (*Server.Run) is an interface **Handler** which has one method **ProcessTask.** 

**Handler**

- A Handler processes task
- ProcessTask should return nil if the processing of a task is successful
- If ProcessTask returns a non nil error or panics, the task will be retried after delay if retry-count is remaining, otherwise the task will be archived
- One exception to this rule is when ProcessTask returns a SkipRetry error. If the returned error is SkipRetry or an error wraps SkipRetry, retry is skipped and task will be immediately archived instead.

```go
type Handler interface {
	// ProcessTask should return nil if the task was processed successfully
	// If ProcessTask returns a non-nil error or panics, the task will be retried again later
	ProcessTask(context.Context, *Task) error
} 
```

The simplest way to implement a handler, is to define a function with the same signature *(context.Context, *Task)*

and use asynq.HandlerFunc adapter type when passing it to Run

**HandlerFunc Adapter**

- The HandlerFunc type is an adapter to allow the use of ordinary functions as a Handler
- **If f is a function with the appropriate signature, HandlerFunc(f) is a Handler that calls f.**

```go
// The HandlerFunc type is an adapter to allow the use of
// ordinary functions as a Handler. If f is a function
// with the appropriate signature, HandlerFunc(f) is a
// Handler that calls f.
type HandlerFunc func(context.Context, *Task) error
```

**Run**

Run starts the task processing and blocks 

until an os signal to exit the program is received.

Once it receives a signal, it gracefully shuts down all active workers and other goroutines to process the tasks. 

Run returns any error encountered at server startup time. 

If the server has already been shutdown, (Run is called after the server has already shutdown), ErrServerClosed is returned

```go
func (srv *Server) Run(handler Handler) error {
	if err := srv.Start(handler); err != nil {
		return err
	}
	srv.waitForSignals()
	srv.Shutdown()
	return nil
}

// ErrServerClosed indicates that the operation is now illegal because of the server 
// has been shutdown
var ErrServerClosed = errors.New("asynq: Server closed")
```

Impementation using asynq.HandlerFunc adapter

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/hibiken/asynq"
)

func main() {
	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: "localhost:6379"},
		asynq.Config{Concurrency: 10},
	)
	fmt.Printf("%+v", srv)

	// Use asynq.HandlerFunc adapter for a handler function
	if err := srv.Run(asynq.HandlerFunc(handler)); err != nil {
		log.Fatal(err)
	}
}

/* type Handler interface {
	// ProcessTask should return nil if the task was processed successfully
	// If ProcessTask returns a non-nil error or panics, the task will be retried again later
	ProcessTask(context.Context, *Task) error
} */

func handler(ctx context.Context, t *asynq.Task) error {
	switch t.Type() {
	case "email:welcome":
		var p EmailTaskPayload
		if err := json.Unmarshal(t.Payload(), &p); err != nil {
			return err
		}
		fmt.Printf(" [*] Send Welcome Email to User %d", p.UserID)

	case "email:reminder":
		var p EmailTaskPayload
		if err := json.Unmarshal(t.Payload(), &p); err != nil {
			return err
		}
		fmt.Printf(" [*] Send Reminder Email to User %d", p.UserID)

	default:
		return fmt.Errorf("unexpected task type: %s", t.Type())
	}
	return nil
}
```

 

Keys that are formed in redis when the worker server starts

```go
127.0.0.1:6379> keys *
1) "asynq:workers"
2) "deepak"
3) "asynq:servers"
4) "asynq:servers:{DESKTOP-KE1MMG0:8936:eb4e2f3b-0f38-4504-9475-84d2364299c8}"
127.0.0.1:6379> type "asynq:workers"
zset
127.0.0.1:6379> zrange "asynq:workers" 0 -1
1) "asynq:workers:{DESKTOP-KE1MMG0:8936:eb4e2f3b-0f38-4504-9475-84d2364299c8}"
127.0.0.1:6379> type "asynq:servers"
zset
127.0.0.1:6379> zrange "asynq:servers"
(error) ERR wrong number of arguments for 'zrange' command
127.0.0.1:6379> zrange "asynq:servers" 0 -1
1) "asynq:servers:{DESKTOP-KE1MMG0:8936:eb4e2f3b-0f38-4504-9475-84d2364299c8}"
127.0.0.1:6379> type asynq:servers:{DESKTOP-KE1MMG0:8936:eb4e2f3b-0f38-4504-9475-84d2364299c8}
string
127.0.0.1:6379> get asynq:servers:{DESKTOP-KE1MMG0:8936:eb4e2f3b-0f38-4504-9475-84d2364299c8}
"\n\x0fDESKTOP-KE1MMG0\x10\xe8E\x1a$eb4e2f3b-0f38-4504-9475-84d2364299c8 \n*\x0b\n\adefault\x10\x01:\x06activeB\x0b\b\xf4\xb5\xcd\xad\x06\x10\x90\xcd\xed="
127.0.0.1:6379>
```

We could keep adding switch cases to this handler function, but in a realistic application, it is convenient to define the logic for each case in a different function.

We can use `ServeMux` to create our handler. Just like the `ServeMux` from `net/http` package, you register a handler by calling the `Handle` or `HandleFunc.`  (HandleFunc internally calls Handle)  `ServeMux` satisfies the `[Handler](https://www.notion.so/Asynq-4657740a10be417d91b32c90e7de32bf?pvs=21)` interface so that you can pass it to `(*Server).Run`

Implementation of Handler Interface ⬇️

```go
// ProcessTask dispatches the task to the handler whose
// pattern most closely matches the task type. 

func (mux *ServeMux) ProcessTask(ctx context.Context, task *Task) error {
	h, _ := mux.Handler(task)
	return h.ProcessTask(ctx, task)
}
```

There are 5 things

1. `Handler` (interface)
    1. [https://www.notion.so/Asynq-4657740a10be417d91b32c90e7de32bf?pvs=4#bc2e3c5ddd6f4756a4a4f1cf03a9a311](https://www.notion.so/Asynq-4657740a10be417d91b32c90e7de32bf?pvs=21)
2. `Handler` (function called from [here](https://www.notion.so/Asynq-4657740a10be417d91b32c90e7de32bf?pvs=21))
    1. Handler returns the handler to use for the given task. It always returns a non-nil handler.
    2. Handler also returns the registered pattern that matches the task
    3. If there is no registered handler that applies to the task, handler returns a ‘not found’ handler which returns a error 
    4. Implementation 
    
    ```
    func (mux *ServeMux) Handler(t *Task) (h Handler, pattern string) {
    	mux.mu.RLock()
    	defer mux.mu.RUnlock()
    
    	h, pattern = mux.match(t.Type())
    	if h == nil {
    		h, pattern = NotFoundHandler(), ""
    	}
    	for i := len(mux.mws) - 1; i >= 0; i-- {
    		h = mux.mws[i](h)
    	}
    	return h, pattern
    }
    ```
    

1. `HandlerFunc` adapter
    1. [https://www.notion.so/Asynq-4657740a10be417d91b32c90e7de32bf?pvs=4#8105561c0f2644cab60f84ed424821ce](https://www.notion.so/Asynq-4657740a10be417d91b32c90e7de32bf?pvs=21)

1. `Handle` function
    1. Handle Registers the handler for the given pattern
    2. If a handler already exists for pattern, Handle panics
    3. Implementation
    
    ```go
    func (mux *ServeMux) Handle(pattern string, handler Handler) {
    	mux.mu.Lock()
    	defer mux.mu.Unlock()
    
    	if strings.TrimSpace(pattern) == "" {
    		panic("asynq: invalid pattern")
    	}
    	if handler == nil {
    		panic("asynq: nil handler")
    	}
    	if _, exist := mux.m[pattern]; exist {
    		panic("asynq: multiple registrations for " + pattern)
    	}
    
    	if mux.m == nil {
    		mux.m = make(map[string]muxEntry)
    	}
    	e := muxEntry{h: handler, pattern: pattern}
    	mux.m[pattern] = e
    	mux.es = appendSorted(mux.es, e)
    }
    ```
    

1. `HandleFunc` function
    1. Wrapper around [Handle](https://www.notion.so/Asynq-4657740a10be417d91b32c90e7de32bf?pvs=21)
    2. Internally calls [Handle](https://www.notion.so/Asynq-4657740a10be417d91b32c90e7de32bf?pvs=21)
    3. HandleFunc registers the handler function for the given pattern
    4. Internally also uses the [HandlerFunc](https://www.notion.so/Asynq-4657740a10be417d91b32c90e7de32bf?pvs=21) adapter
    5. Implementation
    
    ```
    func (mux *ServeMux) HandleFunc(pattern string, handler func(context.Context, *Task) error) {
    	if handler == nil {
    		panic("asynq: nil handler")
    	}
    	mux.Handle(pattern, HandlerFunc(handler))
    }
    ```
    

1. This also works
    1. You can use either HandleFunc or Handle
    
    ```go
    mux.HandleFunc("email:welcome", sendWelcomeEmail)
    mux.Handle("email:welcome", asynq.HandlerFunc(sendWelcomeEmail))
    ```
    

**ServeMux**

ServeMux is a multiplexer for asynchronous tasks.

~ A multiplexer in electronics

 
![Untitled](https://prod-files-secure.s3.us-west-2.amazonaws.com/9c2cca07-8579-43d0-b578-d16891f5ca43/7f729c0a-4456-46d1-ad6b-1434271a81b5/Untitled.png)
It matches the type of each task against a list of registered patterns 

and calls the handler for the pattern that most closely matches the task’s type name

[task type] → [pattern] → [handler for each pattern] 

Longer patterns take precedence over shorter ones, 

so that if there are handlers registered for both “`images`” and “`images:thumbnails`” patterns, 

- the `images:thumbnails`handler will be called for tasks with the type beginning with “`images:thumbanils`”
    
    [ it does not matter that the task type `images:thumbanils` contains the word `images` , `images` handler would not be called and only `images:thumbnails` handler would be called ]
    
- and the `images` handler will receive tasks with the type name beginning with “`images`”

```go
type ServeMux struct {
	mu  sync.RWMutex
	m   map[string]muxEntry
	es  []muxEntry // slice of entries sorted from longest to shortest.
	mws []MiddlewareFunc
}

// NewServeMux allocates and returns a new ServeMux
func NewServeMux() *ServeMux {
	return new(ServeMux)
}

// The new built-in function allocates memory. The first argument is a type, not a value,
//  and the value returned is a pointer to a newly allocated zero value of that type
func new(Type) *Type
```

Worker Program

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
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
	mux.HandleFunc("email:welcome", sendWelcomeEmail)
	/* 
	This also works
	mux.Handle("email:welcome", asynq.HandlerFunc(sendWelcomeEmail)) 
	*/
	mux.HandleFunc("email:reminder", sendReminderEmail)

	if err := srv.Run(mux); err != nil {
		log.Fatal(err)
	}
}

func sendWelcomeEmail(ctx context.Context, t *asynq.Task) error {
	var p EmailTaskPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return err
	}
	fmt.Printf(" [*] Send Welcome Email to User %d", p.UserID)
	return nil
}

func sendReminderEmail(ctx context.Context, t *asynq.Task) error {
	var p EmailTaskPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return err
	}
	fmt.Printf(" [*] Send Reminder Email to User %d", p.UserID)
	return nil
}
```

**Life of a Task**

Points

• Asynq supports multiple queues with different priorities, allowing you to manage the order in which tasks are processed.

- Note that the archive size is finite and once it reaches its max size, oldest tasks in the archive will be deleted

# Asynq CLI

Asynq CLI is a command line tool to monitor the queues and tasks managed by `asynq` package.

## Table of Contents

- [Installation](https://github.com/hibiken/asynq/blob/master/tools/asynq/README.md#installation)
- [Usage](https://github.com/hibiken/asynq/blob/master/tools/asynq/README.md#usage)
- [Config File](https://github.com/hibiken/asynq/blob/master/tools/asynq/README.md#config-file)

## Installation

In order to use the tool, compile it using the following command:

```
go install github.com/hibiken/asynq/tools/asynq

```

This will create the asynq executable under your `$GOPATH/bin` directory.

## Usage

### Commands

To view details on any command, use `asynq help <command> <subcommand>`.

- `asynq dash`
- `asynq stats`
- `asynq queue [ls inspect history rm pause unpause]`
- `asynq task [ls cancel delete archive run delete-all archive-all run-all]`
- `asynq server [ls]`

```

# with Docker (connect to a Redis server running on the host machine)
docker run --rm \
    --name asynqmon \
    -p 3000:3000 \
    hibiken/asynqmon --port=3000 --redis-addr=host.docker.internal:6380

# with Docker (connect to a Redis server running in the Docker container)
docker run --rm \
    --name asynqmon \
    --network dev-network \
    -p 8080:8080 \
    hibiken/asynqmon --redis-addr=dev-redis:6379
```

To use the defaults, simply run and open [http://localhost:8080](http://localhost:8080/).

```
# with a binary
./asynqmon

# with a docker image
docker run --rm \
    --name asynqmon \
    -p 8080:8080 \
    hibiken/asynqmon
```

By default, Asynqmon web server listens on port `8080` and connects to a Redis server running on `127.0.0.1:6379`

To see all available flags, run:

```
# with a binary
./asynqmon --help

# with a docker image
docker run hibiken/asynqmon --help
```

Here's the available flags:

*Note*: Use `--redis-url` to specify address, db-number, and password with one flag value; Alternatively, use `--redis-addr`, `--redis-db`, and `--redis-password` to specify each value.

| Flag | Env | Description | Default |
| --- | --- | --- | --- |
| --port(int) | PORT | port number to use for web ui server | 8080 |
| ---redis-url(string) | REDIS_URL | URL to redis or sentinel server. See https://pkg.go.dev/github.com/hibiken/asynq#ParseRedisURI for supported format | "" |
| --redis-addr(string) | REDIS_ADDR | address of redis server to connect to | "127.0.0.1:6379" |
| --redis-db(int) | REDIS_DB | redis database number | 0 |
| --redis-password(string) | REDIS_PASSWORD | password to use when connecting to redis server | "" |
| --redis-cluster-nodes(string) | REDIS_CLUSTER_NODES | comma separated list of host:port addresses of cluster nodes | "" |
| --redis-tls(string) | REDIS_TLS | server name for TLS validation used when connecting to redis server | "" |
| --redis-insecure-tls(bool) | REDIS_INSECURE_TLS | disable TLS certificate host checks | false |
| --enable-metrics-exporter(bool) | ENABLE_METRICS_EXPORTER | enable prometheus metrics exporter to expose queue metrics | false |
| --prometheus-addr(string) | PROMETHEUS_ADDR | address of prometheus server to query time series | "" |
| --read-only(bool) | READ_ONLY | use web UI in read-only mode | false |

### Connecting to Redis

To connect to a **single redis server**, use either `--redis-url` or (`--redis-addr`, `--redis-db`, and `--redis-password`).

Example:

```
$ ./asynqmon --redis-url=redis://:mypassword@localhost:6380/2

$ ./asynqmon --redis-addr=localhost:6380 --redis-db=2 --redis-password=mypassword
```

To connect to **redis-sentinels**, use `--redis-url`.

Example:

```
$ ./asynqmon --redis-url=redis-sentinel://:mypassword@localhost:5000,localhost:5001,localhost:5002?master=mymaster
```

To connect to a **redis-cluster**, use `--redis-cluster-nodes`.

Example:

```
$ ./asynqmon --redis-cluster-nodes=localhost:7000,localhost:7001,localhost:7002,localhost:7003,localhost:7004,localhost:7006
```

### 

*Note*: Use `--redis-url` to specify address, db-number, and password with one flag value; Alternatively, use `--redis-addr`, `--redis-db`, and `--redis-password` to specify each value.

| Flag | Env | Description | Default |
| --- | --- | --- | --- |
| --port(int) | PORT | port number to use for web ui server | 8080 |
| ---redis-url(string) | REDIS_URL | URL to redis or sentinel server. See https://pkg.go.dev/github.com/hibiken/asynq#ParseRedisURI for supported format | "" |
| --redis-addr(string) | REDIS_ADDR | address of redis server to connect to | "127.0.0.1:6379" |
| --redis-db(int) | REDIS_DB | redis database number | 0 |
| --redis-password(string) | REDIS_PASSWORD | password to use when connecting to redis server | "" |
| --redis-cluster-nodes(string) | REDIS_CLUSTER_NODES | comma separated list of host:port addresses of cluster nodes | "" |
| --redis-tls(string) | REDIS_TLS | server name for TLS validation used when connecting to redis server | "" |
| --redis-insecure-tls(bool) | REDIS_INSECURE_TLS | disable TLS certificate host checks | false |
| --enable-metrics-exporter(bool) | ENABLE_METRICS_EXPORTER | enable prometheus metrics exporter to expose queue metrics | false |
| --prometheus-addr(string) | PROMETHEUS_ADDR | address of prometheus server to query time series | "" |
| --read-only(bool) | READ_ONLY | use web UI in read-only mode | false |

### Connecting to Redis

To connect to a **single redis server**, use either `--redis-url` or (`--redis-addr`, `--redis-db`, and `--redis-password`).

Example:

```
$ ./asynqmon --redis-url=redis://:mypassword@localhost:6380/2

$ ./asynqmon --redis-addr=localhost:6380 --redis-db=2 --redis-password=mypassword
```

To connect to **redis-sentinels**, use `--redis-url`.

Example:

```
$ ./asynqmon --redis-url=redis-sentinel://:mypassword@localhost:5000,localhost:5001,localhost:5002?master=mymaster
```

To connect to a **redis-cluster**, use `--redis-cluster-nodes`.

Example:

```
$ ./asynqmon --redis-cluster-nodes=localhost:7000,localhost:7001,localhost:7002,localhost:7003,localhost:7004,localhost:7006
```

### 

[hibiken/asynqmon: Web UI for Asynq task queue (github.com)](https://github.com/hibiken/asynqmon?tab=readme-ov-file#readme)
