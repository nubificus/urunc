package urunc

import (
	"os/exec"
	"strings"
	"testing"
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
