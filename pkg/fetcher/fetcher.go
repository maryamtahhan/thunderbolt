package fetcher

import (
	"fmt"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/gpuman/thunderbolt/pkg/utils"
	"k8s.io/klog/v2"
)

type Fetcher interface {
	FetchImg(imgName string) (v1.Image, error)
}

type fetcher struct {
	local  Fetcher
	remote Fetcher
}

// Factory function to create a new Fetcher with the specified backend.
func NewFetcher() Fetcher {
	var localFetcher Fetcher
	var fetcherType string

	if utils.HasApp("podman") {
		fetcherType = "podman"
	} else if utils.HasApp("docker") {
		fetcherType = "docker"
	}

	switch fetcherType {
	case "docker":
		localFetcher = &dockerFetcher{}
	case "podman":
		localFetcher = &podmanFetcher{}
	default:
		localFetcher = nil
	}

	return &fetcher{local: localFetcher, remote: &remoteFetcher{}}
}

func (f *fetcher) FetchImg(imgName string) (v1.Image, error) {
	// Try fetching the image locally
	img, err := f.local.FetchImg(imgName)
	if img != nil {
		return img, nil
	}
	klog.V(4).Infof("couldn't retrieve the image locally %v, will try to retrieve from remote image registry", err)

	// If local fetch fails, try fetching the image remotely
	img, err = f.remote.FetchImg(imgName)
	if err != nil || img == nil {
		return nil, fmt.Errorf("failed to fetch image: %w", err)
	}

	return img, nil
}
