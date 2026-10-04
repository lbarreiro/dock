package update

import "context"

func DockerHubToken(image string) (string, error) {
	ref, err := ParseImage(image)
	if err != nil {
		return "", err
	}
	return tokenFor(context.Background(), ref)
}
