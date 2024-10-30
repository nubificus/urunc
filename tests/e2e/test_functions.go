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

package urunce2etesting

import (
	"fmt"
	"strings"

	common "github.com/nubificus/urunc/tests"
)

func seccompTest(tool testTool) error {
	args := tool.getTestArgs()
	unikernelPID, err := tool.inspectAndGet("Pid")
	if err != nil {
		return fmt.Errorf("Failed to extract unikernel PID: %v", err)
	}
	procPath := "/proc/" + unikernelPID + "/status"
	seccompLine, err := common.FindLineInFile(procPath, "Seccomp")
	if err != nil {
		return err
	}
	wordsInLine := strings.Split(seccompLine, ":")
	if strings.TrimSpace(wordsInLine[1]) == "2" {
		if !args.Seccomp {
			return fmt.Errorf("Seccomp should not be enabled")
		}
	} else {
		if args.Seccomp {
			return fmt.Errorf("Seccomp should be enabled")
		}
	}

	return nil
}

func pingTest(tool testTool) error {
	extractedIPAddr, err := tool.inspectAndGet("IPAddress")
	if err != nil {
		return fmt.Errorf("Failed to extract container IP: %v", err)
	}
	err = common.PingUnikernel(extractedIPAddr)
	if err != nil {
		return fmt.Errorf("ping failed: %v", err)
	}

	return nil
}
