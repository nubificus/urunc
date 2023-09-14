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

func (q *Qemu) Execve(data ExecData) error {
	cmdString := q.Path() + " --mem=16"
	if data.BlkDev != "" {
		cmdString += " --disk=" + data.BlkDev
	}
	if data.TapDev != "" {
		cmdString += " --net=" + data.TapDev
	}
	// TODO: Add cmdline

	cmdString += " " + data.Unikernel
	vmmLog.Info(cmdString)

	args := strings.Split(cmdString, " ")
	vmmLog.Info(args)
	return syscall.Exec(q.Path(), args, data.Environment) //nolint: gosec
}
