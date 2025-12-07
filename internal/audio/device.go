package audio

import (
	"fmt"
	"strings"

	"github.com/gen2brain/malgo"
)

// DeviceType represents the type of audio device
type DeviceType int

const (
	DeviceTypePlayback DeviceType = iota
	DeviceTypeCapture
	DeviceTypeDuplex
)

// DeviceInfo contains information about an audio device
type DeviceInfo struct {
	ID                string     // Unique device identifier
	Name              string     // Human-readable device name
	Type              DeviceType // Device type (playback, capture, duplex)
	IsDefault         bool       // Whether this is the default device
	MaxChannels       uint32     // Maximum number of supported channels
	SupportedRates    []uint32   // Supported sample rates
	NativeFormat      string     // Native audio format
	MinBufferSize     uint32     // Minimum buffer size in frames
	MaxBufferSize     uint32     // Maximum buffer size in frames
	DefaultBufferSize uint32     // Default buffer size in frames
}

// String returns a human-readable representation of the device
func (d DeviceInfo) String() string {
	defaultMarker := ""
	if d.IsDefault {
		defaultMarker = " [DEFAULT]"
	}
	return fmt.Sprintf("%s: %s%s (channels: %d, rates: %v)",
		d.ID, d.Name, defaultMarker, d.MaxChannels, d.SupportedRates)
}

// ListDevices returns a list of all available audio devices
func ListDevices() ([]DeviceInfo, error) {
	ctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize malgo context: %w", err)
	}
	defer func() {
		_ = ctx.Uninit()
		ctx.Free()
	}()

	// Get capture devices
	infos, err := ctx.Devices(malgo.Capture)
	if err != nil {
		return nil, fmt.Errorf("failed to enumerate devices: %w", err)
	}

	devices := make([]DeviceInfo, 0, len(infos))
	for i, info := range infos {
		device := DeviceInfo{
			ID:          fmt.Sprintf("capture-%d", i),
			Name:        info.Name(),
			Type:        DeviceTypeCapture,
			IsDefault:   info.IsDefault > 0,
			MaxChannels: 2, // Default to stereo, malgo doesn't expose this directly
		}
		devices = append(devices, device)
	}

	return devices, nil
}

// GetDefaultDevice returns the default capture device
func GetDefaultDevice() (*DeviceInfo, error) {
	devices, err := ListDevices()
	if err != nil {
		return nil, err
	}

	for _, device := range devices {
		if device.IsDefault {
			return &device, nil
		}
	}

	// If no default is found, return the first device
	if len(devices) > 0 {
		return &devices[0], nil
	}

	return nil, fmt.Errorf("no capture devices found")
}

// FindDeviceByID finds a device by its ID
func FindDeviceByID(id string) (*DeviceInfo, error) {
	devices, err := ListDevices()
	if err != nil {
		return nil, err
	}

	for _, device := range devices {
		if device.ID == id {
			return &device, nil
		}
	}

	return nil, fmt.Errorf("device not found: %s", id)
}

// FindDeviceByName finds a device by name (case-insensitive partial match)
func FindDeviceByName(name string) (*DeviceInfo, error) {
	devices, err := ListDevices()
	if err != nil {
		return nil, err
	}

	searchName := strings.ToLower(name)
	for _, device := range devices {
		deviceName := strings.ToLower(device.Name)
		if strings.Contains(deviceName, searchName) {
			return &device, nil
		}
	}

	return nil, fmt.Errorf("no device found matching name: %s", name)
}
