package server

import (
        "net/http"
        "strconv"
        "time"

        "dock/internal/docker"

        "github.com/go-chi/chi/v5"
)

type Server struct {
        router  *chi.Mux
        docker  *docker.Client
        version string
}

func New() (*Server, error) {

        dockerClient, err := docker.New()
        if err != nil {
                return nil, err
        }

        s := &Server{
                router:  chi.NewRouter(),
                docker:  dockerClient,
                version: strconv.FormatInt(time.Now().Unix(), 10),
        }

        fs := http.FileServer(http.Dir("web/static"))
        s.router.Handle("/static/*", http.StripPrefix("/static/", fs))

        s.RegisterRoutes()

        return s, nil
}

func (s *Server) Version() string {
        return s.version
}

func (s *Server) Router() *chi.Mux {
        return s.router
}

func (s *Server) Listen(addr string) error {
        return http.ListenAndServe(addr, s.router)
}
