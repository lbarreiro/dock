package server

import (
	"dock/internal/server/api"
)

func (s *Server) RegisterRoutes() {

	s.router.Get("/", s.Home)
	s.router.Get("/maintenance", s.Maintenance)

	containerHandler := api.NewContainerHandler(s.docker)
	systemHandler := api.NewSystemHandler(s.docker)
	operations := api.NewOperations()
	updateHandler := api.NewUpdateHandler(s.docker, operations)
	cleanupHandler := api.NewCleanupHandler()
	emergencyHandler := api.NewEmergencyHandler(s.docker)

	s.router.Get("/api/containers", containerHandler.List)
	s.router.Get("/api/system", systemHandler.Get)
	s.router.Get("/api/updates", updateHandler.Get)
	s.router.Get("/api/cleanup", cleanupHandler.Get)
	s.router.Post("/api/cleanup", operations.Guard(cleanupHandler.Clean))
	s.router.Post("/api/emergency", operations.Guard(emergencyHandler.Activate))
	s.router.Post("/api/update/{id}", updateHandler.Update)
	s.router.Get("/api/update/{id}/status", updateHandler.JobStatus)

	s.router.Get("/api/container/{id}", containerHandler.Get)
	s.router.Post("/api/container/{id}/toggle", operations.Guard(containerHandler.Toggle))
	s.router.Get("/api/container/{id}/logs", containerHandler.Logs)

}
