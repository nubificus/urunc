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

package hypervisors

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
)

const (
	FirecrackerVmm    VmmType = "firecracker"
	FirecrackerBinary string  = "firecracker"
	FCJsonFilename    string  = "fc.json"
)

type Firecracker struct {
	binaryPath string
	binary     string
}

type FirecrackerBootSource struct {
	ImagePath  string `json:"kernel_image_path"`
	BootArgs   string `json:"boot_args"`
	InitrdPath string `json:"initrd_path,omitempty"`
}

type FirecrackerMachine struct {
	VcpuCount       uint   `json:"vcpu_count"`
	MemSizeMiB      uint64 `json:"mem_size_mib"`
	Smt             bool   `json:"smt"`
	TrackDirtyPages bool   `json:"track_dirty_pages"`
}

type FirecrackerDrive struct {
	DriveID   string `json:"drive_id"`
	IsRO      bool   `json:"is_read_only"`
	IsRootDev bool   `json:"is_root_device"`
	HostPath  string `json:"path_on_host"`
}

type FirecrackerNet struct {
	IfaceID  string `json:"iface_id"`
	GuestMAC string `json:"guest_mac,omitempty"`
	HostIF   string `json:"host_dev_name"`
}

type FirecrackerConfig struct {
	Source  FirecrackerBootSource `json:"boot-source"`
	Machine FirecrackerMachine    `json:"machine-config"`
	Drives  []FirecrackerDrive    `json:"drives"`
	NetIfs  []FirecrackerNet      `json:"network-interfaces"`
}

func (fc *Firecracker) Stop(_ string) error {
	return nil
}

func (fc *Firecracker) Ok() error {
	return nil
}
func (fc *Firecracker) Path() string {
	return fc.binaryPath
}

func (fc *Firecracker) Execve(args ExecArgs) error {
	cmdString := fc.Path() + " --no-api --config-file "
	JSONConfigDir := filepath.Dir(args.UnikernelPath)
	JSONConfigFile := filepath.Join(JSONConfigDir, FCJsonFilename)
	cmdString += JSONConfigFile
	if !args.Seccomp {
		cmdString += " --no-seccomp"
	}

	// VM config for Firecracker
	fcMem := DefaultMemory
	if args.MemSizeB != 0 {
		fcMem = bytesToMiB(args.MemSizeB)
		// Check if memory is too small
		if fcMem == 0 {
			fcMem = DefaultMemory
		}
	}
	FCMachine := FirecrackerMachine{
		VcpuCount:       1, // TODO: Use value from configuration or Environment variable
		MemSizeMiB:      fcMem,
		Smt:             false,
		TrackDirtyPages: false,
	}

	// Net config for Firecracker
	FCNet := make([]FirecrackerNet, 0)
	AnIF := FirecrackerNet{
		IfaceID:  "net1",
		GuestMAC: args.GuestMAC,
		HostIF:   args.TapDevice,
	}
	FCNet = append(FCNet, AnIF)

	// Block config for Firecracker
	// TODO: Add support for block devices in FIrecracker
	FCDrives := make([]FirecrackerDrive, 0)

	// TODO: Check if this check causes any performance drop
	// or explore alternative implementations
	if runtime.GOARCH == "arm64" {
		consoleStr := " console=ttyS0"
		args.Command += consoleStr
	}

	FCSource := FirecrackerBootSource{
		ImagePath:  args.UnikernelPath,
		BootArgs:   args.Command,
		InitrdPath: args.InitrdPath,
	}
	FCConfig := &FirecrackerConfig{
		Source:  FCSource,
		Machine: FCMachine,
		Drives:  FCDrives,
		NetIfs:  FCNet,
	}
	FCConfigJSON, _ := json.Marshal(FCConfig)
	if err := os.WriteFile(JSONConfigFile, FCConfigJSON, 0o644); err != nil { //nolint: gosec
		return fmt.Errorf("failed to save Firecracker json config: %w", err)
	}
	vmmLog.WithField("Json=", string(FCConfigJSON)).Info("Firecracker json config")

	exArgs := strings.Split(cmdString, " ")
	vmmLog.WithField("Firecracker command", exArgs).Info("Ready to execve Firecracker")

	return syscall.Exec(fc.Path(), exArgs, args.Environment) //nolint: gosec
}
