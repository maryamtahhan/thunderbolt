package wrap

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/distribution/reference"
	dockerregistrytypes "github.com/docker/docker/api/types/registry"
	docker "github.com/fsouza/go-dockerclient"
	"github.com/openshift/imagebuilder"
	"github.com/openshift/imagebuilder/dockerclient"
	"k8s.io/klog"
)

// DockerfileTemplate defines the structure of the Dockerfile.
const DockerfileTemplate = `FROM scratch
LABEL org.opencontainers.image.title={{ .ImageTitle }}
COPY "{{ .CacheDir }}/" ./io.triton.cache/
`

type tritonCacheWrapper struct{}

type imgBuilder struct {
	wrapper TritonCacheWrapper
}

// TritonCacheExtractor extracts the Triton cache from an image.
type TritonCacheWrapper interface {
	CreateImage(imgName string, cacheDir string) error
}

// ImgBuilder wraps cache directories in an OCI image
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

// DockerfileData holds the dynamic values for the Dockerfile.
type DockerfileData struct {
	ImageTitle string
	CacheDir   string
}

// generateDockerfile generates a Dockerfile with the given parameters.
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
	// Generate the Dockerfile
	dockerfilePath := "./Dockerfile"
	options := dockerclient.NewClientExecutor(nil)

	// Extract the last part after the last '/' (the full image name with optional tag)
	parts := strings.Split(imageName, "/")
	fullImageName := parts[len(parts)-1]

	// Remove the tag if present
	title := strings.Split(fullImageName, ":")[0]

	err := generateDockerfile(title, cacheDir, dockerfilePath)
	if err != nil {
		return fmt.Errorf("failed to generate Dockerfile: %w", err)
	}
	defer os.Remove(dockerfilePath) // Cleanup Dockerfile after build

	options.Out, options.ErrOut = os.Stdout, os.Stderr
	authConfigurations, err := docker.NewAuthConfigurationsFromDockerCfg()
	options.AuthFn = func(name string) ([]dockerregistrytypes.AuthConfig, bool) {
		if authConfigurations != nil {
			if authConfig, ok := authConfigurations.Configs[name]; ok {
				klog.V(4).Infof("Found authentication secret for registry %q", name)
				return []dockerregistrytypes.AuthConfig{{
					Username:      authConfig.Username,
					Password:      authConfig.Password,
					Email:         authConfig.Email,
					ServerAddress: authConfig.ServerAddress,
				}}, true
			}
			if named, err := reference.ParseNormalizedNamed(name); err == nil {
				domain := reference.Domain(named)
				if authConfig, ok := authConfigurations.Configs[domain]; ok {
					klog.V(4).Infof("Found authentication secret for registry %q", domain)
					return []dockerregistrytypes.AuthConfig{{
						Username:      authConfig.Username,
						Password:      authConfig.Password,
						Email:         authConfig.Email,
						ServerAddress: authConfig.ServerAddress,
					}}, true
				}
				if domain == "docker.io" || strings.HasSuffix(domain, ".docker.io") {
					var auths []dockerregistrytypes.AuthConfig
					for _, aka := range []string{"docker.io", "index.docker.io", "https://index.docker.io/v1/"} {
						if aka == domain {
							continue
						}
						if authConfig, ok := authConfigurations.Configs[aka]; ok {
							klog.V(4).Infof("Found authentication secret for registry %q", aka)
							auths = append(auths, dockerregistrytypes.AuthConfig{
								Username:      authConfig.Username,
								Password:      authConfig.Password,
								Email:         authConfig.Email,
								ServerAddress: authConfig.ServerAddress,
							})
						}
					}
					if len(auths) > 0 {
						return auths, true
					}
				}
			}
		}
		return nil, false
	}
	options.LogFn = func(format string, args ...interface{}) {
		if klog.V(2) {
			log.Printf("Builder: "+format, args...)
		} else {
			fmt.Fprintf(options.Out, "--> %s\n", fmt.Sprintf(format, args...))
		}
	}

	arguments := stringMapFlag{}
	var imageFrom string
	var target string
	var tags stringSliceFlag

	tags.Set(imageName)
	if len(tags) > 0 {
		options.Tag = tags[0]
		options.AdditionalTags = tags[1:]
	}

	dockerfiles := filepath.SplitList(dockerfilePath)
	if len(dockerfiles) == 0 {
		dockerfiles = []string{filepath.Join(options.Directory, "Dockerfile")}
	}
	options.Directory = filepath.Dir(dockerfilePath)

	if err := build(dockerfiles[0], dockerfiles[1:], arguments, imageFrom, target, options); err != nil {
		log.Fatal(err.Error())
		return err
	}

	fmt.Println("Docker image built successfully")
	return nil

}

func build(dockerfile string, additionalDockerfiles []string, arguments map[string]string, from string, target string, e *dockerclient.ClientExecutor) error {
	if err := e.DefaultExcludes(); err != nil {
		return fmt.Errorf("error: Could not parse default .dockerignore: %v", err)
	}

	client, err := docker.NewClientFromEnv()
	if err != nil {
		return fmt.Errorf("error: No connection to Docker available: %v", err)
	}
	e.Client = client

	// TODO: handle signals
	defer func() {
		for _, err := range e.Release() {
			fmt.Fprintf(e.ErrOut, "error: Unable to clean up build: %v\n", err)
		}
	}()

	node, err := imagebuilder.ParseFile(dockerfile)
	if err != nil {
		return err
	}
	for _, s := range additionalDockerfiles {
		additionalNode, err := imagebuilder.ParseFile(s)
		if err != nil {
			return err
		}
		node.Children = append(node.Children, additionalNode.Children...)
	}

	b := imagebuilder.NewBuilder(arguments)
	stages, err := imagebuilder.NewStages(node, b)
	if err != nil {
		return err
	}
	stages, ok := stages.ByTarget(target)
	if !ok {
		return fmt.Errorf("error: The target %q was not found in the provided Dockerfile", target)
	}

	lastExecutor, err := e.Stages(b, stages, from)
	if err != nil {
		return err
	}

	return lastExecutor.Commit(stages[len(stages)-1].Builder)
}

type stringSliceFlag []string

func (f *stringSliceFlag) Set(s string) error {
	*f = append(*f, s)
	return nil
}

func (f *stringSliceFlag) String() string {
	return strings.Join(*f, " ")
}

type stringMapFlag map[string]string

func (f *stringMapFlag) String() string {
	args := []string{}
	for k, v := range *f {
		args = append(args, strings.Join([]string{k, v}, "="))
	}
	return strings.Join(args, " ")
}

func (f *stringMapFlag) Set(value string) error {
	kv := strings.Split(value, "=")
	(*f)[kv[0]] = kv[1]
	return nil
}
