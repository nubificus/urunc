// Copyright 2023 Nubificus LTD.

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package hypervisors

import (
	"fmt"
	"sync"
	"time"

	"github.com/hpcloud/tail"
	hedge "github.com/nubificus/hedge_cli/hedge_api"
	"github.com/sirupsen/logrus"
)

const (
	HedgeVmm         VmmType = "hedge"
	maxVMListRetries int     = 20
	ConsoleEndpoint          = "/proc/vmcons"
)

type Hedge struct {
}

func (h *Hedge) Ok() error {
	return hedge.Status()
}

func (h *Hedge) Stop(t string) error {
	return hedge.StopVM(t)
}

func (h *Hedge) Path() string {
	return ""
}

func (h *Hedge) Execve(data ExecData) error {
	vmmLog.WithFields(logrus.Fields{
		"binary":  data.Unikernel,
		"CmdLine": data.CmdLine,
		"BlkDev":  data.BlkDev,
		"TapDev":  data.TapDev,
	}).Info("Hedge execve")
	rumprunConfig, err := NewRumprunConfig(data)
	if err != nil {
		return err
	}
	cmdLine, err := rumprunConfig.ToJSONString()
	if err != nil {
		return err
	}

	conf := hedge.VMConfig{
		Name:    data.Container,
		Binary:  data.Unikernel,
		CPU:     0,
		Mem:     512,
		Blk:     data.BlkDev,
		Net:     data.TapDev,
		CmdLine: cmdLine,
	}
	vmmLog.WithField("hedge_conf", conf).Info("STARTING hedge")

	err = hedge.StartVM(conf)
	if err != nil {
		vmmLog.WithError(err).Error("Failed to start hedge")

		return err
	}
	vmmLog.Info("STARTED hedge")

	var wg sync.WaitGroup
	retries := 0
	wg.Add(1)
	found := false
	var id int
	// TODO: rewrite this to actually work
	go func() {
		defer wg.Done()
		for {
			defer time.Sleep(100 * time.Millisecond)
			if retries > maxVMListRetries {
				break
			}
			hedgeVms, err := hedge.ListVMs()
			if err != nil {
				retries++
				continue
			}
			for _, vm := range hedgeVms {
				if vm.Name == data.Container {
					found = true
					id = vm.ID
					continue
				}
			}
			if !found {
				retries++
			}
		}
	}()
	go func() {
		for {
			time.Sleep(100 * time.Millisecond)
			if found {
				outputCh, err := ConsoleChannel(id)
				if err != nil {
					vmmLog.WithError(err).Error("Failed to get console channel")
					return
				}

				// Consume the output from the channel
				for line := range outputCh {
					fmt.Println(line)
					vmmLog.WithField("out", line).Error("hedge output")
				}
			}
		}

	}()
	wg.Wait()
	vmmLog.Error("hedge DONE")

	return nil
}

func ConsoleChannel(id int) (<-chan string, error) {
	consoleEndpoint := fmt.Sprintf("%s/vm%d", ConsoleEndpoint, id)

	t, err := tail.TailFile(consoleEndpoint, tail.Config{
		Follow: true,
	})
	if err != nil {
		return nil, err
	}

	outputCh := make(chan string)

	go func() {
		defer close(outputCh)

		for line := range t.Lines {
			outputCh <- line.Text
		}
	}()

	return outputCh, nil
}
