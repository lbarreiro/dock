package update

import (
	"fmt"
	"net/http"
)

func GHCRDigest(image, tag string) (string, error) {

	token, err := GHCRToken(image)
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf(
		"https://ghcr.io/v2/%s/manifests/%s",
		image,
		tag,
	)

	req, err := http.NewRequest(http.MethodHead, url, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+token)

	req.Header.Set(
		"Accept",
		"application/vnd.oci.image.manifest.v1+json, application/vnd.docker.distribution.manifest.v2+json, application/vnd.oci.image.index.v1+json, application/vnd.docker.distribution.manifest.list.v2+json",
	)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("registry returned %s", resp.Status)
	}

	return resp.Header.Get("Docker-Content-Digest"), nil
}
