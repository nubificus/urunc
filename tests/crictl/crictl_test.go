package urunc

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/go-ping/ping"
	common "github.com/nubificus/urunc/tests"
)

const PodConfig = `{
    "metadata": {
        "name": "redis-sandbox",
        "namespace": "default",
        "attempt": 1,
        "uid": "abcshd83djaidwnduwk28bcsb"
    },
    "linux": {
    }
}
`

const ContainerConfig = `{
	"metadata": {
		"name": "redis-hvt"
	},
	"image":{
		"image": "harbor.nbfc.io/nubificus/urunc/redis-hvt-rump:latest"
	},
	"command": [
		"/unikernel/redis6.hvt"
	],
	"linux": {
	}
  }
`

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

func TestCrictlHvtRumprunRedis(t *testing.T) {
	// create config files
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal("Failed to retrieve current directory")
	}
	absPodConf := filepath.Join(cwd, "pod.json")
	absContConf := filepath.Join(cwd, "cont.json")
	err = writeToFile(absPodConf, PodConfig)
	if err != nil {
		t.Fatalf("Failed to write pod config: %v", err)
	}
	defer os.Remove(absPodConf)
	err = writeToFile(absContConf, ContainerConfig)
	if err != nil {
		t.Fatalf("Failed to write container config: %v", err)
	}
	defer os.Remove(absContConf)

	// pull image
	params := strings.Split("crictl pull harbor.nbfc.io/nubificus/urunc/redis-hvt-rump:latest", " ")
	cmd := exec.Command(params[0], params[1:]...) //nolint:gosec
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to pull image: %v\n%s", err, output)
	}

	// start unikernel in pod
	params = strings.Split("crictl run --runtime=urunc cont.json pod.json", " ")
	cmd = exec.Command(params[0], params[1:]...) //nolint:gosec
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run unikernel: %v\n%s", err, output)
	}
	time.Sleep(2 * time.Second)
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

	// Find pod ID
	params = strings.Split("crictl pods -q", " ")
	cmd = exec.Command(params[0], params[1:]...) //nolint:gosec
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to find pod: %v\n%s", err, output)
	}

	podID := string(output)
	podID = strings.TrimSpace(podID)

	// Stop and remove pod
	params = strings.Split("crictl rmp --force "+podID, " ")
	cmd = exec.Command(params[0], params[1:]...) //nolint:gosec
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to stop and remove pod: %v\n%s", err, output)
	}

	// Check if solo5 process still alive
	proc, _ = common.FindProc("solo5-hvt")
	if proc != nil {
		t.Fatal("solo5-hvt process is still alive")
	}
}
