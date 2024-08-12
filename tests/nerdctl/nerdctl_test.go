package urunc

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"strconv"
	"testing"

	common "github.com/nubificus/urunc/tests"

	"time"
)

type testMethod func(testArgs) error

type nerdctlTestArgs struct {
	Name string
	Image string
	Devmapper bool
	Seccomp bool
	Skippable bool
}

type testArgs struct {
	ContainerID string
	Seccomp bool
	Expected string
}

func TestNerdctlHvtRumprunHello(t *testing.T) {
	containerImage := "harbor.nbfc.io/nubificus/urunc/hello-hvt-rump:latest"
	containerName := "hvt-rumprun-hello-test"
	err := runTest(containerName, containerImage, true, true, "Hello world", matchTest)
	if err != nil {
		t.Fatal(err.Error())
	}
}

func TestNerdctlHvtRumprunRedis(t *testing.T) {
	containerImage := "harbor.nbfc.io/nubificus/urunc/redis-hvt-rump:latest"
	containerName := "hvt-rumprun-redis-test"
	err := runTest(containerName, containerImage, true, true, "", pingTest)
	if err != nil {
		t.Fatal(err.Error())
	}
}

func TestNerdctlHvtSeccompOn(t *testing.T) {
	containerImage := "harbor.nbfc.io/nubificus/urunc/redis-hvt-rump:latest"
	containerName := "hvt-rumprun-redis-test"
	err := runTest(containerName, containerImage, true, true, "", seccompTest)
	if err != nil {
		t.Fatal(err.Error())
	}
}

func TestNerdctlHvtSeccompOff(t *testing.T) {
	containerImage := "harbor.nbfc.io/nubificus/urunc/redis-hvt-rump:latest"
	containerName := "hvt-rumprun-redis-test"
	err := runTest(containerName, containerImage, true, false, "", seccompTest)
	if err != nil {
		t.Fatal(err.Error())
	}
}

func TestNerdctlSptRumprunRedis(t *testing.T) {
	containerImage := "harbor.nbfc.io/nubificus/urunc/redis-spt-rump:latest"
	containerName := "spt-rumprun-redis-test"
	err := runTest(containerName, containerImage, true, true, "", pingTest)
	if err != nil {
		t.Fatal(err.Error())
	}
}

func TestNerdctlQemuUnikraftRedis(t *testing.T) {
	containerImage := "harbor.nbfc.io/nubificus/urunc/redis-qemu-unikraft-initrd:latest"
	containerName := "qemu-unik-redis-test"
	err := runTest(containerName, containerImage, false, true, "", pingTest)
	if err != nil {
		t.Fatal(err.Error())
	}
}

func TestNerdctlQemuUnikraftNginx(t *testing.T) {
	containerImage := "harbor.nbfc.io/nubificus/urunc/nginx-qemu-unikraft:latest"
	containerName := "qemu-unik-nginx-test"
	err := runTest(containerName, containerImage, false, true, "", pingTest)
	if err != nil {
		t.Fatal(err.Error())
	}
}

func TestNerdctlQemuSeccompOn(t *testing.T) {
	containerImage := "harbor.nbfc.io/nubificus/urunc/redis-qemu-unikraft-initrd:latest"
	containerName := "qemu-unik-redis-test"
	err := runTest(containerName, containerImage, true, true, "", seccompTest)
	if err != nil {
		t.Fatal(err.Error())
	}
}

func TestNerdctlQemuSeccompOff(t *testing.T) {
	containerImage := "harbor.nbfc.io/nubificus/urunc/redis-qemu-unikraft-initrd:latest"
	containerName := "qemu-unik-redis-test"
	err := runTest(containerName, containerImage, true, false, "", seccompTest)
	if err != nil {
		t.Fatal(err.Error())
	}
}

func TestNerdctlFCUnikraftNginx(t *testing.T) {
	containerImage := "harbor.nbfc.io/nubificus/urunc/nginx-fc-unik:latest"
	containerName := "fc-unik-nginx-test"
	err := runTest(containerName, containerImage, false, true, "", pingTest)
	if err != nil {
		t.Fatal(err.Error())
	}
}

func TestNerdctlFCSeccompOn(t *testing.T) {
	containerImage := "harbor.nbfc.io/nubificus/urunc/nginx-fc-unik:latest"
	containerName := "fc-unik-nginx-test"
	err := runTest(containerName, containerImage, true, true, "", seccompTest)
	if err != nil {
		t.Fatal(err.Error())
	}
}

func TestNerdctlFCSeccompOff(t *testing.T) {
	containerImage := "harbor.nbfc.io/nubificus/urunc/nginx-fc-unik:latest"
	containerName := "fc-unik-nginx-test"
	err := runTest(containerName, containerImage, true, false, "", seccompTest)
	if err != nil {
		t.Fatal(err.Error())
	}
}

func runTest(containerName string, containerImage string, devmapper bool, seccomp bool, pattern string, fn testMethod) error {
	nerdctlArgs := nerdctlTestArgs {
		Image : containerImage,
		Name : containerName,
		Devmapper : devmapper,
		Seccomp : seccomp,
	}
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
	testArguments := testArgs {
		ContainerID : containerID,
		Seccomp : seccomp,
		Expected : pattern,
	}
	return fn(testArguments)
}

func seccompTest(args testArgs) error {
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

func matchTest(args testArgs) error {
	return findInUnikernelLogs(args.ContainerID, args.Expected)
}

func pingTest(args testArgs) error {
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
