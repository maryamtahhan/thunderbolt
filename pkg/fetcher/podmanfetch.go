package fetcher

import (
	"context"
	"fmt"
	"os"
	"path"

	"github.com/containers/podman/v5/pkg/bindings"
	"github.com/containers/podman/v5/pkg/bindings/images"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/gpuman/thunderbolt/pkg/constants"
)

type podmanFetcher struct{}

func (p *podmanFetcher) FetchImg(imgName string) (v1.Image, error) {
	// Initialize Podman client
	ctx, err := bindings.NewConnection(context.Background(), "unix:/run/podman/podman.sock")
	if err != nil {
		return nil, fmt.Errorf("failed to create Podman client: %w", err)
	}

	options := images.ExistsOptions{}
	exists, err := images.Exists(ctx, imgName, &options)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve Podman images: %w", err)
	}

	if exists {
		tmpDir, err := os.MkdirTemp("", constants.PodmanCacheDirPrefix)
		if err != nil {
			return nil, err
		}

		// Create the tarball file
		tarballFilePath := path.Join(tmpDir, "tmp.tar")
		tarballFile, err := os.Create(tarballFilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to create tarball file: %w", err)
		}
		defer tarballFile.Close()

		// Use Export to save the image
		err = images.Export(ctx, []string{imgName}, tarballFile, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to export image: %v", err)
		}

		fmt.Printf("Saved image: %s\n", tarballFile.Name())
		return loadImageFromTarball(tarballFilePath)
	}

	return nil, fmt.Errorf("image not found: %s", imgName)
}
