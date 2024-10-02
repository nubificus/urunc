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
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"syscall"
)

const (
	QemuVmm    VmmType = "qemu"
	QemuBinary string  = "qemu-system-"
)

type Qemu struct {
	binaryPath string
	binary     string
}

func (q *Qemu) Stop(_ string) error {
	return nil
}

func (q *Qemu) Ok() error {
	return nil
}
func (q *Qemu) Path() string {
	return q.binaryPath
}

func (q *Qemu) Execve(args ExecArgs) error {
	var cmdString string
	if args.MemSizeMiB != "" {
		memory, err := strconv.ParseInt(args.MemSizeMiB, 10, 64)
		if err != nil {
			return fmt.Errorf("%s failed to parse memory size %w", args.MemSizeMiB, err)
		}
		memory = int64(bytesToMB(memory))
		cmdString = fmt.Sprintf("%s -cpu host -m %d -enable-kvm -nographic -vga none", q.Path(), memory)
	} else {
		cmdString = fmt.Sprintf("%s -cpu host -m %s -enable-kvm -nographic -vga none", q.Path(), DefaultMemory)
	}

	if args.Seccomp {
		// Enable Seccomp in QEMU
		cmdString += " --sandbox on"
		// Allow or Deny Obsolete system calls
		cmdString += ",obsolete=deny"
		// Allow or Deny set*uid|gid system calls
		cmdString += ",elevateprivileges=deny"
		// Allow or Deny *fork and execve
		cmdString += ",spawn=deny"
		// Allow or Deny process affinity and schedular priority
		cmdString += ",resourcecontrol=deny"
	}

	// TODO: Check if this check causes any performance drop
	// or explore alternative implementations
	if runtime.GOARCH == "arm64" {
		machineType := " -M virt"
		cmdString += machineType
	}

	cmdString += " -kernel " + args.UnikernelPath
	if args.TapDevice != "" {
		cmdString += " -net nic,model=virtio -net tap,script=no,ifname=" + args.TapDevice
	}
	if args.BlockDevice != "" {
		// TODO: For the time being, we only have support for initrd with
		// QEMU and Unikraft. We will need to add support for block device
		// and other storage options in QEMU (e.g. shared fs)
		vmmLog.Warn("Block device is currently not supported in QEMU execution")
	}
	if args.InitrdPath != "" {
		cmdString += " -initrd " + args.InitrdPath
	}
	exArgs := strings.Split(cmdString, " ")
	exArgs = append(exArgs, "-append", args.Command)
	vmmLog.WithField("qemu command", exArgs).Info("Ready to execve qemu")
	return syscall.Exec(q.Path(), exArgs, args.Environment) //nolint: gosec
}
