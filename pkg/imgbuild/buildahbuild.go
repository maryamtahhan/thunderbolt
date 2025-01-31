package imgbuild

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/containers/buildah"
	"github.com/containers/common/pkg/config"
	is "github.com/containers/image/v5/storage"
	"github.com/containers/storage"
	"github.com/gpuman/thunderbolt/pkg/constants"
	"github.com/gpuman/thunderbolt/pkg/utils"
	"k8s.io/klog/v2"
)

type buildahBuilder struct{}

func (b *buildahBuilder) CreateImage(imageName, cacheDir string) error {
	// Export cacheDir into temporary dir
	tmpDir, err := os.MkdirTemp("", constants.BuildahCacheDirPrefix)
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)
	err = copyDir(cacheDir, tmpDir)
	if err != nil {
		return fmt.Errorf("error copying contents using cp: %v", err)
	}
	klog.V(4).Infof("%s", tmpDir)

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

	imageRef, err := is.Transport.ParseStoreReference(buildStore, imageName)
	if err != nil {
		return fmt.Errorf("error creating the image reference: %v", err)
	}

	// Define Buildah build options
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

	builder.SetAnnotation("module.triton.image/variant", "compat")

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
		return fmt.Errorf("error commiting the image: %v", err)
	}

	klog.Infof("Image built! %s\n", imageId)
	utils.CleanupTmpDirs()
	return nil
}

func copyDir(srcDir, dstDir string) error {

	cmd := exec.Command("cp", "-r", srcDir, dstDir)

	// Run the command
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("error executing cp command: %v", err)
	}

	return nil
}
