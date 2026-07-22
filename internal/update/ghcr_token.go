package update

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

func GHCRToken(image string) (string, error) {

	u := url.URL{
		Scheme: "https",
		Host:   "ghcr.io",
		Path:   "/token",
	}

	q := u.Query()
	q.Set("service", "ghcr.io")
	q.Set("scope", fmt.Sprintf("repository:%s:pull", image))
	u.RawQuery = q.Encode()

	resp, err := http.Get(u.String())
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var data struct {
		Token string `json:"token"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", err
	}

	return data.Token, nil
}
