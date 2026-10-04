package update

import "context"

func DockerHubDigest(image, tag string) (string, error) {
	return registryDigest(context.Background(), image+":"+tag)
}
