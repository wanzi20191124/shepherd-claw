package main

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

type RunRequest struct {
	TaskID string `json:"task_id"`
	Prompt string `json:"prompt"`
}

type CancelRequest struct {
	TaskID string `json:"task_id"`
}

type EventRequest struct {
	TaskID   string `json:"task_id"`
	AgentID  string `json:"agent_id"`
	Status   string `json:"status"`
	Step     string `json:"step"`
	Progress int    `json:"progress"`
}

func main() {
	agentID := os.Getenv("WORKER_ID")
	if agentID == "" {
		agentID = "worker-1"
	}

	orchestratorAddr := os.Getenv("ORCHESTRATOR_ADDR")
	if orchestratorAddr == "" {
		orchestratorAddr = "http://127.0.0.1:9000"
	}

	r := gin.Default()

	r.POST("/run", func(c *gin.Context) {
		var req RunRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if req.TaskID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing task_id"})
			return
		}

		startTask(req.TaskID, req.Prompt, orchestratorAddr, agentID)
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	r.POST("/cancel", func(c *gin.Context) {
		var req CancelRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if req.TaskID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing task_id"})
			return
		}

		if !cancelTask(req.TaskID) {
			c.JSON(http.StatusNotFound, gin.H{"error": "task not running"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	r.GET("/health", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	if err := r.Run(":8080"); err != nil {
		panic(err)
	}
}
