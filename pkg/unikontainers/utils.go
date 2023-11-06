package unikontainers

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/shirou/gopsutil/disk"
)

// RootFs represents a root file system and its properties.
type RootFs struct {
	Path      string             // The path of the root file system.
	IsBlock   bool               // Indicates if it's a block device.
	BlkDevice disk.PartitionStat // Information about the block device.
}

// getBlockDevice retrieves information about the block device associated with a given path.
// It searches for a mounted block device with the specified path and returns its details.
// If the path is not a block device or there is an error, it returns an empty RootFs struct and an error.
func getBlockDevice(path string) (RootFs, error) {
	var result RootFs
	result.IsBlock = false

	// Retrieve a list of mounted partitions
	parts, err := disk.Partitions(true)
	if err != nil {
		return result, err
	}

	// Search for the partition with the specified path
	// FIXME: Looping through all mounted devices could hinder performance. Explore alternatives.
	for _, p := range parts {
		if p.Mountpoint == path {
			result.Path = path
			result.BlkDevice = p
			break
		}
	}

	// Check if the file system type is ext4 or ext2 and the device name contains "dm" (indicating a block device)
	if (result.BlkDevice.Fstype == "ext4" || result.BlkDevice.Fstype == "ext2") && strings.Contains(result.BlkDevice.Device, "dm") {
		result.IsBlock = true
	}

	return result, nil
}

// getInitPid extracts "init_process_pid" value from the given JSON file
func getInitPid(filePath string) (float64, error) {
	// Open the JSON file for reading
	file, err := os.Open(filePath)
	if err != nil {
		return 0, nil
	}
	defer file.Close()

	// Decode the JSON data into a map[string]interface{}
	var jsonData map[string]interface{}
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&jsonData); err != nil {
		return 0, nil

	}

	// Extract the specific value "init_process_pid"
	initProcessPID, found := jsonData["init_process_pid"].(float64) // Assuming it's a numeric value
	if !found {
		return 0, nil
	}
	return initProcessPID, nil
}

// copy sourceFile to targetDir
// creates targetDir and all necessary parent directories
func copyFile(sourceFile string, targetDir string) error {
	source, err := os.Open(sourceFile)
	if err != nil {
		return err
	}
	defer source.Close()
	err = os.MkdirAll(targetDir, 0755)
	if err != nil {
		return err
	}

	_, filename := filepath.Split(sourceFile)
	targetPath := filepath.Join(targetDir, filename)
	target, err := os.Create(targetPath)
	if err != nil {
		return err
	}
	defer target.Close()
	_, err = io.Copy(target, source)
	if err != nil {
		return err
	}
	return nil
}

// move sourceFile to targetDir
// creates targetDir and all necessary parent directories
func moveFile(sourceFile string, targetDir string) error {
	err := copyFile(sourceFile, targetDir)
	if err != nil {
		return err
	}
	return os.Remove(sourceFile)
}

// loadSpec returns the Spec found in the given bundle directory
func loadSpec(bundleDir string) (*specs.Spec, error) {
	var spec specs.Spec
	absDir, err := filepath.Abs(bundleDir)
	if err != nil {
		return &spec, fmt.Errorf("failed to find absolute bundle path: %w", err)
	}
	data, err := os.ReadFile(filepath.Join(absDir, "config.json"))
	if err != nil {
		return &spec, fmt.Errorf("failed to read config.json: %w", err)
	}
	if err := json.Unmarshal(data, &spec); err != nil {
		return &spec, fmt.Errorf("failed to parse config.json: %w", err)
	}
	return &spec, nil
}

// writePidFile writes the content of pid to the file defined by path
func writePidFile(path string, pid int) error {
	var (
		tmpDir  = filepath.Dir(path)
		tmpName = filepath.Join(tmpDir, "."+filepath.Base(path))
	)
	f, err := os.OpenFile(tmpName, os.O_RDWR|os.O_CREATE|os.O_EXCL|os.O_SYNC, 0o666)
	if err != nil {
		return err
	}
	_, err = f.WriteString(strconv.Itoa(pid))
	f.Close()
	if err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

// isBimaContainer attempts to find any bima related annotations
// in the given bundle to verify the image is compatible with urunc
func IsBimaContainer(bundle string) bool {
	spec, err := loadSpec(bundle)
	if err != nil {
		Log.WithError(err).WithField("bundle", bundle).Error("Couldn't load spec from bundle")
		return false
	}

	_, err = GetUnikernelConfig(bundle, spec)
	return err == nil
}
