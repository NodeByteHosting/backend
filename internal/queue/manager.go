package queue

import (
	"encoding/json"
	"time"

	"github.com/hibiken/asynq"
)

// Task types
const (
	TypeSyncFull        = "sync:full"
	TypeSyncLocations   = "sync:locations"
	TypeSyncNodes       = "sync:nodes"
	TypeSyncAllocations = "sync:allocations"
	TypeSyncNests       = "sync:nests"
	TypeSyncServers     = "sync:servers"
	TypeSyncDatabases   = "sync:databases"
	TypeSyncUsers       = "sync:users"

	TypeEmailSend = "email:send"
	TypeEmailBulk = "email:bulk"

	TypeWebhookDiscord = "webhook:discord"
	TypeWebhookSlack   = "webhook:slack"

	TypeCleanupLogs = "cleanup:logs"
)

// Queue names (for priority)
const (
	QueueCritical = "critical" // High priority (sync cancellations, urgent notifications)
	QueueDefault  = "default"  // Normal priority (most sync operations)
	QueueLow      = "low"      // Low priority (cleanup, non-urgent tasks)
)

// Manager handles task enqueueing
type Manager struct {
	client *asynq.Client
}

// NewManager creates a new queue manager
func NewManager(client *asynq.Client) *Manager {
	return &Manager{client: client}
}

// Client returns the underlying Asynq client for direct task enqueueing
func (m *Manager) Client() *asynq.Client {
	return m.client
}

// SyncFullPayload contains data for a full sync task
type SyncFullPayload struct {
	SyncLogID   string `json:"sync_log_id"`
	RequestedBy string `json:"requested_by,omitempty"`
	SkipUsers   bool   `json:"skip_users,omitempty"`
}

// SyncPayload contains data for individual sync tasks
type SyncPayload struct {
	SyncLogID string `json:"sync_log_id"`
	ParentID  string `json:"parent_id,omitempty"` // Parent sync log if part of full sync
}

// Specific sync payloads for type-safe enqueueing
type SyncLocationsPayload struct {
	SyncLogID string `json:"sync_log_id"`
}

type SyncNodesPayload struct {
	SyncLogID string `json:"sync_log_id"`
}

type SyncAllocationsPayload struct {
	SyncLogID string `json:"sync_log_id"`
}

type SyncNestsPayload struct {
	SyncLogID string `json:"sync_log_id"`
}

type SyncServersPayload struct {
	SyncLogID string `json:"sync_log_id"`
}

type SyncDatabasesPayload struct {
	SyncLogID string `json:"sync_log_id"`
}

type SyncUsersPayload struct {
	SyncLogID string `json:"sync_log_id"`
}

// EmailPayload contains data for sending an email
type EmailPayload struct {
	To       string            `json:"to"`
	Subject  string            `json:"subject"`
	Template string            `json:"template"`
	Data     map[string]string `json:"data,omitempty"`
}

// WebhookPayload contains data for sending a webhook
type WebhookPayload struct {
	WebhookID string                 `json:"webhook_id"`
	Event     string                 `json:"event"`
	Data      map[string]interface{} `json:"data"`
}

// EnqueueSyncFull enqueues a full sync task
func (m *Manager) EnqueueSyncFull(payload SyncFullPayload) (*asynq.TaskInfo, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	task := asynq.NewTask(TypeSyncFull, data,
		asynq.Queue(QueueDefault),
		asynq.MaxRetry(3),
		asynq.Timeout(30*time.Minute),
		asynq.Unique(10*time.Minute), // Prevent duplicate syncs
	)

	return m.client.Enqueue(task)
}

// EnqueueSyncLocations enqueues a locations sync task
func (m *Manager) EnqueueSyncLocations(payload SyncPayload) (*asynq.TaskInfo, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	task := asynq.NewTask(TypeSyncLocations, data,
		asynq.Queue(QueueDefault),
		asynq.MaxRetry(3),
		asynq.Timeout(5*time.Minute),
	)

	return m.client.Enqueue(task)
}

// EnqueueSyncNodes enqueues a nodes sync task
func (m *Manager) EnqueueSyncNodes(payload SyncPayload) (*asynq.TaskInfo, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	task := asynq.NewTask(TypeSyncNodes, data,
		asynq.Queue(QueueDefault),
		asynq.MaxRetry(3),
		asynq.Timeout(5*time.Minute),
	)

	return m.client.Enqueue(task)
}

// EnqueueSyncServers enqueues a servers sync task
func (m *Manager) EnqueueSyncServers(payload SyncPayload) (*asynq.TaskInfo, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	task := asynq.NewTask(TypeSyncServers, data,
		asynq.Queue(QueueDefault),
		asynq.MaxRetry(3),
		asynq.Timeout(15*time.Minute), // Servers can take longer
	)

	return m.client.Enqueue(task)
}

// EnqueueSyncUsers enqueues a users sync task
func (m *Manager) EnqueueSyncUsers(payload SyncPayload) (*asynq.TaskInfo, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	task := asynq.NewTask(TypeSyncUsers, data,
		asynq.Queue(QueueDefault),
		asynq.MaxRetry(3),
		asynq.Timeout(15*time.Minute),
	)

	return m.client.Enqueue(task)
}

// EnqueueSyncAllocations enqueues an allocations sync task
func (m *Manager) EnqueueSyncAllocations(payload SyncPayload) (*asynq.TaskInfo, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	task := asynq.NewTask(TypeSyncAllocations, data,
		asynq.Queue(QueueDefault),
		asynq.MaxRetry(3),
		asynq.Timeout(10*time.Minute),
	)
	return m.client.Enqueue(task)
}

// EnqueueSyncNests enqueues a nests and eggs sync task
func (m *Manager) EnqueueSyncNests(payload SyncPayload) (*asynq.TaskInfo, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	task := asynq.NewTask(TypeSyncNests, data,
		asynq.Queue(QueueDefault),
		asynq.MaxRetry(3),
		asynq.Timeout(5*time.Minute),
	)
	return m.client.Enqueue(task)
}

// EnqueueSyncDatabases enqueues a databases sync task
func (m *Manager) EnqueueSyncDatabases(payload SyncPayload) (*asynq.TaskInfo, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	task := asynq.NewTask(TypeSyncDatabases, data,
		asynq.Queue(QueueDefault),
		asynq.MaxRetry(3),
		asynq.Timeout(15*time.Minute),
	)
	return m.client.Enqueue(task)
}

// EnqueueEmail enqueues an email send task
func (m *Manager) EnqueueEmail(payload EmailPayload) (*asynq.TaskInfo, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	task := asynq.NewTask(TypeEmailSend, data,
		asynq.Queue(QueueDefault),
		asynq.MaxRetry(5),
		asynq.Timeout(30*time.Second),
	)

	return m.client.Enqueue(task)
}

// EnqueueWebhook enqueues a webhook dispatch task
func (m *Manager) EnqueueWebhook(payload WebhookPayload) (*asynq.TaskInfo, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	task := asynq.NewTask(TypeWebhookDiscord, data,
		asynq.Queue(QueueDefault),
		asynq.MaxRetry(3),
		asynq.Timeout(10*time.Second),
	)

	return m.client.Enqueue(task)
}

// EnqueueCleanupLogs enqueues a log cleanup task
func (m *Manager) EnqueueCleanupLogs(olderThanDays int) (*asynq.TaskInfo, error) {
	data, _ := json.Marshal(map[string]int{"older_than_days": olderThanDays})

	task := asynq.NewTask(TypeCleanupLogs, data,
		asynq.Queue(QueueLow),
		asynq.MaxRetry(1),
		asynq.Timeout(5*time.Minute),
	)

	return m.client.Enqueue(task)
}

// GetTaskInfo returns information about a specific task
func (m *Manager) GetTaskInfo(queueName, taskID string) (*asynq.TaskInfo, error) {
	inspector := asynq.NewInspector(asynq.RedisClientOpt{
		Addr: "localhost:6379", // TODO: Get from config
	})
	return inspector.GetTaskInfo(queueName, taskID)
}
