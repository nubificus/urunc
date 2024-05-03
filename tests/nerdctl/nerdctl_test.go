package urunc

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	common "github.com/nubificus/urunc/tests"

	"time"
)

const NotImplemented = "Not implemented"

func TestNerdctlHvtRumprunHello(t *testing.T) {
	params := strings.Fields("nerdctl run --name hello --rm --snapshotter devmapper --runtime io.containerd.urunc.v2 harbor.nbfc.io/nubificus/urunc/hello-hvt-rump:latest")
	cmd := exec.Command(params[0], params[1:]...) //nolint:gosec
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%v: Error executing rumprun unikernel with solo5-hvt using nerdctl: %s", err, output)
	}
	if !strings.Contains(string(output), "Hello world") {
		t.Fatalf("Expected: %s, Got: %s", "Hello world", output)
	}
}

func TestNerdctlHvtRumprunRedis(t *testing.T) {
	containerImage := "harbor.nbfc.io/nubificus/urunc/redis-hvt-rump:latest"
	containerName := "hvt-rumprun-redis-test"
	err := nerdctlTest(containerName, containerImage, true)
	if err != nil {
		t.Fatal(err.Error())
	}
}

func TestNerdctlSptRumprunRedis(t *testing.T) {
	containerImage := "harbor.nbfc.io/nubificus/urunc/redis-spt-rump:latest"
	containerName := "spt-rumprun-redis-test"
	err := nerdctlTest(containerName, containerImage, true)
	if err != nil {
		t.Fatal(err.Error())
	}
}

func TestNerdctlQemuUnikraftRedis(t *testing.T) {
	containerImage := "harbor.nbfc.io/nubificus/urunc/redis-qemu-unikraft-initrd:latest"
	containerName := "qemu-unik-redis-test"
	err := nerdctlTest(containerName, containerImage, false)
	if err != nil {
		t.Fatal(err.Error())
	}
}

func TestNerdctlQemuUnikraftNginx(t *testing.T) {
	containerImage := "harbor.nbfc.io/nubificus/urunc/nginx-qemu-unikraft:latest"
	containerName := "qemu-unik-nginx-test"
	err := nerdctlTest(containerName, containerImage, false)
	if err != nil {
		t.Fatal(err.Error())
	}
}

func TestNerdctlFCUnikraftNginx(t *testing.T) {
	containerImage := "harbor.nbfc.io/nubificus/urunc/nginx-fc-unik:latest"
	containerName := "fc-unik-nginx-test"
	err := nerdctlTest(containerName, containerImage, false)
	if err != nil {
		t.Fatal(err.Error())
	}
}

func nerdctlTest(containerName string, containerImage string, devmapper bool) error {
	containerID, err := startNerdctlUnikernel(containerImage, containerName, devmapper)
	if err != nil {
		return fmt.Errorf("Failed to start unikernel container: %v", err)
	}
	time.Sleep(4 * time.Second)
	extractedIPAddr, err := findUnikernelIP(containerID)
	if err != nil {
		return fmt.Errorf("Failed to extract container IP: %v", err)
	}
	err = common.PingUnikernel(extractedIPAddr)
	if err != nil {
		return fmt.Errorf("ping failed: %v", err)
	}
	err = stopNerdctlUnikernel(containerID)
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
	err = verifyNoStaleFiles(containerID)
	if err != nil {
		return fmt.Errorf("Failed to remove all stale files: %v", err)
	}
	return nil
}

func findUnikernelIP(containerID string) (string, error) {
	params := strings.Fields(fmt.Sprintf("nerdctl inspect %s", containerID))
	cmd := exec.Command(params[0], params[1:]...) //nolint:gosec
	var result []map[string]any
	var networkSettings map[string]any
	time.Sleep(4 * time.Second)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to inspect %s", output)
	}
	err = json.Unmarshal(output, &result)
	if err != nil {
		return "", err
	}
	containerInfo := result[0]
	for key, value := range containerInfo {
		// Each value is an `any` type, that is type asserted as a string
		if key == "NetworkSettings" {
			// t.Log(key, fmt.Sprintf("%v", value))
			networkSettings = value.(map[string]any)
			break
		}
	}
	for key, value := range networkSettings {
		if key == "IPAddress" {
			return value.(string), nil
		}
	}
	return "", nil
}

func startNerdctlUnikernel(containerImage string, containerName string, devmapper bool) (containerID string, err error) {
	cmdBase := "nerdctl run "
	if devmapper {
		cmdBase += "--snapshotter devmapper "
	}
	cmdline := fmt.Sprintf("%s--name %s -d --runtime io.containerd.urunc.v2 %s unikernel", cmdBase, containerName, containerImage)
	params := strings.Fields(cmdline)
	cmd := exec.Command(params[0], params[1:]...) //nolint:gosec
	containerIDBytes, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("%s - %v", string(containerIDBytes), err)
	}
	time.Sleep(4 * time.Second)
	containerID = string(containerIDBytes)
	containerID = strings.TrimSpace(containerID)
	return containerID, nil
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

func verifyNoStaleFiles(containerID string) error {
	// Check /run/containerd/runc/default/containerID directory does not exist
	dirPath := "/run/containerd/runc/default/" + containerID
	_, err := os.Stat(dirPath)
	if !os.IsNotExist(err) {
		return fmt.Errorf("root directory %s still exists", dirPath)
	}

	// Check /run/containerd/io.containerd.runtime.v2.task/default/containerID directory does not exist
	dirPath = "run/containerd/io.containerd.runtime.v2.task/default/" + containerID
	_, err = os.Stat(dirPath)
	if !os.IsNotExist(err) {
		return fmt.Errorf("bundle directory %s still exists", dirPath)
	}
	return nil
}
