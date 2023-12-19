package urunc

import (
	"os/exec"
	"strings"
	"testing"

	common "github.com/nubificus/urunc/tests"
)

const NotImplemented = "Not implemented"

func TestCtrHvtUnikraft(t *testing.T) {
	t.Log(NotImplemented)
}

func TestCtrHvtRumprun(t *testing.T) {
	pullParams := strings.Split("ctr image pull harbor.nbfc.io/nubificus/urunc/hello-hvt-nonet@sha256:6d005854fdc6760898f4c9832cfbb8310103b510ef518e1ea996013ca44040bd", " ")
	pullCmd := exec.Command(pullParams[0], pullParams[1:]...) //nolint:gosec
	err := pullCmd.Run()
	if err != nil {
		t.Fatalf("Error pulling hello-hvt-rump:latest image: %v", err)
	}
	params := strings.Split("ctr run --rm --snapshotter devmapper --runtime io.containerd.urunc.v2 harbor.nbfc.io/nubificus/urunc/hello-hvt-nonet@sha256:6d005854fdc6760898f4c9832cfbb8310103b510ef518e1ea996013ca44040bd testhvtunikraft", " ")
	cmd := exec.Command(params[0], params[1:]...) //nolint:gosec
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%v: Error executing rumprun unikernel with solo5-hvt using ctr: %s", err, output)
	}
	expectedContain := "Hello world"
	if !strings.Contains(string(output), expectedContain) {
		t.Fatalf("Expected: %s, Got: %s", expectedContain, output)
	}
}

func TestCtrSptUnikraft(t *testing.T) {
	t.Log(NotImplemented)
}

func TestCtrSptRumprun(t *testing.T) {
	t.Log(NotImplemented)
}

func TestCtrQemuUnikraftNginx(t *testing.T) {
	pullParams := strings.Split("ctr image pull harbor.nbfc.io/nubificus/urunc/nginx-qemu-unikraft:latest", " ")
	pullCmd := exec.Command(pullParams[0], pullParams[1:]...) //nolint:gosec
	err := pullCmd.Run()
	if err != nil {
		t.Fatalf("Error pulling nginx-qemu-unikraft:latest image: %v", err)
	}
	params := strings.Split("ctr run -d --runtime io.containerd.urunc.v2 harbor.nbfc.io/nubificus/urunc/nginx-qemu-unikraft:latest ctrqmunik", " ")
	cmd := exec.Command(params[0], params[1:]...) //nolint:gosec
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%v: Error executing unikraft unikernel with qemu using ctr: %s", err, output)
	}
	params = strings.Split("ctr c ls -q", " ")
	cmd = exec.Command(params[0], params[1:]...) //nolint:gosec
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%v: Error listing containers using ctr: %s", err, output)
	}
	expectedContain := "ctrqmunik"
	if !strings.Contains(string(output), expectedContain) {
		t.Fatalf("Container not running. Expected: %s, Got: %s", expectedContain, output)
	}
	proc, _ := common.FindProc("ctrqmunik")
	err = proc.Kill()
	if err != nil {
		t.Fatalf("%v: Error killing urunc process", err)
	}
	params = strings.Split("ctr c rm ctrqmunik", " ")
	cmd = exec.Command(params[0], params[1:]...) //nolint:gosec
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%v: Error deleting container using ctr: %s", err, output)
	}
	params = strings.Split("ctr c ls -q", " ")
	cmd = exec.Command(params[0], params[1:]...) //nolint:gosec
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%v: Error listing containers using ctr: %s", err, output)
	}
	if strings.Contains(string(output), expectedContain) {
		t.Fatalf("Container still running. Expected: %s, Got: %s", expectedContain, output)
	}
}
