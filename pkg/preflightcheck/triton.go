package preflightcheck

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/gpuman/thunderbolt/pkg/accelerator"
	"github.com/gpuman/thunderbolt/pkg/accelerator/devices"
	"github.com/gpuman/thunderbolt/pkg/config"
	logging "github.com/sirupsen/logrus"
)

// Define the struct matching the JSON structure
type TritonCacheData struct {
	Hash                      string     `json:"hash"`
	Target                    Target     `json:"target"`
	NumWarps                  int        `json:"num_warps"`
	NumCtas                   int        `json:"num_ctas"`
	NumStages                 int        `json:"num_stages"`
	MaxNReg                   *int       `json:"maxnreg"`
	ClusterDims               []int      `json:"cluster_dims"`
	PtxVersion                *int       `json:"ptx_version"`
	EnableFpFusion            bool       `json:"enable_fp_fusion"`
	SupportedFp8Dtypes        []string   `json:"supported_fp8_dtypes"`
	DeprecatedFp8Dtypes       []string   `json:"deprecated_fp8_dtypes"`
	DefaultDotInputPrecision  string     `json:"default_dot_input_precision"`
	AllowedDotInputPrecisions []string   `json:"allowed_dot_input_precisions"`
	MaxNumImpreciseAccDefault int        `json:"max_num_imprecise_acc_default"`
	ExternLibs                [][]string `json:"extern_libs"`
	Debug                     bool       `json:"debug"`
	BackendName               string     `json:"backend_name"`
	SanitizeOverflow          bool       `json:"sanitize_overflow"`
	Shared                    int        `json:"shared"`
	GlobalScratchSize         int        `json:"global_scratch_size"`
	GlobalScratchAlign        int        `json:"global_scratch_align"`
	Name                      string     `json:"name"`
}

// Nested struct for the "target" field
type Target struct {
	Backend  string `json:"backend"`
	Arch     int    `json:"arch"`
	WarpSize int    `json:"warp_size"`
}

func GetTritonCacheJSONData(filePath string) (*TritonCacheData, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		logging.Errorf("Failed to read file %s: %v", filePath, err)
		return nil, fmt.Errorf("failed to read file %s: %v", filePath, err)
	}

	var data TritonCacheData
	if err = json.Unmarshal(content, &data); err != nil {
		logging.Errorf("Failed to parse JSON in file %s: %v", filePath, err)
		return nil, fmt.Errorf("failed to parse JSON in file %s: %v", filePath, err)
	}

	// Check if the "hash" field is present and valid
	if data.Hash == "" {
		logging.Debugf("File %s does not contain the required 'hash' field", filePath)
		// DO NOT return an error.
		return nil, nil
	}

	prettyJSON, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		logging.Errorf("Failed to pretty print JSON in file %s: %v", filePath, err)
		return nil, fmt.Errorf("failed to pretty print JSON in file %s: %v", filePath, err)
	}

	logging.Debugf("Cache JSON output:\n%s", string(prettyJSON))
	return &data, nil
}

func CompareTritonCacheToGPU(cacheData *TritonCacheData, acc accelerator.Accelerator) error {
	if cacheData == nil {
		return errors.New("cache data is nil")
	}
	if acc == nil {
		return errors.New("acc is nil")
	}
	var devInfo []devices.TritonGPUInfo
	if config.IsGPUEnabled() {
		if gpu := accelerator.GetActiveAcceleratorByType(config.GPU); gpu != nil {
			d := gpu.Device()
			if tritonDevInfo, err := d.GetAllGPUInfo(); err == nil {
				devInfo = tritonDevInfo
			} else {
				return errors.New("couldn't retrieve the GPU Triton info")
			}
		}
	}

	for _, gpuInfo := range devInfo {
		gpuArch, err := strconv.Atoi(gpuInfo.Arch) // Convert GPU Arch string to int
		if err != nil {
			return fmt.Errorf("invalid GPU Arch format: %s", gpuInfo.Arch)
		}

		if cacheData.Target.Arch != gpuArch {
			return fmt.Errorf("mismatch in architecture: cache=%d, gpu=%d", cacheData.Target.Arch, gpuArch)
		}

		if cacheData.Target.WarpSize != gpuInfo.WarpSize {
			return fmt.Errorf("mismatch in warp size: cache=%d, gpu=%d", cacheData.Target.WarpSize, gpuInfo.WarpSize)
		}

		if cacheData.PtxVersion != nil && *cacheData.PtxVersion != gpuInfo.PTXVersion {
			return fmt.Errorf("mismatch in PTX version: cache=%d, gpu=%d", *cacheData.PtxVersion, gpuInfo.PTXVersion)
		}
	}

	return nil
}
