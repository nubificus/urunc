// Copyright 2023 Nubificus LTD.

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"strings"

	"github.com/shirou/gopsutil/disk"
)

// RootFs represents a root file system and its properties.
type RootFs struct {
	Path      string             // The path of the root file system.
	IsBlock   bool               // Indicates if it's a block device.
	BlkDevice disk.PartitionStat // Information about the block device.
}

// GetBlockDevice retrieves information about the block device associated with a given path.
// It searches for a mounted block device with the specified path and returns its details.
// If the path is not a block device or there is an error, it returns an empty RootFs struct and an error.
func GetBlockDevice(path string) (RootFs, error) {
	var result RootFs
	result.IsBlock = false

	// Retrieve a list of mounted partitions
	parts, err := disk.Partitions(true)
	if err != nil {
		return result, err
	}

	// Search for the partition with the specified path
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
