package urunc

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"

	common "github.com/nubificus/urunc/tests"
)

type testMethod func(testSpecificArgs) error

var matchTest testMethod = nil

type ctrTestArgs struct {
	Name string
	Image string
	Devmapper bool
	Seccomp bool
	Skippable bool
	TestFunc testMethod
	TestArgs testSpecificArgs
}

type testSpecificArgs struct {
	ContainerID string
	Seccomp bool
	Expected string
}

//func TestsWithCtr(t *testing.T) {
func TestCtr(t *testing.T) {
	tests := []ctrTestArgs {
		{
			Image : "harbor.nbfc.io/nubificus/urunc/hello-hvt-nonet:latest",
			Name : "hvt-rumprun-hello",
			Devmapper : true,
			Seccomp : true,
			Skippable: false,
			TestArgs : testSpecificArgs {
				Expected : "Hello world",
			},
			TestFunc: matchTest,
		},
		{
			Image : "harbor.nbfc.io/nubificus/urunc/hello-spt-nonet:latest",
			Name : "spt-rumprun-hello",
			Devmapper : true,
			Seccomp : true,
			Skippable: false,
			TestArgs : testSpecificArgs {
				Expected : "Hello world",
			},
			TestFunc: matchTest,
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


func runTest(ctrArgs ctrTestArgs) error {
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

// TODO: Need to replace this test
func TestCtrQemuUnikraftNginx(t *testing.T) {
	containerImage := "harbor.nbfc.io/nubificus/urunc/nginx-qemu-unikraft:latest"
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

// TODO: Need to replace this test
func TestCtrFCUnikraftNginx(t *testing.T) {
	containerImage := "harbor.nbfc.io/nubificus/urunc/nginx-fc-unik:latest"
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
