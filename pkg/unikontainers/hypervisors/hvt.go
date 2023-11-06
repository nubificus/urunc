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
	"os/exec"
	"strings"
	"syscall"
)

const (
	HvtVmm    VmmType = "hvt"
	HvtBinary string  = "solo5-hvt"
)

type HVT struct {
	binaryPath string
	binary     string
}

// Stop is an empty function to satisfy VMM interface compatibility requirements.
// It does not perform any actions and always returns nil.
func (h *HVT) Stop(_ string) error {
	return nil
}

// Path returns the path to the hvt binary.
func (h *HVT) Path() string {
	return h.binaryPath
}

// Ok checks if the hvt binary is available in the system's PATH.
func (h *HVT) Ok() error {
	if _, err := exec.LookPath(HvtBinary); err != nil {
		return ErrVMMNotInstalled
	}
	return nil
}

func (h *HVT) Execve(args ExecArgs) error {
	cmdString := h.binaryPath + " --mem=512"
	cmdString = appendNonEmpty(cmdString, " --net:tap=", args.TapDevice)
	cmdString = appendNonEmpty(cmdString, " --block:rootfs=", args.BlockDevice)
	cmdString += " " + args.UnikernelPath + " " + args.Command
	cmdArgs := strings.Split(cmdString, " ")
	vmmLog.WithField("hvt command", cmdString).Error("Ready to execve hvt")
	return syscall.Exec(h.binaryPath, cmdArgs, args.Environment) //nolint: gosec
}
