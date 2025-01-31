package utils

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/gpuman/thunderbolt/pkg/constants"
	"k8s.io/klog/v2"
)

func FilePathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// HasApp checks if the host has a particular app installed and returns a boolean.
func HasApp(app string) bool {

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

func CleanupTmpDirs() error {
	tmpDirPrefixes := []string{
		constants.BuildahCacheDirPrefix,
		constants.DockerCacheDirPrefix,
		constants.PodmanCacheDirPrefix,
	}

	for _, prefix := range tmpDirPrefixes {
		cmd := exec.Command("rm", "-rf", "/tmp/"+prefix)

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to delete /tmp/%s: %w", prefix, err)
		}
	}

	klog.V(4).Info("Temporary directories successfully deleted.")
	return nil
}
