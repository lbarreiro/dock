package docker

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"dock/internal/models"

	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/moby/moby/client"
)

type Client struct {
	cli *client.Client
}

func New() (*Client, error) {

	cli, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, err
	}

	return &Client{
		cli: cli,
	}, nil

}

func (c *Client) ListContainers(ctx context.Context) ([]models.Container, error) {

	result, err := c.cli.ContainerList(ctx, client.ContainerListOptions{
		All: true,
	})
	if err != nil {
		return nil, err
	}

	containers := make([]models.Container, 0, len(result.Items))

	for _, item := range result.Items {

		name := ""
		if len(item.Names) > 0 {
			name = strings.TrimPrefix(item.Names[0], "/")
		}

		ctn, err := c.Container(ctx, item.ID)
		if err != nil {
			return nil, fmt.Errorf("inspect %s: %w", name, err)
		}
		ctn.Status = item.Status
		containers = append(containers, ctn)

	}

	sort.Slice(containers, func(i, j int) bool {

		if containers[i].State != containers[j].State {

			if containers[i].State == "exited" {
				return true
			}

			if containers[j].State == "exited" {
				return false
			}

		}

		return strings.ToLower(containers[i].Name) < strings.ToLower(containers[j].Name)

	})

	return containers, nil

}

func (c *Client) StartContainer(ctx context.Context, id string) error {
	ctn, err := c.Container(ctx, id)
	if err != nil {
		return err
	}
	if ctn.State == "paused" {
		_, err = c.cli.ContainerUnpause(ctx, id, client.ContainerUnpauseOptions{})
		return err
	}
	if ctn.State == "running" || ctn.State == "restarting" {
		return nil
	}
	_, err = c.cli.ContainerStart(ctx, id, client.ContainerStartOptions{})
	return err
}

func (c *Client) StopContainer(ctx context.Context, id string) error {
	ctn, err := c.Container(ctx, id)
	if err != nil {
		return err
	}
	if ctn.State == "paused" {
		if _, err = c.cli.ContainerUnpause(ctx, id, client.ContainerUnpauseOptions{}); err != nil {
			return err
		}
	}
	_, err = c.cli.ContainerStop(ctx, id, client.ContainerStopOptions{})
	return err
}

func (c *Client) RestartContainer(ctx context.Context, id string) error {

	_, err := c.cli.ContainerRestart(ctx, id, client.ContainerRestartOptions{})

	return err

}

func (c *Client) ComposeWorkingDir(ctx context.Context, id string) (string, error) {

	inspect, err := c.cli.ContainerInspect(
		ctx,
		id,
		client.ContainerInspectOptions{},
	)

	if err != nil {
		return "", err
	}

	if inspect.Container.Config == nil {
		return "", nil
	}

	return inspect.Container.Config.Labels["com.docker.compose.project.working_dir"], nil

}

func (c *Client) ComposeService(ctx context.Context, id string) (string, error) {

	inspect, err := c.cli.ContainerInspect(
		ctx,
		id,
		client.ContainerInspectOptions{},
	)

	if err != nil {
		return "", err
	}

	if inspect.Container.Config == nil {
		return "", nil
	}

	return inspect.Container.Config.Labels["com.docker.compose.service"], nil

}

func (c *Client) ContainerLogs(ctx context.Context, id string, tail int) (io.ReadCloser, error) {

	reader, err := c.cli.ContainerLogs(ctx, id, client.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       strconv.Itoa(tail),
	})
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	var buf bytes.Buffer

	if _, err := stdcopy.StdCopy(&buf, &buf, reader); err != nil {
		return nil, err
	}

	return io.NopCloser(bytes.NewReader(buf.Bytes())), nil

}

func (c *Client) Container(ctx context.Context, id string) (models.Container, error) {
	result, err := c.cli.ContainerInspect(ctx, id, client.ContainerInspectOptions{})
	if err != nil {
		return models.Container{}, err
	}
	in := result.Container
	if in.Config == nil || in.State == nil {
		return models.Container{}, fmt.Errorf("incomplete inspect for %s", id)
	}
	name := strings.TrimPrefix(in.Name, "/")
	labels := in.Config.Labels
	state := string(in.State.Status)
	if in.State.Paused {
		state = "paused"
	} else if in.State.Restarting {
		state = "restarting"
	}
	hostname, _ := os.Hostname()
	self := name == "dock" || labels["com.docker.compose.service"] == "dock" || (len(hostname) >= 12 && strings.HasPrefix(in.ID, hostname))
	hasHealth := in.Config.Healthcheck != nil && len(in.Config.Healthcheck.Test) > 0 && in.Config.Healthcheck.Test[0] != "NONE"
	health := ""
	if in.State.Health != nil {
		health = string(in.State.Health.Status)
		hasHealth = true
	}
	return models.Container{ID: in.ID, Name: name, Image: in.Config.Image, ImageID: in.Image, State: state, Status: state, URL: labels["dock.url"], Service: labels["com.docker.compose.service"], Project: labels["com.docker.compose.project"], Health: health, HasHealthcheck: hasHealth, Self: self}, nil
}
