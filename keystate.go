package main

import "github.com/holoplot/go-evdev"

// ----------------------------------------------------------------------

var abortCombo = [...]evdev.EvCode{
	evdev.KEY_LEFTCTRL,
	evdev.KEY_LEFTALT,
	evdev.KEY_LEFTSHIFT,
	evdev.KEY_F12,
}

// ----------------------------------------------------------------------

type KeyState struct {
	keys map[evdev.EvCode]int32
}

// ----------------------------------------------------------------------

func newKeyState() *KeyState {
	return &KeyState{keys: make(map[evdev.EvCode]int32)}
}

// ----------------------------------------------------------------------

func (keyState *KeyState) update(event *evdev.InputEvent) {
	if event.Type == evdev.EV_KEY {
		keyState.keys[event.Code] = event.Value
	}
}

// ----------------------------------------------------------------------

func (keyState *KeyState) isPressed(codes ...evdev.EvCode) bool {
	for _, code := range codes {
		if keyState.keys[code] == 0 {
			return false
		}
	}

	return true
}

// ----------------------------------------------------------------------
