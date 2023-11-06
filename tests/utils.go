package tests

import (
	"fmt"
	"strings"

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
