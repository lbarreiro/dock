package update

import "strings"

func SplitImage(image string) (string, string) {

	lastSlash := strings.LastIndex(image, "/")
	lastColon := strings.LastIndex(image, ":")

	if lastColon > lastSlash {
		return image[:lastColon], image[lastColon+1:]
	}

	return image, "latest"
}