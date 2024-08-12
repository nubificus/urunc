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
	"fmt"
	"os/exec"
	"strings"
	"testing"

	common "github.com/nubificus/urunc/tests"
)

type testMethod func(testSpecificArgs) error

var matchTest testMethod

type ctrTestArgs struct {
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

// func TestsWithCtr(t *testing.T) {
func TestCtr(t *testing.T) {
	tests := []ctrTestArgs{
		{
			Image:     "harbor.nbfc.io/nubificus/urunc/hello-hvt-rumprun-nonet:latest",
			Name:      "Hvt-rumprun-hello",
			Devmapper: true,
			Seccomp:   true,
			Skippable: false,
			TestArgs: testSpecificArgs{
				Expected: "Hello world",
			},
			TestFunc: matchTest,
		},
		{
			Image:     "harbor.nbfc.io/nubificus/urunc/hello-spt-rumprun-nonet:latest",
			Name:      "Spt-rumprun-hello",
			Devmapper: true,
			Seccomp:   true,
			Skippable: false,
			TestArgs: testSpecificArgs{
				Expected: "Hello world",
			},
			TestFunc: matchTest,
		},
		{
			Image:     "harbor.nbfc.io/nubificus/urunc/hello-qemu-unikraft:latest",
			Name:      "Qemu-unikraft-hello",
			Devmapper: false,
			Seccomp:   true,
			Skippable: false,
			TestArgs: testSpecificArgs{
				Expected: "\"Urunc\" \"Unikraft\" \"Qemu\"",
			},
			TestFunc: matchTest,
		},
		{
			Image:     "harbor.nbfc.io/nubificus/urunc/hello-firecracker-unikraft:latest",
			Name:      "Firecracker-unikraft-hello",
			Devmapper: false,
			Seccomp:   true,
			Skippable: false,
			TestArgs: testSpecificArgs{
				Expected: "\"Urunc\" \"Unikraft\" \"FC\"",
			},
			TestFunc: matchTest,
		},
	}
	for _, tc := range tests {
		t.Run(tc.Name, func(t *testing.T) {
			err := pullImage(tc.Image)
			if err != nil {
				t.Fatal(err.Error())
			}
			err = runTest(tc)
			if err != nil {
				t.Fatal(err.Error())
			}
		})
	}
}

func pullImage(image string) error {
	pullParams := strings.Fields("ctr image pull " + image)
	pullCmd := exec.Command(pullParams[0], pullParams[1:]...) //nolint:gosec
	err := pullCmd.Run()
	if err != nil {
		return fmt.Errorf("Error pulling %s: %v", image, err)
	}

	return nil
}

func runTest(ctrArgs ctrTestArgs) error {
	output, err := startCtrUnikernel(ctrArgs)
	if err != nil {
		return fmt.Errorf("Failed to start unikernel container: %v", err)
	}
	if !strings.Contains(string(output), ctrArgs.TestArgs.Expected) {
		return fmt.Errorf("Expected: %s, Got: %s", ctrArgs.TestArgs.Expected, output)
	}
	defer func() {
		// We do not want a successful cleanup to overwrite any previous error
		if tempErr := ctrCleanup(ctrArgs.Name); tempErr != nil {
			err = tempErr
		}
	}()
	return nil
}

func startCtrUnikernel(ctrArgs ctrTestArgs) (output []byte, err error) {
	cmdBase := "ctr "
	cmdBase += "run "
	cmdBase += "--rm "
	cmdBase += "--runtime io.containerd.urunc.v2 "
	if ctrArgs.Devmapper {
		cmdBase += "--snapshotter devmapper "
	}
	cmdBase += ctrArgs.Image + " "
	cmdBase += ctrArgs.Name
	params := strings.Fields(cmdBase)
	cmd := exec.Command(params[0], params[1:]...) //nolint:gosec
	return cmd.CombinedOutput()
}

func ctrCleanup(containerID string) error {
	err := removeCtrUnikernel(containerID)
	if err != nil {
		return fmt.Errorf("Failed to remove container: %v", err)
	}
	err = verifyCtrRemoved(containerID)
	if err != nil {
		return fmt.Errorf("Failed to remove container: %v", err)
	}
	err = common.VerifyNoStaleFiles(containerID)
	if err != nil {
		return fmt.Errorf("Failed to remove all stale files: %v", err)
	}

	return nil
}

func removeCtrUnikernel(containerID string) error {
	params := strings.Fields(fmt.Sprintf("ctr rm %s", containerID))
	cmd := exec.Command(params[0], params[1:]...) //nolint:gosec
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("deleting %s failed: - %v", containerID, err)
	}
	return nil
}

func verifyCtrRemoved(containerID string) error {
	params := strings.Fields("ctr c ls -q")
	cmd := exec.Command(params[0], params[1:]...) //nolint:gosec
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: Error listing containers using ctr: %s", err, output)
	}
	if strings.Contains(string(output), containerID) {
		return fmt.Errorf("Container still running. Got: %s", output)
	}
	return nil
}
