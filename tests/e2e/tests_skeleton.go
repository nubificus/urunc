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
	"errors"
	"fmt"
	"strings"
	"testing"

	common "github.com/nubificus/urunc/tests"

	"time"
)

type testTool interface {
	getTestArgs() containerTestArgs
	getContainerID() string
	setContainerID(string)
	pullImage() error
	rmImage() error
	createContainer() (string, error)
	startContainer(bool) (string, error)
	runContainer(bool) (string, error)
	stopContainer() error
	rmContainer() error
	logContainer() (string, error)
	searchContainer(string) (bool, error)
	inspectAndGet(string) (string, error)
}

var matchTest testMethod
var errToolDoesNotSUpport = errors.New("Operarion not support")

func runTest(tool testTool, t *testing.T) {
	cntrArgs := tool.getTestArgs()
	err := tool.pullImage()
	if err != nil {
		t.Fatalf("Failed to pull container image: %s - %v", cntrArgs.Image, err)
	}
	t.Cleanup(func() {
		err = tool.rmImage()
		if err != nil {
			t.Errorf("Failed to remove container image: %s - %v", cntrArgs.Image, err)
		}

	})
	if cntrArgs.TestFunc == nil {
		output, err := tool.runContainer(false)
		if err != nil {
			t.Fatalf("Failed to run unikernel container: %s -- %v", output, err)
		}
		tool.setContainerID(cntrArgs.Name)
		if !strings.Contains(string(output), cntrArgs.ExpectOut) {
			t.Fatalf("Expected: %s, Got: %s", cntrArgs.ExpectOut, output)
		}
		err = testCleanup(tool)
		if err != nil {
			t.Errorf("Cleaning up: %v", err)
		}
		return
	}
	output, err := tool.createContainer()
	if err != nil {
		t.Fatalf("Failed to create unikernel container: %v", err)
	}
	tool.setContainerID(output)
	t.Cleanup(func() {
		err = testVerifyRm(tool)
		if err != nil {
			t.Errorf("Failed to verify container removal: %s - %v", cntrArgs.Image, err)
		}

	})
	t.Cleanup(func() {
		err = tool.rmContainer()
		if err != nil {
			t.Errorf("Failed to remove container: %s - %v", cntrArgs.Image, err)
		}

	})
	output, err = tool.startContainer(true)
	if err != nil {
		t.Fatalf("Failed to start unikernel container: %s - %v", output, err)
	}
	t.Cleanup(func() {
		err = tool.stopContainer()
		if err != nil {
			t.Errorf("Failed to stop container: %s - %v", cntrArgs.Image, err)
		}

	})
	// Give some time till the unikernel is up and running.
	// Maybe we need to revisit this in the future.
	time.Sleep(1 * time.Second)
	err = cntrArgs.TestFunc(tool)
	if err != nil {
		t.Fatalf("Failed test: %v", err)
	}
}

func testVerifyRm(tool testTool) error {
	containerID := tool.getContainerID()
	exists, err := tool.searchContainer(containerID)
	if exists || err != nil {
		return fmt.Errorf("Container %s is not removed: %v", containerID, err)
	}
	err = common.VerifyNoStaleFiles(containerID)
	if err != nil {
		return fmt.Errorf("Failed to remove all stale files: %v", err)
	}

	return nil
}

func testCleanup(tool testTool) error {
	err := tool.stopContainer()
	if err != nil {
		return fmt.Errorf("Failed to stop container: %v", err)
	}

	err = tool.rmContainer()
	if err != nil {
		return fmt.Errorf("Failed to remove container: %v", err)
	}

	return testVerifyRm(tool)
}
