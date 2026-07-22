package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"dock/internal/docker"
	"dock/internal/update"

	"github.com/go-chi/chi/v5"
)

type UpdateHandler struct {
	docker docker.Service

	mu   sync.RWMutex
	jobs map[string]string
}

func NewUpdateHandler(d docker.Service) *UpdateHandler {
	return &UpdateHandler{
		docker: d,
		jobs:   make(map[string]string),
	}
}

type UpdateResponse struct {
	Status     string          `json:"status"`
	Containers []update.Result `json:"containers"`
}

func (h *UpdateHandler) setJob(name, status string) {
	h.mu.Lock()
	h.jobs[name] = status
	h.mu.Unlock()
}

func (h *UpdateHandler) JobStatus(w http.ResponseWriter, r *http.Request) {

	id := chi.URLParam(r, "id")

	h.mu.RLock()
	status, exists := h.jobs[id]
	h.mu.RUnlock()

	if !exists {
		status = "idle"
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(map[string]string{
		"status": status,
	})
}

func (h *UpdateHandler) Update(w http.ResponseWriter, r *http.Request) {

	id := chi.URLParam(r, "id")

	h.mu.RLock()
	current := h.jobs[id]
	h.mu.RUnlock()

	if current == "running" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status": "running",
		})
		return
	}

	h.setJob(id, "running")

	log.Printf("Update requested: %s", id)

	go func() {

		err := h.docker.ComposeUpdate(
			context.Background(),
			id,
		)

		if err != nil {
			log.Printf("Update failed: %s: %v", id, err)
			h.setJob(id, "error")
			return
		}

		log.Printf("Update completed: %s", id)
		h.setJob(id, "completed")

	}()

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(map[string]string{
		"status": "started",
	})
}

func (h *UpdateHandler) Get(w http.ResponseWriter, r *http.Request) {

	containers, err := h.docker.ListContainers(context.Background())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	results := make([]update.Result, 0, len(containers))

	for _, c := range containers {

		image, current := update.SplitImage(c.Image)
		provider := update.Provider(c.Image)

		result := update.Result{
			Name:     c.Name,
			Provider: provider,
			Image:    image,
			Current:  current,
			Status:   update.StatusChecking,
		}

		switch provider {

		case "Docker Hub", "GHCR":

			localDigest, err := h.docker.ImageDigest(
				context.Background(),
				c.Image,
			)

			if err != nil {
				result.Status = update.StatusError
				break
			}

			var remoteDigest string

			if provider == "Docker Hub" {
				remoteDigest, err = update.DockerHubDigest(image, current)
			} else {
				remoteDigest, err = update.GHCRDigest(image, current)
			}

			if err != nil {
				result.Status = update.StatusError
				result.Current = err.Error()
			} else if localDigest == remoteDigest {
				result.Status = update.StatusCurrent
			} else {
				result.Status = update.StatusUpdate
			}
		}

		results = append(results, result)
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(UpdateResponse{
		Status:     "done",
		Containers: results,
	})
}
