//go:build darwin

package input

import "golang.design/x/hotkey"

// modAlt returns the Option modifier for macOS
func modAlt() hotkey.Modifier {
	return hotkey.ModOption
}

// modSuper returns the Command modifier for macOS
func modSuper() hotkey.Modifier {
	return hotkey.ModCmd
}
