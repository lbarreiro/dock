package api

import (
    "encoding/json"
    "io"
    "net/http"

    "dock/internal/docker"

    "github.com/go-chi/chi/v5"
)

type ContainerHandler struct {
    docker *docker.Client
}

func NewContainerHandler(d *docker.Client) *ContainerHandler {
    return &ContainerHandler{
        docker: d,
    }
}

func (h *ContainerHandler) List(w http.ResponseWriter, r *http.Request) {

    containers, err := h.docker.ListContainers(r.Context())
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")

    encoder := json.NewEncoder(w)
    encoder.SetIndent("", "  ")

    if err := encoder.Encode(containers); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }

}

func (h *ContainerHandler) Get(w http.ResponseWriter, r *http.Request) {

    id := chi.URLParam(r, "id")

    containers, err := h.docker.ListContainers(r.Context())
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    for _, c := range containers {

        if c.ID != id {
            continue
        }

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(c)
        return
    }

    http.NotFound(w, r)

}

func (h *ContainerHandler) Toggle(w http.ResponseWriter, r *http.Request) {

    id := chi.URLParam(r, "id")

    containers, err := h.docker.ListContainers(r.Context())
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    for _, c := range containers {

        if c.ID != id {
            continue
        }

        if c.State == "running" {
            err = h.docker.StopContainer(r.Context(), id)
        } else {
            err = h.docker.StartContainer(r.Context(), id)
        }

        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }

        w.WriteHeader(http.StatusNoContent)
        return
    }

    http.NotFound(w, r)

}

func (h *ContainerHandler) Logs(w http.ResponseWriter, r *http.Request) {

    id := chi.URLParam(r, "id")

    reader, err := h.docker.ContainerLogs(r.Context(), id, 300)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    defer reader.Close()

    w.Header().Set("Content-Type", "text/plain; charset=utf-8")

    io.Copy(w, reader)

}
