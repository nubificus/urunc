package uruncE2ETesting

import (
	"fmt"
	"os/exec"
	"strings"

	common "github.com/nubificus/urunc/tests"
)

func pullImage(Image string) error {
	pullParams := strings.Fields("ctr image pull " + Image)
	pullCmd := exec.Command(pullParams[0], pullParams[1:]...) //nolint:gosec
	err := pullCmd.Run()
	if err != nil {
		return fmt.Errorf("Error pulling %s: %v", Image, err)
	}

	return nil
}

func runTest(ctrArgs containerTestArgs) error {
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

