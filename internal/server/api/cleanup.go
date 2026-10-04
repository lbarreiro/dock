package api

import (
	"context"
	"encoding/json"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type CleanupHandler struct{}

type CleanupInfo struct {
	Images      string `json:"images"`
	Containers  string `json:"containers"`
	Volumes     string `json:"volumes"`
	BuildCache  string `json:"build_cache"`
	UnusedCount int    `json:"unused_count"`
	UnusedSize  string `json:"unused_size"`
}

func NewCleanupHandler() *CleanupHandler {
	return &CleanupHandler{}
}

func (h *CleanupHandler) Get(w http.ResponseWriter, r *http.Request) {
	count, size := unusedImages()

	info := CleanupInfo{
		Images:      dockerDF("Images"),
		Containers:  dockerDF("Containers"),
		Volumes:     dockerDF("Local Volumes"),
		BuildCache:  dockerDF("Build Cache"),
		UnusedCount: count,
		UnusedSize:  size,
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(info); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func dockerDF(kind string) string {
	out, err := exec.Command(
		"docker",
		"system",
		"df",
		"--format",
		"{{.Type}}|{{.TotalCount}}|{{.Active}}|{{.Size}}|{{.Reclaimable}}",
	).Output()

	if err != nil {
		return "unavailable"
	}

	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		fields := strings.Split(line, "|")

		if len(fields) == 5 && fields[0] == kind {
			return strings.Join(fields[1:], "|")
		}
	}

	return "0|0|0B|0B"
}

func unusedImages() (int, string) {
	out, err := exec.Command(
		"docker",
		"system",
		"df",
		"-v",
	).Output()

	if err != nil {
		return 0, "unavailable"
	}

	lines := strings.Split(string(out), "\n")

	inImages := false
	count := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if line == "Images space usage:" {
			inImages = true
			continue
		}

		if inImages && line == "Containers space usage:" {
			break
		}

		if !inImages || line == "" ||
			strings.HasPrefix(line, "REPOSITORY") {
			continue
		}

		fields := strings.Fields(line)

		if len(fields) < 2 {
			continue
		}

		// Última coluna da secção Images é CONTAINERS.
		if fields[len(fields)-1] == "0" {
			count++
		}
	}

	// O próprio Docker calcula o espaço reclaimable das Images.
	df := dockerDF("Images")
	parts := strings.Split(df, "|")

	if len(parts) != 4 {
		return count, "unavailable"
	}

	reclaimable := strings.Fields(parts[3])

	if len(reclaimable) == 0 {
		return count, "0B"
	}

	return count, reclaimable[0]
}

func parseDockerSize(size string) float64 {
	size = strings.TrimSpace(size)

	units := []struct {
		suffix     string
		multiplier float64
	}{
		{"GB", 1000 * 1000 * 1000},
		{"MB", 1000 * 1000},
		{"kB", 1000},
		{"B", 1},
	}

	for _, unit := range units {

		if strings.HasSuffix(size, unit.suffix) {

			value := strings.TrimSpace(
				strings.TrimSuffix(size, unit.suffix),
			)

			n, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return 0
			}

			return n * unit.multiplier
		}
	}

	return 0
}

func formatBytes(bytes float64) string {
	switch {

	case bytes >= 1000*1000*1000:
		return strconv.FormatFloat(
			bytes/(1000*1000*1000), 'f', 2, 64,
		) + "GB"

	case bytes >= 1000*1000:
		return strconv.FormatFloat(
			bytes/(1000*1000), 'f', 0, 64,
		) + "MB"

	case bytes >= 1000:
		return strconv.FormatFloat(
			bytes/1000, 'f', 0, 64,
		) + "kB"

	default:
		return strconv.FormatFloat(
			bytes, 'f', 0, 64,
		) + "B"
	}
}

type CleanupResult struct {
	Status string `json:"status"`
}

func (h *CleanupHandler) Clean(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	if out, err := exec.CommandContext(ctx,
		"docker",
		"image",
		"prune",
		"-a",
		"-f",
	).CombinedOutput(); err != nil {
		http.Error(
			w,
			"image cleanup failed: "+strings.TrimSpace(string(out)),
			http.StatusInternalServerError,
		)
		return
	}

	if out, err := exec.CommandContext(ctx,
		"docker",
		"builder",
		"prune",
		"-f",
	).CombinedOutput(); err != nil {
		http.Error(
			w,
			"build cache cleanup failed: "+strings.TrimSpace(string(out)),
			http.StatusInternalServerError,
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(CleanupResult{
		Status: "completed",
	})
}
