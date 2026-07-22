package update

type DockerHubChecker struct{}

func (DockerHubChecker) Name() string {
	return "Docker Hub"
}

func (DockerHubChecker) Check(image string) (Result, error) {

	return Result{
		Provider: "Docker Hub",
		Status:   StatusCurrent,
	}, nil
}