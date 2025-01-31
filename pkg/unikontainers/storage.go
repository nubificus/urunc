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
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/moby/sys/mount"
	"github.com/sirupsen/logrus"
)

var ErrMountpoint = errors.New("No FS is mounted in this mountpoint")

// RootFs contains information regarding a mount
type RootFs struct {
	Path   string // The path of the root file system.
	Device string // The device which is mounted as the container rootfs
	FsType string // The filesystem type of the mounted device
}

// getBlockDevice retrieves information about the block device associated with a given path.
// It searches for a mounted block device with the specified path and returns its details.
// If the path is not a block device or there is an error, it returns an empty RootFs struct and an error.
func getBlockDevice(path string) (RootFs, error) {
	var result RootFs
	selfProcMountInfo := "/proc/self/mountinfo"

	file, err := os.Open(selfProcMountInfo)
	if err != nil {
		return result, nil
	}

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, " - ")
		if len(parts) != 2 {
			return result, fmt.Errorf("Invalid mountinfo line in /proc/self/mountinfo")
		}

		fields := strings.Fields(parts[0])
		mountPoint := fields[4]
		if mountPoint != path {
			continue
		}
		result.Path = mountPoint
		fields = strings.Fields(parts[1])
		result.FsType = fields[0]
		result.Device = fields[1]
		Log.WithFields(logrus.Fields{
			"mountpoint": result.Path,
			"device":     result.Device,
			"fstype":     result.FsType,
		}).Debug("Found container rootfs mount")

		return result, nil
	}

	return result, ErrMountpoint
}

// extractUnikernelFromBlock creates target directory inside the bundle and moves unikernel & urunc.json
// FIXME: This approach fills up /run with unikernel binaries and urunc.json files for each unikernel we run
func extractFilesFromBlock(unikernel string, uruncJSON string, initrd string, rootfsPath string) (string, error) {
	// create bundle/tmp directory and moves unikernel binary and urunc.json
	rootBase := filepath.Dir(rootfsPath)
	tmpDir := filepath.Join(rootBase, "/tmp")
	err := os.Mkdir(tmpDir, 0755)
	if err != nil {
		return "", err
	}

	currentUnikernelPath := filepath.Join(rootfsPath, unikernel)
	targetUnikernelPath := filepath.Join(tmpDir, unikernel)
	targetUnikernelDir, _ := filepath.Split(targetUnikernelPath)
	err = moveFile(currentUnikernelPath, targetUnikernelDir)
	if err != nil {
		err1 := os.RemoveAll(tmpDir)
		if err1 != nil {
			Log.Errorf("Could not remove directory %s", tmpDir)
		}
		return "", err
	}

	if initrd != "" {
		currentInitrdPath := filepath.Join(rootfsPath, initrd)
		targetInitrdPath := filepath.Join(tmpDir, initrd)
		targetInitrdDir, _ := filepath.Split(targetInitrdPath)
		err = moveFile(currentInitrdPath, targetInitrdDir)
		if err != nil {
			err1 := os.RemoveAll(tmpDir)
			if err1 != nil {
				Log.Errorf("Could not remove directory %s", tmpDir)
			}
			return "", err
		}
	}

	currentConfigPath := filepath.Join(rootfsPath, uruncJSON)
	err = moveFile(currentConfigPath, tmpDir)
	if err != nil {
		err1 := os.RemoveAll(tmpDir)
		if err1 != nil {
			Log.Errorf("Could not remove directory %s", tmpDir)
		}
		return "", err
	}

	return tmpDir, nil
}

// prepareDMAsBLock copies the files needed for the unikernel boot (e.g.
// unikernel binary, initrd file) and the urunc.json file in a new temporary
// directory. Then it unmounts the devmapper device and renames the temporary
// directory as the container rootfs. This is needed to keep the same paths
// for the unikernel files.
func prepareDMAsBlock(rootfsPath string, unikernel string, uruncJSON string, initrd string) error {
	// extract unikernel
	// FIXME: This approach fills up /run with unikernel binaries and
	// urunc.json files for each unikernel instance we run
	tmpDir, err := extractFilesFromBlock(unikernel, uruncJSON, initrd, rootfsPath)
	if err != nil {
		return err
	}
	// unmount block device
	// FIXME: umount and rm might need some retries
	err = mount.Unmount(rootfsPath)
	if err != nil {
		return err
	}
	// rename tmp to rootfs
	err = os.Remove(rootfsPath)
	if err != nil {
		return err
	}
	err = os.Rename(tmpDir, rootfsPath)
	if err != nil {
		return err
	}

	return nil
}

// cleanupExtractedFiles cleans up all the files that we copied to unmount
// container's rootfs. In particular it should delete three files: the unikernel
// binary the initrd and the urunc.json file.
// For the time being it acts as a placeholder for future changes, where we might
// need to do more advanced things than removing files.
func cleanupExtractedFiles(rootfsPath string) error {
	return os.RemoveAll(rootfsPath)
}
