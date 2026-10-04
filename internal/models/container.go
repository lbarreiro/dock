package models

type Container struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Image          string `json:"image"`
	ImageID        string `json:"image_id"`
	State          string `json:"state"`
	Status         string `json:"status"`
	URL            string `json:"url"`
	Service        string `json:"service"`
	Project        string `json:"project"`
	Health         string `json:"health,omitempty"`
	HasHealthcheck bool   `json:"has_healthcheck"`
	Self           bool   `json:"self"`
}

func (c Container) Ready() bool {
	return c.State == "running" && (!c.HasHealthcheck || c.Health == "healthy")
}

type Image struct {
	ManifestDigest string
	ID             string
	OS             string
	Architecture   string
	Variant        string
}
