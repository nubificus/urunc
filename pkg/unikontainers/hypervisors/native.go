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
	"time"
)

const (
	NativeVmm    VmmType = "native"
	NativeBinary string  = ""
)

type Native struct {
	binaryPath string
	binary     string
}

func (n *Native) Stop(_ string) error {
	return nil
}

func (n *Native) Ok() error {
	return nil
}
func (n *Native) Path() string {
	return n.binaryPath
}

func (n *Native) Execve(args ExecArgs) error {
	cmdString := args.UnikernelPath + args.Command
	exArgs := strings.Split(cmdString, " ")
	vmmLog.WithField("native command", exArgs).Info("Ready to execve native")
	time.Sleep(20*time.Second)
	return syscall.Exec(args.UnikernelPath, exArgs, args.Environment) //nolint: gosec
}
