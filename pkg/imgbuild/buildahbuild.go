package imgbuild

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"github.com/containers/buildah"
	"github.com/containers/common/pkg/config"
	is "github.com/containers/image/v5/storage"
	"github.com/containers/storage"
	"github.com/gpuman/thunderbolt/pkg/constants"
	"github.com/gpuman/thunderbolt/pkg/preflightcheck"
	"github.com/gpuman/thunderbolt/pkg/utils"
	logging "github.com/sirupsen/logrus"
)

type buildahBuilder struct{}

func (b *buildahBuilder) CreateImage(imageName, cacheDir string) error {
	// Export cacheDir into temporary dir
	tmpDir, err := os.MkdirTemp("", constants.BuildahCacheDirPrefix)
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)
	json, err := preflightcheck.FindTritonCacheJSON(cacheDir)
	if err != nil {
		// TODO CLEAN UP on failure
		return fmt.Errorf("failed to retrieve cache json file from %s: %w", cacheDir, err)
	}
	data, err := preflightcheck.GetTritonCacheJSONData(json)
	if err != nil {
		// TODO CLEAN UP on failure
		return fmt.Errorf("failed to retrieve cache data %s: %w", cacheDir, err)
	}
	err = copyDir(cacheDir, tmpDir)
	if err != nil {
		return fmt.Errorf("error copying contents using cp: %v", err)
	}
	logging.Debugf("%s", tmpDir)

	buildStoreOptions, _ := storage.DefaultStoreOptions()
	conf, err := config.Default()
	if err != nil {
		return fmt.Errorf("error configuring buildah: %v", err)
	}

	capabilitiesForRoot, err := conf.Capabilities("root", nil, nil)
	if err != nil {
		return fmt.Errorf("capabilitiesForRoot error: %v", err)
	}

	buildStore, err := storage.GetStore(buildStoreOptions)
	if err != nil {
		return fmt.Errorf("failed to init storage: %v", err)
	}
	defer buildStore.Shutdown(false)

	imageWithTag := fmt.Sprintf("%s:%s", imageName, data.Hash)

	imageRef, err := is.Transport.ParseStoreReference(buildStore, imageWithTag)
	if err != nil {
		return fmt.Errorf("error creating the image reference: %v", err)
	}

	builderOpts := buildah.BuilderOptions{
		Capabilities: capabilitiesForRoot,
		FromImage:    "scratch",
	}

	ctx := context.TODO()
	// Initialize Buildah
	builder, err := buildah.NewBuilder(ctx, buildStore, builderOpts)
	if err != nil {
		return fmt.Errorf("error creating Buildah builder: %v", err)
	}
	defer builder.Delete()

	builder.SetLabel("cache.triton.image/variant", "compat")
	builder.SetLabel("cache.triton.image/hash", data.Hash)
	builder.SetLabel("cache.triton.image/arch", preflightcheck.ConvertArchToString(data.Target.Arch))
	builder.SetLabel("cache.triton.image/backend", data.Target.Backend)
	builder.SetLabel("cache.triton.image/warp-size", strconv.Itoa(data.Target.WarpSize))
	if data.PtxVersion != nil && *data.PtxVersion != 0 {
		builder.SetLabel("cache.triton.image/ptx-version", strconv.Itoa(*data.PtxVersion))
	}

	addOptions := buildah.AddAndCopyOptions{}
	err = builder.Add("", false, addOptions, tmpDir)
	if err != nil {
		return fmt.Errorf("error adding %s to builder: %v", cacheDir, err)
	}

	commitOptions := buildah.CommitOptions{
		Squash: true,
	}

	imageId, _, _, err := builder.Commit(ctx, imageRef, commitOptions)
	if err != nil {
		return fmt.Errorf("error committing the image: %v", err)
	}

	logging.Infof("Image built! %s\n", imageId)
	utils.CleanupTmpDirs()
	return nil
}

func copyDir(srcDir, dstDir string) error {

	cmd := exec.Command("cp", "-r", srcDir, dstDir)

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("error executing cp command: %v", err)
	}

	return nil
}
