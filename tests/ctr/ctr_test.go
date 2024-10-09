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

func TestCtrHvtRumprun(t *testing.T) {
	containerImage := "harbor.nbfc.io/nubificus/urunc/hello-hvt-rumprun-nonet:latest"
	containerName := "hvt-rumprun-hello"
	procName := "solo5-hvt"
	pullParams := strings.Fields("ctr image pull " + containerImage)
	pullCmd := exec.Command(pullParams[0], pullParams[1:]...) //nolint:gosec
	err := pullCmd.Run()
	if err != nil {
		t.Fatalf("Error pulling %s: %v", containerImage, err)
	}
	cmdString := fmt.Sprintf("ctr run --rm --snapshotter devmapper --runtime io.containerd.urunc.v2 %s %s", containerImage, containerName)
	params := strings.Fields(cmdString)
	cmd := exec.Command(params[0], params[1:]...) //nolint:gosec
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%v: Error executing unikernel %s with %s using ctr: %s", err, containerName, procName, output)
	}
	expectedContain := "Hello world"
	if !strings.Contains(string(output), expectedContain) {
		t.Fatalf("Expected: %s, Got: %s", expectedContain, output)
	}
}

func TestCtrSptRumprun(t *testing.T) {
	containerImage := "harbor.nbfc.io/nubificus/urunc/hello-spt-rumprun-nonet:latest"
	containerName := "spt-rumprun-hello"
	procName := "solo5-spt"
	pullParams := strings.Fields("ctr image pull " + containerImage)
	pullCmd := exec.Command(pullParams[0], pullParams[1:]...) //nolint:gosec
	err := pullCmd.Run()
	if err != nil {
		t.Fatalf("Error pulling %s: %v", containerImage, err)
	}
	cmdString := fmt.Sprintf("ctr run --rm --snapshotter devmapper --runtime io.containerd.urunc.v2 %s %s", containerImage, containerName)
	params := strings.Fields(cmdString)
	cmd := exec.Command(params[0], params[1:]...) //nolint:gosec
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%v: Error executing unikernel %s with %s using ctr: %s", err, containerName, procName, output)
	}
	expectedContain := "Hello world"
	if !strings.Contains(string(output), expectedContain) {
		t.Fatalf("Expected: %s, Got: %s", expectedContain, output)
	}
}

func TestCtrQemuUnikraftNginx(t *testing.T) {
	containerImage := "harbor.nbfc.io/nubificus/urunc/nginx-qemu-unikraft-initrd:latest"
	containerName := "qemu-unikraft-nginx"
	procName := "qemu-system"
	pullParams := strings.Fields("ctr image pull " + containerImage)
	pullCmd := exec.Command(pullParams[0], pullParams[1:]...) //nolint:gosec
	err := pullCmd.Run()
	if err != nil {
		t.Fatalf("Error pulling %s: %v", containerImage, err)
	}
	cmdString := fmt.Sprintf("ctr run -d --runtime io.containerd.urunc.v2 %s %s", containerImage, containerName)
	params := strings.Fields(cmdString)
	cmd := exec.Command(params[0], params[1:]...) //nolint:gosec
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%v: Error executing unikernel %s with %s using ctr: %s", err, containerName, procName, output)
	}
	params = strings.Fields("ctr c ls -q")
	cmd = exec.Command(params[0], params[1:]...) //nolint:gosec
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%v: Error listing containers using ctr: %s", err, output)
	}
	if !strings.Contains(string(output), containerName) {
		t.Fatalf("Container not running. Expected: %s, Got: %s", containerName, output)
	}
	proc, _ := common.FindProc(containerName)
	err = proc.Kill()
	if err != nil {
		t.Fatalf("%v: Error killing urunc process", err)
	}
	params = strings.Fields("ctr c rm " + containerName)
	cmd = exec.Command(params[0], params[1:]...) //nolint:gosec
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%v: Error deleting container using ctr: %s", err, output)
	}
	params = strings.Fields("ctr c ls -q")
	cmd = exec.Command(params[0], params[1:]...) //nolint:gosec
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%v: Error listing containers using ctr: %s", err, output)
	}
	if strings.Contains(string(output), containerName) {
		t.Fatalf("Container still running. Got: %s", output)
	}
}

func TestCtrFCUnikraftNginx(t *testing.T) {
	containerImage := "harbor.nbfc.io/nubificus/urunc/nginx-firecracker-unikraft-initrd:latest"
	containerName := "fc-unikraft-nginx"
	procName := "firecracker"
	pullParams := strings.Fields("ctr image pull " + containerImage)
	pullCmd := exec.Command(pullParams[0], pullParams[1:]...) //nolint:gosec
	err := pullCmd.Run()
	if err != nil {
		t.Fatalf("Error pulling %s: %v", containerImage, err)
	}
	cmdString := fmt.Sprintf("ctr run -d --runtime io.containerd.urunc.v2 %s %s", containerImage, containerName)
	params := strings.Fields(cmdString)
	cmd := exec.Command(params[0], params[1:]...) //nolint:gosec
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%v: Error executing unikernel %s with %s using ctr: %s", err, containerName, procName, output)
	}
	params = strings.Fields("ctr c ls -q")
	cmd = exec.Command(params[0], params[1:]...) //nolint:gosec
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%v: Error listing containers using ctr: %s", err, output)
	}
	if !strings.Contains(string(output), containerName) {
		t.Fatalf("Container not running. Expected: %s, Got: %s", containerName, output)
	}
	proc, _ := common.FindProc(containerName)
	err = proc.Kill()
	if err != nil {
		t.Fatalf("%v: Error killing urunc process", err)
	}
	params = strings.Fields("ctr c rm " + containerName)
	cmd = exec.Command(params[0], params[1:]...) //nolint:gosec
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%v: Error deleting container using ctr: %s", err, output)
	}
	params = strings.Fields("ctr c ls -q")
	cmd = exec.Command(params[0], params[1:]...) //nolint:gosec
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%v: Error listing containers using ctr: %s", err, output)
	}
	if strings.Contains(string(output), containerName) {
		t.Fatalf("Container still running. Got: %s", output)
	}
}
