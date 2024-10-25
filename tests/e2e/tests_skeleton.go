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
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	common "github.com/nubificus/urunc/tests"

	"time"
)

type testTool interface {
	getTestArgs() containerTestArgs
	getContainerID() string
	setContainerID(string)
	pullImage() error
	rmImage() error
	createContainer() (string, error)
	startContainer(bool) (string, error)
	runContainer(bool) (string, error)
	stopContainer() error
	rmContainer() error
	logContainer() (string, error)
	searchContainer(string) (bool, error)
	inspectAndGet(string) (string, error)
}

var matchTest testMethod
var errToolDoesNotSUpport = errors.New("Operarion not support")

func runTest1(tool testTool) (err error) {
	cntrArgs := tool.getTestArgs()
	err = tool.pullImage()
	if err != nil {
		return fmt.Errorf("Failed to pull container imeage: %s - %v", cntrArgs.Image, err)
	}
	var output string
	if cntrArgs.TestFunc == nil {
		output, err = tool.runContainer(false)
		tool.setContainerID(cntrArgs.Name)
	} else {
		output, err = tool.runContainer(true)
		tool.setContainerID(output)
	}
	if err != nil {
		return fmt.Errorf("Failed to run unikernel container: %v", err)
	}
	defer func() {
		// We do not want a successful cleanup to overwrite any previous error
		if tempErr := testCleanup1(tool); tempErr != nil {
			err = tempErr
		}
	}()
	if cntrArgs.TestFunc == nil {
		if !strings.Contains(string(output), cntrArgs.ExpectOut) {
			return fmt.Errorf("Expected: %s, Got: %s", cntrArgs.ExpectOut, output)
		}
		return err
	}
	// Give some time till the unikernel is up and running.
	// Maybe we need to revisit this in the future.
	time.Sleep(2 * time.Second)
	return cntrArgs.TestFunc(tool)
}

func runTest(tool string, cntrArgs containerTestArgs) (err error) {
	var output string
	if cntrArgs.TestFunc == nil {
		output, err = startContainer(tool, cntrArgs, false)
	} else {
		output, err = startContainer(tool, cntrArgs, true)
	}
	if err != nil {
		return fmt.Errorf("Failed to start unikernel container: %v", err)
	}
	defer func() {
		// We do not want a successful cleanup to overwrite any previous error
		if tempErr := testCleanup(tool, cntrArgs.Name); tempErr != nil {
			err = tempErr
		}
	}()
	if cntrArgs.TestFunc == nil {
		if !strings.Contains(string(output), cntrArgs.ExpectOut) {
			return fmt.Errorf("Expected: %s, Got: %s", cntrArgs.ExpectOut, output)
		}
		return err
	}
	// Give some time till the unikernel is up and running.
	// Maybe we need to revisit this in the future.
	time.Sleep(2 * time.Second)
	// return cntrArgs.TestFunc(cntrArgs, output)
	return nil
}

func testCleanup1(tool testTool) error {
	err := tool.stopContainer()
	if err != nil {
		return fmt.Errorf("Failed to stop container: %v", err)
	}
	err = tool.rmContainer()
	if err != nil {
		return fmt.Errorf("Failed to remove container: %v", err)
	}
	containerID := tool.getContainerID()
	exists, err := tool.searchContainer(containerID)
	if exists || err != nil {
		return fmt.Errorf("Container %s is not removed: %v", containerID, err)
	}
	err = common.VerifyNoStaleFiles(containerID)
	if err != nil {
		return fmt.Errorf("Failed to remove all stale files: %v", err)
	}

	return nil
}

func testCleanup(tool string, containerID string) error {
	if tool != "ctr" && tool != "nerdctl" {
		return fmt.Errorf("Unknown tool %s", tool)
	}
	err := stopContainer(tool, containerID)
	if err != nil {
		return fmt.Errorf("Failed to stop container: %v", err)
	}
	err = removeContainer(tool, containerID)
	if err != nil {
		return fmt.Errorf("Failed to remove container: %v", err)
	}
	err = verifyContainerRemoved(tool, containerID)
	if err != nil {
		return fmt.Errorf("Failed to remove container: %v", err)
	}
	err = common.VerifyNoStaleFiles(containerID)
	if err != nil {
		return fmt.Errorf("Failed to remove all stale files: %v", err)
	}

	return nil
}

func findValOfKey(searchArea string, key string) (string, error) {
	keystr := "\"" + key + "\":[^,;\\]}]*"
	r, err := regexp.Compile(keystr)
	if err != nil {
		return "", err
	}
	match := r.FindString(searchArea)
	keyValMatch := strings.Split(match, ":")
	val := strings.ReplaceAll(keyValMatch[1], "\"", "")
	return strings.TrimSpace(val), nil
}

func commonNewContainerCmd(a containerTestArgs) string {
	cmdBase := "--runtime io.containerd.urunc.v2 "
	if a.Devmapper {
		cmdBase += "--snapshotter devmapper "
	}
	if !a.Seccomp {
		cmdBase += "--security-opt seccomp=unconfined "
	}
	cmdBase += a.Image + " "
	cmdBase += a.Name
	return cmdBase
}

func commonCmdExec(command string) (output string, err error) {
	params := strings.Fields(command)
	cmd := exec.Command(params[0], params[1:]...) //nolint:gosec
	outBytes, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s - %v", output, err)
	}
	output = string(outBytes)
	output = strings.TrimSpace(output)
	return output, nil
}

func commonPull(tool string, image string) error {
	pullCmd := tool + " image pull " + image

	_, err := commonCmdExec(pullCmd)
	return err
}

func commonRmImage(tool string, image string) error {
	pullCmd := tool + " image rm " + image

	_, err := commonCmdExec(pullCmd)
	return err
}

// nolint:unused
func commonCreate(tool string, cntrArgs containerTestArgs) (output string, err error) {
	cmdBase := tool + " create "
	cmdBase += commonNewContainerCmd(cntrArgs)
	return commonCmdExec(cmdBase)
}

func commonStart(tool string, cID string, attach bool) (output string, err error) {
	cmdBase := tool + " start "
	cmdBase += "--runtime io.containerd.urunc.v2 "
	if attach {
		if tool != "ctr t" {
			cmdBase += "--attach "
		}
	} else {
		if tool == "ctr t" {
			cmdBase += "--detach "
		}
	}
	cmdBase += cID
	return commonCmdExec(cmdBase)
}

// nolint:unused
func commonRun(tool string, cntrArgs containerTestArgs, detach bool) (output string, err error) {
	cmdBase := tool
	cmdBase += " run "
	if detach {
		cmdBase += "-d "
	}
	cmdBase += commonNewContainerCmd(cntrArgs)
	return commonCmdExec(cmdBase)
}

// nolint:unused
func commonLogs(tool string, cID string) (string, error) {
	logCmd := tool + " logs " + cID

	return commonCmdExec(logCmd)
}

// nolint:unused
func commonSearchContainer(tool string, cID string) (bool, error) {
	cmd := tool + " ps -a --no-trunc -q"

	output, err := commonCmdExec(cmd)
	if err != nil {
		return true, err
	}
	return searchCID(output, cID), nil
}

// nolint:unused
func commonInspectAndGet(tool string, containerID string, key string) (string, error) {
	cmdBase := tool
	cmdBase += " inspect "
	cmdBase += containerID
	output, err := commonCmdExec(cmdBase)
	if err != nil {
		return "", err
	}

	return findValOfKey(output, key)
}

// nolint:unused
func commonStopContainer(tool string, containerID string) error {
	cmdBase := tool
	cmdBase += " stop "
	cmdBase += containerID
	_, err := commonCmdExec(cmdBase)
	return err
}

func commonRmContainer(tool string, containerID string) (string, error) {
	cmdBase := tool
	cmdBase += " rm "
	cmdBase += containerID
	return commonCmdExec(cmdBase)
}

func startContainer(tool string, cntrArgs containerTestArgs, detach bool) (output string, err error) {
	cmdBase := "run "
	if detach {
		cmdBase += "-d "
	}
	cmdBase += "--runtime io.containerd.urunc.v2 "
	if cntrArgs.Devmapper {
		cmdBase += "--snapshotter devmapper "
	}
	switch tool {
	case "ctr":
		if cntrArgs.Seccomp {
			cmdBase += "--seccomp "
		}
		cmdBase += cntrArgs.Image + " "
		cmdBase += cntrArgs.Name
	case "nerdctl":
		if !cntrArgs.Seccomp {
			cmdBase += "--security-opt seccomp=unconfined "
		}
		cmdBase += "--name " + cntrArgs.Name
		cmdBase += " " + cntrArgs.Image
	default:
		return "", fmt.Errorf("Unknown tool %s", tool)
	}
	params := strings.Fields(cmdBase)
	cmd := exec.Command(tool, params...) //nolint:gosec
	outBytes, err := cmd.CombinedOutput()
	output = string(outBytes)
	output = strings.TrimSpace(output)
	if err != nil {
		return "", fmt.Errorf("%s - %v", output, err)
	}
	return output, nil
}

func stopContainer(tool string, containerID string) error {
	var params []string
	switch tool {
	case "ctr":
		params = strings.Fields("ctr t kill " + containerID)
	case "nerdctl":
		params = strings.Fields("nerdctl stop " + containerID)
	default:
		return fmt.Errorf("Unknown tool %s", tool)
	}
	cmd := exec.Command(params[0], params[1:]...) //nolint:gosec
	output, err := cmd.CombinedOutput()
	retMsg := strings.TrimSpace(string(output))
	if err != nil && (tool != "ctr" || !strings.Contains(retMsg, "not found")) {
		return fmt.Errorf("stop %s failed: %s - %v", containerID, retMsg, err)
	}
	if tool == "nerdctl" && containerID != retMsg {
		return fmt.Errorf("unexpected output when stopping %s. expected: %s, got: %s", containerID, containerID, retMsg)
	}
	return nil
}

func removeContainer(tool string, containerID string) error {
	var rmcmd string
	if tool == "ctr" {
		rmcmd = " c"
	}
	rmcmd += " rm " + containerID
	params := strings.Fields(rmcmd)
	cmd := exec.Command(tool, params...) //nolint:gosec
	output, err := cmd.CombinedOutput()
	retMsg := strings.TrimSpace(string(output))
	if err != nil {
		return fmt.Errorf("deleting %s failed: %s - %v", containerID, retMsg, err)
	}
	if tool == "nerdctl" && containerID != retMsg {
		return fmt.Errorf("unexpected output when deleting %s. expected: %s, got: %s", containerID, containerID, retMsg)
	}
	return nil
}

func searchCID(searchArea string, containerID string) bool {
	found := false
	lines := strings.Split(searchArea, "\n")
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
	return found
}

func verifyContainerRemoved(tool string, containerID string) error {
	var params []string
	switch tool {
	case "ctr":
		params = strings.Fields("ctr c ls -q")
	case "nerdctl":
		params = strings.Fields("nerdctl ps -a --no-trunc -q")
	default:
		return fmt.Errorf("Unknown tool %s", tool)
	}
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
