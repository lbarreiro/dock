package docker

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os/exec"
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

		id := item.ID
		if len(id) > 12 {
			id = id[:12]
		}

		url := ""
		image := item.Image

		inspect, err := c.cli.ContainerInspect(
			ctx,
			item.ID,
			client.ContainerInspectOptions{},
		)

		if err == nil && inspect.Container.Config != nil {
			if inspect.Container.Config.Image != "" {
				image = inspect.Container.Config.Image
			}

			if inspect.Container.Config.Labels != nil {
				url = inspect.Container.Config.Labels["dock.url"]
			}
		}

		containers = append(containers, models.Container{
			ID:     id,
			Name:   name,
			Image:  image,
			State:  string(item.State),
			Status: item.Status,
			URL:    url,
		})

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

	_, err := c.cli.ContainerStart(ctx, id, client.ContainerStartOptions{})

	return err

}

func (c *Client) StopContainer(ctx context.Context, id string) error {

	_, err := c.cli.ContainerStop(ctx, id, client.ContainerStopOptions{})

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

func (c *Client) ComposeUpdate(ctx context.Context, id string) error {

	// Guardar o estado original do container.
	inspect, err := c.cli.ContainerInspect(
		ctx,
		id,
		client.ContainerInspectOptions{},
	)
	if err != nil {
		return err
	}

	wasRunning := false

	if inspect.Container.State != nil {
		wasRunning = inspect.Container.State.Running
	}

	dir, err := c.ComposeWorkingDir(ctx, id)
	if err != nil {
		return err
	}

	service, err := c.ComposeService(ctx, id)
	if err != nil {
		return err
	}

	if dir == "" {
		return fmt.Errorf("compose working directory not found")
	}

	if service == "" {
		return fmt.Errorf("compose service not found")
	}

	log.Printf(
		"Updating %s (service=%s, running=%t)",
		id,
		service,
		wasRunning,
	)

	// Atualizar apenas a imagem do serviço selecionado.
	pull := exec.CommandContext(
		ctx,
		"docker",
		"compose",
		"pull",
		service,
	)

	pull.Dir = dir

	out, err := pull.CombinedOutput()

	if err != nil {
		log.Printf("Pull failed for %s: %s", id, string(out))
		return fmt.Errorf("docker compose pull: %w", err)
	}

	// Recriar/arrancar o serviço com a nova imagem.
	up := exec.CommandContext(
		ctx,
		"docker",
		"compose",
		"up",
		"-d",
		service,
	)

	up.Dir = dir

	out, err = up.CombinedOutput()

	if err != nil {

		log.Printf("Compose up failed for %s: %s", id, string(out))

		// Se estava parado, tentar preservar o estado original
		// mesmo perante uma falha parcial no compose up.
		if !wasRunning {

			stop := exec.CommandContext(
				context.Background(),
				"docker",
				"compose",
				"stop",
				service,
			)

			stop.Dir = dir
			_, _ = stop.CombinedOutput()
		}

		return fmt.Errorf("docker compose up: %w", err)
	}

	// Se estava parado antes da atualização,
	// voltar a pará-lo depois da recriação.
	if !wasRunning {

		log.Printf(
			"%s was stopped before update; restoring stopped state",
			id,
		)

		stop := exec.CommandContext(
			ctx,
			"docker",
			"compose",
			"stop",
			service,
		)

		stop.Dir = dir

		out, err = stop.CombinedOutput()

		if err != nil {
			log.Printf("Compose stop failed for %s: %s", id, string(out))
			return fmt.Errorf("docker compose stop: %w", err)
		}

		log.Printf("%s restored to stopped state", id)
	}

	return nil
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
