//go:build linux

package input

import "golang.design/x/hotkey"

// modAlt returns the Alt modifier for Linux (Mod1)
func modAlt() hotkey.Modifier {
	return hotkey.Mod1
}

// modSuper returns the Super/Win modifier for Linux (Mod4)
func modSuper() hotkey.Modifier {
	return hotkey.Mod4
}
