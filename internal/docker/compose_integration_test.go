package docker

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Runs only on an isolated CI Docker daemon, never against the user's Pi.
func TestComposeIntegration(t *testing.T) {
	if os.Getenv("DOCK_INTEGRATION") != "1" {
		t.Skip("requires isolated Docker daemon")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	dir := t.TempDir()
	project := fmt.Sprintf("dock-test-%d", time.Now().UnixNano())
	file := filepath.Join(dir, "custom.yml")
	run := func(args ...string) string {
		t.Helper()
		out, e := exec.CommandContext(ctx, "docker", args...).CombinedOutput()
		if e != nil {
			t.Fatalf("docker %v: %v\n%s", args, e, out)
		}
		return strings.TrimSpace(string(out))
	}
	base := []string{"compose", "--project-directory", dir, "-p", project, "-f", file}
	compose := func(args ...string) string { return run(append(append([]string{}, base...), args...)...) }
	os.WriteFile(file, []byte("services:\n  db:\n    image: busybox:1.37\n    command: ['sleep', '600']\n  app:\n    image: busybox:1.37\n    command: ['sleep', '600']\n    depends_on: [db]\n    healthcheck:\n      test: ['CMD', 'true']\n      interval: 1s\n      timeout: 1s\n      retries: 5\n"), 0600)
	defer func() {
		cleanup, done := context.WithTimeout(context.Background(), 30*time.Second)
		defer done()
		_ = exec.CommandContext(cleanup, "docker", append(base, "down", "--volumes")...).Run()
	}()
	run("pull", "busybox:1.36")
	run("pull", "busybox:1.37")
	wanted := run("image", "inspect", "busybox:1.37", "--format", "{{.Id}}")
	run("tag", "busybox:1.36", "busybox:1.37")
	compose("up", "-d", "--pull", "never")
	old := compose("ps", "-q", "app")
	db := compose("ps", "-q", "db")
	c, err := New()
	if err != nil {
		t.Fatal(err)
	}
	if err = c.ComposeUpdate(ctx, old); err != nil {
		t.Fatal(err)
	}
	newID := compose("ps", "-q", "app")
	if old == newID {
		t.Fatal("image update did not recreate container")
	}
	result, err := c.Container(ctx, newID)
	if err != nil || result.ImageID != wanted || !result.Ready() {
		t.Fatal(result, err)
	}
	if compose("ps", "-q", "db") != db {
		t.Fatal("dependency recreated")
	}
	compose("stop", "app", "db")
	if err = c.ComposeUpdate(ctx, newID); err != nil {
		t.Fatal(err)
	}
	containers, err := c.ListContainers(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, ctn := range containers {
		if ctn.Project == project && ctn.State == "running" {
			t.Fatalf("stopped service/dependency started: %+v", ctn)
		}
	}
}
