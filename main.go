package main

import (
	"fmt"
	"time"

	"github.com/holoplot/go-evdev"
)

// ----------------------------------------------------------------------

func startListener(appCtx *AppContext, device *evdev.InputDevice, events chan *evdev.InputEvent) {

	defer appCtx.wg.Done()

	for {
		select {
		case <-appCtx.Context.Done():
			return
		default:
			device.NonBlock()
			event, err := device.ReadOne()

			if appCtx.Context.Err() != nil {
				return
			}

			if err != nil {
				appCtx.Errors <- err
				return
			}

			if event != nil {
				events <- event
			}
		}
	}
}

var SCAN_INPUT_EVENT = evdev.InputEvent{
	Type:  evdev.EV_MSC,
	Code:  evdev.MSC_SCAN,
	Value: 458976,
}

var SYN_INPUT_EVENT = evdev.InputEvent{
	Type:  evdev.EV_SYN,
	Code:  evdev.SYN_REPORT,
	Value: 0,
}

func sendKey(device *evdev.InputDevice, key evdev.EvCode, value int32) {

	/*
		keyboard event: type: 0x04 [EV_MSC], code: 0x04 [MSC_SCAN], value: 458976
		keyboard event: type: 0x01 [EV_KEY], code: 0x1d [KEY_LEFTCTRL], value: 1
		keyboard event: type: 0x00 [EV_SYN], code: 0x00 [SYN_REPORT], value: 0
		keyboard event: type: 0x04 [EV_MSC], code: 0x04 [MSC_SCAN], value: 458976
		keyboard event: type: 0x01 [EV_KEY], code: 0x1d [KEY_LEFTCTRL], value: 0
		keyboard event: type: 0x00 [EV_SYN], code: 0x00 [SYN_REPORT], value: 0
	*/

	device.WriteOne(&SCAN_INPUT_EVENT)

	event := &evdev.InputEvent{
		Type:  evdev.EV_KEY,
		Code:  key,
		Value: value,
	}

	device.WriteOne(event)

	device.WriteOne(&SYN_INPUT_EVENT)
}

// ----------------------------------------------------------------------

func queueSendKey(device *evdev.InputDevice, key evdev.EvCode, value int32, timeout int) {
	time.AfterFunc(time.Duration(timeout)*time.Millisecond, func() {
		sendKey(device, key, value)
	})
}

// ----------------------------------------------------------------------

func queueEvent(eventChannel chan *evdev.InputEvent, timeout int, event *evdev.InputEvent) {

	time.AfterFunc(time.Duration(timeout)*time.Millisecond, func() {
		eventChannel <- &SCAN_INPUT_EVENT
		eventChannel <- event
		eventChannel <- &SYN_INPUT_EVENT
	})
}

// ----------------------------------------------------------------------

const MODIFIER_TIMEOUT = 20

// ----------------------------------------------------------------------

func processMouseEvent(event *evdev.InputEvent, appCtx *AppContext) {

	transformed := false

	if appCtx.KeyState.isPressed(evdev.KEY_CAPSLOCK) {
		if event.Type == evdev.EV_KEY {
			switch event.Code {
			case evdev.BTN_LEFT:
				transformed = true
				event.Code = evdev.BTN_MIDDLE

				if event.Value == 1 {
					// down
					sendKey(appCtx.KeyboardOutput, evdev.KEY_LEFTSHIFT, 1)
					queueEvent(appCtx.MouseEvents, MODIFIER_TIMEOUT, event)

				} else {
					appCtx.MouseOutput.WriteOne(event)
					queueSendKey(appCtx.KeyboardOutput, evdev.KEY_LEFTSHIFT, 0, MODIFIER_TIMEOUT)
				}

			case evdev.BTN_RIGHT:
				event.Code = evdev.BTN_MIDDLE
				transformed = true

				if event.Value == 1 {
					// down
					sendKey(appCtx.KeyboardOutput, evdev.KEY_LEFTCTRL, 1)
					queueEvent(appCtx.MouseEvents, MODIFIER_TIMEOUT, event)

				} else {
					appCtx.MouseOutput.WriteOne(event)
					queueSendKey(appCtx.KeyboardOutput, evdev.KEY_LEFTCTRL, 0, MODIFIER_TIMEOUT)
				}
			}
		}
	}

	if !transformed {
		// send the mouse event immediately
		appCtx.MouseOutput.WriteOne(event)
	}
}

// ----------------------------------------------------------------------

func main() {

	appCtx := AppContext{}
	defer appCtx.Dispose()

	err := appCtx.initialize()

	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("Starting listeners\n\n")

	go startListener(&appCtx, appCtx.Keyboard, appCtx.KeyboardEvents)
	go startListener(&appCtx, appCtx.Mouse, appCtx.MouseEvents)

	for {
		select {
		case event := <-appCtx.KeyboardEvents:
			// fmt.Println("keyboard event:", event)
			appCtx.KeyState.update(event)

			if appCtx.KeyState.isPressed(abortCombo[:]...) {
				fmt.Println("aborting")
				return
			}

			// consume capslock event
			if event.Code == evdev.KEY_CAPSLOCK {
				continue
			}

			appCtx.KeyboardOutput.WriteOne(event)

		case event := <-appCtx.MouseEvents:
			// fmt.Println("mouse event:", event)
			// appCtx.KeyState.update(event) // not needed?

			processMouseEvent(event, &appCtx)

		case err := <-appCtx.Errors:
			fmt.Println("error:", err)
			appCtx.Cancel()
			return
		case <-appCtx.Context.Done():
			return
		}
	}

}
