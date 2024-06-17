package network

import (
	"errors"
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockNetInterfaceFetcher is a mock implementation of NetInterfaceFetcher.
type MockNetInterfaceFetcher struct {
	mock.Mock
}

// Interfaces mocks the Interfaces method of NetInterfaceFetcher.
func (m *MockNetInterfaceFetcher) Interfaces() ([]net.Interface, error) {
	args := m.Called()
	return args.Get(0).([]net.Interface), args.Error(1)
}

func TestEnsureEth0Exists(t *testing.T) {
	t.Run("eth0 found", func(t *testing.T) {
		t.Parallel()
		mockFetcher := new(MockNetInterfaceFetcher)
		mockFetcher.On("Interfaces").Return([]net.Interface{
			{Name: DefaultInterface},
		}, nil)
		err := ensureEth0Exists(mockFetcher)
		assert.NoError(t, err)
		mockFetcher.AssertExpectations(t)
	})

	t.Run("eth0 not found", func(t *testing.T) {
		t.Parallel()
		mockFetcher := new(MockNetInterfaceFetcher)
		mockFetcher.On("Interfaces").Return([]net.Interface{
			{Name: "lo"},
		}, nil)
		err := ensureEth0Exists(mockFetcher)
		assert.EqualError(t, err, "eth0 device not found")
		mockFetcher.AssertExpectations(t)
	})

	t.Run("eth0 error", func(t *testing.T) {
		mockFetcher := &MockNetInterfaceFetcher{}
		mockFetcher.On("Interfaces").Return([]net.Interface{}, errors.New("failed to get interfaces"))
		err := ensureEth0Exists(mockFetcher)
		assert.EqualError(t, err, "failed to get interfaces")
		mockFetcher.AssertExpectations(t)
	})
}

func TestGetTapIndex(t *testing.T) {
	t.Run("No TAP interfaces", func(t *testing.T) {
		t.Parallel()
		mockFetcher := new(MockNetInterfaceFetcher)
		mockFetcher.On("Interfaces").Return([]net.Interface{
			{Name: "eth0"},
			{Name: "tap0"},
			{Name: "tap1"},
			{Name: "tap2"},
			{Name: "lo"},
		}, nil)
		count, err := getTapIndex(mockFetcher)
		assert.NoError(t, err)
		assert.Equal(t, 3, count)
		mockFetcher.AssertExpectations(t)

	})

	t.Run("Some TAP interfaces", func(t *testing.T) {
		t.Parallel()
		mockFetcher := new(MockNetInterfaceFetcher)
		mockFetcher.On("Interfaces").Return([]net.Interface{
			{Name: "eth0"},
			{Name: "lo"},
		}, nil)
		count, err := getTapIndex(mockFetcher)
		assert.NoError(t, err)
		assert.Equal(t, 0, count)
		mockFetcher.AssertExpectations(t)

	})

	t.Run("Error fetching TAP interfaces", func(t *testing.T) {
		t.Parallel()
		mockFetcher := new(MockNetInterfaceFetcher)
		mockFetcher.On("Interfaces").Return([]net.Interface{}, errors.New("failed to get interfaces"))
		count, err := getTapIndex(mockFetcher)
		assert.EqualError(t, err, "failed to get interfaces")
		assert.Equal(t, 0, count)
		mockFetcher.AssertExpectations(t)
	})

	t.Run("Too many TAP interfaces", func(t *testing.T) {
		t.Parallel()
		mockFetcher := new(MockNetInterfaceFetcher)
		var interfaces []net.Interface
		for i := 0; i < 256; i++ {
			interfaces = append(interfaces, net.Interface{Name: "tap" + fmt.Sprint(i)})
		}
		mockFetcher.On("Interfaces").Return(interfaces, nil)
		count, err := getTapIndex(mockFetcher)
		assert.EqualError(t, err, "TAP interfaces count higher than 255")
		assert.Equal(t, 256, count)
		mockFetcher.AssertExpectations(t)
	})
}

func TestNewNetworkManager(t *testing.T) {
	// Test for "static" network type.
	t.Run("Static", func(t *testing.T) {
		t.Parallel()
		manager, err := NewNetworkManager("static")
		assert.NoError(t, err)
		_, ok := manager.(*StaticNetwork)
		assert.True(t, ok)
	})

	// Test for "dynamic" network type.
	t.Run("Dynamic", func(t *testing.T) {
		t.Parallel()
		manager, err := NewNetworkManager("dynamic")
		assert.NoError(t, err)
		_, ok := manager.(*DynamicNetwork)
		assert.True(t, ok)
	})

	// Test for unsupported network type.
	t.Run("UnsupportedType", func(t *testing.T) {
		t.Parallel()
		manager, err := NewNetworkManager("invalid")
		assert.Nil(t, manager)
		assert.EqualError(t, err, "network manager invalid not supported")
	})
}
