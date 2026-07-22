package update

import (
	"fmt"
	"net/http"
)

func DockerHubDigest(image, tag string) (string, error) {

	token, err := DockerHubToken(image)
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf(
		"https://registry-1.docker.io/v2/%s/manifests/%s",
		image,
		tag,
	)

	req, err := http.NewRequest(http.MethodHead, url, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set(
		"Authorization",
		"Bearer "+token,
	)

	req.Header.Set(
		"Accept",
		"application/vnd.docker.distribution.manifest.v2+json",
	)

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("registry returned %s", resp.Status)
	}

	return resp.Header.Get("Docker-Content-Digest"), nil
}