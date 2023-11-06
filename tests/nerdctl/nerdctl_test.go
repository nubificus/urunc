package urunc

import (
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/go-ping/ping"
	common "github.com/nubificus/urunc/tests"
)

const NotImplemented = "Not implemented"

func TestNerdctlHvtRumprunHello(t *testing.T) {
	params := strings.Split("nerdctl run --name hello --rm --snapshotter devmapper --runtime io.containerd.urunc.v2 harbor.nbfc.io/nubificus/urunc/hello-hvt-rump:latest", " ")
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
	params := strings.Split("nerdctl run --name redis-test -d --snapshotter devmapper --runtime io.containerd.urunc.v2 harbor.nbfc.io/nubificus/urunc/redis-hvt-rump:latest", " ")
	cmd := exec.Command(params[0], params[1:]...) //nolint:gosec
	containerIDBytes, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%v: Error executing redis rumprun unikernel with solo5-hvt using nerdctl: %s", err, containerIDBytes)
	}
	time.Sleep(2 * time.Second)
	// Get the container ID
	containerID := string(containerIDBytes)
	containerID = strings.TrimSpace(containerID)

	// Find the solo5-hvt process
	proc, err := common.FindProc("solo5-hvt")
	if err != nil {
		t.Fatalf("Failed to find solo5-hvt process: %v", err)
	}
	cmdLine, err := proc.Cmdline()
	if err != nil {
		t.Fatalf("Failed to find solo5-hvt process' command line: %v", err)
	}

	// Extract the IP address
	re := regexp.MustCompile(`"addr":"([0-9]+\.[0-9]+\.[0-9]+\.[0-9]+)"`)
	match := re.FindStringSubmatch(cmdLine)
	extractedIPAddr := ""
	if len(match) == 2 {
		extractedIPAddr = match[1]
	} else {
		t.Fatal("Failed to extract IP address for solo5 process")
	}

	// Create a new Pinger and ping the redis IP
	pinger, err := ping.NewPinger(extractedIPAddr)
	if err != nil {
		t.Fatalf("Failed to create Pinger: %v", err)
	}
	pinger.Count = 3
	err = pinger.Run() // Blocks until finished.
	if err != nil {
		t.Fatalf("Failed to ping %s: %v", extractedIPAddr, err)
	}
	// Stop the unikernel
	stopCmdString := "nerdctl stop " + containerID
	params = strings.Split(stopCmdString, " ")
	cmd = exec.Command(params[0], params[1:]...) //nolint:gosec
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("%v: Error stopping redis rumprun unikernel with solo5-hvt using nerdctl", err)
	}
	retMsg := strings.TrimSpace(string(output))
	if containerID != retMsg {
		t.Fatalf("Unexpected output when stopping redis. Expected: %s, got: %s", containerID, retMsg)
	}

	// Delete the unikernel
	deleteCmdString := "nerdctl rm " + containerID
	params = strings.Split(deleteCmdString, " ")
	cmd = exec.Command(params[0], params[1:]...) //nolint:gosec
	output, err = cmd.Output()
	if err != nil {
		t.Fatalf("%v: Error deleting redis rumprun unikernel with solo5-hvt using nerdctl", err)
	}
	retMsg = strings.TrimSpace(string(output))
	if containerID != retMsg {
		t.Fatalf("Unexpected output when deleting redis. Expected: %s, got: %s", containerID, retMsg)
	}

	// Check the unikernel is removed from nerdctl ps -a
	listCmdString := "nerdctl ps -a --no-trunc -q"
	params = strings.Split(listCmdString, " ")
	cmd = exec.Command(params[0], params[1:]...) //nolint:gosec
	output, err = cmd.Output()
	if err != nil {
		t.Fatalf("%v: Error listing all containers using nerdctl", err)
	}
	outputStr := string(output)
	lines := strings.Split(outputStr, "\n")
	for _, line := range lines {
		// Skip empty lines
		if line == "" {
			continue
		}
		cID := strings.TrimSpace(line)
		if cID == containerID {
			t.Fatalf("Unikernel %s was not successfully removed from nerdctl", containerID)
		}
	}

	// Check /run/containerd/runc/default/containerID directory does not exist
	dirPath := "/run/containerd/runc/default/" + containerID
	_, err = os.Stat(dirPath)
	if !os.IsNotExist(err) {
		t.Fatalf("root directory %s still exists", dirPath)
	}

	// Check /run/containerd/io.containerd.runtime.v2.task/default/containerID directory does not exist
	dirPath = "run/containerd/io.containerd.runtime.v2.task/default/" + containerID
	_, err = os.Stat(dirPath)
	if !os.IsNotExist(err) {
		t.Fatalf("bundle directory %s still exists", dirPath)
	}
}

func TestNerdctlSptUnikraft(t *testing.T) {
	t.Log(NotImplemented)
}

func TestNerdctlSptRumprun(t *testing.T) {
	t.Log(NotImplemented)
}
