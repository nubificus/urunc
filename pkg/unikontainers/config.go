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

package unikontainers

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sirupsen/logrus"
)

var ErrEmptyAnnotations = errors.New("spec annotations are empty")

// A UnikernelConfig struct holds the info provided by bima image on how to execute our unikernel
type UnikernelConfig struct {
	UnikernelType   string `json:"com.urunc.unikernel.unikernelType"`
	UnikernelCmd    string `json:"com.urunc.unikernel.cmdline,omitempty"`
	UnikernelBinary string `json:"com.urunc.unikernel.binary"`
	Hypervisor      string `json:"com.urunc.unikernel.hypervisor"`
	Initrd          string `json:"com.urunc.unikernel.initrd,omitempty"`
}

// GetUnikernelConfig tries to get the Unikernel config from the bundle annotations.
// If that fails, it tries to read the ENV variable of the spec.
// If that also fails, it gets the Unikernel config from the urunc.json file inside the rootfs.
// FIXME: custom annotations are unreachable, we nned to investigate why to skip adding the urunc.json file
// For more details, see: https://github.com/nubificus/urunc/issues/12
func GetUnikernelConfig(bundleDir string, spec *specs.Spec) (*UnikernelConfig, error) {
	conf, err := getConfigFromSpec(spec)
	if err == nil {
		conf.decode()
		return conf, nil
	}

	conf, err = getConfigFromEnv(spec)
	if err == nil {
		conf.decode()
		return conf, nil
	}

	conf, err = getConfigFromJSON(bundleDir)
	if err == nil {
		conf.decode()
		return conf, nil
	}

	return nil, errors.New("failed to retrieve Unikernel config")
}

// getConfigFromSpec retrieves the urunc specific annotations from the spec and populates the Unikernel config.
func getConfigFromSpec(spec *specs.Spec) (*UnikernelConfig, error) {
	unikernelType := spec.Annotations["com.urunc.unikernel.unikernelType"]
	unikernelCmd := spec.Annotations["com.urunc.unikernel.cmdline"]
	unikernelBinary := spec.Annotations["com.urunc.unikernel.binary"]
	hypervisor := spec.Annotations["com.urunc.unikernel.hypervisor"]
	initrd := spec.Annotations["com.urunc.unikernel.initrd"]

	Log.WithFields(logrus.Fields{
		"unikernelType":   unikernelType,
		"unikernelCmd":    unikernelCmd,
		"unikernelBinary": unikernelBinary,
		"hypervisor":      hypervisor,
		"initrd":          initrd,
	}).Info("urunc annotations")

	newConf := &UnikernelConfig{
		UnikernelBinary: unikernelBinary,
		UnikernelType:   unikernelType,
		UnikernelCmd:    unikernelCmd,
		Hypervisor:      hypervisor,
		Initrd:          initrd,
	}
	if !newConf.IsSet() {
		return nil, ErrEmptyAnnotations

	}
	return newConf, nil
}

// getConfigFromEnv retrieves the urunc specific annotations from the spec's cmd ENV variables and populates the Unikernel config.
func getConfigFromEnv(spec *specs.Spec) (*UnikernelConfig, error) {
	newConfig := &UnikernelConfig{}
	envs := spec.Process.Env
	for _, env := range envs {
		if !strings.Contains(env, "com.urunc.unikernel") {
			continue
		}
		parts := strings.SplitN(env, "=", 2)
		switch key := parts[0]; key {
		case "com.urunc.unikernel.unikernelType":
			newConfig.UnikernelType = parts[1]
		case "com.urunc.unikernel.cmdline":
			newConfig.UnikernelCmd = parts[1]
		case "com.urunc.unikernel.binary":
			newConfig.UnikernelBinary = parts[1]
		case "com.urunc.unikernel.hypervisor":
			newConfig.Hypervisor = parts[1]
		case "com.urunc.unikernel.initrd":
			newConfig.Initrd = parts[1]
		}
	}
	if !newConfig.IsSet() {
		return nil, ErrEmptyAnnotations
	}
	return newConfig, nil
}

// getConfigFromJSON retrieves the Unikernel config parameters from the urunc.json file inside the rootfs.
func getConfigFromJSON(bundleDir string) (*UnikernelConfig, error) {
	jsonFilePath := filepath.Join(bundleDir, "rootfs", "urunc.json")
	file, err := os.Open(jsonFilePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if fileInfo.IsDir() {
		return nil, errors.New("urunc.json is a directory")
	}

	byteData, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	var conf UnikernelConfig
	err = json.Unmarshal(byteData, &conf)
	if err != nil {
		return nil, err
	}
	Log.WithFields(logrus.Fields{
		"unikernelType":   conf.UnikernelType,
		"unikernelCmd":    conf.UnikernelCmd,
		"unikernelBinary": conf.UnikernelBinary,
		"hypervisor":      conf.Hypervisor,
		"initrd":          conf.Initrd,
	}).Info("urunc.json annotations")
	return &conf, nil
}

// decode decodes the base64 encoded values of the Unikernel config
func (c *UnikernelConfig) decode() {
	decoded, err := base64.StdEncoding.DecodeString(c.UnikernelCmd)
	if err != nil {
		Log.WithError(err).Fatal("failed to decode UnikernelCmd")
	}
	c.UnikernelCmd = string(decoded)

	decoded, err = base64.StdEncoding.DecodeString(c.Hypervisor)
	if err != nil {
		Log.WithError(err).Fatal("failed to decode Hypervisor")
	}
	c.Hypervisor = string(decoded)

	decoded, err = base64.StdEncoding.DecodeString(c.UnikernelType)
	if err != nil {
		Log.WithError(err).Fatal("failed to decode UnikernelType")
	}
	c.UnikernelType = string(decoded)

	decoded, err = base64.StdEncoding.DecodeString(c.UnikernelBinary)
	if err != nil {
		Log.WithError(err).Fatal("failed to decode UnikernelBinary")
	}
	c.UnikernelBinary = string(decoded)

	decoded, err = base64.StdEncoding.DecodeString(c.Initrd)
	if err != nil {
		Log.WithError(err).Fatal("failed to decode Initrd")
	}
	c.Initrd = string(decoded)
}

// Map returns a map containing the Unikernel config data
func (c *UnikernelConfig) Map() map[string]string {
	myMap := make(map[string]string)
	if c.UnikernelCmd != "" {
		myMap["com.urunc.unikernel.cmdline"] = c.UnikernelCmd
	}
	if c.UnikernelType != "" {
		myMap["com.urunc.unikernel.unikernelType"] = c.UnikernelType
	}
	if c.Hypervisor != "" {
		myMap["com.urunc.unikernel.hypervisor"] = c.Hypervisor
	}
	if c.UnikernelBinary != "" {
		myMap["com.urunc.unikernel.binary"] = c.UnikernelBinary
	}
	if c.Initrd != "" {
		myMap["com.urunc.unikernel.initrd"] = c.Initrd
	}
	return myMap
}

// IsSet returns true if any field is set
func (c *UnikernelConfig) IsSet() bool {
	conf := fmt.Sprintf("%s%s%s%s%s", c.UnikernelType, c.UnikernelCmd, c.UnikernelBinary, c.Hypervisor, c.Initrd)
	return conf != ""
}
