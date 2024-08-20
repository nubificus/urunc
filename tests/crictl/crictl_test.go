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

package urunc

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/asaskevich/govalidator"
	common "github.com/nubificus/urunc/tests"
	_ "github.com/opencontainers/runc/libcontainer/nsenter"
	"github.com/vishvananda/netns"
)

type testMethod func(containerInfo) error

type containerTestArgs struct {
	Name           string
	Image          string
	Devmapper      bool
	Seccomp        bool
	StaticNet      bool
	SideContainers []string
	Skippable      bool
	TestPath       string
	TestFunc       testMethod
	ExpectOut      string
}

type containerInfo struct {
	PodID       string
	ContainerID string
	Name        string
	Seccomp     bool
}

func TestCrictl(t *testing.T) {
	tests := []containerTestArgs{
		{
			Image:          "harbor.nbfc.io/nubificus/urunc/redis-hvt-rumprun:latest",
			Name:           "Hvt-rumptun-redis",
			Devmapper:      true,
			Seccomp:        true,
			StaticNet:      false,
			SideContainers: []string{},
			Skippable:      false,
			TestPath:       t.TempDir(),
			TestFunc:       pingTest,
		},
		{
			Image:          "harbor.nbfc.io/nubificus/urunc/redis-spt-rumprun:latest",
			Name:           "Spt-rumptun-redis",
			Devmapper:      true,
			Seccomp:        true,
			StaticNet:      false,
			SideContainers: []string{},
			Skippable:      false,
			TestPath:       t.TempDir(),
			TestFunc:       pingTest,
		},
		{
			Image:          "harbor.nbfc.io/nubificus/urunc/redis-qemu-unikraft-initrd:latest",
			Name:           "Qemu-unikraft-redis",
			Devmapper:      false,
			Seccomp:        true,
			StaticNet:      false,
			SideContainers: []string{},
			Skippable:      false,
			TestPath:       t.TempDir(),
			TestFunc:       pingTest,
		},
		{
			Image:          "harbor.nbfc.io/nubificus/urunc/nginx-firecracker-unikraft:latest",
			Name:           "Firecracker-unikraft-nginx",
			Devmapper:      false,
			Seccomp:        true,
			StaticNet:      false,
			SideContainers: []string{},
			Skippable:      false,
			TestPath:       t.TempDir(),
			TestFunc:       pingTest,
		},
		{
			Image:          "harbor.nbfc.io/nubificus/urunc/httpreply-firecracker-unikraft:latest",
			Name:           "Firecracker-unikraft-httpreply-static-net",
			Devmapper:      false,
			Seccomp:        true,
			StaticNet:      true,
			SideContainers: []string{},
			Skippable:      false,
			TestPath:       t.TempDir(),
			TestFunc:       httpStaticNetTest,
		},
	}
	for _, tc := range tests {
		t.Run(tc.Name, func(t *testing.T) {
			err := pullImage(tc.Image)
			if err != nil {
				t.Fatal(err.Error())
			}
			err = runTest(tc)
			if err != nil {
				t.Fatal(err.Error())
			}
		})
	}
}

func pullImage(image string) error {
	params := []string{"crictl", "pull", image}
	cmd := exec.Command(params[0], params[1:]...) //nolint:gosec
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Failed to pull image %s: %v\n%s", image, err, output)
	}

	return nil
}

func runTest(args containerTestArgs) error {
	// podConfig := crictlSandboxConfig(args.Name + "-sandbox")

	containerID, err := startContainer(args)
	if err != nil {
		return fmt.Errorf("Failed to start container: %v", err)
	}
	time.Sleep(2 * time.Second)

	podID, err := inspectAndFind(false, containerID, "sandboxID")
	if err != nil {
		return fmt.Errorf("Failed to extract pod ID: %v", err)
	}
	defer func() {
		// We do not want a successful cleanup to overwrite any previous error
		if tempErr := testCleanup(podID); tempErr != nil {
			err = tempErr
		}
	}()
	testContainer := containerInfo{
		PodID:       podID,
		ContainerID: containerID,
		Name:        args.Name,
		Seccomp:     args.Seccomp,
	}
	return args.TestFunc(testContainer)
}

func startContainer(args containerTestArgs) (string, error) {
	// TODO: Handle Sidecar container
	// First runp the pod
	// Then create containers inside pod
	// Then start containers
	var containerConfig string
	if args.StaticNet {
		containerConfig = crictlContainerConfig("user-container", args.Image)
	} else {
		containerConfig = crictlContainerConfig(args.Name, args.Image)
	}
	podConfig := crictlSandboxConfig(args.Name)

	absPodConf := filepath.Join(args.TestPath, "pod.json")
	absContConf := filepath.Join(args.TestPath, "cont.json")
	err := writeToFile(absPodConf, podConfig)
	if err != nil {
		return "", fmt.Errorf("Failed to write pod config: %v", err)
	}
	err = writeToFile(absContConf, containerConfig)
	if err != nil {
		return "", fmt.Errorf("Failed to write container config: %v", err)
	}
	// start unikernel in pod
	params := strings.Fields("crictl run --runtime=urunc " + absContConf + " " + absPodConf)
	cmd := exec.Command(params[0], params[1:]...) //nolint:gosec
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("Failed to run unikernel: %v\n%s", err, output)
	}
	return string(output), nil
}

func testCleanup(podID string) error {
	// Stop and remove pod
	params := strings.Fields("crictl rmp --force " + podID)
	cmd := exec.Command(params[0], params[1:]...) //nolint:gosec
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Failed to stop and remove pod: %v\n%s", err, output)
	}

	return nil
}

func inspectAndFind(pod bool, id, key string) (string, error) {
	var params []string
	if pod {
		params = strings.Fields(fmt.Sprintf("crictl inspectp --output json %s", id))
	} else {
		params = strings.Fields(fmt.Sprintf("crictl inspect --output json %s", id))
	}
	cmd := exec.Command(params[0], params[1:]...) //nolint:gosec
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	keystr := "\"" + key + "\":[^,;\\]}]*"
	r, err := regexp.Compile(keystr)
	if err != nil {
		return "", err
	}
	match := r.FindString(string(output))
	keyValMatch := strings.Split(match, ":")
	val := strings.ReplaceAll(keyValMatch[1], "\"", "")
	return strings.TrimSpace(val), nil
}

func httpStaticNetTest(containerInfo) (err error) {
	procName := "firecracker"

	proc, err := common.FindProc(procName)
	if err != nil {
		return fmt.Errorf("Failed to find %s process: %v", procName, err)
	}
	netNs, err := netns.GetFromPid(int(proc.Pid))
	if err != nil {
		return fmt.Errorf("Failed to find %s process network namespace: %v", procName, err)
	}
	origns, _ := netns.Get()
	defer origns.Close()
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	err = netns.Set(netNs)
	defer func() {
		tempErr := netns.Set(origns)
		if tempErr != nil {
			err = fmt.Errorf("Failed to revert to default network nampespace: %v", err)
		}
	}()
	if err != nil {
		return fmt.Errorf("Failed to change network namespace: %v", err)
	}
	ifaces, err := net.Interfaces()
	if err != nil {
		return fmt.Errorf("Failed to get all interfaces in current network namespace: %v", err)
	}
	var tapUrunc net.Interface
	for _, iface := range ifaces {
		if strings.Contains(iface.Name, "urunc") {
			tapUrunc = iface
			break
		}
	}
	if tapUrunc.Name == "" {
		var names []string
		for _, iface := range ifaces {
			names = append(names, iface.Name)
		}
		err = fmt.Errorf("Expected tap0_urunc, got %v", names)
		return fmt.Errorf("Failed to find urunc's tap device: %v", err)
	}

	addrs, err := tapUrunc.Addrs()
	if err != nil {
		return fmt.Errorf("Failed to get %s interface's IP addresses: %v", tapUrunc.Name, err)
	}
	ipAddr := ""
	for _, addr := range addrs {
		tmp := strings.Split(addr.String(), "/")[0]
		if govalidator.IsIPv4(tmp) {
			ipAddr = tmp
			break
		}
	}
	if ipAddr == "" {
		return fmt.Errorf("Failed to get %s interface's IPv4 address", tapUrunc.Name)
	}
	parts := strings.Split(ipAddr, ".")
	newIP := fmt.Sprintf("%s.%s.%s.2", parts[0], parts[1], parts[2])
	url := fmt.Sprintf("http://%s:8080", newIP)
	curlCmd := fmt.Sprintf("curl %s", url)
	params := strings.Fields(curlCmd)
	cmd := exec.Command(params[0], params[1:]...) //nolint:gosec
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Failed to run curl: %v\n%s", err, output)
	}
	if string(output) == "" {
		return fmt.Errorf("Failed to receive valid response")
	}

	// FIXME: Investigate why the GET request using net/http fails, while is successful using curl
	//
	// client := http.DefaultClient
	// client.Timeout = 10 * time.Second
	// resp, err := client.Get(url)
	// if err != nil {
	// 	t.Logf("Failed to perform GET request to %s: %v", url, err)
	// }
	// defer resp.Body.Close()
	// body, err := io.ReadAll(resp.Body)
	// if err != nil {
	// 	t.Logf("Error reading response body: %v", err)
	// }
	// t.Log(string(body))

	// Find pod ID
	params = strings.Fields("crictl pods -q")
	cmd = exec.Command(params[0], params[1:]...) //nolint:gosec
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Failed to find pod: %v\n%s", err, output)
	}

	podID := string(output)
	podID = strings.TrimSpace(podID)

	// Stop and remove pod
	params = strings.Fields("crictl rmp --force " + podID)
	cmd = exec.Command(params[0], params[1:]...) //nolint:gosec
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Failed to stop and remove pod: %v\n%s", err, output)
	}

	proc, _ = common.FindProc(procName)
	if proc != nil {
		return fmt.Errorf("%s process is still alive", procName)
	}

	return nil
}

func crictlSandboxConfig(name string) string {
	return fmt.Sprintf(`{
		"metadata": {
			"name": "%s",
			"namespace": "default",
			"attempt": 1,
			"uid": "abcshd83djaidwnduwk28bcsb"
		},
		"linux": {
		}
	}
	`, name)
}

func crictlContainerConfig(name string, image string) string {
	return fmt.Sprintf(`{
		"metadata": {
			"name": "%s"
		},
		"image":{
			"image": "%s"
		},
		"command": [
			"/unikernel"
		],
		"linux": {
		}
	  }
	`, name, image)
}

func writeToFile(filename string, content string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(content)
	if err != nil {
		return err
	}
	return nil
}

func pingTest(cntrInfo containerInfo) error {
	extractedIPAddr, err := inspectAndFind(true, cntrInfo.PodID, "ip")
	if err != nil {
		return fmt.Errorf("Failed to extract container IP: %v", err)
	}
	err = common.PingUnikernel(extractedIPAddr)
	if err != nil {
		return fmt.Errorf("ping failed: %v", err)
	}

	return nil
}
