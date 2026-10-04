package api

import (
	"context"
	"dock/internal/models"
	"fmt"
	"net/http"
	"time"
)

type emergencyService interface {
	ListContainers(context.Context) ([]models.Container, error)
	Container(context.Context, string) (models.Container, error)
	StartContainer(context.Context, string) error
	StopContainer(context.Context, string) error
}
type EmergencyHandler struct {
	docker        emergencyService
	healthTimeout time.Duration
}
type EmergencyResult struct {
	Status           string   `json:"status"`
	Stopped          int      `json:"stopped"`
	EssentialRunning int      `json:"essential_running"`
	Errors           []string `json:"errors,omitempty"`
}

func NewEmergencyHandler(d emergencyService) *EmergencyHandler {
	return &EmergencyHandler{docker: d, healthTimeout: 90 * time.Second}
}

var requiredServices = []string{"dock", "cloudflared", "whatssend", "ntfy"}

func essentialName(c models.Container) string {
	if c.Self {
		return "dock"
	}
	for _, name := range requiredServices {
		if c.Name == name || c.Service == name {
			return name
		}
	}
	return ""
}
func (h *EmergencyHandler) ready(ctx context.Context, essential map[string]models.Container) error {
	for _, name := range requiredServices {
		c, err := h.docker.Container(ctx, essential[name].ID)
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		if !c.Ready() {
			return fmt.Errorf("%s not ready (state=%s, health=%s)", name, c.State, c.Health)
		}
	}
	return nil
}
func (h *EmergencyHandler) Activate(w http.ResponseWriter, r *http.Request) {
	// Finish the bounded operation even when a proxy being stopped disconnects HTTP.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	result := EmergencyResult{Status: "blocked"}
	fail := func(err error) {
		result.Errors = append(result.Errors, err.Error())
		writeJSON(w, http.StatusConflict, result)
	}
	containers, err := h.docker.ListContainers(ctx)
	if err != nil {
		fail(err)
		return
	}
	essential := map[string]models.Container{}
	for _, c := range containers {
		name := essentialName(c)
		if name == "" {
			continue
		}
		if _, exists := essential[name]; exists {
			fail(fmt.Errorf("multiple candidates for essential %s; nothing stopped", name))
			return
		}
		essential[name] = c
	}
	for _, name := range requiredServices {
		if _, ok := essential[name]; !ok {
			fail(fmt.Errorf("essential %s not found; nothing stopped", name))
			return
		}
	}
	for _, name := range requiredServices {
		c := essential[name]
		if c.State != "running" {
			if err = h.docker.StartContainer(ctx, c.ID); err != nil {
				fail(fmt.Errorf("cannot start %s: %w; nothing stopped", name, err))
				return
			}
		}
	}
	// All four must simultaneously be running and healthy when a healthcheck exists.
	healthCtx, healthCancel := context.WithTimeout(ctx, h.healthTimeout)
	defer healthCancel()
	var readinessError error
	for {
		readinessError = h.ready(healthCtx, essential)
		if readinessError == nil {
			break
		}
		timer := time.NewTimer(time.Second)
		select {
		case <-healthCtx.Done():
			timer.Stop()
			fail(fmt.Errorf("essential services not ready; nothing stopped: %w", readinessError))
			return
		case <-timer.C:
		}
	}
	result.EssentialRunning = len(requiredServices)
	for _, c := range containers {
		if essentialName(c) != "" {
			continue
		}
		// Re-read state: restarting/paused containers also need a real stop.
		current, e := h.docker.Container(ctx, c.ID)
		if e != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", c.Name, e))
			continue
		}
		if current.State != "running" && current.State != "restarting" && current.State != "paused" {
			continue
		}
		if err = h.ready(ctx, essential); err != nil {
			fail(fmt.Errorf("shutdown interrupted because an essential service lost readiness: %w", err))
			return
		}
		if err = h.docker.StopContainer(ctx, c.ID); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", c.Name, err))
			continue
		}
		after, e := h.docker.Container(ctx, c.ID)
		if e != nil || (after.State != "exited" && after.State != "created" && after.State != "dead") {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: stopped state not confirmed", c.Name))
			continue
		}
		result.Stopped++
	}
	if err = h.ready(ctx, essential); err != nil {
		fail(fmt.Errorf("final essential check: %w", err))
		return
	}
	result.Status = "completed"
	if len(result.Errors) > 0 {
		result.Status = "partial"
	}
	writeJSON(w, http.StatusOK, result)
}
