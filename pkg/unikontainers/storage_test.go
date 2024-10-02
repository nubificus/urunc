// Copyright (c) 2023-2024, Nubificus LTD
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

package unikontainers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetBlockDevice(t *testing.T) {
	// Create a mock partition
	tmpMnt := RootFs{
		Path:   "/proc",
		Device: "proc",
		FsType: "proc",
	}

	rootFs, err := getBlockDevice(tmpMnt.Path)
	assert.NoError(t, err, "Expected no error in getting block device")
	assert.Equal(t, tmpMnt.Path, rootFs.Path, "Expected path to be /mock/path")
	assert.Equal(t, tmpMnt.Device, rootFs.Device, "Expected device to be dm-0")
	assert.Equal(t, tmpMnt.FsType, rootFs.FsType, "Expected filesystem type to be ext4")
}
