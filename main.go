package main

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/hashicorp/go-multierror"
	"github.com/spf13/cobra"
)

// fetchImage pulls the image from the registry and prints its details.
func fetchImage(imageName string) error {
	// Parse the image name into a reference (e.g., quay.io/mtahhan/triton)
	ref, err := name.ParseReference(imageName)
	if err != nil {
		return fmt.Errorf("failed to parse image name: %w", err)
	}

	// Fetch the image descriptor (including the manifest)
	img, err := remote.Image(ref, remote.WithAuthFromKeychain(authn.DefaultKeychain))
	if err != nil {
		return fmt.Errorf("failed to fetch image: %w", err)
	}

	// Print the image details
	fmt.Println("Image fetched successfully!!!!!!!!")

	// Get the image digest and handle the error
	digest, err := img.Digest()
	if err != nil {
		return fmt.Errorf("failed to get image digest: %w", err)
	}
	// Print the image digest
	fmt.Println("Image Digest:", digest)

	size, err := img.Size()
	if err != nil {
		return fmt.Errorf("failed to get image digest: %w", err)
	}
	// Print the image size
	fmt.Printf("Image Size: %v\n", size)

	manifest, err := img.Manifest()
	if err != nil {
		return fmt.Errorf("failed to fetch manifest: %w", err)
	}

	if manifest.MediaType == types.DockerManifestSchema2 {
		// This case, assume we have docker images with "application/vnd.docker.distribution.manifest.v2+json"
		// as the manifest media type. Note that the media type of manifest is Docker specific and
		// all OCI images would have an empty string in .MediaType field.
		_, err := extractDockerImage(img)
		if err != nil {
			return fmt.Errorf("could not extract the Triton Cache from the container image %v", err)
		}
		return nil
	}

	// We try to parse it as the "compat" variant image with a single "application/vnd.oci.image.layer.v1.tar+gzip" layer.
	_, errCompat := extractOCIStandardImage(img)
	if errCompat == nil {
		return nil
	}

	// Otherwise, we try to parse it as the *oci* variant image with custom artifact media types.
	_, errOCI := extractOCIArtifactImage(img)
	if errOCI == nil {
		return nil
	}

	// We failed to parse the image in any format, so wrap the errors and return.
	return fmt.Errorf("the given image is in invalid format as an OCI image: %v",
		multierror.Append(err,
			fmt.Errorf("could not parse as compat variant: %v", errCompat),
			fmt.Errorf("could not parse as oci variant: %v", errOCI),
		),
	)
}

// extractOCIArtifactImage extracts the triton cache from the
// *oci* variant Triton Kernel Cache image: https://github.com/solo-io/wasm/blob/master/spec/spec.md#format //TODO UPDATE
func extractOCIArtifactImage(img v1.Image) ([]byte, error) {
	layers, err := img.Layers()
	if err != nil {
		return nil, fmt.Errorf("could not fetch layers: %v", err)
	}

	// The image must be single-layered.
	if len(layers) != 1 {
		return nil, fmt.Errorf("number of layers must be 1 but got %d", len(layers))
	}

	// The layer type of the Triton cache itself in *oci* variant.
	//const cacheLayerMediaType = "application/vnd.module.wasm.content.layer.v1+wasm"
	const cacheLayerMediaType = "application/cache.triton.content.layer.v1+triton"

	// Find the target layer walking through the layers.
	var layer v1.Layer
	for _, l := range layers {
		mt, err := l.MediaType()
		if err != nil {
			return nil, fmt.Errorf("could not retrieve the media type: %v", err)
		}
		if mt == cacheLayerMediaType {
			layer = l
			break
		}
	}

	if layer == nil {
		return nil, fmt.Errorf("could not find the layer of type %s", cacheLayerMediaType)
	}

	// Somehow go-container registry recognizes custom artifact layers as compressed ones,
	// while the Solo's Wasm layer is actually uncompressed and therefore
	// the content itself is a raw Wasm binary. So using "Uncompressed()" here result in errors
	// since internally it tries to umcompress it as gzipped blob.
	r, err := layer.Compressed()
	if err != nil {
		return nil, fmt.Errorf("could not get layer content: %v", err)
	}
	defer r.Close()

	// Just read it since the content is already a raw Wasm binary as mentioned above.
	ret, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("could not extract io.triton.cache: %v", err)
	}
	return ret, nil
}

// extractDockerImage extracts the Triton Kernel Cache from the
// *compat* variant Wasm image with the standard Docker media type: application/vnd.docker.image.rootfs.diff.tar.gzip.
// https://github.com/solo-io/wasm/blob/master/spec/spec-compat.md#specification
func extractDockerImage(img v1.Image) ([]byte, error) {
	layers, err := img.Layers()
	if err != nil {
		return nil, fmt.Errorf("could not fetch layers: %v", err)
	}

	// The image must have at least one layer.
	if len(layers) == 0 {
		return nil, errors.New("number of layers must be greater than zero")
	}

	layer := layers[len(layers)-1]
	mt, err := layer.MediaType()
	if err != nil {
		return nil, fmt.Errorf("could not get media type: %v", err)
	}

	// Media type must be application/vnd.docker.image.rootfs.diff.tar.gzip.
	if mt != types.DockerLayer {
		return nil, fmt.Errorf("invalid media type %s (expect %s)", mt, types.DockerLayer)
	}

	r, err := layer.Compressed()
	if err != nil {
		return nil, fmt.Errorf("could not get layer content: %v", err)
	}
	defer r.Close()

	ret, err := extractTritonCacheDirectory(r)
	if err != nil {
		return nil, fmt.Errorf("could not extract Triton Kernel Cache: %v", err)
	}
	return ret, nil
}

// extractOCIStandardImage extracts the Triton Kernel Cache from the
// *compat* variant Triton Kernel image with the standard OCI media type: application/vnd.oci.image.layer.v1.tar+gzip.
// https://github.com/solo-io/wasm/blob/master/spec/spec-compat.md#specification //TODO UPDATE
func extractOCIStandardImage(img v1.Image) ([]byte, error) {
	layers, err := img.Layers()
	if err != nil {
		return nil, fmt.Errorf("could not fetch layers: %v", err)
	}

	// The image must have at least one layer.
	if len(layers) == 0 {
		return nil, fmt.Errorf("number of layers must be greater than zero")
	}

	layer := layers[len(layers)-1]
	mt, err := layer.MediaType()
	if err != nil {
		return nil, fmt.Errorf("could not get media type: %v", err)
	}

	// Check if the layer is "application/vnd.oci.image.layer.v1.tar+gzip".
	if types.OCILayer != mt {
		return nil, fmt.Errorf("invalid media type %s (expect %s)", mt, types.OCILayer)
	}

	r, err := layer.Compressed()
	if err != nil {
		return nil, fmt.Errorf("could not get layer content: %v", err)
	}
	defer r.Close()

	ret, err := extractTritonCacheDirectory(r)
	if err != nil {
		return nil, fmt.Errorf("could not extract Triton Kernel Cache: %v", err)
	}
	return ret, nil
}

// Extracts the triton named "io.triton.cache" in a given reader for tar.gz.
// This is only used for *compat* variant.
func extractTritonCacheDirectory(r io.Reader) ([]byte, error) {
	targetDir := os.Getenv("HOME") + "/.triton/cache"
	gr, err := gzip.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("failed to parse layer as tar.gz: %v", err)
	}

	// The target directory name to skip (but process its contents)
	const TritonCacheDirName = "io.triton.cache/"

	// Tar reader to iterate through the archive
	tr := tar.NewReader(gr)

	for {
		h, err := tr.Next()
		if err == io.EOF {
			break // End of archive
		} else if err != nil {
			return nil, fmt.Errorf("error reading tar archive: %w", err)
		}

		// Skip directories and files that are not part of io.triton.cache
		if !strings.HasPrefix(h.Name, TritonCacheDirName) {
			continue
		}

		// Strip the prefix "io.triton.cache/" from the file path
		relativePath := strings.TrimPrefix(h.Name, TritonCacheDirName)
		if relativePath == "" {
			continue // Skip the directory itself
		}

		// Resolve the new file path under the target directory
		filePath := filepath.Join(targetDir, relativePath)

		switch h.Typeflag {
		case tar.TypeDir:
			// Create the directory in the target location
			if err := os.MkdirAll(filePath, os.FileMode(h.Mode)); err != nil {
				return nil, fmt.Errorf("failed to create directory %s: %w", filePath, err)
			}

		case tar.TypeReg:
			// Create the file in the target location
			err := writeFile(filePath, tr, os.FileMode(h.Mode))
			if err != nil {
				return nil, fmt.Errorf("failed to create file %s: %w", filePath, err)
			}

		default:
			// Skip unsupported types
			fmt.Printf("Skipping unsupported type: %c in file %s\n", h.Typeflag, h.Name)
		}
	}

	return nil, nil
}

// TODO
// extractLayer extracts the contents of a gzip-compressed tar archive into the target directory
// func extractLayer(layerPath, targetDir string) error {
// 	// Open the .tar.gz file
// 	layerFile, err := os.Open(layerPath)
// 	if err != nil {
// 		return fmt.Errorf("failed to open layer file: %w", err)
// 	}
// 	defer layerFile.Close()

// 	// Create a gzip reader to decompress the file
// 	gzipReader, err := gzip.NewReader(layerFile)
// 	if err != nil {
// 		return fmt.Errorf("failed to create gzip reader: %w", err)
// 	}
// 	defer gzipReader.Close()

// 	// Create a tar reader to read the tar contents
// 	tarReader := tar.NewReader(gzipReader)

// 	// Iterate over each file in the tar archive
// 	for {
// 		header, err := tarReader.Next()
// 		if err == io.EOF {
// 			// End of archive
// 			break
// 		}
// 		if err != nil {
// 			return fmt.Errorf("error reading tar archive: %w", err)
// 		}

// 		// Resolve the file path to ensure it's safe
// 		filePath := filepath.Join(targetDir, header.Name)

// 		switch header.Typeflag {
// 		case tar.TypeDir:
// 			// Create a directory
// 			if err := os.MkdirAll(filePath, os.FileMode(header.Mode)); err != nil {
// 				return fmt.Errorf("failed to create directory %s: %w", filePath, err)
// 			}

// 		case tar.TypeReg:
// 			// Create a file
// 			err := writeFile(filePath, tarReader, os.FileMode(header.Mode))
// 			if err != nil {
// 				return fmt.Errorf("failed to create file %s: %w", filePath, err)
// 			}

// 		default:
// 			// Handle other file types (symlinks, etc.) if necessary
// 			fmt.Printf("Skipping unsupported type: %c in file %s\n", header.Typeflag, header.Name)
// 		}
// 	}
// 	return nil
// }

// writeFile writes a file's content to disk from the tar reader
func writeFile(filePath string, tarReader io.Reader, mode os.FileMode) error {
	// Create any parent directories if needed
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return fmt.Errorf("failed to create parent directories for %s: %w", filePath, err)
	}

	// Create the file
	outFile, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filePath, err)
	}
	defer outFile.Close()

	// Copy the file content
	if _, err := io.Copy(outFile, tarReader); err != nil {
		return fmt.Errorf("failed to copy content to file %s: %w", filePath, err)
	}

	// Set file permissions
	if err := os.Chmod(filePath, mode); err != nil {
		return fmt.Errorf("failed to set file permissions for %s: %w", filePath, err)
	}

	return nil
}

// main function that defines the command-line interface (CLI)
func main() {
	var imageName string

	// Define the CLI command using cobra
	var rootCmd = &cobra.Command{
		Use:   "triton-cache-fetcher",
		Short: "A tool to fetch OCI images and their layers",
		Run: func(cmd *cobra.Command, args []string) {
			if err := fetchImage(imageName); err != nil {
				log.Fatalf("Error: %v\n", err)
			}
		},
	}

	// Define the flags for the command-line arguments
	rootCmd.Flags().StringVarP(&imageName, "image", "i", "", "OCI image name to fetch")
	rootCmd.MarkFlagRequired("image")

	// Execute the root command
	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("Error: %v\n", err)
	}
}
