package update

import (
	"fmt"
	"github.com/distribution/reference"
)

type Reference struct {
	Registry, Repository, Version string
	Pinned                        bool
}

func ParseImage(image string) (Reference, error) {
	named, err := reference.ParseNormalizedNamed(image)
	if err != nil {
		return Reference{}, err
	}
	ref := Reference{Registry: reference.Domain(named), Repository: reference.Path(named), Version: "latest"}
	if tagged, ok := named.(reference.Tagged); ok {
		ref.Version = tagged.Tag()
	}
	if digested, ok := named.(reference.Digested); ok {
		ref.Version = digested.Digest().String()
		ref.Pinned = true
	}
	if ref.Registry == "index.docker.io" {
		ref.Registry = "docker.io"
	}
	if ref.Repository == "" {
		return Reference{}, fmt.Errorf("missing image repository")
	}
	return ref, nil
}
func SplitImage(image string) (string, string) {
	ref, err := ParseImage(image)
	if err != nil {
		return image, ""
	}
	return ref.Repository, ref.Version
}
