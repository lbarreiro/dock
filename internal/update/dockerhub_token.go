package update

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type dockerHubTokenResponse struct {
	Token string `json:"token"`
}

func DockerHubToken(image string) (string, error) {

	url := fmt.Sprintf(
		"https://auth.docker.io/token?service=registry.docker.io&scope=repository:%s:pull",
		image,
	)

	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var token dockerHubTokenResponse

	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return "", err
	}

	return token.Token, nil
}