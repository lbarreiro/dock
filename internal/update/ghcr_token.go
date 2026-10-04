package update

import (
	"context"
	"strings"
)

func GHCRToken(image string) (string, error) {
	if !strings.HasPrefix(image, "ghcr.io/") {
		image = "ghcr.io/" + image
	}
	ref, err := ParseImage(image)
	if err != nil {
		return "", err
	}
	return tokenFor(context.Background(), ref)
}
