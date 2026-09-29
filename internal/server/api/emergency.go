package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"dock/internal/docker"
)

type EmergencyHandler struct {
	docker *docker.Client
}

type EmergencyResult struct {
	Stopped          int      `json:"stopped"`
	EssentialRunning int      `json:"essential_running"`
	Errors           []string `json:"errors,omitempty"`
}

func NewEmergencyHandler(d *docker.Client) *EmergencyHandler {
	return &EmergencyHandler{docker: d}
}

func (h *EmergencyHandler) Activate(w http.ResponseWriter, r *http.Request) {
	containers, err := h.docker.ListContainers(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	essential := map[string]bool{
		"dock":       true,
		"cloudflared": true,
		"ntfy":       true,
		"whatssend":   true,
	}

	result := EmergencyResult{}
	found := make(map[string]bool, len(essential))

	// First guarantee that every essential service is running. Dock itself is
	// never stopped, so the request can complete even while the other
	// containers are being shut down.
	for _, container := range containers {
		name := strings.ToLower(container.Name)
		if !essential[name] {
			continue
		}

		found[name] = true

		if container.State != "running" {
			if err := h.docker.StartContainer(r.Context(), container.ID); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", container.Name, err))
				continue
			}
		}

		result.EssentialRunning++
	}

	for name := range essential {
		if !found[name] {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: container not found", name))
		}
	}

	// Stop everything that is not part of the emergency allowlist.
	for _, container := range containers {
		if essential[strings.ToLower(container.Name)] || container.State != "running" {
			continue
		}

		if err := h.docker.StopContainer(r.Context(), container.ID); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", container.Name, err))
			continue
		}

		result.Stopped++
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}
