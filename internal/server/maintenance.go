package server

import (
	"html/template"
	"net/http"
	"path/filepath"

	"dock/internal/system"
	"dock/internal/update"
)

func (s *Server) Maintenance(w http.ResponseWriter, r *http.Request) {

	stats, err := system.GetStats()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	containers, err := s.docker.ListContainers(r.Context())
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

	maintenance := make([]MaintenanceContainer, 0, len(containers))

	for _, c := range containers {
		maintenance = append(maintenance, MaintenanceContainer{
			Name:     c.Name,
			Image:    c.Image,
			Provider: update.Provider(c.Image),
		})
	}

	data := HomeData{
		Version:      s.Version(),
		HeaderAction: "← Containers",
		HeaderLink:   "/",

		CPU:             stats.CPU,
		RAM:             stats.RAM,
		Temp:            stats.Temp,
		Uptime:          stats.Uptime,
		Running:         running,
		Total:           len(containers),
		Maintenance:     maintenance,
		MaintenancePage: true,
	}

	tmpl, err := template.ParseFiles(
		filepath.Join("web", "templates", "layout.html"),
		filepath.Join("web", "templates", "header.html"),
		filepath.Join("web", "templates", "maintenance.html"),
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := tmpl.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
