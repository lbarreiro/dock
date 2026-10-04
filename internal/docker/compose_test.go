package docker

import (
	"context"
	"encoding/json"
	"github.com/moby/moby/client"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func composeFixture(t *testing.T, initial, mode string) (*Client, string, string) {
	t.Helper()
	dir := t.TempDir()
	log := filepath.Join(dir, "calls")
	signal := filepath.Join(dir, "up")
	for _, name := range []string{"custom.yml", "override.yml"} {
		os.WriteFile(filepath.Join(dir, name), []byte("services: {}\n"), 0600)
	}
	script := `#!/bin/sh
printf '%s\n' "$*" >> "$DOCK_TEST_LOG"
case "$*" in
 *'config --format json'*) printf '{"services":{"app":{"image":"example/app:latest"}}}';;
 *'pull app'*) if [ "$DOCK_TEST_MODE" = pullfail ]; then echo 'pull rejected'; exit 1; fi;;
 *'up --detach'*) touch "$DOCK_TEST_UP"; if [ "$DOCK_TEST_MODE" = upfail ]; then echo 'create failed'; exit 1; fi;;
esac
`
	os.WriteFile(filepath.Join(dir, "docker"), []byte(script), 0700)
	t.Setenv("PATH", dir+":"+os.Getenv("PATH"))
	t.Setenv("DOCK_TEST_LOG", log)
	t.Setenv("DOCK_TEST_UP", signal)
	t.Setenv("DOCK_TEST_MODE", mode)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "/images/") {
			json.NewEncoder(w).Encode(map[string]any{"Id": "sha256:new", "Os": "linux", "Architecture": "arm64"})
			return
		}
		if strings.HasSuffix(r.URL.Path, "/containers/json") {
			json.NewEncoder(w).Encode([]map[string]any{{"Id": "new-container", "Names": []string{"/app"}, "State": "running", "Image": "example/app:latest"}})
			return
		}
		state := initial
		id := "old-container"
		image := "sha256:old"
		if strings.Contains(r.URL.Path, "new-container") {
			id = "new-container"
			image = "sha256:new"
			if initial == "exited" {
				state = "created"
			} else {
				state = "running"
			}
		}
		labels := map[string]string{"com.docker.compose.project": "original-project", "com.docker.compose.service": "app", "com.docker.compose.project.working_dir": dir, "com.docker.compose.project.config_files": filepath.Join(dir, "custom.yml") + "," + filepath.Join(dir, "override.yml")}
		containerState := map[string]any{"Status": state, "Running": state == "running"}
		if mode == "unhealthy" && id == "new-container" {
			containerState["Health"] = map[string]string{"Status": "unhealthy"}
		}
		json.NewEncoder(w).Encode(map[string]any{"Id": id, "Name": "/app", "Image": image, "Config": map[string]any{"Image": "example/app:latest", "Labels": labels}, "State": containerState})
	}))
	t.Cleanup(srv.Close)
	cli, err := client.NewClientWithOpts(client.WithHost(srv.URL), client.WithVersion("1.55"))
	if err != nil {
		t.Fatal(err)
	}
	return &Client{cli: cli}, log, dir
}
func TestComposeUsesOriginalFilesAndProjectAndVerifiesAppliedImage(t *testing.T) {
	c, log, dir := composeFixture(t, "running", "")
	if err := c.ComposeUpdate(context.Background(), "old-container"); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(log)
	calls := string(b)
	for _, want := range []string{"--project-name original-project", "--file " + filepath.Join(dir, "custom.yml"), "--file " + filepath.Join(dir, "override.yml"), "config --format json", "up --detach --no-deps --no-build --pull never app"} {
		if !strings.Contains(calls, want) {
			t.Fatal(want, calls)
		}
	}
}
func TestComposeStoppedServiceNeverStarts(t *testing.T) {
	c, log, _ := composeFixture(t, "exited", "")
	if err := c.ComposeUpdate(context.Background(), "old-container"); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(log)
	if !strings.Contains(string(b), "--no-start app") || strings.Contains(string(b), "stop app") {
		t.Fatal(string(b))
	}
}
func TestComposeAbortsOnMissingOriginalFileBeforePull(t *testing.T) {
	c, log, dir := composeFixture(t, "running", "")
	os.Remove(filepath.Join(dir, "custom.yml"))
	if err := c.ComposeUpdate(context.Background(), "old-container"); err == nil {
		t.Fatal("missing file accepted")
	}
	if _, err := os.Stat(log); !os.IsNotExist(err) {
		t.Fatal("command executed")
	}
}
func TestComposePullFailureNeverRecreates(t *testing.T) {
	c, log, _ := composeFixture(t, "running", "pullfail")
	if err := c.ComposeUpdate(context.Background(), "old-container"); err == nil || !strings.Contains(err.Error(), "pull rejected") {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(log)
	if strings.Contains(string(b), "up --detach") {
		t.Fatal(string(b))
	}
}
func TestComposeUnhealthyCannotComplete(t *testing.T) {
	c, _, _ := composeFixture(t, "running", "unhealthy")
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if err := c.ComposeUpdate(ctx, "old-container"); err == nil {
		t.Fatal("unhealthy accepted")
	}
}
func TestComposeRejectsUnstableStateBeforeCommands(t *testing.T) {
	c, log, _ := composeFixture(t, "paused", "")
	if err := c.ComposeUpdate(context.Background(), "old-container"); err == nil {
		t.Fatal("paused accepted")
	}
	if _, err := os.Stat(log); !os.IsNotExist(err) {
		t.Fatal("command executed")
	}
}
