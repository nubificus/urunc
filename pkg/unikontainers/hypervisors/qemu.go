// Copyright (c) 2023-2025, Nubificus LTD
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
	"runtime"
	"strings"
	"syscall"

	"github.com/nubificus/urunc/pkg/unikontainers/unikernels"
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

func (q *Qemu) Execve(args ExecArgs, ukernel unikernels.Unikernel) error {
	qemuString := string(QemuVmm)
	qemuMem := bytesToStringMB(args.MemSizeB)
	cmdString := q.binaryPath + " -m " + qemuMem + "M"
	cmdString += " -cpu host"            // Choose CPU
	cmdString += " -enable-kvm"          // Enable KVM to use CPU virt extensions
	cmdString += " -nographic -vga none" // Disable graphic output

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
		netcli := ukernel.MonitorNetCli(qemuString)
		if netcli == "" {
			netcli += " -net nic,model=virtio"
			netcli += " -net tap,script=no,downscript=no,ifname="
		}
		netcli += args.TapDevice
		cmdString += netcli
	} else {
		cmdString += " -nic none"
	}
	if args.BlockDevice != "" {
		blockCli := ukernel.MonitorBlockCli(qemuString)
		if blockCli == "" {
			blockCli += " -device virtio-blk-pci,id=blk0,drive=hd0,scsi=off"
			blockCli += " -drive format=raw,if=none,id=hd0,file="
		}
		blockCli += args.BlockDevice
		cmdString += blockCli
	}
	if args.InitrdPath != "" {
		cmdString += " -initrd " + args.InitrdPath
	}

	if len(args.NinePFSVols) > 0 {
		cmdString += " -fsdev local,id=p9idfs1,path=" + args.NinePFSVols[0].Src + ",security_model=mapped-xattr"
		cmdString += " -device virtio-9p-pci,fsdev=p9idfs1,mount_tag=fs1"
	}

	cmdString += ukernel.MonitorCli(qemuString)
	exArgs := strings.Split(cmdString, " ")
	exArgs = append(exArgs, "-append", args.Command)
	vmmLog.WithField("qemu command", exArgs).Info("Ready to execve qemu")
	return syscall.Exec(q.Path(), exArgs, args.Environment) //nolint: gosec
}
