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

package unikontainers

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

// Important: Unfortunately GOlang does not allow to use constant values for
// struct tagsAs a result, please always keep the constant definitions and the
// UnikernelConfig struct below in sync.

// Urunc specific annotations
// ALways keep it in sync with the struct UnikernelConfig struct
const (
	annotType          = "com.urunc.unikernel.unikernelType"
	annotVersion       = "com.urunc.unikernel.unikernelVersion"
	annotBinary        = "com.urunc.unikernel.binary"
	annotCmdLine       = "com.urunc.unikernel.cmdline"
	annotHypervisor    = "com.urunc.unikernel.hypervisor"
	annotInitrd        = "com.urunc.unikernel.initrd"
	annotBlock         = "com.urunc.unikernel.block"
	annotBlockMntPoint = "com.urunc.unikernel.blkMntPoint"
	annotUseDMBlock    = "com.urunc.unikernel.useDMBlock"
)

// A UnikernelConfig struct holds the info provided by bima image on how to execute our unikernel
type UnikernelConfig struct {
	UnikernelType    string `json:"com.urunc.unikernel.unikernelType"`
	UnikernelVersion string `json:"com.urunc.unikernel.unikernelVersion"`
	UnikernelCmd     string `json:"com.urunc.unikernel.cmdline,omitempty"`
	UnikernelBinary  string `json:"com.urunc.unikernel.binary"`
	Hypervisor       string `json:"com.urunc.unikernel.hypervisor"`
	Initrd           string `json:"com.urunc.unikernel.initrd,omitempty"`
	Block            string `json:"com.urunc.unikernel.block,omitempty"`
	BlkMntPoint      string `json:"com.urunc.unikernel.blkMntPoint,omitempty"`
	UseDMBlock       string `json:"com.urunc.unikernel.useDMBlock"`
}

// GetUnikernelConfig tries to get the Unikernel config from the bundle annotations.
// If that fails, it gets the Unikernel config from the urunc.json file inside the rootfs.
// FIXME: custom annotations are unreachable, we need to investigate why to skip adding the urunc.json file
// For more details, see: https://github.com/nubificus/urunc/issues/12
func GetUnikernelConfig(bundleDir string, spec *specs.Spec) (*UnikernelConfig, error) {
	conf, err := getConfigFromSpec(spec)
	if err == nil {
		err1 := conf.decode()
		if err1 != nil {
			return nil, err1
		}
		return conf, nil
	}
	rootFSDir := spec.Root.Path

	var jsonFilePath string
	if filepath.IsAbs(rootFSDir) {
		jsonFilePath = filepath.Join(rootFSDir, uruncJSONFilename)
	} else {
		jsonFilePath = filepath.Join(bundleDir, rootFSDir, uruncJSONFilename)
	}
	conf, err = getConfigFromJSON(jsonFilePath)
	if err == nil {
		err1 := conf.decode()
		if err1 != nil {
			return nil, err1
		}
		return conf, nil
	}

	return nil, errors.New("failed to retrieve Unikernel config")
}

// getConfigFromSpec retrieves the urunc specific annotations from the spec and populates the Unikernel config.
func getConfigFromSpec(spec *specs.Spec) (*UnikernelConfig, error) {
	unikernelType := spec.Annotations[annotType]
	unikernelVersion := spec.Annotations[annotVersion]
	unikernelCmd := spec.Annotations[annotCmdLine]
	unikernelBinary := spec.Annotations[annotBinary]
	hypervisor := spec.Annotations[annotHypervisor]
	initrd := spec.Annotations[annotInitrd]
	block := spec.Annotations[annotBlock]
	blkMntPoint := spec.Annotations[annotBlockMntPoint]
	useDMBlock := spec.Annotations[annotUseDMBlock]

	Log.WithFields(logrus.Fields{
		"unikernelType":    unikernelType,
		"unikernelVersion": unikernelVersion,
		"unikernelCmd":     unikernelCmd,
		"unikernelBinary":  unikernelBinary,
		"hypervisor":       hypervisor,
		"initrd":           initrd,
		"block":            block,
		"blkMntPoint":      blkMntPoint,
		"useDMBlock":       useDMBlock,
	}).Info("urunc annotations")

	// TODO: We need to use a better check to see if annotations were empty
	conf := fmt.Sprintf("%s%s%s%s%s%s%s%s", unikernelType, unikernelVersion, unikernelCmd, unikernelBinary, hypervisor, initrd, block, blkMntPoint)
	if conf == "" {
		return nil, ErrEmptyAnnotations
	}
	return &UnikernelConfig{
		UnikernelBinary:  unikernelBinary,
		UnikernelVersion: unikernelVersion,
		UnikernelType:    unikernelType,
		UnikernelCmd:     unikernelCmd,
		Hypervisor:       hypervisor,
		Initrd:           initrd,
		Block:            block,
		BlkMntPoint:      blkMntPoint,
		UseDMBlock:       useDMBlock,
	}, nil
}

// getConfigFromJSON retrieves the Unikernel config parameters from the urunc.json file inside the rootfs.
func getConfigFromJSON(jsonFilePath string) (*UnikernelConfig, error) {
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
		return nil, errors.New(uruncJSONFilename + " is a directory")
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
		"unikernelType":    conf.UnikernelType,
		"unikernelVersion": conf.UnikernelVersion,
		"unikernelCmd":     conf.UnikernelCmd,
		"unikernelBinary":  conf.UnikernelBinary,
		"hypervisor":       conf.Hypervisor,
		"initrd":           conf.Initrd,
		"block":            conf.Block,
		"blkMntPoint":      conf.BlkMntPoint,
		"useDMBlock":       conf.UseDMBlock,
	}).Info(uruncJSONFilename + " annotations")
	return &conf, nil
}

// decode decodes the base64 encoded values of the Unikernel config
func (c *UnikernelConfig) decode() error {
	decoded, err := base64.StdEncoding.DecodeString(c.UnikernelCmd)
	if err != nil {
		return fmt.Errorf("failed to decode UnikernelCmd: %v", err)
	}
	c.UnikernelCmd = string(decoded)

	decoded, err = base64.StdEncoding.DecodeString(c.Hypervisor)
	if err != nil {
		return fmt.Errorf("failed to decode Hypervisor: %v", err)
	}
	c.Hypervisor = string(decoded)

	decoded, err = base64.StdEncoding.DecodeString(c.UnikernelType)
	if err != nil {
		return fmt.Errorf("failed to decode UnikernelType: %v", err)
	}
	c.UnikernelType = string(decoded)

	decoded, err = base64.StdEncoding.DecodeString(c.UnikernelVersion)
	if err != nil {
		return fmt.Errorf("failed to decode UnikernelVersion: %v", err)
	}
	c.UnikernelVersion = string(decoded)

	decoded, err = base64.StdEncoding.DecodeString(c.UnikernelBinary)
	if err != nil {
		return fmt.Errorf("failed to decode UnikernelBinary: %v", err)
	}
	c.UnikernelBinary = string(decoded)

	decoded, err = base64.StdEncoding.DecodeString(c.Initrd)
	if err != nil {
		return fmt.Errorf("failed to decode Initrd: %v", err)
	}
	c.Initrd = string(decoded)

	decoded, err = base64.StdEncoding.DecodeString(c.Block)
	if err != nil {
		return fmt.Errorf("failed to decode Block: %v", err)
	}
	c.Block = string(decoded)

	decoded, err = base64.StdEncoding.DecodeString(c.BlkMntPoint)
	if err != nil {
		return fmt.Errorf("failed to decode BlockMntPoint: %v", err)
	}
	c.BlkMntPoint = string(decoded)

	decoded, err = base64.StdEncoding.DecodeString(c.UseDMBlock)
	if err != nil {
		return fmt.Errorf("failed to decode UseDMBlock: %v", err)
	}
	c.UseDMBlock = string(decoded)

	return nil
}

// Map returns a map containing the Unikernel config data
func (c *UnikernelConfig) Map() map[string]string {
	myMap := make(map[string]string)
	if c.UnikernelCmd != "" {
		myMap[annotCmdLine] = c.UnikernelCmd
	}
	if c.UnikernelType != "" {
		myMap[annotType] = c.UnikernelType
	}
	if c.UnikernelVersion != "" {
		myMap[annotVersion] = c.UnikernelVersion
	}
	if c.Hypervisor != "" {
		myMap[annotHypervisor] = c.Hypervisor
	}
	if c.UnikernelBinary != "" {
		myMap[annotBinary] = c.UnikernelBinary
	}
	if c.Initrd != "" {
		myMap[annotInitrd] = c.Initrd
	}
	if c.Block != "" {
		myMap[annotBlock] = c.Block
	}
	if c.BlkMntPoint != "" {
		myMap[annotBlockMntPoint] = c.BlkMntPoint
	}
	if c.UseDMBlock != "" {
		myMap[annotUseDMBlock] = c.UseDMBlock
	} else {
		myMap[annotUseDMBlock] = os.Getenv("USE_DEVMAPPER_AS_BLOCK")
	}

	return myMap
}
