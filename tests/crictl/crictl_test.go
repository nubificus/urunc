package urunc

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	common "github.com/nubificus/urunc/tests"
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
