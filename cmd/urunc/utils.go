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

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/nubificus/urunc/internal/constants"
	"github.com/nubificus/urunc/pkg/unikontainers"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

// Argument check types for the `checkArgs` function.
const (
	exactArgs = iota // Checks for an exact number of arguments.
	minArgs          // Checks for a minimum number of arguments.
	maxArgs          // Checks for a maximum number of arguments.
)

// checkArgs checks the number of arguments provided in the command-line context
// against the expected number, based on the specified checkType.
func checkArgs(context *cli.Context, expected, checkType int) error {
	var err error
	cmdName := context.Command.Name

	switch checkType {
	case exactArgs:
		if context.NArg() != expected {
			err = fmt.Errorf("%s: %q requires exactly %d argument(s)", os.Args[0], cmdName, expected)
		}
	case minArgs:
		if context.NArg() < expected {
			err = fmt.Errorf("%s: %q requires a minimum of %d argument(s)", os.Args[0], cmdName, expected)
		}
	case maxArgs:
		if context.NArg() > expected {
			err = fmt.Errorf("%s: %q requires a maximum of %d argument(s)", os.Args[0], cmdName, expected)
		}
	}

	if err != nil {
		fmt.Printf("Incorrect Usage.\n\n")
		_ = cli.ShowCommandHelp(context, cmdName)
		return err
	}
	return nil
}

// handleNonBimaContainer check if bundle is supported by urunc
// if not, it execve's itself using the exact same arguments and runc
func handleNonBimaContainer(context *cli.Context) error {
	containerID := context.Args().First()
	metrics.Capture(containerID, "cTS00")
	defer func() {
		metrics.Capture(containerID, "cTS01")
	}()
	if containerID == "" {
		// cli.ShowAppHelpAndExit(context, 129)
		return nil
	}
	bundle := context.String("bundle")
	if bundle == "" {
		rootDir := context.GlobalString("root")
		// get Unikontainer data from state.json
		unikontainer, err := unikontainers.Get(containerID, rootDir)
		if err != nil {
			return err
		}
		bundle = unikontainer.State.Bundle
	}

	if unikontainers.IsBimaContainer(bundle) {
		logrus.Info("This is a bima container! Proceeding...")
		return nil
	}
	logrus.Info("This is a normal container. Calling runc...")
	return runcExec()
}

func runcExec() error {
	args := os.Args
	binPath, err := exec.LookPath("runc")
	if err != nil {
		return err
	}
	args[0] = binPath
	return syscall.Exec(args[0], args, os.Environ()) //nolint: gosec
}

func logrusToStderr() bool {
	l, ok := logrus.StandardLogger().Out.(*os.File)
	return ok && l.Fd() == os.Stderr.Fd()
}

// fatal prints the error's details if it is a libcontainer specific error type
// then exits the program with an exit status of 1.
func fatal(err error) {
	fatalWithCode(err, 1)
}

func fatalWithCode(err error, ret int) {
	// Make sure the error is written to the logger.
	logrus.Error(err)
	if !logrusToStderr() {
		fmt.Fprintln(os.Stderr, err)
	}

	os.Exit(ret)
}

// handleQueueProxy checks if the provided bundle contains a queue-proxy container
// and adds a hardcoded IP to the process's environment.
// Then, the container is identified as a non-bima container
// is spawned using runc.
func handleQueueProxy(context *cli.Context) error {
	logrus.Error("handleQueueProxy")
	containerID := context.Args().First()
	if containerID == "" {
		return nil
	}
	bundle := context.String("bundle")
	if bundle == "" {
		rootDir := context.GlobalString("root")
		// get Unikontainer data from state.json
		unikontainer, err := unikontainers.Get(containerID, rootDir)
		if err != nil {
			return err
		}
		bundle = unikontainer.State.Bundle
	}


	var spec specs.Spec
	absDir, err := filepath.Abs(bundle)
	if err != nil {
		return fmt.Errorf("failed to find absolute bundle path: %w", err)
	}
	configDir := filepath.Join(absDir, "config.json")
	data, err := os.ReadFile(configDir)
	if err != nil {
		return fmt.Errorf("failed to read config.json: %w", err)
	}
	if err := json.Unmarshal(data, &spec); err != nil {
		return fmt.Errorf("failed to parse config.json: %w", err)
	}
	containerName := spec.Annotations["io.kubernetes.cri.container-name"]
	if containerName == "queue-proxy" {
		logrus.Error("This is a queue-proxy container. Adding IP env.")
		for i, envVar := range spec.Process.Env {
			if strings.HasPrefix(envVar, "SERVING_READINESS_PROBE") {
				spec.Process.Env = remove(spec.Process.Env, i)
				break
			}
		}
		readinessProbeEnv := fmt.Sprintf("SERVING_READINESS_PROBE={\"tcpSocket\":{\"port\":8080,\"host\":\"%s\"},\"successThreshold\":1}", constants.QueueProxyRedirectIP)
		redirectIPEnv := fmt.Sprintf("REDIRECT_IP=%s", constants.QueueProxyRedirectIP)
		envs := []string{readinessProbeEnv, redirectIPEnv}
		spec.Process.Env = append(spec.Process.Env, envs...)
		fileInfo, err := os.Stat(configDir)
		if err != nil {
			return fmt.Errorf("error getting file info: %v", err)
		}
		permissions := fileInfo.Mode()
		// Write the modified struct back to the JSON file
		updatedData, err := json.MarshalIndent(spec, "", "  ")
		if err != nil {
			return fmt.Errorf("error marshalling JSON: %v", err)
		}

		err = os.WriteFile(configDir, updatedData, permissions)
		if err != nil {
			return fmt.Errorf("error writing to file: %v", err)
		}
		return runcExec()
	}
	return nil
}

func remove(s []string, i int) []string {
	s[i] = s[len(s)-1]
	return s[:len(s)-1]
}
