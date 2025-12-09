package app

import (
	"fmt"
	"os"

	"github.com/emmett/diaz/internal/audio"
)

// DeviceManager handles audio device selection and listing
type DeviceManager struct{}

// NewDeviceManager creates a new DeviceManager instance
func NewDeviceManager() *DeviceManager {
	return &DeviceManager{}
}

// ListDevices lists all available audio input devices
func (dm *DeviceManager) ListDevices() error {
	fmt.Println("Detecting audio input devices...")
	fmt.Println()

	devices, err := audio.ListDevices()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to list devices: %v\n", err)
		return err
	}

	if len(devices) == 0 {
		fmt.Println("No audio capture devices found.")
		return fmt.Errorf("no devices found")
	}

	fmt.Printf("Found %d capture device(s):\n\n", len(devices))

	for i, device := range devices {
		marker := ""
		if device.IsDefault {
			marker = " [DEFAULT]"
		}
		fmt.Printf("%d. %s%s\n", i+1, device.Name, marker)
		fmt.Printf("   ID: %s\n", device.ID)
		if device.MaxChannels > 0 {
			fmt.Printf("   Max Channels: %d\n", device.MaxChannels)
		}
		if len(device.SupportedRates) > 0 {
			fmt.Printf("   Supported Rates: %v Hz\n", device.SupportedRates)
		}
		fmt.Println()
	}

	fmt.Println("To use a specific device, run:")
	fmt.Println("  diaz --device \"<device-name>\"")
	fmt.Println()
	fmt.Println("Example:")
	if len(devices) > 0 {
		fmt.Printf("  diaz --device \"%s\"\n", devices[0].Name)
	}

	return nil
}

// SelectDevice selects an audio device by name/ID, or returns default
func (dm *DeviceManager) SelectDevice(deviceName string) (*audio.DeviceInfo, error) {
	// List available devices
	fmt.Println("\nDetecting audio devices...")
	devices, err := audio.ListDevices()
	if err != nil {
		return nil, fmt.Errorf("failed to list devices: %w", err)
	}

	if len(devices) == 0 {
		return nil, fmt.Errorf("no audio capture devices found")
	}

	fmt.Printf("Found %d capture device(s):\n", len(devices))
	for _, device := range devices {
		fmt.Printf("  - %s\n", device.String())
	}
	fmt.Println()

	var selectedDevice *audio.DeviceInfo

	if deviceName != "" {
		// User specified a device, search for it
		for i := range devices {
			if devices[i].Name == deviceName || devices[i].ID == deviceName {
				selectedDevice = &devices[i]
				break
			}
		}

		if selectedDevice == nil {
			fmt.Fprintf(os.Stderr, "Error: Device '%s' not found\n\n", deviceName)
			fmt.Println("Available devices:")
			for i, device := range devices {
				marker := ""
				if device.IsDefault {
					marker = " [DEFAULT]"
				}
				fmt.Printf("  %d. %s%s\n", i+1, device.Name, marker)
			}
			fmt.Println()
			fmt.Println("Use --list-devices for more details")
			return nil, fmt.Errorf("invalid audio device specified")
		}
	} else {
		// Use default device
		defaultDevice, err := audio.GetDefaultDevice()
		if err != nil {
			return nil, fmt.Errorf("failed to get default device: %w", err)
		}
		selectedDevice = defaultDevice
	}

	fmt.Printf("Using device: %s\n", selectedDevice.Name)
	fmt.Println()

	return selectedDevice, nil
}
