package update

import (
	"context"
	"dock/internal/models"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func response(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Status: http.StatusText(status), Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}
}
func TestReferenceParsing(t *testing.T) {
	for _, tc := range []struct{ input, registry, repo, version string }{
		{"alpine", "docker.io", "library/alpine", "latest"},
		{"ghcr.io/lbarreiro/dock:latest", "ghcr.io", "lbarreiro/dock", "latest"},
		{"example/app@sha256:" + strings.Repeat("a", 64), "docker.io", "example/app", "sha256:" + strings.Repeat("a", 64)},
		{"localhost:5000/app:v1", "localhost:5000", "app", "v1"},
	} {
		ref, e := ParseImage(tc.input)
		if e != nil || ref.Registry != tc.registry || ref.Repository != tc.repo || ref.Version != tc.version {
			t.Fatal(tc, ref, e)
		}
	}
}
func TestRegistrySelectsARM64ConfigAndCorrectGHCRScope(t *testing.T) {
	old := registryClient
	defer func() { registryClient = old }()
	paths := []string{}
	registryClient = &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
		paths = append(paths, r.URL.String())
		switch r.URL.Path {
		case "/token":
			if r.URL.Query().Get("scope") != "repository:lbarreiro/dock:pull" {
				t.Fatal(r.URL)
			}
			return response(200, `{"token":"test"}`), nil
		case "/v2/lbarreiro/dock/manifests/latest":
			return response(200, `{"manifests":[{"digest":"sha256:amd","platform":{"os":"linux","architecture":"amd64"}},{"digest":"sha256:arm","platform":{"os":"linux","architecture":"arm64","variant":"v8"}}]}`), nil
		case "/v2/lbarreiro/dock/manifests/sha256:arm":
			return response(200, `{"config":{"digest":"sha256:applied"}}`), nil
		default:
			t.Fatal(r.URL)
			return nil, nil
		}
	})}
	digest, e := ConfigDigest(context.Background(), "ghcr.io/lbarreiro/dock:latest", models.Image{OS: "linux", Architecture: "arm64"})
	if e != nil || digest != "sha256:applied" {
		t.Fatal(digest, e, paths)
	}
}
func TestRegistryRejectsFailedAndEmptyTokens(t *testing.T) {
	for _, tc := range []struct {
		code int
		body string
	}{{403, `{"message":"denied"}`}, {200, `{"token":""}`}} {
		old := registryClient
		registryClient = &http.Client{Transport: roundTrip(func(*http.Request) (*http.Response, error) { return response(tc.code, tc.body), nil })}
		_, err := DockerHubToken("alpine")
		registryClient = old
		if err == nil {
			t.Fatal(tc)
		}
	}
}
func TestRegistryRejectsManifestWithoutConfigOrMatchingPlatform(t *testing.T) {
	old := registryClient
	defer func() { registryClient = old }()
	registryClient = &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path == "/token" {
			return response(200, `{"token":"test"}`), nil
		}
		return response(200, `{"manifests":[]}`), nil
	})}
	if _, e := ConfigDigest(context.Background(), "alpine", models.Image{OS: "linux", Architecture: "arm64"}); e == nil {
		t.Fatal("empty manifest accepted")
	}
}
func TestUnsupportedRegistryExplicit(t *testing.T) {
	if Provider("quay.io/example/app:latest") != "Unsupported" {
		t.Fatal("misclassified")
	}
}
