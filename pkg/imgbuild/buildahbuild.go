package imgbuild

import (
	"context"
	"encoding/json"
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

	var allMetadata []CacheMetadataWithDummy

	// Export cacheDir into temporary dir
	tmpDir, err := os.MkdirTemp("", constants.BuildahCacheDirPrefix)
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	jsonFiles, err := preflightcheck.FindAllTritonCacheJSON(cacheDir)
	if err != nil {
		return fmt.Errorf("failed to find cache files: %w", err)
	}

	for _, jsonFile := range jsonFiles {
		data, ret := preflightcheck.GetTritonCacheJSONData(jsonFile)
		if ret != nil {
			return fmt.Errorf("failed to extract data from %s: %w", jsonFile, ret)
		}
		if data == nil {
			continue
		}

		dummyKey, ret := preflightcheck.ComputeDummyTritonKey(data)
		if ret != nil {
			return fmt.Errorf("failed to calculate dummy triton key for %s: %w", jsonFile, ret)
		}

		allMetadata = append(allMetadata, CacheMetadataWithDummy{
			Hash:       data.Hash,
			Backend:    data.Target.Backend,
			Arch:       preflightcheck.ConvertArchToString(data.Target.Arch),
			WarpSize:   data.Target.WarpSize,
			PTXVersion: data.PtxVersion,
			DummyKey:   dummyKey,
		})
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

	imageWithTag := fmt.Sprintf("%s:%s", imageName, "latest")

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

	metadataJSON, err := json.Marshal(allMetadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata for labels: %w", err)
	}

	builder.SetLabel("cache.triton.image/variant", "multi")
	builder.SetLabel("cache.triton.image/entry-count", strconv.Itoa(len(allMetadata)))
	builder.SetLabel("cache.triton.image/metadata", string(metadataJSON))
	addOptions := buildah.AddAndCopyOptions{}
	err = builder.Add("./io.triton.cache/", false, addOptions, tmpDir+"/.")
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

	cmd := exec.Command("cp", "-r", srcDir+"/.", dstDir)

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("error executing cp command: %v", err)
	}

	return nil
}
