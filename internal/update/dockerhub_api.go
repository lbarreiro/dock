package update

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type dockerHubTag struct {
	Name string `json:"name"`
}

func DockerHubLatest(image string) (string, error) {

	url := fmt.Sprintf(
		"https://hub.docker.com/v2/repositories/%s/tags/latest",
		image,
	)

	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("docker hub returned %s", resp.Status)
	}

	var tag dockerHubTag

	if err := json.NewDecoder(resp.Body).Decode(&tag); err != nil {
		return "", err
	}

	return tag.Name, nil
}