// Copyright Red Hat Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"log"
	"os"

	"github.com/gpuman/thunderbolt/pkg/fetcher"
	"github.com/gpuman/thunderbolt/pkg/push"
	"github.com/spf13/cobra"
)

func getCacheImage(imageName string) error {
	f := fetcher.New()
	return f.FetchAndExtractCache(imageName)
}

func createCacheImage(imageName string) error {
	// TODO Ensure that the cache directory exists
	// Make this configuration
	cacheDir := os.Getenv("HOME") + "/.triton/cache"

	// Create a new Pusher instance
	pusher, err := push.New(imageName, cacheDir)
	if err != nil {
		log.Fatalf("Failed to create Pusher: %v\n", err)
	}

	// Push the layer and manifest to the registry
	err = pusher.Push()
	if err != nil {
		log.Fatalf("Failed to push layer and manifest: %v\n", err)
	}

	fmt.Println("OCI image pushed successfully.")
	return nil
}

func main() {
	var imageName string
	var createFlag bool
	var extractFlag bool

	// Define the CLI command using cobra
	var rootCmd = &cobra.Command{
		Use:   "thunderbolt",
		Short: "A GPU Kernel runtime container image management utility",
		Run: func(cmd *cobra.Command, args []string) {
			if createFlag {
				if err := createCacheImage(imageName); err != nil {
					log.Fatalf("Error creating image: %v\n", err)
				}
			}

			if extractFlag {
				if err := getCacheImage(imageName); err != nil {
					log.Fatalf("Error extracting image: %v\n", err)
				}
			}

			if !createFlag && !extractFlag {
				log.Println("No action specified. Use --create or --extract flag.")
			}
		},
	}

	// Define the flags for the command-line arguments
	rootCmd.Flags().StringVarP(&imageName, "image", "i", "", "OCI image name")
	rootCmd.Flags().BoolVarP(&createFlag, "create", "c", false, "Create - TODO")
	rootCmd.Flags().BoolVarP(&extractFlag, "extract", "e", false, "Extract a Triton cache from an OCI image")

	// Mark the image flag as required
	rootCmd.MarkFlagRequired("image")

	// Execute the root command
	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("Error: %v\n", err)
	}
}
