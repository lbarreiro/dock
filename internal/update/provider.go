package update

import "strings"

func Provider(image string) string {

	switch {

	case strings.HasPrefix(image, "ghcr.io/"):
		return "GHCR"

	case strings.HasPrefix(image, "quay.io/"):
		return "Quay"

	default:
		return "Docker Hub"
	}
}
