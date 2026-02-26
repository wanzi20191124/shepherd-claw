package main

import (
	"math/rand"
)

func SelectWorker(workers []WorkerNode, tasks map[string]*Task) WorkerNode {
	if len(workers) == 0 {
		return WorkerNode{}
	}

	minCount := -1
	candidates := make([]WorkerNode, 0, len(workers))

	for _, w := range workers {
		count := 0
		for _, t := range tasks {
			if t.AgentID == w.ID && (t.Status == "pending" || t.Status == "running") {
				count++
			}
		}

		if minCount == -1 || count < minCount {
			minCount = count
			candidates = []WorkerNode{w}
			continue
		}
		if count == minCount {
			candidates = append(candidates, w)
		}
	}

	return candidates[rand.Intn(len(candidates))]
}
