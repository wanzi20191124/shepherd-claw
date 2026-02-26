package main

import "time"

type Task struct {
	ID         string    `json:"id"`
	AgentID    string    `json:"agent_id"`
	Prompt     string    `json:"prompt"`
	Status     string    `json:"status"` // pending/running/finished/error/cancelled
	Step       string    `json:"step"`
	Progress   int       `json:"progress"`
	StartTime  time.Time `json:"start_time"`
	LastUpdate time.Time `json:"last_update"`
}

type WorkerNode struct {
	ID   string `json:"id"`
	Addr string `json:"addr"`
}

type CommandRequest struct {
	Text string `json:"text"`
}

type EventRequest struct {
	TaskID   string `json:"task_id"`
	AgentID  string `json:"agent_id"`
	Status   string `json:"status"`
	Step     string `json:"step"`
	Progress int    `json:"progress"`
}

type RunRequest struct {
	TaskID string `json:"task_id"`
	Prompt string `json:"prompt"`
}

type CancelRequest struct {
	TaskID string `json:"task_id"`
}
