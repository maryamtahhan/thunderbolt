package push

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// Pusher handles pushing the layer and manifest to the registry
type Pusher struct {
	RegistryURL string
	Repository  string
	AuthToken   string
	Layer       *Layer
	Manifest    *Manifest
}

// New creates a new instance of Pusher
func New(imageName, cacheDir string) (*Pusher, error) {
	registryURL := "https://example.com" // Replace with actual registry
	repository := "myrepo/myimage"       // Replace with actual image name
	authToken := "your-auth-token"       // Replace with actual auth token

	// Create the layer from the cache directory
	layer, err := CreateLayerFromCache(cacheDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create OCI layer: %v", err)
	}

	// Generate the OCI image manifest
	manifest := generateOCIManifest(layer)

	// Return the Pusher instance
	return &Pusher{
		RegistryURL: registryURL,
		Repository:  repository,
		AuthToken:   authToken,
		Layer:       layer,
		Manifest:    manifest,
	}, nil
}

// Manifest represents the OCI image manifest
type Manifest struct {
	SchemaVersion int          `json:"schemaVersion"`
	MediaType     string       `json:"mediaType"`
	Config        Descriptor   `json:"config"`
	Layers        []Descriptor `json:"layers"`
}

// Descriptor represents the descriptor used in the manifest
type Descriptor struct {
	Digest    string `json:"digest"`
	MediaType string `json:"mediaType"`
	Size      int64  `json:"size"`
}

// generateOCIManifest generates the OCI image manifest for a single layer
func generateOCIManifest(layer *Layer) *Manifest {
	return &Manifest{
		SchemaVersion: 2,
		MediaType:     "application/vnd.oci.image.manifest.v1+json",
		Config: Descriptor{
			Digest:    "", // Empty config for simplicity
			MediaType: "application/json",
			Size:      0,
		},
		Layers: []Descriptor{
			{
				Digest:    layer.Digest,
				MediaType: layer.MediaType,
				Size:      layer.Size,
			},
		},
	}
}

// Push pushes the layer and manifest to the image registry
func (p *Pusher) Push() error {
	// Push the layer to the registry (simplified for now)
	layerURL := fmt.Sprintf("%s/v2/%s/blobs/uploads/", p.RegistryURL, p.Repository)
	fmt.Printf("Uploading layer %s to %s...\n", p.Layer.Digest, layerURL)

	// Push the manifest to the registry
	manifestURL := fmt.Sprintf("%s/v2/%s/manifests/latest", p.RegistryURL, p.Repository)

	manifestData, err := json.MarshalIndent(p.Manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %v", err)
	}

	req, err := http.NewRequest("PUT", manifestURL, bytes.NewReader(manifestData))
	if err != nil {
		return fmt.Errorf("failed to create request for manifest push: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.AuthToken)
	req.Header.Set("Content-Type", "application/vnd.oci.image.manifest.v1+json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to push manifest: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("failed to push manifest, status code: %v", resp.StatusCode)
	}

	fmt.Printf("Successfully pushed manifest to %s\n", manifestURL)
	return nil
}
