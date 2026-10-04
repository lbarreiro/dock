package docker

import (
	"context"
	"io"

	"dock/internal/models"
)

// Service defines all Docker operations.
// The rest of the application communicates only
// through this interface.
type Service interface {
	Container(ctx context.Context, id string) (models.Container, error)
	ImageInfo(ctx context.Context, id string) (models.Image, error)
	ListContainers(ctx context.Context) ([]models.Container, error)

	ImageDigest(ctx context.Context, image string) (string, error)

	ComposeWorkingDir(ctx context.Context, id string) (string, error)
	ComposeUpdate(ctx context.Context, id string) error

	StartContainer(ctx context.Context, id string) error
	StopContainer(ctx context.Context, id string) error
	RestartContainer(ctx context.Context, id string) error

	ContainerLogs(ctx context.Context, id string, tail int) (io.ReadCloser, error)
}
