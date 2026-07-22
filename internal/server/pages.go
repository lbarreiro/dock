package server

import (
	"context"
	"html/template"
	"net/http"
	"path/filepath"

	"dock/internal/models"
	"dock/internal/system"
)

type HomeData struct {
	Version string

	HeaderAction string
	HeaderLink   string

	CPU     float64
	RAM     float64
	Temp    float64
	Uptime  string
	Running int
	Total   int

	Containers      []models.Container
	Maintenance     []MaintenanceContainer
	MaintenancePage bool
}

func (s *Server) Home(w http.ResponseWriter, r *http.Request) {

	containers, err := s.docker.ListContainers(context.Background())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	stats, err := system.GetStats()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	running := 0

	for _, c := range containers {
		if c.State == "running" {
			running++
		}
	}

	data := HomeData{
		Version:      s.Version(),
		HeaderAction: "Updates",
		HeaderLink:   "/maintenance",

		CPU:        stats.CPU,
		RAM:        stats.RAM,
		Temp:       stats.Temp,
		Uptime:     stats.Uptime,
		Running:    running,
		Total:      len(containers),
		Containers: containers,
	}

	tmpl, err := template.ParseFiles(
		filepath.Join("web", "templates", "layout.html"),
		filepath.Join("web", "templates", "header.html"),
		filepath.Join("web", "templates", "home.html"),
		filepath.Join("web", "templates", "container.html"),
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := tmpl.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
