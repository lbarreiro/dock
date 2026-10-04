package api

import (
	"context"
	"crypto/rand"
	"dock/internal/docker"
	"dock/internal/update"
	"encoding/hex"
	"fmt"
	"github.com/go-chi/chi/v5"
	"log"
	"net/http"
	"regexp"
	"sync"
	"time"
)

type UpdateJob struct {
	ID        string    `json:"job"`
	Status    string    `json:"status"`
	Error     string    `json:"error,omitempty"`
	Container string    `json:"container"`
	Finished  time.Time `json:"-"`
}
type UpdateHandler struct {
	docker     docker.Service
	operations *Operations
	mu         sync.Mutex
	jobs       map[string]UpdateJob
	aliases    map[string]string
}

func NewUpdateHandler(d docker.Service, operations *Operations) *UpdateHandler {
	return &UpdateHandler{docker: d, operations: operations, jobs: map[string]UpdateJob{}, aliases: map[string]string{}}
}

type UpdateResponse struct {
	Status     string          `json:"status"`
	Containers []update.Result `json:"containers"`
}

var jobPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{8,100}$`)

func (h *UpdateHandler) JobStatus(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	defer h.mu.Unlock()
	key := r.URL.Query().Get("job")
	if key == "" {
		key = h.aliases[chi.URLParam(r, "id")]
	}
	job, exists := h.jobs[key]
	if !exists {
		writeJSON(w, http.StatusOK, map[string]string{"status": "unknown", "error": "No tracked job. Verify container state before submitting another update."})
		return
	}
	writeJSON(w, http.StatusOK, job)
}
func (h *UpdateHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	key := r.URL.Query().Get("job")
	if key == "" {
		var random [16]byte
		if _, err := rand.Read(random[:]); err != nil {
			writeJSON(w, 500, map[string]string{"error": err.Error()})
			return
		}
		key = hex.EncodeToString(random[:])
	}
	if !jobPattern.MatchString(key) {
		writeJSON(w, 400, map[string]string{"error": "Invalid job identifier"})
		return
	}
	h.mu.Lock()
	if job, ok := h.jobs[key]; ok {
		h.mu.Unlock()
		writeJSON(w, 200, job)
		return
	}
	h.mu.Unlock()
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	container, err := h.docker.Container(ctx, id)
	if err != nil {
		writeJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}
	if container.Self {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "Update Dock from the host; it cannot safely recreate its own process."})
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if old, ok := h.jobs[key]; ok {
		writeJSON(w, 200, old)
		return
	}
	if old, ok := h.jobs[h.aliases[container.ID]]; ok && old.Status == "running" {
		writeJSON(w, 200, old)
		return
	}
	if !h.operations.Try() {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "Another Docker operation is in progress; wait for it to finish."})
		return
	}
	// Keep recent results, but do not accumulate an unbounded history.
	for k, job := range h.jobs {
		if !job.Finished.IsZero() && time.Since(job.Finished) > 24*time.Hour {
			delete(h.jobs, k)
			for alias, v := range h.aliases {
				if v == k {
					delete(h.aliases, alias)
				}
			}
		}
	}
	job := UpdateJob{ID: key, Status: "running", Container: container.Name}
	h.jobs[key] = job
	h.aliases[id] = key
	h.aliases[container.ID] = key
	h.aliases[container.Name] = key
	go func() {
		defer h.operations.Done()
		updateCtx, finish := context.WithTimeout(context.Background(), 15*time.Minute)
		defer finish()
		log.Printf("Update requested: %s (job=%s)", container.Name, key)
		err := h.docker.ComposeUpdate(updateCtx, container.ID)
		h.mu.Lock()
		defer h.mu.Unlock()
		result := h.jobs[key]
		result.Finished = time.Now()
		if err != nil {
			result.Status = "error"
			result.Error = err.Error()
			log.Printf("Update failed: %s: %v", container.Name, err)
		} else {
			result.Status = "completed"
			log.Printf("Update completed: %s", container.Name)
		}
		h.jobs[key] = result
	}()
	writeJSON(w, http.StatusAccepted, job)
}
func (h *UpdateHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Minute)
	defer cancel()
	containers, err := h.docker.ListContainers(ctx)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	results := make([]update.Result, 0, len(containers))
	for _, c := range containers {
		image, version := update.SplitImage(c.Image)
		result := update.Result{Name: c.Name, Image: image, Current: version, Provider: update.Provider(c.Image), Status: update.StatusUnsupported, CanUpdate: !c.Self && c.Project != "" && c.Service != ""}
		if c.Self {
			result.Message = "Update Dock from the host"
		} else if !result.CanUpdate {
			result.Message = "Original Compose project is required"
		}
		if result.Provider == "Unsupported" {
			result.Message = "Registry is not supported"
			result.CanUpdate = false
			results = append(results, result)
			continue
		}
		local, e := h.docker.ImageInfo(ctx, c.ImageID)
		if e == nil {
			var remote string
			remote, e = update.ConfigDigest(ctx, c.Image, local)
			if e == nil {
				result.Status = update.StatusUpdate
				if local.ID == remote {
					result.Status = update.StatusCurrent
				}
			}
		}
		if e != nil {
			result.Status = update.StatusError
			result.Message = fmt.Sprintf("Unable to check image: %v", e)
		}
		results = append(results, result)
	}
	writeJSON(w, 200, UpdateResponse{Status: "done", Containers: results})
}
