package docker

import (
	"context"
	"strings"
)

func (c *Client) ImageDigest(ctx context.Context, image string) (string, error) {

	inspect, err := c.cli.ImageInspect(ctx, image)
	if err != nil {
		return "", err
	}

	if len(inspect.RepoDigests) == 0 {
		return inspect.ID, nil
	}

	digest := inspect.RepoDigests[0]

	if i := strings.Index(digest, "@"); i >= 0 {
		digest = digest[i+1:]
	}

	return digest, nil
}