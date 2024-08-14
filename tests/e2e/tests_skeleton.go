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

func nerdctlRunTest(nerdctlArgs containerTestArgs) error {
	containerID, err := startNerdctlUnikernel(nerdctlArgs)
	if err != nil {
		return fmt.Errorf("Failed to start unikernel container: %v", err)
	}
	// Give some time till the unikernel is up and running.
	// Maybe we need to revisit this in the future.
	time.Sleep(2 * time.Second)
	defer func() {
		// We do not want a succesful cleanup to overwrite any previous error
		if tempErr := nerdctlCleanup(containerID); tempErr != nil {
			err = tempErr
		}
	}()
	testArguments := testSpecificArgs {
		ContainerID : containerID,
		Seccomp : nerdctlArgs.Seccomp,
		Expected : nerdctlArgs.TestArgs.Expected,
	}
	return nerdctlArgs.TestFunc(testArguments)
}

func seccompTest(args testSpecificArgs) error {
	unikernelPID, err := findUnikernelKey(args.ContainerID, "State", "Pid")
	if err != nil {
		return fmt.Errorf("Failed to extract container IP: %v", err)
	}
	procPath := "/proc/" + unikernelPID + "/status"
	seccompLine, err:= common.FindLineInFile(procPath, "Seccomp")
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
			} else {
				return strconv.FormatFloat(value.(float64), 'f', -1, 64), nil
			}
		}
	}
	return "", nil
}

func startNerdctlUnikernel(nerdctlArgs containerTestArgs) (containerID string, err error) {
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

func pullImage(Image string) error {
	pullParams := strings.Fields("ctr image pull " + Image)
	pullCmd := exec.Command(pullParams[0], pullParams[1:]...) //nolint:gosec
	err := pullCmd.Run()
	if err != nil {
		return fmt.Errorf("Error pulling %s: %v", Image, err)
	}

	return nil
}

func ctrRunTest(ctrArgs containerTestArgs) error {
	output, err := startCtrUnikernel(ctrArgs)
	if err != nil {
		return fmt.Errorf("Failed to start unikernel container: %v", err)
	}
	if !strings.Contains(string(output), ctrArgs.TestArgs.Expected) {
		return fmt.Errorf("Expected: %s, Got: %s", ctrArgs.TestArgs.Expected, output)
	}
	defer func() {
		// We do not want a succesful cleanup to overwrite any previous error
		if tempErr := ctrCleanup(ctrArgs.Name); tempErr != nil {
			err = tempErr
		}
	}()
	return nil
}

func startCtrUnikernel(ctrArgs containerTestArgs) (output []byte, err error) {
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
