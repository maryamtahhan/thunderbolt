package utils

import (
	"errors"
	"os"
	"os/exec"
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
