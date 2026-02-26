package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

var (
	runningTasks = map[string]context.CancelFunc{}
	rmu         sync.RWMutex
)

func startTask(taskID, prompt string, orchestratorAddr, agentID string) {
	ctx, cancel := context.WithCancel(context.Background())

	rmu.Lock()
	runningTasks[taskID] = cancel
	rmu.Unlock()

	go func() {
		defer func() {
			rmu.Lock()
			delete(runningTasks, taskID)
			rmu.Unlock()
		}()

		reportEvent(orchestratorAddr, EventRequest{
			TaskID:   taskID,
			AgentID:  agentID,
			Status:   "running",
			Step:     "started",
			Progress: 0,
		})

		for i := 1; i <= 5; i++ {
			select {
			case <-ctx.Done():
				reportEvent(orchestratorAddr, EventRequest{
					TaskID:   taskID,
					AgentID:  agentID,
					Status:   "cancelled",
					Step:     "cancelled",
					Progress: i - 1,
				})
				return
			case <-time.After(2 * time.Second):
				reportEvent(orchestratorAddr, EventRequest{
					TaskID:   taskID,
					AgentID:  agentID,
					Status:   "running",
					Step:     fmt.Sprintf("step-%d", i),
					Progress: i,
				})
			}
		}

		reportEvent(orchestratorAddr, EventRequest{
			TaskID:   taskID,
			AgentID:  agentID,
			Status:   "finished",
			Step:     "done",
			Progress: 5,
		})
	}()
}

func cancelTask(taskID string) bool {
	rmu.RLock()
	cancel, ok := runningTasks[taskID]
	rmu.RUnlock()
	if !ok {
		return false
	}
	cancel()
	return true
}

func reportEvent(orchestratorAddr string, ev EventRequest) {
	body, _ := json.Marshal(ev)
	url := fmt.Sprintf("%s/event", orchestratorAddr)
	_, _ = http.Post(url, "application/json", bytes.NewBuffer(body))
}
