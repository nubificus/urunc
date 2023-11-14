package unikontainers

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/stretchr/testify/assert"
)

func TestGetUnikernelConfigFromSpec(t *testing.T) {
	spec := &specs.Spec{
		Annotations: map[string]string{
			"com.urunc.unikernel.unikernelType": "dGVzdFR5cGU=",
			"com.urunc.unikernel.cmdline":       "dGVzdENtZA==",
			"com.urunc.unikernel.binary":        "dGVzdEJpbmFyeQ==",
			"com.urunc.unikernel.hypervisor":    "dGVzdEh5cGVydmlzb3I=",
		},
	}

	conf, err := GetUnikernelConfig("", spec)
	assert.NoError(t, err)
	assert.NotNil(t, conf)
	assert.Equal(t, "testType", conf.UnikernelType)
	assert.Equal(t, "testCmd", conf.UnikernelCmd)
	assert.Equal(t, "testBinary", conf.UnikernelBinary)
	assert.Equal(t, "testHypervisor", conf.Hypervisor)
}

func TestGetUnikernelConfigEmptyAnnotations(t *testing.T) {
	// Create a sample spec with no annotations
	spec := &specs.Spec{}

	// Test GetUnikernelConfig with empty annotations
	conf, err := GetUnikernelConfig("", spec)
	assert.Error(t, err)
	assert.Nil(t, conf)
	assert.Equal(t, "failed to retrieve Unikernel config", err.Error())
}

func TestGetUnikernelConfigFromJSON(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := "testbundle"
	err := os.MkdirAll(filepath.Join(".", tmpDir, "rootfs"), os.ModePerm)
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create a sample urunc.json file with test data
	jsonData := `{
		"com.urunc.unikernel.unikernelType": "dGVzdFR5cGU=",
		"com.urunc.unikernel.cmdline":       "dGVzdENtZA==",
		"com.urunc.unikernel.binary":        "dGVzdEJpbmFyeQ==",
		"com.urunc.unikernel.hypervisor":    "dGVzdEh5cGVydmlzb3I="
	}`
	err = os.WriteFile(filepath.Join(tmpDir, "rootfs", "urunc.json"), []byte(jsonData), 0644)
	assert.NoError(t, err)

	// Test GetUnikernelConfig with urunc.json
	conf, err := GetUnikernelConfig(tmpDir, &specs.Spec{})
	assert.NoError(t, err)
	assert.NotNil(t, conf)
	assert.Equal(t, "testType", conf.UnikernelType)
	assert.Equal(t, "testCmd", conf.UnikernelCmd)
	assert.Equal(t, "testBinary", conf.UnikernelBinary)
	assert.Equal(t, "testHypervisor", conf.Hypervisor)
}

func TestGetConfigFromSpec(t *testing.T) {
	// Create a sample spec with annotations
	spec := &specs.Spec{
		Annotations: map[string]string{
			"com.urunc.unikernel.unikernelType": "testType",
			"com.urunc.unikernel.cmdline":       "testCmd",
			"com.urunc.unikernel.binary":        "testBinary",
			"com.urunc.unikernel.hypervisor":    "testHypervisor",
		},
	}

	// Test getConfigFromSpec
	conf, err := getConfigFromSpec(spec)
	assert.NoError(t, err)
	assert.NotNil(t, conf)
	assert.Equal(t, "testType", conf.UnikernelType)
	assert.Equal(t, "testCmd", conf.UnikernelCmd)
	assert.Equal(t, "testBinary", conf.UnikernelBinary)
	assert.Equal(t, "testHypervisor", conf.Hypervisor)
}

func TestGetConfigFromSpecEmptyAnnotations(t *testing.T) {
	// Create a sample spec with no annotations
	spec := &specs.Spec{}

	// Test getConfigFromSpec with empty annotations
	conf, err := getConfigFromSpec(spec)
	assert.Error(t, err)
	assert.Nil(t, conf)
	assert.Equal(t, ErrEmptyAnnotations, err)
}

func TestGetConfigFromJSON(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := "testbundle"
	err := os.MkdirAll(filepath.Join(".", tmpDir, "rootfs"), os.ModePerm)
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create a sample urunc.json file with test data
	jsonData := `{
		"com.urunc.unikernel.unikernelType": "testType",
		"com.urunc.unikernel.cmdline": "testCmd",
		"com.urunc.unikernel.binary": "testBinary",
		"com.urunc.unikernel.hypervisor": "testHypervisor"
	}`
	err = os.WriteFile(filepath.Join(tmpDir, "rootfs", "urunc.json"), []byte(jsonData), 0644)
	assert.NoError(t, err)

	// Test getConfigFromJSON
	conf, err := getConfigFromJSON(tmpDir)
	assert.NoError(t, err)
	assert.NotNil(t, conf)
	assert.Equal(t, "testType", conf.UnikernelType)
	assert.Equal(t, "testCmd", conf.UnikernelCmd)
	assert.Equal(t, "testBinary", conf.UnikernelBinary)
	assert.Equal(t, "testHypervisor", conf.Hypervisor)
}

func TestDecode(t *testing.T) {
	// Create a sample UnikernelConfig with base64-encoded values
	conf := &UnikernelConfig{
		UnikernelType:   base64.StdEncoding.EncodeToString([]byte("encodedType")),
		UnikernelCmd:    base64.StdEncoding.EncodeToString([]byte("encodedCmd")),
		UnikernelBinary: base64.StdEncoding.EncodeToString([]byte("encodedBinary")),
		Hypervisor:      base64.StdEncoding.EncodeToString([]byte("encodedHypervisor")),
	}

	// Test decode
	conf.decode()

	// Check if values are correctly decoded
	assert.Equal(t, "encodedType", conf.UnikernelType)
	assert.Equal(t, "encodedCmd", conf.UnikernelCmd)
	assert.Equal(t, "encodedBinary", conf.UnikernelBinary)
	assert.Equal(t, "encodedHypervisor", conf.Hypervisor)
}

func TestMap(t *testing.T) {
	// Create a sample UnikernelConfig
	conf := &UnikernelConfig{
		UnikernelType:   "testType",
		UnikernelCmd:    "testCmd",
		UnikernelBinary: "testBinary",
		Hypervisor:      "testHypervisor",
	}

	// Test Map
	result := conf.Map()

	// Check if the map is correctly generated
	expected := map[string]string{
		"com.urunc.unikernel.unikernelType": "testType",
		"com.urunc.unikernel.cmdline":       "testCmd",
		"com.urunc.unikernel.binary":        "testBinary",
		"com.urunc.unikernel.hypervisor":    "testHypervisor",
	}

	assert.Equal(t, expected, result)
}
