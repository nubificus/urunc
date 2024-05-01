package urunc

import (
	"encoding/json"
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

func TestCrictlHvtRumprunRedis(t *testing.T) {
	containerImage := "harbor.nbfc.io/nubificus/urunc/redis-hvt-rump:latest"
	procName := "solo5-hvt"
	podConfig := crictlSandboxConfig("hvt-rumprun-redis-sandbox")
	containerConfig := crictlContainerConfig("hvt-rumprun-redis", containerImage)

	// create config files
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal("Failed to retrieve current directory")
	}
	absPodConf := filepath.Join(cwd, "pod.json")
	absContConf := filepath.Join(cwd, "cont.json")
	err = writeToFile(absPodConf, podConfig)
	if err != nil {
		t.Fatalf("Failed to write pod config: %v", err)
	}
	defer os.Remove(absPodConf)
	err = writeToFile(absContConf, containerConfig)
	if err != nil {
		t.Fatalf("Failed to write container config: %v", err)
	}
	defer os.Remove(absContConf)

	// pull image
	params := []string{"crictl", "pull", containerImage}
	cmd := exec.Command(params[0], params[1:]...) //nolint:gosec
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to pull image: %v\n%s", err, output)
	}

	// start unikernel in pod
	params = strings.Fields("crictl run --runtime=urunc cont.json pod.json")
	cmd = exec.Command(params[0], params[1:]...) //nolint:gosec
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run unikernel: %v\n%s", err, output)
	}
	time.Sleep(2 * time.Second)
	proc, err := common.FindProc(procName)
	if err != nil {
		t.Fatalf("Failed to find %s process: %v", procName, err)
	}
	cmdLine, err := proc.Cmdline()
	if err != nil {
		t.Fatalf("Failed to find %s process' command line: %v", procName, err)
	}

	// Extract the IP address
	re := regexp.MustCompile(`"addr":"([0-9]+\.[0-9]+\.[0-9]+\.[0-9]+)"`)
	match := re.FindStringSubmatch(cmdLine)
	extractedIPAddr := ""
	if len(match) == 2 {
		extractedIPAddr = match[1]
	} else {
		t.Fatalf("Failed to extract IP address for %s process", procName)

	}

	err = common.PingUnikernel(extractedIPAddr)
	if err != nil {
		t.Fatalf("ping failed: %v", err)
	}

	// Find pod ID
	params = strings.Fields("crictl pods -q")
	cmd = exec.Command(params[0], params[1:]...) //nolint:gosec
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to find pod: %v\n%s", err, output)
	}

	podID := string(output)
	podID = strings.TrimSpace(podID)

	// Stop and remove pod
	params = strings.Fields("crictl rmp --force " + podID)
	cmd = exec.Command(params[0], params[1:]...) //nolint:gosec
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to stop and remove pod: %v\n%s", err, output)
	}

	proc, _ = common.FindProc(procName)
	if proc != nil {
		t.Fatalf("%s process is still alive", procName)
	}
}

func TestCrictlSptRumprunRedis(t *testing.T) {
	containerImage := "harbor.nbfc.io/nubificus/urunc/redis-spt-rump:latest"
	procName := "solo5-spt"
	podConfig := crictlSandboxConfig("spt-rumprun-redis-sandbox")
	containerConfig := crictlContainerConfig("spt-rumprun-redis", containerImage)

	// create config files
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal("Failed to retrieve current directory")
	}
	absPodConf := filepath.Join(cwd, "pod.json")
	absContConf := filepath.Join(cwd, "cont.json")
	err = writeToFile(absPodConf, podConfig)
	if err != nil {
		t.Fatalf("Failed to write pod config: %v", err)
	}
	defer os.Remove(absPodConf)
	err = writeToFile(absContConf, containerConfig)
	if err != nil {
		t.Fatalf("Failed to write container config: %v", err)
	}
	defer os.Remove(absContConf)

	// pull image
	params := []string{"crictl", "pull", containerImage}
	cmd := exec.Command(params[0], params[1:]...) //nolint:gosec
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to pull image: %v\n%s", err, output)
	}

	// start unikernel in pod
	params = strings.Fields("crictl run --runtime=urunc cont.json pod.json")
	cmd = exec.Command(params[0], params[1:]...) //nolint:gosec
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run unikernel: %v\n%s", err, output)
	}
	time.Sleep(2 * time.Second)
	proc, err := common.FindProc(procName)
	if err != nil {
		t.Fatalf("Failed to find %s process: %v", procName, err)
	}
	cmdLine, err := proc.Cmdline()
	if err != nil {
		t.Fatalf("Failed to find %s process' command line: %v", procName, err)
	}

	// Extract the IP address
	re := regexp.MustCompile(`"addr":"([0-9]+\.[0-9]+\.[0-9]+\.[0-9]+)"`)
	match := re.FindStringSubmatch(cmdLine)
	extractedIPAddr := ""
	if len(match) == 2 {
		extractedIPAddr = match[1]
	} else {
		t.Fatalf("Failed to extract IP address for %s process", procName)

	}

	err = common.PingUnikernel(extractedIPAddr)
	if err != nil {
		t.Fatalf("ping failed: %v", err)
	}

	// Find pod ID
	params = strings.Fields("crictl pods -q")
	cmd = exec.Command(params[0], params[1:]...) //nolint:gosec
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to find pod: %v\n%s", err, output)
	}

	podID := string(output)
	podID = strings.TrimSpace(podID)

	// Stop and remove pod
	params = strings.Fields("crictl rmp --force " + podID)
	cmd = exec.Command(params[0], params[1:]...) //nolint:gosec
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to stop and remove pod: %v\n%s", err, output)
	}

	proc, _ = common.FindProc(procName)
	if proc != nil {
		t.Fatalf("%s process is still alive", procName)
	}
}

func TestCrictlQemuUnikraftRedis(t *testing.T) {
	containerImage := "harbor.nbfc.io/nubificus/urunc/redis-qemu-unikraft-initrd:latest"
	procName := "qemu-system"
	podConfig := crictlSandboxConfig("qemu-unikraft-redis-sandbox")
	containerConfig := crictlContainerConfig("qemu-unikraft-redis", containerImage)

	// create config files
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal("Failed to retrieve current directory")
	}
	absPodConf := filepath.Join(cwd, "pod.json")
	absContConf := filepath.Join(cwd, "cont.json")
	err = writeToFile(absPodConf, podConfig)
	if err != nil {
		t.Fatalf("Failed to write pod config: %v", err)
	}
	defer os.Remove(absPodConf)
	err = writeToFile(absContConf, containerConfig)
	if err != nil {
		t.Fatalf("Failed to write container config: %v", err)
	}
	defer os.Remove(absContConf)

	// pull image
	params := []string{"crictl", "pull", containerImage}
	cmd := exec.Command(params[0], params[1:]...) //nolint:gosec
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to pull image: %v\n%s", err, output)
	}

	// start unikernel in pod
	params = strings.Fields("crictl run --runtime=urunc cont.json pod.json")
	cmd = exec.Command(params[0], params[1:]...) //nolint:gosec
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run unikernel: %v\n%s", err, output)
	}
	time.Sleep(2 * time.Second)
	proc, err := common.FindProc(procName)
	if err != nil {
		t.Fatalf("Failed to find %s process: %v", procName, err)
	}
	cmdLine, err := proc.Cmdline()
	if err != nil {
		t.Fatalf("Failed to find %s process' command line: %v", procName, err)
	}

	// Extract the IP address
	re := regexp.MustCompile(`netdev.ipv4_addr=([0-9]+\.[0-9]+\.[0-9]+\.[0-9]+)`)
	match := re.FindStringSubmatch(cmdLine)
	extractedIPAddr := ""
	if len(match) == 2 {
		extractedIPAddr = match[1]
	} else {
		t.Fatalf("Failed to extract IP address for %s process", procName)
	}

	err = common.PingUnikernel(extractedIPAddr)
	if err != nil {
		t.Fatalf("ping failed: %v", err)
	}

	// Find pod ID
	params = strings.Fields("crictl pods -q")
	cmd = exec.Command(params[0], params[1:]...) //nolint:gosec
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to find pod: %v\n%s", err, output)
	}

	podID := string(output)
	podID = strings.TrimSpace(podID)

	// Stop and remove pod
	params = strings.Fields("crictl rmp --force " + podID)
	cmd = exec.Command(params[0], params[1:]...) //nolint:gosec
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to stop and remove pod: %v\n%s", err, output)
	}

	proc, _ = common.FindProc(procName)
	if proc != nil {
		t.Fatalf("%s process is still alive", procName)
	}
}

func TestCrictlFCUnikraftNginx(t *testing.T) {
	containerImage := "harbor.nbfc.io/nubificus/urunc/nginx-fc-unik:latest"
	procName := "firecracker"
	podConfig := crictlSandboxConfig("fc-unikraft-nginx-sandbox")
	containerConfig := crictlContainerConfig("fc-unikraft-nginx", containerImage)

	// create config files
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal("Failed to retrieve current directory")
	}
	absPodConf := filepath.Join(cwd, "pod.json")
	absContConf := filepath.Join(cwd, "cont.json")
	err = writeToFile(absPodConf, podConfig)
	if err != nil {
		t.Fatalf("Failed to write pod config: %v", err)
	}
	defer os.Remove(absPodConf)
	err = writeToFile(absContConf, containerConfig)
	if err != nil {
		t.Fatalf("Failed to write container config: %v", err)
	}
	defer os.Remove(absContConf)

	// pull image
	params := []string{"crictl", "pull", containerImage}
	cmd := exec.Command(params[0], params[1:]...) //nolint:gosec
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to pull image: %v\n%s", err, output)
	}

	// start unikernel in pod
	params = strings.Fields("crictl run --runtime=urunc cont.json pod.json")
	cmd = exec.Command(params[0], params[1:]...) //nolint:gosec
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run unikernel: %v\n%s", err, output)
	}
	time.Sleep(2 * time.Second)
	proc, err := common.FindProc(procName)
	if err != nil {
		t.Fatalf("Failed to find %s process: %v", procName, err)
	}
	cmdLine, err := proc.Cmdline()
	if err != nil {
		t.Fatalf("Failed to find %s process' command line: %v", procName, err)
	}

	var extractedFCconfig string
	re := regexp.MustCompile(`--config-file\s+([^\s]+)`)
	match := re.FindStringSubmatch(cmdLine)
	if len(match) == 2 {
		extractedFCconfig = match[1]
	} else {
		t.Fatalf("Failed to extract config file for %s", procName)
	}
	var fcConfig struct {
		BootSource struct {
			BootArgs string `json:"boot_args"`
		} `json:"boot-source"`
	}
	jsonData, err := os.ReadFile(extractedFCconfig)
	if err != nil {
		t.Fatalf("Failed to read config file for %s", procName)
	}
	err = json.Unmarshal(jsonData, &fcConfig)
	if err != nil {
		t.Fatalf("Failed to unmarshal config file for %s", procName)
	}

	// Extract the IP address
	re = regexp.MustCompile(`netdev.ipv4_addr=([0-9]+\.[0-9]+\.[0-9]+\.[0-9]+)`)
	match = re.FindStringSubmatch(fcConfig.BootSource.BootArgs)
	extractedIPAddr := ""
	if len(match) == 2 {
		extractedIPAddr = match[1]
	} else {
		t.Fatalf("Failed to extract IP address for %s process", procName)
	}

	err = common.PingUnikernel(extractedIPAddr)
	if err != nil {
		t.Fatalf("ping failed: %v", err)
	}

	// Find pod ID
	params = strings.Fields("crictl pods -q")
	cmd = exec.Command(params[0], params[1:]...) //nolint:gosec
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to find pod: %v\n%s", err, output)
	}

	podID := string(output)
	podID = strings.TrimSpace(podID)

	// Stop and remove pod
	params = strings.Fields("crictl rmp --force " + podID)
	cmd = exec.Command(params[0], params[1:]...) //nolint:gosec
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to stop and remove pod: %v\n%s", err, output)
	}

	proc, _ = common.FindProc(procName)
	if proc != nil {
		t.Fatalf("%s process is still alive", procName)
	}
}

func TestCrictlHTTPStaticNet(t *testing.T) {
	containerImage := "harbor.nbfc.io/nubificus/httpreply-fc:x86_64"
	procName := "firecracker"
	podConfig := crictlSandboxConfig("fc-unikraft-knative-sandbox")

	// user-container is used as "io.kubernetes.cri.container-name" annotation by crictl
	// in order to trigger the static net mode
	containerConfig := crictlContainerConfig("user-container", containerImage)

	// create config files
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal("Failed to retrieve current directory")
	}
	absPodConf := filepath.Join(cwd, "pod.json")
	absContConf := filepath.Join(cwd, "cont.json")
	err = writeToFile(absPodConf, podConfig)
	if err != nil {
		t.Fatalf("Failed to write pod config: %v", err)
	}
	defer os.Remove(absPodConf)
	err = writeToFile(absContConf, containerConfig)
	if err != nil {
		t.Fatalf("Failed to write container config: %v", err)
	}
	defer os.Remove(absContConf)

	// pull image
	params := []string{"crictl", "pull", containerImage}
	cmd := exec.Command(params[0], params[1:]...) //nolint:gosec
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to pull image: %v\n%s", err, output)
	}

	// start unikernel in pod
	params = strings.Fields("crictl run --runtime=urunc cont.json pod.json")
	cmd = exec.Command(params[0], params[1:]...) //nolint:gosec
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run unikernel: %v\n%s", err, output)
	}
	time.Sleep(2 * time.Second)
	proc, err := common.FindProc(procName)
	if err != nil {
		t.Fatalf("Failed to find %s process: %v", procName, err)
	}
	netNs, err := netns.GetFromPid(int(proc.Pid))
	if err != nil {
		t.Fatalf("Failed to find %s process network namespace: %v", procName, err)
	}
	origns, _ := netns.Get()
	defer origns.Close()
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	err = netns.Set(netNs)
	defer func() {
		err := netns.Set(origns)
		if err != nil {
			t.Fatalf("Failed to revert to default network nampespace: %v", err)
		}
	}()
	if err != nil {
		t.Fatalf("Failed to change network namespace: %v", err)
	}
	ifaces, err := net.Interfaces()
	if err != nil {
		t.Fatalf("Failed to get all interfaces in current network namespace: %v", err)
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
		t.Fatalf("Failed to find urunc's tap device: %v", err)
	}

	addrs, err := tapUrunc.Addrs()
	if err != nil {
		t.Fatalf("Failed to get %s interface's IP addresses: %v", tapUrunc.Name, err)
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
		t.Fatalf("Failed to get %s interface's IPv4 address", tapUrunc.Name)
	}
	parts := strings.Split(ipAddr, ".")
	newIP := fmt.Sprintf("%s.%s.%s.2", parts[0], parts[1], parts[2])
	url := fmt.Sprintf("http://%s:8080", newIP)
	curlCmd := fmt.Sprintf("curl %s", url)
	params = strings.Fields(curlCmd)
	cmd = exec.Command(params[0], params[1:]...) //nolint:gosec
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run curl: %v\n%s", err, output)
	}
	if string(output) == "" {
		t.Fatal("Failed to receive valid response")
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
		t.Fatalf("Failed to find pod: %v\n%s", err, output)
	}

	podID := string(output)
	podID = strings.TrimSpace(podID)

	// Stop and remove pod
	params = strings.Fields("crictl rmp --force " + podID)
	cmd = exec.Command(params[0], params[1:]...) //nolint:gosec
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to stop and remove pod: %v\n%s", err, output)
	}

	proc, _ = common.FindProc(procName)
	if proc != nil {
		t.Fatalf("%s process is still alive", procName)
	}
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
