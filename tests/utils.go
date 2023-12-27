package tests

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-ping/ping"
	"github.com/shirou/gopsutil/v3/process"
)

func FindProc(executable string) (*process.Process, error) {
	var proc *process.Process
	processes, err := process.Processes()
	if err != nil {
		return nil, err
	}
	var cmdLine string
	found := false
	for _, p := range processes {
		cmdLine, _ = p.Cmdline()
		if strings.Contains(cmdLine, executable) {
			found = true
			proc = p
			break
		}
	}
	if found {
		return proc, nil
	}
	return nil, fmt.Errorf("process %s not found", executable)
}

func PingUnikernel(ipAddress string) error {
	pinger, err := ping.NewPinger(ipAddress)
	if err != nil {
		return fmt.Errorf("failed to create Pinger: %v", err)
	}
	pinger.Count = 3
	pinger.Timeout = 5 * time.Second
	err = pinger.Run()
	if err != nil {
		return fmt.Errorf("failed to ping %s: %v", ipAddress, err)
	}
	if pinger.PacketsRecv != pinger.PacketsSent {
		return fmt.Errorf("packets received (%d) not equal to packets sent (%d)", pinger.PacketsRecv, pinger.PacketsSent)
	}
	if pinger.PacketsSent == 0 {
		return fmt.Errorf("no packets were sent")
	}
	return nil
}
