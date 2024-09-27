// Copyright 2024 Nubificus LTD.

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package unikontainers

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/nubificus/urunc/internal/constants"
	"github.com/opencontainers/runtime-spec/specs-go"
)

const (
	configFilename    = "config.json"
	stateFilename     = "state.json"
	initPidFilename   = "init.pid"
	uruncJsonFilename = "urunc.json"
	rootfsDirName     = "rootfs"
)

// getInitPid extracts "init_process_pid" value from the given JSON file
func getInitPid(filePath string) (float64, error) {
	// Open the JSON file for reading
	file, err := os.Open(filePath)
	if err != nil {
		return 0, nil
	}
	defer file.Close()

	// Decode the JSON data into a map[string]interface{}
	var jsonData map[string]interface{}
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&jsonData); err != nil {
		return 0, nil

	}

	// Extract the specific value "init_process_pid"
	initProcessPID, found := jsonData["init_process_pid"].(float64) // Assuming it's a numeric value
	if !found {
		return 0, nil
	}
	return initProcessPID, nil
}

// copy sourceFile to targetDir
// creates targetDir and all necessary parent directories
func copyFile(sourceFile string, targetDir string) error {
	source, err := os.Open(sourceFile)
	if err != nil {
		return err
	}
	defer source.Close()
	err = os.MkdirAll(targetDir, 0755)
	if err != nil {
		return err
	}

	_, filename := filepath.Split(sourceFile)
	targetPath := filepath.Join(targetDir, filename)
	target, err := os.Create(targetPath)
	if err != nil {
		return err
	}
	defer target.Close()
	_, err = io.Copy(target, source)
	if err != nil {
		return err
	}
	return nil
}

// move sourceFile to targetDir
// creates targetDir and all necessary parent directories
func moveFile(sourceFile string, targetDir string) error {
	err := copyFile(sourceFile, targetDir)
	if err != nil {
		return err
	}
	return os.Remove(sourceFile)
}

// loadSpec returns the Spec found in the given bundle directory
func loadSpec(bundleDir string) (*specs.Spec, error) {
	var spec specs.Spec

	absBundleDir, err := filepath.Abs(bundleDir)
	if err != nil {
		return nil, fmt.Errorf("failed to find absolute path of bundle: %w", err)
	}

	configFile := filepath.Join(absBundleDir, configFilename)
	specData, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read specification file: %w", err)
	}

	if err := json.Unmarshal(specData, &spec); err != nil {
		return nil, fmt.Errorf("failed to parse specification json: %w", err)
	}

	return &spec, nil
}

// writePidFile writes the content of pid to the file defined by path
func writePidFile(path string, pid int) error {
	var (
		tmpDir  = filepath.Dir(path)
		tmpName = filepath.Join(tmpDir, "."+filepath.Base(path))
	)
	f, err := os.OpenFile(tmpName, os.O_RDWR|os.O_CREATE|os.O_EXCL|os.O_SYNC, 0o666)
	if err != nil {
		return err
	}
	_, err = f.WriteString(strconv.Itoa(pid))
	f.Close()
	if err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

// handleQueueProxy adds a hardcoded IP to the process's environment.
// Then, the container is identified as a non-bima container
// is spawned using runc.
func handleQueueProxy(spec specs.Spec, configFile string) error {
	for i, envVar := range spec.Process.Env {
		if strings.HasPrefix(envVar, "SERVING_READINESS_PROBE") {
			spec.Process.Env = remove(spec.Process.Env, i)
			break
		}
	}

	// Set new environment variables for Queue Proxy container
	readinessProbeEnv := fmt.Sprintf("SERVING_READINESS_PROBE={\"tcpSocket\":{\"port\":8080,\"host\":\"%s\"},\"successThreshold\":1}", constants.QueueProxyRedirectIP)
	redirectIPEnv := fmt.Sprintf("REDIRECT_IP=%s", constants.QueueProxyRedirectIP)
	envs := []string{readinessProbeEnv, redirectIPEnv}
	spec.Process.Env = append(spec.Process.Env, envs...)

	// Get permissions of specification file
	fileInfo, err := os.Stat(configFile)
	if err != nil {
		return fmt.Errorf("error getting file info: %v", err)
	}
	permissions := fileInfo.Mode()

	// Write the modified struct back to the JSON file
	updatedData, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshalling JSON: %v", err)
	}

	err = os.WriteFile(configFile, updatedData, permissions)
	if err != nil {
		return fmt.Errorf("error writing to file: %v", err)
	}

	// Exec runc to handle the Queue Proxy container
	return nil
}

func remove(s []string, i int) []string {
	s[i] = s[len(s)-1]
	return s[:len(s)-1]
}
