// Copyright 2024 Nubificus LTD.

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package unikontainers

import (
	"testing"

	"github.com/shirou/gopsutil/disk"
	"github.com/stretchr/testify/assert"
)

func TestGetBlockDevice(t *testing.T) {
	// Create a mock partition
	partitions := []disk.PartitionStat{
		{
			Device:     "dm-0",
			Mountpoint: "/mock/path",
			Fstype:     "ext4",
		},
	}

	// Mock the disk.Partitions function
	mockGetPartitions := func(all bool) ([]disk.PartitionStat, error) {
		return partitions, nil
	}
	rootFs, err := getBlockDevice("/mock/path", mockGetPartitions)
	assert.NoError(t, err, "Expected no error in getting block device")
	assert.Equal(t, "/mock/path", rootFs.Path, "Expected path to be /mock/path")
	assert.True(t, rootFs.IsBlock, "Expected IsBlock to be true")
	assert.Equal(t, "ext4", rootFs.BlkDevice.Fstype, "Expected filesystem type to be ext4")
}
