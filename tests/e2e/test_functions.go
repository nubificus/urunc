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
	"fmt"
	"net"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/asaskevich/govalidator"
	common "github.com/nubificus/urunc/tests"
	"github.com/vishvananda/netns"
)

func seccompTest(tool testTool) error {
	args := tool.getTestArgs()
	unikernelPID, err := tool.inspectCAndGet("Pid")
	if err != nil {
		return fmt.Errorf("Failed to extract unikernel PID: %v", err)
	}
	procPath := "/proc/" + unikernelPID + "/status"
	seccompLine, err := common.FindLineInFile(procPath, "Seccomp")
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

func pingTest(tool testTool) error {
	extractedIPAddr, err := tool.inspectPAndGet("ip")
	if err != nil && err != errToolDoesNotSUpport {
		return fmt.Errorf("Failed to extract container IP: %v", err)
	} else if err == errToolDoesNotSUpport {
		extractedIPAddr, err = tool.inspectCAndGet("IPAddress")
		if err != nil {
			return fmt.Errorf("Failed to extract container IP: %v", err)
		}
	}
	err = common.PingUnikernel(extractedIPAddr)
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
