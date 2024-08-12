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

package urunc

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"testing"

	common "github.com/nubificus/urunc/tests"

	"time"
)

type testMethod func(testSpecificArgs) error

type nerdctlTestArgs struct {
	Name      string
	Image     string
	Devmapper bool
	Seccomp   bool
	Skippable bool
	TestFunc  testMethod
	TestArgs  testSpecificArgs
}

type testSpecificArgs struct {
	ContainerID string
	Seccomp     bool
	Expected    string
}

// func TestsWithNerdctl(t *testing.T) {
func TestNerdctl(t *testing.T) {
	tests := []nerdctlTestArgs{
		{
			Image:     "harbor.nbfc.io/nubificus/urunc/hello-hvt-rumprun:latest",
			Name:      "Hvt-rumprun-capture-hello",
			Devmapper: true,
			Seccomp:   true,
			Skippable: false,
			TestArgs: testSpecificArgs{
				Expected: "Hello world",
			},
			TestFunc: matchTest,
		},
		{
			Image:     "harbor.nbfc.io/nubificus/urunc/redis-hvt-rumprun:latest",
			Name:      "Hvt-rumprun-ping-redis",
			Devmapper: true,
			Seccomp:   true,
			Skippable: false,
			TestFunc:  pingTest,
		},
		{
			Image:     "harbor.nbfc.io/nubificus/urunc/redis-hvt-rumprun-block:latest",
			Name:      "Hvt-rumprun-ping-redis-with-block",
			Devmapper: true,
			Seccomp:   true,
			Skippable: false,
			TestFunc:  pingTest,
		},
		{
			Image:     "harbor.nbfc.io/nubificus/urunc/redis-hvt-rumprun:latest",
			Name:      "Hvt-rumprun-with-seccomp",
			Devmapper: true,
			Seccomp:   true,
			Skippable: false,
			TestFunc:  seccompTest,
		},
		{
			Image:     "harbor.nbfc.io/nubificus/urunc/redis-hvt-rumprun:latest",
			Name:      "Hvt-rumprun-without-seccomp",
			Devmapper: true,
			Seccomp:   false,
			Skippable: false,
			TestFunc:  seccompTest,
		},
		{
			Image:     "harbor.nbfc.io/nubificus/urunc/redis-spt-rumprun:latest",
			Name:      "Spt-rumprun-ping-redis",
			Devmapper: true,
			Seccomp:   true,
			Skippable: false,
			TestFunc:  pingTest,
		},
		{
			Image:     "harbor.nbfc.io/nubificus/urunc/redis-qemu-unikraft-initrd:latest",
			Name:      "Qemu-unikraft-ping-redis",
			Devmapper: false,
			Seccomp:   true,
			Skippable: false,
			TestFunc:  pingTest,
		},
		{
			Image:     "harbor.nbfc.io/nubificus/urunc/nginx-qemu-unikraft-initrd:latest",
			Name:      "Qemu-unikraft-ping-nginx",
			Devmapper: false,
			Seccomp:   true,
			Skippable: false,
			TestFunc:  pingTest,
		},
		{
			Image:     "harbor.nbfc.io/nubificus/urunc/redis-qemu-unikraft-initrd:latest",
			Name:      "Qemu-unikraft-with-seccomp",
			Devmapper: false,
			Seccomp:   true,
			Skippable: false,
			TestFunc:  seccompTest,
		},
		{
			Image:     "harbor.nbfc.io/nubificus/urunc/redis-qemu-unikraft-initrd:latest",
			Name:      "Qemu-unikraft-without-seccomp",
			Devmapper: false,
			Seccomp:   false,
			Skippable: false,
			TestFunc:  seccompTest,
		},
		{
			Image:     "harbor.nbfc.io/nubificus/urunc/nginx-firecracker-unikraft-initrd:latest",
			Name:      "Firecracker-unikraft-ping-nginx",
			Devmapper: false,
			Seccomp:   true,
			Skippable: false,
			TestFunc:  pingTest,
		},
		{
			Image:     "harbor.nbfc.io/nubificus/urunc/nginx-firecracker-unikraft-initrd:latest",
			Name:      "Firecracker-unikraft-with-seccomp",
			Devmapper: false,
			Seccomp:   true,
			Skippable: false,
			TestFunc:  seccompTest,
		},
		{
			Image:     "harbor.nbfc.io/nubificus/urunc/nginx-firecracker-unikraft-initrd:latest",
			Name:      "Firecracker-unikraft-without-seccomp",
			Devmapper: false,
			Seccomp:   false,
			Skippable: false,
			TestFunc:  seccompTest,
		},
	}
	for _, tc := range tests {
		t.Run(tc.Name, func(t *testing.T) {
			err := runTest(tc)
			if err != nil {
				t.Fatal(err.Error())
			}
		})
	}
}

func runTest(nerdctlArgs nerdctlTestArgs) error {
	containerID, err := startNerdctlUnikernel(nerdctlArgs)
	if err != nil {
		return fmt.Errorf("Failed to start unikernel container: %v", err)
	}
	// Give some time till the unikernel is up and running.
	// Maybe we need to revisit this in the future.
	time.Sleep(2 * time.Second)
	defer func() {
		// We do not want a successful cleanup to overwrite any previous error
		if tempErr := nerdctlCleanup(containerID); tempErr != nil {
			err = tempErr
		}
	}()
	testArguments := testSpecificArgs{
		ContainerID: containerID,
		Seccomp:     nerdctlArgs.Seccomp,
		Expected:    nerdctlArgs.TestArgs.Expected,
	}
	return nerdctlArgs.TestFunc(testArguments)
}

func seccompTest(args testSpecificArgs) error {
	unikernelPID, err := findUnikernelKey(args.ContainerID, "State", "Pid")
	if err != nil {
		return fmt.Errorf("Failed to extract container IP: %v", err)
	}
	procPath := "/proc/" + unikernelPID + "/status"
	seccompLine, err := common.FindLineInFile(procPath, "Seccomp")
	if err != nil {
		return err
	}
	wordsInLine := strings.Split(seccompLine, ":")
	if strings.TrimSpace(wordsInLine[1]) == "2" {
		if args.Seccomp == false {
			return fmt.Errorf("Seccomp should not be enabled")
		}
	} else {
		if args.Seccomp == true {
			return fmt.Errorf("Seccomp should be enabled")
		}
	}

	return nil
}

func matchTest(args testSpecificArgs) error {
	return findInUnikernelLogs(args.ContainerID, args.Expected)
}

func pingTest(args testSpecificArgs) error {
	extractedIPAddr, err := findUnikernelKey(args.ContainerID, "NetworkSettings", "IPAddress")
	if err != nil {
		return fmt.Errorf("Failed to extract container IP: %v", err)
	}
	err = common.PingUnikernel(extractedIPAddr)
	if err != nil {
		return fmt.Errorf("ping failed: %v", err)
	}

	return nil
}

func nerdctlCleanup(containerID string) error {
	err := stopNerdctlUnikernel(containerID)
	if err != nil {
		return fmt.Errorf("Failed to stop container: %v", err)
	}
	err = removeNerdctlUnikernel(containerID)
	if err != nil {
		return fmt.Errorf("Failed to remove container: %v", err)
	}
	err = verifyNerdctlRemoved(containerID)
	if err != nil {
		return fmt.Errorf("Failed to remove container: %v", err)
	}
	err = common.VerifyNoStaleFiles(containerID)
	if err != nil {
		return fmt.Errorf("Failed to remove all stale files: %v", err)
	}

	return nil
}

func findUnikernelKey(containerID string, field string, key string) (string, error) {
	params := strings.Fields(fmt.Sprintf("nerdctl inspect %s", containerID))
	cmd := exec.Command(params[0], params[1:]...) //nolint:gosec
	var result []map[string]any
	var fieldInfo map[string]any
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to inspect %s", output)
	}
	err = json.Unmarshal(output, &result)
	if err != nil {
		return "", err
	}
	containerInfo := result[0]
	for object, value := range containerInfo {
		// Each value is an `any` type, that is type asserted as a string
		if object == field {
			// t.Log(key, fmt.Sprintf("%v", value))
			fieldInfo = value.(map[string]any)
			break
		}
	}
	for object, value := range fieldInfo {
		if object == key {
			retVal, ok := value.(string)
			if ok {
				return retVal, nil
			}
			return strconv.FormatFloat(value.(float64), 'f', -1, 64), nil
		}
	}
	return "", nil
}

func startNerdctlUnikernel(nerdctlArgs nerdctlTestArgs) (containerID string, err error) {
	cmdBase := "nerdctl "
	cmdBase += "run "
	cmdBase += "-d "
	cmdBase += "--runtime io.containerd.urunc.v2 "
	if nerdctlArgs.Devmapper {
		cmdBase += "--snapshotter devmapper "
	}
	if nerdctlArgs.Seccomp == false {
		cmdBase += "--security-opt seccomp=unconfined "
	}
	if nerdctlArgs.Name != "" {
		cmdBase += "--name " + nerdctlArgs.Name + " "
	}
	cmdBase += nerdctlArgs.Image + " "
	cmdBase += "unikernel "
	params := strings.Fields(cmdBase)
	cmd := exec.Command(params[0], params[1:]...) //nolint:gosec
	containerIDBytes, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("%s - %v", string(containerIDBytes), err)
	}
	containerID = string(containerIDBytes)
	containerID = strings.TrimSpace(containerID)
	return containerID, nil
}

func findInUnikernelLogs(containerID string, pattern string) error {
	cmdStr := "nerdctl logs " + containerID
	params := strings.Fields(cmdStr)
	cmd := exec.Command(params[0], params[1:]...) //nolint:gosec
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("Could not retrieve logs for container %s: %v", containerID, err)
	}
	if !strings.Contains(string(output), pattern) {
		return fmt.Errorf("Expected: %s, Got: %s", pattern, output)
	}
	return nil
}

func stopNerdctlUnikernel(containerID string) error {
	params := strings.Fields(fmt.Sprintf("nerdctl stop %s", containerID))
	cmd := exec.Command(params[0], params[1:]...) //nolint:gosec
	output, err := cmd.Output()
	retMsg := strings.TrimSpace(string(output))
	if err != nil {
		return fmt.Errorf("stop %s failed: %s - %v", containerID, retMsg, err)
	}
	if containerID != retMsg {
		return fmt.Errorf("unexpected output when stopping %s. expected: %s, got: %s", containerID, containerID, retMsg)
	}
	return nil
}

func removeNerdctlUnikernel(containerID string) error {
	params := strings.Fields(fmt.Sprintf("nerdctl rm %s", containerID))
	cmd := exec.Command(params[0], params[1:]...) //nolint:gosec
	output, err := cmd.Output()
	retMsg := strings.TrimSpace(string(output))
	if err != nil {
		return fmt.Errorf("deleting %s failed: %s - %v", containerID, retMsg, err)
	}
	if containerID != retMsg {
		return fmt.Errorf("unexpected output when deleting %s. expected: %s, got: %s", containerID, containerID, retMsg)
	}
	return nil
}

func verifyNerdctlRemoved(containerID string) error {
	params := strings.Fields("nerdctl ps -a --no-trunc -q")
	cmd := exec.Command(params[0], params[1:]...) //nolint:gosec
	output, err := cmd.Output()
	retMsg := strings.TrimSpace(string(output))
	if err != nil {
		return fmt.Errorf("listing all nerdctl containers failed: %s - %v", retMsg, err)
	}
	found := false
	lines := strings.Split(retMsg, "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		cID := strings.TrimSpace(line)
		if cID == containerID {
			found = true
			break
		}
	}
	if found {
		return fmt.Errorf("unikernel %s was not successfully removed from nerdctl", containerID)
	}
	return nil
}
