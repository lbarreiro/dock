package docker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/moby/moby/client"
)

// ComposeUpdate changes only the selected service, using its original project/files.
// Dock must be updated from outside its own container.
func (c *Client) ComposeUpdate(ctx context.Context, id string) error {
	current, err := c.Container(ctx, id)
	if err != nil {
		return err
	}
	if current.Self {
		return fmt.Errorf("update Dock from the host: its own process cannot safely recreate itself")
	}
	if current.State == "paused" || current.State == "restarting" {
		return fmt.Errorf("%s is %s; resolve its state before updating", current.Name, current.State)
	}
	in, err := c.cli.ContainerInspect(ctx, id, client.ContainerInspectOptions{})
	if err != nil {
		return err
	}
	labels := in.Container.Config.Labels
	dir := labels["com.docker.compose.project.working_dir"]
	project := labels["com.docker.compose.project"]
	files := strings.Split(labels["com.docker.compose.project.config_files"], ",")
	if dir == "" || project == "" || current.Service == "" || files[0] == "" {
		return fmt.Errorf("missing original Compose project, directory, service or files for %s", current.Name)
	}
	base := []string{"compose", "--project-directory", dir, "--project-name", project}
	for _, file := range files {
		if !filepath.IsAbs(file) {
			file = filepath.Join(dir, file)
		}
		st, e := os.Stat(file)
		if e != nil || !st.Mode().IsRegular() {
			return fmt.Errorf("original Compose file is unavailable: %s", file)
		}
		base = append(base, "--file", file)
	}
	run := func(args ...string) ([]byte, error) {
		cmd := exec.CommandContext(ctx, "docker", append(append([]string{}, base...), args...)...)
		cmd.Dir = dir
		var out []byte
		var e error
		if args[0] == "config" {
			var stderr bytes.Buffer
			cmd.Stderr = &stderr
			out, e = cmd.Output()
			if e != nil {
				out = append(out, stderr.Bytes()...)
			}
		} else {
			out, e = cmd.CombinedOutput()
		}
		if e != nil {
			return nil, fmt.Errorf("compose %s: %w: %s", args[0], e, strings.TrimSpace(string(out)))
		}
		return out, nil
	}
	raw, err := run("config", "--format", "json")
	if err != nil {
		return fmt.Errorf("validate configuration: %w", err)
	}
	var cfg struct {
		Services map[string]struct {
			Image string `json:"image"`
		} `json:"services"`
	}
	if err = json.Unmarshal(raw, &cfg); err != nil {
		return fmt.Errorf("invalid Compose configuration: %w", err)
	}
	service, ok := cfg.Services[current.Service]
	if !ok || service.Image == "" {
		return fmt.Errorf("selected service has no pullable image in its original Compose configuration")
	}
	if service.Image != current.Image {
		return fmt.Errorf("Compose image %q differs from running configuration %q; reconcile configuration before updating", service.Image, current.Image)
	}
	if _, err = run("pull", current.Service); err != nil {
		return err
	}
	expected, err := c.ImageInfo(ctx, service.Image)
	if err != nil {
		return fmt.Errorf("inspect pulled image: %w", err)
	}
	args := []string{"up", "--detach", "--no-deps", "--no-build", "--pull", "never"}
	active := current.State == "running"
	if !active {
		args = append(args, "--no-start")
	}
	args = append(args, current.Service)
	if _, err = run(args...); err != nil {
		return fmt.Errorf("recreation failed; check service state before retrying: %w", err)
	}
	// The old ID is gone after recreation; resolve by the original project/service labels.
	containers, err := c.ListContainers(ctx)
	if err != nil {
		return err
	}
	var target string
	for _, ctn := range containers {
		if ctn.Project == current.Project && ctn.Service == current.Service {
			if target != "" {
				return fmt.Errorf("multiple containers for selected service; verify replicas manually")
			}
			target = ctn.ID
		}
	}
	if target == "" {
		return fmt.Errorf("recreated container not found")
	}
	result, err := c.Container(ctx, target)
	if err != nil {
		return err
	}
	if result.ImageID != expected.ID {
		return fmt.Errorf("recreated service does not use the pulled image")
	}
	if !active {
		if result.State != "created" && result.State != "exited" {
			return fmt.Errorf("previously stopped service is unexpectedly %s", result.State)
		}
		return nil
	}
	readyCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	return c.WaitReady(readyCtx, target)
}

func (c *Client) WaitReady(ctx context.Context, id string) error {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		ctn, err := c.Container(ctx, id)
		if err != nil {
			return err
		}
		if ctn.Ready() {
			return nil
		}
		if ctn.State == "exited" || ctn.State == "dead" {
			return fmt.Errorf("%s is %s", ctn.Name, ctn.State)
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("readiness not confirmed for %s: %w", ctn.Name, ctx.Err())
		case <-ticker.C:
		}
	}
}
