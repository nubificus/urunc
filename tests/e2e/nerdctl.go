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

var nerdctlName = "nerdctl"

type nerdctlInfo struct {
	testArgs    containerTestArgs
	containerID string
}

func newNerdctlTool(args containerTestArgs) *nerdctlInfo {
	return &nerdctlInfo{
		testArgs:    args,
		containerID: "",
	}
}

func (i *nerdctlInfo) getTestArgs() containerTestArgs {
	return i.testArgs
}

func (i *nerdctlInfo) setContainerID(cID string) {
	i.containerID = cID
}

func (i *nerdctlInfo) getContainerID() string {
	return i.containerID
}

func (i *nerdctlInfo) pullImage() error {
	return commonPull(nerdctlName, i.testArgs.Image)
}

func (i *nerdctlInfo) rmImage() error {
	return commonRmImage(nerdctlName, i.testArgs.Image)
}

func (i *nerdctlInfo) createContainer() (string, error) {
	return commonCreate(nerdctlName, i.testArgs)
}

func (i *nerdctlInfo) startContainer(detach bool) (string, error) {
	return commonStart(nerdctlName, i.containerID, detach)
}

func (i *nerdctlInfo) runContainer(detach bool) (string, error) {
	return commonRun(nerdctlName, i.testArgs, detach)
}

func (i *nerdctlInfo) stopContainer() error {
	output, err := commonStopContainer(nerdctlName, i.containerID)
	if err != nil {
		return fmt.Errorf("Stopping %s failed: %s - %v", i.containerID, output, err)
	}

	if i.containerID != output {
		return fmt.Errorf("unexpected output when stopping %s. %s", i.containerID, output)
	}

	return nil
}

func (i *nerdctlInfo) rmContainer() error {
	output, err := commonRmContainer(nerdctlName, i.containerID)
	if err != nil {
		return fmt.Errorf("Stopping %s failed: %s - %v", i.containerID, output, err)
	}

	if i.containerID != output {
		return fmt.Errorf("unexpected output when removing %s. %s", i.containerID, output)
	}

	return nil
}

func (i *nerdctlInfo) logContainer() (string, error) {
	return commonLogs(nerdctlName, i.containerID)
}

func (i *nerdctlInfo) searchContainer(cID string) (bool, error) {
	return commonSearchContainer(nerdctlName, cID)
}

func (i *nerdctlInfo) inspectAndGet(key string) (string, error) {
	return commonInspectAndGet(nerdctlName, i.containerID, key)
}
