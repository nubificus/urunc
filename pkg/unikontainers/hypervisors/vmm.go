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
	"errors"
	"fmt"
	"os/exec"

	"github.com/sirupsen/logrus"
)

// ExecArgs holds the data required by Execve to start the VMM
// FIXME: add extra fields if required by additional VMM's
type ExecArgs struct {
	Container     string   // The container ID
	UnikernelPath string   // The path of the unikernel inside rootfs
	TapDevice     string   // The TAP device name
	BlockDevice   string   // The block device path
	Command       string   // The unikernel's command line
	IPAddress     string   // The IP address of the TAP device
	GuestMAC      string   // The MAC address of the guest network device
	Environment   []string // Environment
}

type VmmType string

var ErrVMMNotInstalled = errors.New("vmm not found")
var vmmLog = logrus.WithField("subsystem", "hypervisors")

type VMM interface {
	Execve(args ExecArgs) error
	Stop(t string) error
	Path() string
	Ok() error
}

func NewVMM(vmmType VmmType) (vmm VMM, err error) {
	defer func() {
		if err != nil {
			vmmLog.Error(err.Error())
		}
	}()
	switch vmmType {
	case HvtVmm:
		vmmPath, err := exec.LookPath(HvtBinary)
		if err != nil {
			return nil, ErrVMMNotInstalled
		}
		return &HVT{binary: HvtBinary, binaryPath: vmmPath}, nil
	case QemuVmm:
		vmmPath, err := exec.LookPath(QemuBinary + cpuArch())
		if err != nil {
			return nil, ErrVMMNotInstalled
		}
		return &Qemu{binary: QemuBinary, binaryPath: vmmPath}, nil
	case FirecrackerVmm:
		vmmPath, err := exec.LookPath(FirecrackerBinary + cpuArch())
		if err != nil {
			return nil, ErrVMMNotInstalled
		}
		return &Firecracker{binary: FirecrackerBinary, binaryPath: vmmPath}, nil
	case HedgeVmm:
		hedge := Hedge{}
		err := hedge.Ok()
		if err != nil {
			return nil, ErrVMMNotInstalled
		}
		return &hedge, nil
	default:
		return nil, fmt.Errorf("vmm \"%s\" is not supported", vmmType)
	}
}
