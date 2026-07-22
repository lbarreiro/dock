package api

import (
    "encoding/json"
    "net/http"

    "dock/internal/docker"
    "dock/internal/system"
)

type SystemHandler struct {
    docker *docker.Client
}

func NewSystemHandler(d *docker.Client) *SystemHandler {
    return &SystemHandler{
        docker: d,
    }
}

type SystemResponse struct {
    CPU     float64 `json:"cpu"`
    RAM     float64 `json:"ram"`
    Temp    float64 `json:"temp"`
    Uptime  string  `json:"uptime"`
    Running int     `json:"running"`
    Total   int     `json:"total"`
}

func (h *SystemHandler) Get(w http.ResponseWriter, r *http.Request) {

    stats, err := system.GetStats()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    containers, err := h.docker.ListContainers(r.Context())
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    var running int

    for _, c := range containers {
        if c.State == "running" {
            running++
        }
    }

    w.Header().Set("Content-Type", "application/json")

    json.NewEncoder(w).Encode(SystemResponse{
        CPU:     stats.CPU,
        RAM:     stats.RAM,
        Temp:    stats.Temp,
        Uptime:  stats.Uptime,
        Running: running,
        Total:   len(containers),
    })

}