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

package hypervisors

import (
	"os"
	"fmt"
	"strings"
	"syscall"
	"encoding/json"
	"path/filepath"
)

const (
	FirecrackerVmm    VmmType = "firecracker"
	FirecrackerBinary string  = "firecracker-"
)

type Firecracker struct {
	binaryPath string
	binary     string
}

type FirecrackerBootSource struct {
	ImagePath	string	`json:"kernel_image_path"`
	BootArgs	string	`json:"boot_args"`
	InitrdPath	string	`json:"initrd_path"`
}

type FirecrackerMachine struct {
	VcpuCount	uint	`json:"vcpu_count"`
	MemSizeMiB	uint	`json:"mem_size_mib"`
	Smt		bool	`json:"smt"`
	TrackDirtyPages bool	`json:"track_dirty_pages"`
}

type FirecrackerDrive struct {
	DriveID		string	`json:"drive_id"`
	IsRO		bool	`json:"is_read_only"`
	IsRootDev	bool	`json:"is_root_device"`
	HostPath	string	`json:"path_on_host"`
}

type FirecrackerNet struct {
	IfaceID		string	`json:"iface_id"`
	GuestMAC	*string	`json:"guest_mac"`
	HostIF		string	`json:"host_dev_name"`
}

type FirecrackerConfig struct {
	Source     FirecrackerBootSource `json:"boot-source"`
	Machine    FirecrackerMachine    `json:"machine-config"`
	Drives     []FirecrackerDrive    `json:"drives"`
	NetIfs     []FirecrackerNet	 `json:"network-interfaces"`
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
	JsonConfigDir := filepath.Dir(args.UnikernelPath)
	JsonConfigFile := filepath.Join(JsonConfigDir, "fc.json")
	cmdString += JsonConfigFile

	vmmLog.WithField("FC cmd", cmdString).Info("Firecracker command")

	// VM config for Firecracker
	FCMachine := FirecrackerMachine{
		VcpuCount: 1,
		MemSizeMiB: 32,
		Smt: false,
		TrackDirtyPages: false,
	}

	// Net config for Firecracker
	FCNet := make([]FirecrackerNet, 0)
	GMac := args.GuestMAC //TODO
	AnIF := FirecrackerNet{
		IfaceID: "net1",
		GuestMAC: &GMac,
		HostIF: args.TapDevice,
	}
	FCNet = append(FCNet, AnIF)

	// Block config for Firecracker
	FCDrives := make([]FirecrackerDrive, 0)

	//GUest kernel and CLI options configuration for FIrecracker
	//
	// For some reason Unikraft in Firecracker requires the first
	// argument in boot_args to be the name of the application.
	// Therefore, we use the first word from the CLI options
	// that the user gave, as the name of the application.
	// The rest of cli options are passed normally.
	//GuestCLIOpts := data.CmdLine[:strings.Index(data.CmdLine, " ")]
	//data.CmdLine = strings.TrimLeft(data.CmdLine, GuestCLIOpts)
	//RestGuestCLIOpts, _ := UnikraftCli(data)
	//GuestCLIOpts += " " + RestGuestCLIOpts

	FCSource := FirecrackerBootSource{
		ImagePath:  args.UnikernelPath,
		BootArgs:   args.Command,
		// TODO: We assume that the block is the initrd. We will need to
		// revisit this later
		InitrdPath: args.BlockDevice,
	}
	FCConfig := &FirecrackerConfig{
		Source:  FCSource,
		Machine: FCMachine,
		Drives:  FCDrives,
		NetIfs:  FCNet,
	}
	FCConfigJson, _ := json.Marshal(FCConfig)
	if err := os.WriteFile(JsonConfigFile, FCConfigJson, 0o644); err != nil { //nolint: gosec
		return fmt.Errorf("failed to save Firecracker json config: %w", err)
	}
	if err := os.WriteFile("/tmp/fc.json", FCConfigJson, 0o644); err != nil { //nolint: gosec
		return fmt.Errorf("failed to save Firecracker json config: %w", err)
	}
	vmmLog.WithField("Json=", string(FCConfigJson)).Info("Firecracker json config")

	exArgs := strings.Split(cmdString, " ")
	vmmLog.WithField("Firecracker command", exArgs).Info("Ready to execve Firecracker")
	return syscall.Exec(fc.Path(), exArgs, args.Environment) //nolint: gosec
}
