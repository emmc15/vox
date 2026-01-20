package input

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"golang.design/x/hotkey"
)

// HotkeyManager manages global hotkey registration and events
type HotkeyManager struct {
	mu        sync.Mutex
	hk        *hotkey.Hotkey
	recording bool
	onToggle  func(recording bool)
	cancel    context.CancelFunc
	done      chan struct{}
}

// NewHotkeyManager creates a new HotkeyManager
func NewHotkeyManager(onToggle func(recording bool)) *HotkeyManager {
	return &HotkeyManager{
		onToggle: onToggle,
		done:     make(chan struct{}),
	}
}

// Start begins listening for hotkey events
func (h *HotkeyManager) Start(ctx context.Context, hotkeyStr string) error {
	mods, key, err := parseHotkey(hotkeyStr)
	if err != nil {
		return fmt.Errorf("invalid hotkey: %w", err)
	}

	h.hk = hotkey.New(mods, key)
	if err := h.hk.Register(); err != nil {
		return fmt.Errorf("failed to register hotkey: %w", err)
	}

	ctx, h.cancel = context.WithCancel(ctx)

	go func() {
		defer close(h.done)
		for {
			select {
			case <-ctx.Done():
				return
			case _, ok := <-h.hk.Keydown():
				if !ok {
					return
				}
				h.mu.Lock()
				h.recording = !h.recording
				recording := h.recording
				h.mu.Unlock()

				if h.onToggle != nil {
					h.onToggle(recording)
				}
			}
		}
	}()

	return nil
}

// Stop stops listening for hotkey events
func (h *HotkeyManager) Stop() {
	if h.cancel != nil {
		h.cancel()
	}
	// Unregister hotkey
	if h.hk != nil {
		h.hk.Unregister()
	}
	// Wait briefly for goroutine to exit
	if h.done != nil {
		select {
		case <-h.done:
		case <-time.After(100 * time.Millisecond):
		}
	}
}

// IsRecording returns the current recording state
func (h *HotkeyManager) IsRecording() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.recording
}

// parseHotkey parses a hotkey string like "ctrl+shift+space" into modifiers and key
func parseHotkey(s string) ([]hotkey.Modifier, hotkey.Key, error) {
	parts := strings.Split(strings.ToLower(s), "+")
	if len(parts) == 0 {
		return nil, 0, fmt.Errorf("empty hotkey string")
	}

	var mods []hotkey.Modifier
	var key hotkey.Key
	var keyFound bool

	for _, part := range parts {
		part = strings.TrimSpace(part)
		switch part {
		case "ctrl", "control":
			mods = append(mods, hotkey.ModCtrl)
		case "shift":
			mods = append(mods, hotkey.ModShift)
		case "alt":
			mods = append(mods, modAlt())
		case "cmd", "command", "super", "win":
			mods = append(mods, modSuper())
		default:
			if keyFound {
				return nil, 0, fmt.Errorf("multiple keys specified")
			}
			k, err := parseKey(part)
			if err != nil {
				return nil, 0, err
			}
			key = k
			keyFound = true
		}
	}

	if !keyFound {
		return nil, 0, fmt.Errorf("no key specified")
	}

	return mods, key, nil
}

// parseKey parses a key name to hotkey.Key
func parseKey(s string) (hotkey.Key, error) {
	switch s {
	case "space":
		return hotkey.KeySpace, nil
	case "return", "enter":
		return hotkey.KeyReturn, nil
	case "tab":
		return hotkey.KeyTab, nil
	case "escape", "esc":
		return hotkey.KeyEscape, nil
	case "a":
		return hotkey.KeyA, nil
	case "b":
		return hotkey.KeyB, nil
	case "c":
		return hotkey.KeyC, nil
	case "d":
		return hotkey.KeyD, nil
	case "e":
		return hotkey.KeyE, nil
	case "f":
		return hotkey.KeyF, nil
	case "g":
		return hotkey.KeyG, nil
	case "h":
		return hotkey.KeyH, nil
	case "i":
		return hotkey.KeyI, nil
	case "j":
		return hotkey.KeyJ, nil
	case "k":
		return hotkey.KeyK, nil
	case "l":
		return hotkey.KeyL, nil
	case "m":
		return hotkey.KeyM, nil
	case "n":
		return hotkey.KeyN, nil
	case "o":
		return hotkey.KeyO, nil
	case "p":
		return hotkey.KeyP, nil
	case "q":
		return hotkey.KeyQ, nil
	case "r":
		return hotkey.KeyR, nil
	case "s":
		return hotkey.KeyS, nil
	case "t":
		return hotkey.KeyT, nil
	case "u":
		return hotkey.KeyU, nil
	case "v":
		return hotkey.KeyV, nil
	case "w":
		return hotkey.KeyW, nil
	case "x":
		return hotkey.KeyX, nil
	case "y":
		return hotkey.KeyY, nil
	case "z":
		return hotkey.KeyZ, nil
	case "0":
		return hotkey.Key0, nil
	case "1":
		return hotkey.Key1, nil
	case "2":
		return hotkey.Key2, nil
	case "3":
		return hotkey.Key3, nil
	case "4":
		return hotkey.Key4, nil
	case "5":
		return hotkey.Key5, nil
	case "6":
		return hotkey.Key6, nil
	case "7":
		return hotkey.Key7, nil
	case "8":
		return hotkey.Key8, nil
	case "9":
		return hotkey.Key9, nil
	case "f1":
		return hotkey.KeyF1, nil
	case "f2":
		return hotkey.KeyF2, nil
	case "f3":
		return hotkey.KeyF3, nil
	case "f4":
		return hotkey.KeyF4, nil
	case "f5":
		return hotkey.KeyF5, nil
	case "f6":
		return hotkey.KeyF6, nil
	case "f7":
		return hotkey.KeyF7, nil
	case "f8":
		return hotkey.KeyF8, nil
	case "f9":
		return hotkey.KeyF9, nil
	case "f10":
		return hotkey.KeyF10, nil
	case "f11":
		return hotkey.KeyF11, nil
	case "f12":
		return hotkey.KeyF12, nil
	default:
		return 0, fmt.Errorf("unknown key: %s", s)
	}
}
