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
	"os/exec"
	"runtime"

	"github.com/nubificus/urunc/internal/log"
	unet "github.com/nubificus/urunc/pkg/network"
)

type VmmType string

type ExecData struct {
	Container   string
	Unikernel   string
	TapDev      string
	BlkDev      string
	CmdLine     string
	Environment []string
	Network     unet.UnikernelNetworkInfo
}

var ErrNotSupportedVMM = errors.New("vmm is not supported")
var ErrVMMNotInstalled = errors.New("vmm not found")
var vmmLog = log.BaseLogEntry().WithField("subsystem", "hypervisors")

type VMM interface {
	Execve(data ExecData) error
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
	case HedgeVmm:
		hedge := Hedge{}
		err := hedge.Ok()
		if err != nil {
			return nil, ErrVMMNotInstalled
		}
		return &hedge, nil
	default:
		return nil, ErrNotSupportedVMM
	}
}

func cpuArch() string {
	switch runtime.GOARCH {
	case "arm64":
		return "aarch64"
	case "amd64":
		return "x86_64"
	default:
		return ""
	}
}
