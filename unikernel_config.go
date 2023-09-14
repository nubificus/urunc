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
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sirupsen/logrus"
)

var ErrEmptyAnnotations = errors.New("spec annotations are empty")

// A config struct that holds the necessary info on how to execute our unikernel
type UnikernelConfig struct {
	UnikernelType   string `json:"com.urunc.unikernel.unikernelType"`
	UnikernelCmd    string `json:"com.urunc.unikernel.cmdline,omitempty"`
	UnikernelBinary string `json:"com.urunc.unikernel.binary"`
	Hypervisor      string `json:"com.urunc.unikernel.hypervisor"`
}

// GetUnikernelConfig tries to get the Unikernel config from the bundle annotation.
// If that fails, it gets the Unikernel config from the urunc.json file inside the rootfs.
func GetUnikernelConfig(bundleDir string, spec *specs.Spec) (*UnikernelConfig, error) {
	conf, err := getConfigFromSpec(spec)
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

	Log.WithFields(logrus.Fields{
		"unikernelType":   unikernelType,
		"unikernelCmd":    unikernelCmd,
		"unikernelBinary": unikernelBinary,
		"hypervisor":      hypervisor,
	}).Info("urunc annotations")

	conf := fmt.Sprintf("%s%s%s%s", unikernelType, unikernelCmd, unikernelBinary, hypervisor)
	if conf == "" {
		return nil, ErrEmptyAnnotations
	}

	Log.Info("Found urunc config in spec annotations")
	return &UnikernelConfig{
		UnikernelBinary: unikernelBinary,
		UnikernelType:   unikernelType,
		UnikernelCmd:    unikernelCmd,
		Hypervisor:      hypervisor,
	}, nil
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

	Log.WithField("config", conf).Info("Found urunc config in urunc.json")
	return &conf, nil
}

// Decodes the base64 encoded values of the Unikernel config
func (c *UnikernelConfig) decode() {
	decoded, err := base64.StdEncoding.DecodeString(c.UnikernelCmd)
	if err != nil {
		Log.Error(err.Error())
		return
	}
	c.UnikernelCmd = string(decoded)

	decoded, err = base64.StdEncoding.DecodeString(c.Hypervisor)
	if err != nil {
		Log.Error(err.Error())
		return
	}
	c.Hypervisor = string(decoded)

	decoded, err = base64.StdEncoding.DecodeString(c.UnikernelType)
	if err != nil {
		Log.Error(err.Error())
		return
	}
	c.UnikernelType = string(decoded)

	decoded, err = base64.StdEncoding.DecodeString(c.UnikernelBinary)
	if err != nil {
		Log.Error(err.Error())
		return
	}
	c.UnikernelBinary = string(decoded)
}

// Returns a map containing the Unikernel config data
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
	return myMap
}

type UruncConfig struct {
	UnikernelBinary string
	VmmType         string
	UnikernelCmd    string
	IsSet           bool
}
