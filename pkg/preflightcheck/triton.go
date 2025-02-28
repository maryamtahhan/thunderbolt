package preflightcheck

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

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
	Arch                      string     `json:"arch"`
	SanitizeOverflow          bool       `json:"sanitize_overflow"`
	Shared                    int        `json:"shared"`
	GlobalScratchSize         int        `json:"global_scratch_size"`
	GlobalScratchAlign        int        `json:"global_scratch_align"`
	Name                      string     `json:"name"`
}

// Nested struct for the "target" field
type Target struct {
	Backend  string      `json:"backend"`
	Arch     json.Number `json:"arch"`
	WarpSize int         `json:"warp_size"`
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

	var hasMatch bool
	var backendMismatch bool

	for _, gpuInfo := range devInfo {
		backendMatches := cacheData.Target.Backend == gpuInfo.Backend
		archMatches := cacheData.Target.Arch.String() == gpuInfo.Arch
		warpMatches := cacheData.Target.WarpSize == gpuInfo.WarpSize
		ptxMatches := true

		if gpuInfo.Backend == "cuda" && cacheData.PtxVersion != nil {
			ptxMatches = *cacheData.PtxVersion == gpuInfo.PTXVersion
			if !ptxMatches {
				logging.Debugf("PTX version mismatch - cache=%d, gpu=%d", *cacheData.PtxVersion, gpuInfo.PTXVersion)
			}
		}

		if backendMatches && archMatches && warpMatches && ptxMatches {
			hasMatch = true
			break // No need to check further, at least one match is found
		}

		if !backendMatches {
			backendMismatch = true
			logging.Debugf("Backend mismatch - cache=%s, gpu=%s", cacheData.Target.Backend, gpuInfo.Backend)
		}
	}

	if hasMatch {
		return nil // At least one GPU matches all fields, return no error
	}

	if backendMismatch {
		return fmt.Errorf("incompatibility detected: backendMismatch=%t", backendMismatch)
	}

	return fmt.Errorf("no compatible GPU found")
}
