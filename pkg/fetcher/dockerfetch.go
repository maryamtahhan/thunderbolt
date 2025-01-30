package fetcher

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"

	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	v1 "github.com/google/go-containerregistry/pkg/v1"
)

type dockerFetcher struct{}

func (d *dockerFetcher) FetchImg(imgName string) (v1.Image, error) {
	// Initialize Docker client
	apiClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}
	defer apiClient.Close()

	images, err := apiClient.ImageList(context.Background(), image.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve Docker images: %w", err)
	}

	// Iterate through images and check for matching name
	for _, img := range images {
		for _, tag := range img.RepoTags {
			if tag == imgName {
				fmt.Printf("Found local image: %s\n", tag)
				//return &img, nil // Return the image if found
				// Use the Docker client to save the image to a tarball
				reader, err := apiClient.ImageSave(context.Background(), []string{imgName})
				if err != nil {
					return nil, fmt.Errorf("failed to save image: %v", err)
				}
				defer reader.Close()

				tmpDir, err := os.MkdirTemp("", "docker-cache-dir-")
				if err != nil {
					return nil, err
				}

				// Create a tarball file where the image will be saved
				tarballFilePath := path.Join(tmpDir, "tmp.tar")
				tarballFile, err := os.Create(tarballFilePath)
				if err != nil {
					return nil, fmt.Errorf("failed to create tarball file: %v", err)
				}
				defer tarballFile.Close()

				// Copy the content from the reader to the tarball file
				_, err = io.Copy(tarballFile, reader)
				if err != nil {
					return nil, fmt.Errorf("failed to copy data to tarball file: %v", err)
				}

				fmt.Printf("Saved image: %s\n", tarballFile.Name())
				return loadImageFromTarball(tarballFilePath)
			}
		}
	}

	return nil, fmt.Errorf("failed to create Docker client: %w", err)
}
