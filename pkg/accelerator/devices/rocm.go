package devices

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strconv"

	"github.com/gpuman/thunderbolt/pkg/config"
	"github.com/gpuman/thunderbolt/pkg/utils"
	logging "github.com/sirupsen/logrus"
)

const rocmHwType = config.GPU

var (
	rocmAccImpl = gpuRocm{}
	rocmType    DeviceType
)

type gpuRocm struct {
	devices map[int]GPUDevice // GPU identifiers mapped to device info
}

type ROCMGPUInfo struct {
	Name          string `json:"Card model"`
	UUID          string `json:"Unique ID"`
	SerialNumber  string `json:"Serial Number"`
	DriverVersion string `json:"Driver version"`
	MemoryTotalB  string `json:"VRAM Total Memory (B)"` // Stored as string in JSON
}

func rocmCheck(r *Registry) {
	if err := initROCmLib(); err != nil {
		logging.Infof("Error initializing ROCm: %v", err)
		return
	}
	rocmType = ROCM
	if err := addDeviceInterface(r, rocmType, rocmHwType, rocmDeviceStartup); err == nil {
		logging.Infof("Using %s to obtain processor power", rocmAccImpl.Name())
	} else {
		logging.Infof("Error registering rocm-smi: %v", err)
	}
}

func rocmDeviceStartup() Device {
	a := rocmAccImpl
	if err := a.InitLib(); err != nil {
		logging.Errorf("Error initializing %s: %v", rocmType.String(), err)
		return nil
	}
	if err := a.Init(); err != nil {
		logging.Errorf("Failed to init device: %v", err)
		return nil
	}
	logging.Infof("Using %s to obtain GPU info", rocmType.String())
	return &a
}

func initROCmLib() error {
	if utils.HasApp("rocm-smi") {
		return nil
	}
	return errors.New("couldn't find rocm-smi")
}

func (r *gpuRocm) InitLib() error {
	return initROCmLib()
}

func (r *gpuRocm) Name() string {
	return rocmType.String()
}

func (r *gpuRocm) DevType() DeviceType {
	return rocmType
}

func (r *gpuRocm) HwType() string {
	return rocmHwType
}

// Init initializes and starts the GPU info collection using a **single `rocm-smi` command**
func (r *gpuRocm) Init() error {
	gpuInfoList, err := getAllRocmGPUInfo()
	if err != nil {
		return fmt.Errorf("failed to get GPU information: %v", err)
	}

	// Populate the devices map
	r.devices = make(map[int]GPUDevice, len(gpuInfoList))
	for gpuID, info := range gpuInfoList {
		memTotal, _ := strconv.ParseUint(info.MemoryTotalB, 10, 64)

		r.devices[gpuID] = GPUDevice{
			ID: gpuID,
			TritonInfo: TritonGPUInfo{
				Name:              info.Name,
				UUID:              info.UUID,
				ComputeCapability: "",                       // ROCm doesn't expose compute capability like CUDA
				Arch:              info.DriverVersion,       // Using driver version as a proxy for arch
				WarpSize:          64,                       // AMD GPUs use 64-thread wavefronts
				MemoryTotalMB:     memTotal / (1024 * 1024), // Convert bytes to MB
				GFXVersion:        info.DriverVersion,
			},
		}
	}

	return nil
}

// Shutdown stops the GPU metric collector
func (r *gpuRocm) Shutdown() bool {
	return true
}

// Fetches all GPUs' info in **one single rocm-smi call**
func getAllRocmGPUInfo() (map[int]ROCMGPUInfo, error) {
	cmd := exec.Command("rocm-smi",
		"--json",
		"--showdriverversion",
		"--showuniqueid",
		"--showserial",
		"--showmeminfo",
		"all")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute rocm-smi: %v", err)
	}

	// Parse JSON into map of GPUs
	var gpuInfo map[string]ROCMGPUInfo
	if err := json.Unmarshal(output, &gpuInfo); err != nil {
		return nil, fmt.Errorf("failed to parse rocm-smi output: %v", err)
	}

	// Convert map keys from "GPUX" to int keys
	parsedGPUs := make(map[int]ROCMGPUInfo)
	for key, gpu := range gpuInfo {
		var gpuID int
		_, err := fmt.Sscanf(key, "GPU%d", &gpuID)
		if err == nil {
			parsedGPUs[gpuID] = gpu
		}
	}

	return parsedGPUs, nil
}

// GetAllGPUInfo returns a list of GPU info for all devices
func (r *gpuRocm) GetAllGPUInfo() ([]TritonGPUInfo, error) {
	var allTritonInfo []TritonGPUInfo
	for gpuID, dev := range r.devices {
		allTritonInfo = append(allTritonInfo, dev.TritonInfo)
		logging.Infof("GPU %d: %+v", gpuID, dev.TritonInfo)
	}
	return allTritonInfo, nil
}

// GetGPUInfo retrieves the stored GPU info for a specific device ID.
func (r *gpuRocm) GetGPUInfo(gpuID int) (TritonGPUInfo, error) {
	dev, exists := r.devices[gpuID]
	if !exists {
		return TritonGPUInfo{}, fmt.Errorf("GPU device %d not found", gpuID)
	}
	return dev.TritonInfo, nil
}
