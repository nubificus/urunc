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
)

var ctrName = "ctr"

type ctrInfo struct {
	testArgs    containerTestArgs
	containerID string
	detached    bool
}

func newCtrTool(args containerTestArgs) *ctrInfo {
	return &ctrInfo{
		testArgs:    args,
		containerID: "",
		detached:    false,
	}
}

func ctrNewContainerCmd(a containerTestArgs) string {
	cmdBase := "--runtime io.containerd.urunc.v2 "
	if a.Devmapper {
		cmdBase += "--snapshotter devmapper "
	}
	if a.Seccomp {
		cmdBase += "--seccomp "
	}
	cmdBase += a.Image + " "
	cmdBase += a.Name
	return cmdBase
}

func (i *ctrInfo) getTestArgs() containerTestArgs {
	return i.testArgs
}

func (i *ctrInfo) setContainerID(cID string) {
	i.containerID = cID
}

func (i *ctrInfo) getContainerID() string {
	return i.containerID
}

func (i *ctrInfo) pullImage() error {
	return commonPull(ctrName, i.testArgs.Image)
}

func (i *ctrInfo) rmImage() error {
	return commonRmImage(ctrName, i.testArgs.Image)
}

func (i *ctrInfo) createContainer() (string, error) {
	cmdBase := ctrName
	cmdBase += " c create "
	cmdBase += ctrNewContainerCmd(i.testArgs)
	return commonCmdExec(cmdBase)
}

func (i *ctrInfo) startContainer(detach bool) (string, error) {
	if detach {
		i.detached = true
	}
	return commonStart(ctrName+" t", i.containerID, detach)
}

func (i *ctrInfo) runContainer(detach bool) (string, error) {
	cmdBase := ctrName
	cmdBase += " run "
	if detach {
		cmdBase += "-d "
		i.detached = true
	}
	cmdBase += ctrNewContainerCmd(i.testArgs)
	return commonCmdExec(cmdBase)
}

func (i *ctrInfo) stopContainer() error {
	if !i.detached {
		return nil
	}
	cmdBase := ctrName
	cmdBase += " t kill "
	cmdBase += i.containerID
	output, err := commonCmdExec(cmdBase)
	if output != "" || err != nil {
		return fmt.Errorf("Stopping %s failed: %s - %v", i.containerID, output, err)
	}
	return nil
}

func (i *ctrInfo) rmContainer() error {
	output, err := commonRmContainer(ctrName+" c", i.containerID)
	if output != "" || err != nil {
		return fmt.Errorf("Stopping %s failed: %s - %v", i.containerID, output, err)
	}
	return nil
}

func (i *ctrInfo) logContainer() (string, error) {
	// Not supported by ctr
	return "", errToolDoesNotSUpport
}

func (i *ctrInfo) searchContainer(cID string) (bool, error) {
	cmd := ctrName + " c ls -q"

	output, err := commonCmdExec(cmd)
	if err != nil {
		return true, err
	}
	return searchCID(output, cID), nil
}

func (i *ctrInfo) inspectAndGet(key string) (string, error) {
	cmdBase := ctrName
	cmdBase += " c info "
	cmdBase += i.containerID
	output, err := commonCmdExec(cmdBase)
	if err != nil {
		return "", err
	}

	return findValOfKey(output, key)
}