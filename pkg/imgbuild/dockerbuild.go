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
package imgbuild

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/gpuman/thunderbolt/pkg/preflightcheck"
	"github.com/gpuman/thunderbolt/pkg/utils"
	logging "github.com/sirupsen/logrus"
)

type dockerBuilder struct{}

// Docker implementation of the ImageBuilder interface.
func (d *dockerBuilder) CreateImage(imageName, cacheDir string) error {
	wd, _ := os.Getwd()
	dockerfilePath := fmt.Sprintf("%s/Dockerfile", wd)

	json, err := preflightcheck.FindTritonCacheJSON(cacheDir)
	if err != nil {
		// TODO CLEAN UP on failure
		return fmt.Errorf("failed to retrieve cache json file from %s: %w", cacheDir, err)
	}
	jsondata, err := preflightcheck.GetTritonCacheJSONData(json)
	if err != nil {
		// TODO CLEAN UP on failure
		return fmt.Errorf("failed to retrieve cache data %s: %w", cacheDir, err)
	}

	err = generateDockerfile(imageName, cacheDir, dockerfilePath)
	if err != nil {
		return fmt.Errorf("failed to generate Dockerfile: %w", err)
	}
	defer os.Remove(dockerfilePath)

	if _, err := os.Stat(dockerfilePath); os.IsNotExist(err) {
		return fmt.Errorf("dockerfile not found at %s", dockerfilePath)
	}

	apiClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}
	tar, err := archive.TarWithOptions(wd, &archive.TarOptions{IncludeSourceDir: true})
	if err != nil {
		return fmt.Errorf("error creating tar: %w", err)
	}
	defer tar.Close()

	labels := map[string]string{
		"cache.triton.image/variant":   "compat",
		"cache.triton.image/hash":      jsondata.Hash,
		"cache.triton.image/arch":      string(jsondata.Target.Arch),
		"cache.triton.image/backend":   jsondata.BackendName,
		"cache.triton.image/warp-size": strconv.Itoa(jsondata.Target.WarpSize),
	}

	if jsondata.PtxVersion != nil && *jsondata.PtxVersion != 0 {
		labels["cache.triton.image/ptx-version"] = strconv.Itoa(*jsondata.PtxVersion)
	}

	buildOptions := types.ImageBuildOptions{
		Dockerfile: "Dockerfile",
		Tags:       []string{imageName},
		NoCache:    true,
		Remove:     false,
		Labels:     labels,
	}

	buildResponse, err := apiClient.ImageBuild(context.Background(), tar, buildOptions)
	if err != nil {
		return fmt.Errorf("error building image: %w", err)
	}
	defer buildResponse.Body.Close()

	_, err = io.Copy(os.Stdout, buildResponse.Body)
	if err != nil {
		return fmt.Errorf("error reading build output: %w", err)
	}

	imageWithTag := fmt.Sprintf("%s:%s", imageName, jsondata.Hash)
	err = apiClient.ImageTag(context.Background(), imageName, imageWithTag)
	if err != nil {
		return fmt.Errorf("error tagging image: %w", err)
	}

	utils.CleanupTmpDirs()
	logging.Info("Docker image built successfully")
	return nil
}
