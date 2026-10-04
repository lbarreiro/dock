package update

import (
	"context"
	"dock/internal/models"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

var registryClient = &http.Client{Timeout: 20 * time.Second}

const manifestTypes = "application/vnd.oci.image.index.v1+json, application/vnd.docker.distribution.manifest.list.v2+json, application/vnd.oci.image.manifest.v1+json, application/vnd.docker.distribution.manifest.v2+json"

func tokenFor(ctx context.Context, ref Reference) (string, error) {
	host, service := "auth.docker.io", "registry.docker.io"
	if ref.Registry == "ghcr.io" {
		host, service = "ghcr.io", "ghcr.io"
	} else if ref.Registry != "docker.io" {
		return "", fmt.Errorf("unsupported registry %s", ref.Registry)
	}
	u := url.URL{Scheme: "https", Host: host, Path: "/token"}
	q := u.Query()
	q.Set("service", service)
	q.Set("scope", "repository:"+ref.Repository+":pull")
	u.RawQuery = q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", err
	}
	resp, err := registryClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("registry authentication returned %s", resp.Status)
	}
	var data struct {
		Token       string `json:"token"`
		AccessToken string `json:"access_token"`
	}
	if err = json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&data); err != nil {
		return "", err
	}
	if data.Token == "" {
		data.Token = data.AccessToken
	}
	if data.Token == "" {
		return "", fmt.Errorf("registry returned an empty token")
	}
	return data.Token, nil
}
func manifestRequest(ctx context.Context, ref Reference, version, method, token string) (*http.Response, error) {
	host := "registry-1.docker.io"
	if ref.Registry == "ghcr.io" {
		host = "ghcr.io"
	}
	u := "https://" + host + "/v2/" + ref.Repository + "/manifests/" + url.PathEscape(version)
	req, err := http.NewRequestWithContext(ctx, method, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", manifestTypes)
	resp, err := registryClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("registry returned %s", resp.Status)
	}
	return resp, nil
}

// ConfigDigest compares the remote image config to the ID actually used by the
// container, selecting the correct platform from a multi-architecture index.
func ConfigDigest(ctx context.Context, image string, platform models.Image) (string, error) {
	ref, err := ParseImage(image)
	if err != nil {
		return "", err
	}
	token, err := tokenFor(ctx, ref)
	if err != nil {
		return "", err
	}
	version := ref.Version
	for depth := 0; depth < 3; depth++ {
		resp, e := manifestRequest(ctx, ref, version, http.MethodGet, token)
		if e != nil {
			return "", e
		}
		var data struct {
			Config struct {
				Digest string `json:"digest"`
			} `json:"config"`
			Manifests []struct {
				Digest   string `json:"digest"`
				Platform struct {
					OS           string `json:"os"`
					Architecture string `json:"architecture"`
					Variant      string `json:"variant"`
				} `json:"platform"`
			} `json:"manifests"`
		}
		e = json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&data)
		resp.Body.Close()
		if e != nil {
			return "", e
		}
		if data.Config.Digest != "" {
			return data.Config.Digest, nil
		}
		version = ""
		for _, entry := range data.Manifests {
			if entry.Platform.OS == platform.OS && entry.Platform.Architecture == platform.Architecture && (platform.Variant == "" || entry.Platform.Variant == platform.Variant) {
				version = entry.Digest
				break
			}
		}
		if version == "" {
			return "", fmt.Errorf("no image manifest for %s/%s/%s", platform.OS, platform.Architecture, platform.Variant)
		}
	}
	return "", fmt.Errorf("registry returned a nested or invalid manifest")
}
func registryDigest(ctx context.Context, image string) (string, error) {
	ref, err := ParseImage(image)
	if err != nil {
		return "", err
	}
	token, err := tokenFor(ctx, ref)
	if err != nil {
		return "", err
	}
	resp, err := manifestRequest(ctx, ref, ref.Version, http.MethodHead, token)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	digest := resp.Header.Get("Docker-Content-Digest")
	if digest == "" {
		return "", fmt.Errorf("registry returned an empty digest")
	}
	return digest, nil
}

// AppliedConfigDigest normalizes Docker's two image stores to a config digest.
// Classic image IDs are config digests; containerd IDs identify manifests/indexes.
// Resolve the immutable descriptor of the applied image, never its mutable tag.
func AppliedConfigDigest(ctx context.Context, image string, local models.Image) (string, error) {
	if local.ManifestDigest == "" {
		if local.ID == "" {
			return "", fmt.Errorf("applied image has no identity")
		}
		return local.ID, nil
	}
	ref, err := ParseImage(image)
	if err != nil {
		return "", err
	}
	applied := ref.Registry + "/" + ref.Repository + "@" + local.ManifestDigest
	return ConfigDigest(ctx, applied, local)
}
