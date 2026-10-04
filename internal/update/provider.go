package update

func Provider(image string) string {
	ref, err := ParseImage(image)
	if err != nil {
		return "Unsupported"
	}
	switch ref.Registry {
	case "docker.io":
		return "Docker Hub"
	case "ghcr.io":
		return "GHCR"
	default:
		return "Unsupported"
	}
}
