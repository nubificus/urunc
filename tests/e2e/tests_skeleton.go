package uruncE2ETesting

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"strconv"

	common "github.com/nubificus/urunc/tests"

	"time"
)

var matchTest testMethod = nil

func pullImage(Image string) error {
	pullParams := strings.Fields("ctr image pull " + Image)
	pullCmd := exec.Command(pullParams[0], pullParams[1:]...) //nolint:gosec
	err := pullCmd.Run()
	if err != nil {
		return fmt.Errorf("Error pulling %s: %v", Image, err)
	}

	return nil
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
		// We do not want a succesful cleanup to overwrite any previous error
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
	return cntrArgs.TestFunc(cntrArgs)
}

func testCleanup(tool string,containerID string) error {
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
		if cntrArgs.Seccomp == false {
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
			} else {
				return strconv.FormatFloat(value.(float64), 'f', -1, 64), nil
			}
		}
	}
	return "", nil
}
