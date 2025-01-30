package imgbuild

import "fmt"

type ImageBuilder interface {
	CreateImage(imgName string, cacheDir string) error
}

type imgBuilder struct {
	builder ImageBuilder
}

// Factory function to create a new ImgBuilder with the specified backend.
func New() (ImageBuilder, error) {
	var builder ImageBuilder
	var builderType string

	if hasApp("buildah") {
		builderType = "buildah"
	} else if hasApp("docker") {
		builderType = "docker"
	}

	switch builderType {
	case "docker":
		builder = &dockerBuilder{}
	case "buildah":
		builder = &buildahBuilder{}
	default:
		return nil, fmt.Errorf("unsupported builder type: %s", builderType)
	}

	return &imgBuilder{builder: builder}, nil
}

func (i *imgBuilder) CreateImage(imgName, cacheDir string) error {
	return i.builder.CreateImage(imgName, cacheDir)
}
