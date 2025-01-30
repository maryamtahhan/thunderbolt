package imgbuild

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/template"

	"k8s.io/klog/v2"
)

const DockerfileTemplate = `FROM scratch
LABEL org.opencontainers.image.title={{ .ImageTitle }}
COPY "{{ .CacheDir }}/" ./io.triton.cache/
`

type DockerfileData struct {
	ImageTitle string
	CacheDir   string
}

func generateDockerfile(imageName, CacheDir, outputPath string) error {

	parts := strings.Split(imageName, "/")
	fullImageName := parts[len(parts)-1]
	imageTitle := strings.Split(fullImageName, ":")[0]

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

	klog.Infof("Dockerfile generated successfully at %s", outputPath)
	return nil
}

/*
hasApp checks if the host has a particular app installed and returns a boolean.
It also executes the command: {app} --version and returns the version as
a string.
*/
func hasApp(app string) bool {

	path, err := exec.LookPath(app)
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return false
		}
		return false
	}
	if path == "" {
		return false
	}

	return true
}
