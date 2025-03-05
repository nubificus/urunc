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

package urunce2etesting

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/asaskevich/govalidator"
	"github.com/vishvananda/netns"
	"github.com/opencontainers/runtime-spec/specs-go"
)

var matchTest testMethod

func seccompTest(tool testTool) error {
	args := tool.getTestArgs()
	unikernelPID, err := tool.inspectCAndGet("Pid")
	if err != nil {
		return fmt.Errorf("Failed to extract unikernel PID: %v", err)
	}
	procPath := "/proc/" + unikernelPID + "/status"
	seccompLine, err := findLineInFile(procPath, "Seccomp")
	if err != nil {
		return err
	}
	wordsInLine := strings.Split(seccompLine, ":")
	if strings.TrimSpace(wordsInLine[1]) == "2" {
		if !args.Seccomp {
			return fmt.Errorf("Seccomp should not be enabled")
		}
	} else {
		if args.Seccomp {
			return fmt.Errorf("Seccomp should be enabled")
		}
	}

	return nil
}

func namespaceTest(tool testTool) error {
	// We need to retrieve the container's config, in order to get 
	// the neamspaces that the container should have joined.
	containerID := tool.getContainerID()
	// Try /run/containerd/io.containerd.runtime.v2.task/default/containerID first
	configPath := filepath.Join("/var/run/containerd/io.containerd.runtime.v2.task/default/", containerID, "/config.json")
	_, err := os.Stat(configPath)
	if os.IsNotExist(err) {
		configPath = filepath.Join("/var/run/containerd/io.containerd.runtime.v2.task/k8s.io/", containerID, "/config.json")
		_, err = os.Stat(configPath)
		if os.IsNotExist(err) {
			configPath = filepath.Join("/var/run/containerd/io.containerd.runtime.v2.task/moby/", containerID, "/config.json")
			_, err = os.Stat(configPath)
		}
	}
	if err != nil {
		return fmt.Errorf("Could not retrieve container's config file")
	}

	var spec specs.Spec
	specData, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read specification: %w", err)
	}
	if err := json.Unmarshal(specData, &spec); err != nil {
		return fmt.Errorf("failed to parse specification json: %w", err)
	}

	unikernelPID, err := tool.inspectCAndGet("pid")
	if err != nil {
		return fmt.Errorf("Failed to extract unikernel PID: %v", err)
	}
	cntrNsMap, err := getProcNS(unikernelPID)
	if err != nil {
		return fmt.Errorf("failed to get namespaces of unikernel: %w", err)
	}

	selfNsMap, err := getProcNS("self")
	if err != nil {
		return fmt.Errorf("failed to get namespaces of current process: %w", err)
	}

	for _, ns := range spec.Linux.Namespaces {
		switch ns.Type {
		case specs.UserNamespace:
			err = compareNS(cntrNsMap["user"], selfNsMap["user"], ns.Path)
			if err != nil {
				return fmt.Errorf("user: %w", err)
			}
		case specs.IPCNamespace:
			err = compareNS(cntrNsMap["ipc"], selfNsMap["ipc"], ns.Path)
			if err != nil {
				return fmt.Errorf("ipc: %w", err)
			}
		case specs.UTSNamespace:
			err = compareNS(cntrNsMap["uts"], selfNsMap["uts"], ns.Path)
			if err != nil {
				return fmt.Errorf("uts: %w", err)
			}
		case specs.NetworkNamespace:
			err = compareNS(cntrNsMap["net"], selfNsMap["net"], ns.Path)
			if err != nil {
				return fmt.Errorf("net: %w", err)
			}
		case specs.PIDNamespace:
			err = compareNS(cntrNsMap["pid"], selfNsMap["pid"], ns.Path)
			if err != nil {
				return fmt.Errorf("pid: %w", err)
			}
		case specs.MountNamespace:
			err = compareNS(cntrNsMap["mnt"], selfNsMap["mnt"], ns.Path)
			if err != nil {
				return fmt.Errorf("mnt: %w", err)
			}
		case specs.CgroupNamespace:
			err = compareNS(cntrNsMap["cgroup"], selfNsMap["cgroup"], ns.Path)
			if err != nil {
				return fmt.Errorf("cgroup: %w", err)
			}
		case specs.TimeNamespace:
			err = compareNS(cntrNsMap["uts"], selfNsMap["uts"], ns.Path)
			if err != nil {
				return fmt.Errorf("uts: %w", err)
			}
		default:
			continue
		}
	}

	return nil
}

func pingTest(tool testTool) error {
	extractedIPAddr, err := tool.inspectPAndGet("ip")
	if err != nil && err != errToolDoesNotSupport {
		return fmt.Errorf("Failed to extract container IP: %v", err)
	} else if err == errToolDoesNotSupport {
		extractedIPAddr, err = tool.inspectCAndGet("IPAddress")
		if err != nil {
			return fmt.Errorf("Failed to extract container IP: %v", err)
		}
	}
	err = pingUnikernel(extractedIPAddr)
	if err != nil {
		return fmt.Errorf("ping failed: %v", err)
	}

	return nil
}

func httpStaticNetTest(tool testTool) (err error) {
	extractedPid, err := tool.inspectCAndGet("pid")
	if err != nil {
		return fmt.Errorf("Failed to extract Pid: %v", err)
	}
	pid, err := strconv.Atoi(extractedPid)
	if err != nil {
		return fmt.Errorf("Failed to convert pid %s to int: %v", extractedPid, err)
	}

	netNs, err := netns.GetFromPid(pid)
	if err != nil {
		return fmt.Errorf("Failed to find network namespace of unikernel: %v", err)
	}
	origns, _ := netns.Get()
	defer origns.Close()
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	err = netns.Set(netNs)
	if err != nil {
		return fmt.Errorf("Failed to change network namespace: %v", err)
	}
	defer func() {
		tempErr := netns.Set(origns)
		if tempErr != nil {
			err = fmt.Errorf("Failed to revert to default network nampespace: %v", err)
		}
	}()
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
	return nil
}
