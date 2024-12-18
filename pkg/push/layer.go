package push

import (
	"compress/gzip"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Layer represents the layer data and its properties.
type Layer struct {
	Digest    string
	MediaType string
	Size      int64
}

// CreateLayerFromCache creates a single OCI layer from the cache directory.
func CreateLayerFromCache(cacheDir string) (*Layer, error) {
	layerTarball, err := os.CreateTemp("", "layer-*.tar.gz")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp layer file: %v", err)
	}
	defer layerTarball.Close()

	// Create a tarball from the cache directory
	err = tarDirectory(cacheDir, layerTarball)
	if err != nil {
		return nil, fmt.Errorf("failed to tar directory: %v", err)
	}

	// Compute the SHA256 digest of the tarball
	digest, err := computeDigest(layerTarball.Name())
	if err != nil {
		return nil, fmt.Errorf("failed to compute digest: %v", err)
	}

	fi, err := layerTarball.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to verify file size: %v", err)
	}
	// Return the layer metadata
	layer := &Layer{
		Digest:    digest,
		MediaType: "application/vnd.cache.triton.content.layer.v1+triton",
		Size:      fi.Size(),
	}
	return layer, nil
}

// tarDirectory creates a tarball from the directory.
func tarDirectory(srcDir string, outFile *os.File) error {
	tarWriter := gzip.NewWriter(outFile)
	defer tarWriter.Close()

	// Walk through the "cache" directory and add each file to the tarball.
	return filepath.Walk(srcDir, func(file string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		// Write the file contents to the tarball
		srcFile, err := os.Open(file)
		if err != nil {
			return err
		}
		defer srcFile.Close()

		_, err = io.Copy(tarWriter, srcFile)
		return err
	})
}

// computeDigest computes the SHA256 digest of a file
func computeDigest(filename string) (string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return "", fmt.Errorf("failed to open file for digest calculation: %v", err)
	}
	defer file.Close()

	hash := sha256.New()
	_, err = io.Copy(hash, file)
	if err != nil {
		return "", fmt.Errorf("failed to compute hash: %v", err)
	}

	digest := hash.Sum(nil)
	return "sha256:" + base64.URLEncoding.EncodeToString(digest), nil
}
