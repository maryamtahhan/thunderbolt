/*
Copyright Red Hat Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package dockerbuild

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"text/template"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
)

// A quick list of TODOS:
// 1. make the interface generic
// 2. add buildah support

const DockerfileTemplate = `FROM scratch
LABEL org.opencontainers.image.title={{ .ImageTitle }}
COPY "{{ .CacheDir }}/" ./io.triton.cache/
`

type tritonCacheWrapper struct{}

type imgBuilder struct {
	wrapper TritonCacheWrapper
}

// TritonCacheWrapper wraps the Triton cache in a single layer OCI image.
type TritonCacheWrapper interface {
	CreateImage(imgName string, cacheDir string) error
}

// ImgBuilder wraps triton cache directories in an OCI image
type ImgBuilder interface {
	CreateCacheImage(imgName string, cacheDir string) error
}

// Factory function to create a new ImgMgr.
func New() ImgBuilder {
	return &imgBuilder{
		wrapper: &tritonCacheWrapper{},
	}
}

func (i *imgBuilder) CreateCacheImage(imgName, cacheDir string) error {
	return i.wrapper.CreateImage(imgName, cacheDir)
}

type DockerfileData struct {
	ImageTitle string
	CacheDir   string
}

func generateDockerfile(imageTitle, CacheDir, outputPath string) error {
	data := DockerfileData{
		ImageTitle: imageTitle,
		CacheDir:   CacheDir,
	}

	tmpl, err := template.New("dockerfile").Parse(DockerfileTemplate)
	if err != nil {
		return fmt.Errorf("error parsing template: %w", err)
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("error creating Dockerfile: %w", err)
	}
	defer file.Close()

	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("error executing template: %w", err)
	}

	fmt.Println("Dockerfile generated successfully at", outputPath)
	return nil
}

func (w *tritonCacheWrapper) CreateImage(imageName, cacheDir string) error {
	wd, _ := os.Getwd()

	// Generate the Dockerfile path
	dockerfilePath := fmt.Sprintf("%s/Dockerfile", wd)

	// Extract the last part after the last '/' (the full image name with optional tag)
	parts := strings.Split(imageName, "/")
	fullImageName := parts[len(parts)-1]

	// Remove the tag if present aka get the name of the image prior to the tag
	title := strings.Split(fullImageName, ":")[0]

	// Generate Dockerfile
	err := generateDockerfile(title, cacheDir, dockerfilePath)
	if err != nil {
		return fmt.Errorf("failed to generate Dockerfile: %w", err)
	}
	defer os.Remove(dockerfilePath)

	if _, err := os.Stat(dockerfilePath); os.IsNotExist(err) {
		return fmt.Errorf("dockerfile not found at %s", dockerfilePath)
	}

	// Initialize Docker client
	apiClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}

	// Create the build context (a tar archive of the context directory, including the Dockerfile)
	tar, err := archive.TarWithOptions(wd, &archive.TarOptions{IncludeSourceDir: true})
	if err != nil {
		return fmt.Errorf("error creating tar: w", err)
	}
	defer tar.Close()

	// Set up build options
	buildOptions := types.ImageBuildOptions{
		Dockerfile: "Dockerfile",        // Path to the Dockerfile
		Tags:       []string{imageName}, // Use the provided image name
		NoCache:    true,
		Remove:     false,
	}

	// Build the image
	buildResponse, err := apiClient.ImageBuild(context.Background(), tar, buildOptions)
	if err != nil {
		return fmt.Errorf("error building image: %w", err)
	}
	defer buildResponse.Body.Close()

	// TODO - Make this optional based on debug
	_, err = io.Copy(os.Stdout, buildResponse.Body)
	if err != nil {
		return fmt.Errorf("error reading build output: %w", err)
	}

	// Tag the
	err = apiClient.ImageTag(context.Background(), imageName, imageName+":latest")
	if err != nil {
		return fmt.Errorf("error tagging image: %w", err)
	}

	fmt.Println("Docker image built successfully")

	return nil
}
