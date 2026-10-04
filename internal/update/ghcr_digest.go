package update

import (
	"context"
	"strings"
)

func GHCRDigest(image, tag string) (string, error) {
	if !strings.HasPrefix(image, "ghcr.io/") {
		image = "ghcr.io/" + image
	}
	return registryDigest(context.Background(), image+":"+tag)
}
