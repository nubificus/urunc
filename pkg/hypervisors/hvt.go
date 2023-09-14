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

// Execve executes the hvt binary with the provided execution data.
func (h *HVT) Execve(data ExecData) error {
	// TODO: Perhaps let the user define mem value somehow (?)
	cmdString := h.binaryPath + " --mem=512"
	if data.TapDev != "" {
		cmdString += " --net=" + data.TapDev
	}
	if data.BlkDev != "" {
		cmdString += " --disk=" + data.BlkDev
	}

	// TODO: Implement a mechanism to distinguish between unikernel types (eg rumprun unikraft etc)
	// Create a Rumprun configuration and convert it to JSON
	rumprunConfig, err := NewRumprunConfig(data)
	if err != nil {
		return err
	}
	unikernelCmd, err := rumprunConfig.ToJSONString()
	if err != nil {
		return err
	}
	cmdString += " " + data.Unikernel + " " + unikernelCmd
	vmmLog.WithField("hvt command", cmdString).WithField("IP", data.Network.EthDevice.IP).Debug("Ready to execve hvt")
	args := strings.Split(cmdString, " ")
	return syscall.Exec(h.binaryPath, args, data.Environment) //nolint: gosec
}
