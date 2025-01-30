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

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/containers/buildah"
	"github.com/containers/storage/pkg/unshare"
	"github.com/gpuman/thunderbolt/pkg/fetcher"
	"github.com/gpuman/thunderbolt/pkg/imgbuild"
	"github.com/gpuman/thunderbolt/pkg/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/klog/v2"
)

const (
	exitNormal       = 0
	exitExtractError = 1
	exitCreateError  = 2
)

func getCacheImage(imageName string) error {
	f := fetcher.New()
	return f.FetchAndExtractCache(imageName)
}

func createCacheImage(imageName, cacheDir string) error {

	_, err := utils.FilePathExists(cacheDir)
	if err != nil {
		return fmt.Errorf("error checking cache file path: %v", err)
	}

	// Create a new builder instance
	builder, _ := imgbuild.New()
	if builder == nil {
		return fmt.Errorf("failed to create builder")
	}

	// Push the layer and manifest to the registry
	err = builder.CreateImage(imageName, cacheDir)
	if err != nil {
		return fmt.Errorf("failed to create the OCI image: %v", err)
	}

	klog.Info("OCI image created successfully.")
	return nil
}
func init() {
	// Bind klog flags to Cobra
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
}
func main() {
	var imageName string
	var cacheDirName string
	var createFlag bool
	var extractFlag bool
	var logLevel int

	klog.InitFlags(nil)
	defer klog.Flush()

	// Define the CLI command using cobra
	var rootCmd = &cobra.Command{
		Use:   "thunderbolt",
		Short: "A GPU Kernel runtime container image management utility",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// Set klog verbosity level from `--log-level` flag
			_ = flag.Set("v", fmt.Sprintf("%d", logLevel))
		},
		Run: func(cmd *cobra.Command, args []string) {
			if createFlag {
				if err := createCacheImage(imageName, cacheDirName); err != nil {
					klog.Fatalf("Error creating image: %v\n", err)
					os.Exit(exitCreateError)
				}
			}

			if extractFlag {
				if err := getCacheImage(imageName); err != nil {
					klog.Fatalf("Error extracting image: %v\n", err)
					os.Exit(exitExtractError)
				}
			}

			if !createFlag && !extractFlag {
				klog.Error("No action specified. Use --create or --extract flag.")
				os.Exit(exitNormal)
			}
		},
	}
	// Bind klog flags to Cobra's flag set
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)

	// Define the flags for the command-line arguments
	rootCmd.Flags().StringVarP(&imageName, "image", "i", "", "OCI image name")
	rootCmd.Flags().StringVarP(&cacheDirName, "dir", "d", "", "Triton Cache Directory")
	rootCmd.Flags().BoolVarP(&createFlag, "create", "c", false, "Create OCI image")
	rootCmd.Flags().BoolVarP(&extractFlag, "extract", "e", false, "Extract a Triton cache from an OCI image")
	rootCmd.PersistentFlags().IntVarP(&logLevel, "log-level", "l", 0, "Set the logging verbosity level (0 = minimal, higher is more verbose)")

	// Mark the image flag as required
	rootCmd.MarkFlagRequired("image")

	// Important to call from main()
	if buildah.InitReexec() {
		return
	}
	unshare.MaybeReexecUsingUserNamespace(false)

	// Execute the root command
	if err := rootCmd.Execute(); err != nil {
		klog.Fatalf("Error: %v\n", err)
	}
}
