package preflightcheck

import (
	"encoding/json"
	"fmt"
	"os"

	logging "github.com/sirupsen/logrus"
)

// Define the struct matching the JSON structure
type Data struct {
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

func GetTritonCacheJSONData(filePath string) (*Data, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		logging.Errorf("Failed to read file %s: %v", filePath, err)
		return nil, fmt.Errorf("failed to read file %s: %v", filePath, err)
	}

	var data Data
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
