package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	taskStore = map[string]*Task{}
	mu        sync.RWMutex
)

func main() {
	rand.Seed(time.Now().UnixNano())

	workers := []WorkerNode{
		{ID: "macmini", Addr: "http://macmini.local:8080"},
		{ID: "laptop", Addr: "http://laptop.local:8080"},
	}

	r := gin.Default()

	r.POST("/command", func(c *gin.Context) {
		var req CommandRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		text := strings.TrimSpace(req.Text)
		switch {
		case strings.HasPrefix(text, "/run"):
			handleRunCommand(c, text, workers)
		case strings.HasPrefix(text, "/status"):
			handleStatusCommand(c)
		case strings.HasPrefix(text, "/stop"):
			handleStopCommand(c, text, workers)
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "unknown command"})
		}
	})

	r.POST("/event", func(c *gin.Context) {
		var req EventRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		mu.Lock()
		t, ok := taskStore[req.TaskID]
		if !ok {
			t = &Task{ID: req.TaskID, AgentID: req.AgentID, StartTime: time.Now()}
			taskStore[req.TaskID] = t
		}
		if t.StartTime.IsZero() {
			t.StartTime = time.Now()
		}
		t.AgentID = req.AgentID
		t.Status = req.Status
		t.Step = req.Step
		t.Progress = req.Progress
		t.LastUpdate = time.Now()
		mu.Unlock()

		fmt.Printf("[Agent: %s] Task: %s Status: %s Step: %s\n", req.AgentID, req.TaskID, req.Status, req.Step)
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	r.GET("/tasks", func(c *gin.Context) {
		mu.RLock()
		tasks := make([]*Task, 0, len(taskStore))
		for _, t := range taskStore {
			tasks = append(tasks, t)
		}
		mu.RUnlock()
		c.JSON(http.StatusOK, tasks)
	})

	if err := r.Run(":9000"); err != nil {
		panic(err)
	}
}

func handleRunCommand(c *gin.Context, text string, workers []WorkerNode) {
	args := strings.TrimSpace(strings.TrimPrefix(text, "/run"))
	if args == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing prompt"})
		return
	}

	agentID := ""
	prompt := args

	if strings.HasPrefix(args, "agent=") {
		parts := strings.SplitN(args, " ", 2)
		agentID = strings.TrimPrefix(parts[0], "agent=")
		if len(parts) == 2 {
			prompt = parts[1]
		} else {
			prompt = ""
		}
	}

	if prompt == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing prompt"})
		return
	}

	mu.RLock()
	selected := WorkerNode{}
	if agentID == "" {
		selected = SelectWorker(workers, taskStore)
	} else {
		for _, w := range workers {
			if w.ID == agentID {
				selected = w
				break
			}
		}
	}
	mu.RUnlock()

	if selected.ID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no worker available"})
		return
	}

	taskID := newTaskID()
	newTask := &Task{
		ID:         taskID,
		AgentID:    selected.ID,
		Prompt:     prompt,
		Status:     "pending",
		Step:       "dispatched",
		Progress:   0,
		StartTime:  time.Now(),
		LastUpdate: time.Now(),
	}

	mu.Lock()
	taskStore[taskID] = newTask
	mu.Unlock()

	if err := dispatchRun(selected, taskID, prompt); err != nil {
		mu.Lock()
		taskStore[taskID].Status = "error"
		taskStore[taskID].Step = err.Error()
		taskStore[taskID].LastUpdate = time.Now()
		mu.Unlock()
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	fmt.Printf("[Agent: %s] Task: %s Status: %s Step: %s\n", selected.ID, taskID, "pending", "dispatched")
	c.JSON(http.StatusOK, newTask)
}

func handleStatusCommand(c *gin.Context) {
	mu.RLock()
	resp := make([]*Task, 0, len(taskStore))
	for _, t := range taskStore {
		resp = append(resp, t)
	}
	mu.RUnlock()

	c.JSON(http.StatusOK, resp)
}

func handleStopCommand(c *gin.Context, text string, workers []WorkerNode) {
	args := strings.TrimSpace(strings.TrimPrefix(text, "/stop"))
	if args == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing taskID"})
		return
	}

	taskID := args

	mu.RLock()
	t, ok := taskStore[taskID]
	mu.RUnlock()
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}

	var worker WorkerNode
	for _, w := range workers {
		if w.ID == t.AgentID {
			worker = w
			break
		}
	}
	if worker.ID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "worker not found"})
		return
	}

	if err := dispatchCancel(worker, taskID); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func dispatchRun(worker WorkerNode, taskID, prompt string) error {
	body, _ := json.Marshal(RunRequest{TaskID: taskID, Prompt: prompt})
	url := fmt.Sprintf("%s/run", worker.Addr)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("worker responded %d", resp.StatusCode)
	}
	return nil
}

func dispatchCancel(worker WorkerNode, taskID string) error {
	body, _ := json.Marshal(CancelRequest{TaskID: taskID})
	url := fmt.Sprintf("%s/cancel", worker.Addr)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("worker responded %d", resp.StatusCode)
	}
	return nil
}

func newTaskID() string {
	return fmt.Sprintf("task-%d-%d", time.Now().UnixNano(), rand.Intn(10000))
}
