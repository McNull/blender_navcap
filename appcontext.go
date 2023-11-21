package main

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/holoplot/go-evdev"
)

// ----------------------------------------------------------------------

type AppContext struct {
	Args []string

	KeyboardPath string
	MousePath    string

	Keyboard *evdev.InputDevice
	Mouse    *evdev.InputDevice

	KeyboardOutput *evdev.InputDevice
	MouseOutput    *evdev.InputDevice

	KeyboardEvents chan *evdev.InputEvent
	MouseEvents    chan *evdev.InputEvent
	Errors         chan error

	Cancel  context.CancelFunc
	Context context.Context

	KeyState *KeyState

	wg sync.WaitGroup
}

// ----------------------------------------------------------------------

func (appCtx *AppContext) Dispose() {

	fmt.Println("Disposing")

	if appCtx.Cancel != nil {
		appCtx.Cancel()
	}

	devices := []*evdev.InputDevice{
		appCtx.Keyboard,
		appCtx.Mouse,
		appCtx.KeyboardOutput,
		appCtx.MouseOutput,
	}

	for _, device := range devices {
		if device != nil {
			fmt.Printf("Ungrabbing %s\n", device.Path())
			device.Ungrab()

			fmt.Printf("Closing %s\n", device.Path())
			device.Close()
		}
	}

	appCtx.wg.Wait()
}

// ----------------------------------------------------------------------

func (appCtx *AppContext) selectPathsInteractive() error {

	var err error

	appCtx.KeyboardPath, err = selectDevice("Select a KEYBOARD device:")

	if err != nil {
		return err
	}

	fmt.Println()
	appCtx.MousePath, err = selectDevice("Select a MOUSE device:")

	if err != nil {
		return err
	}

	fmt.Println()

	// sleep 2 seconds to allow any keypresses to finish

	time.Sleep(2 * time.Second)

	return nil
}

// ----------------------------------------------------------------------

func (appCtx *AppContext) selectPaths() error {

	for _, arg := range appCtx.Args {
		if arg == "-i" || arg == "--interactive" {
			return appCtx.selectPathsInteractive()
		}
	}

	appCtx.KeyboardPath = os.Getenv("BLENDER_NAVCAP_KEYBOARD")

	if appCtx.KeyboardPath == "" {
		return fmt.Errorf("environment variable BLENDER_NAVCAP_KEYBOARD not set")
	}

	appCtx.MousePath = os.Getenv("BLENDER_NAVCAP_MOUSE")

	if appCtx.MousePath == "" {
		return fmt.Errorf("environment variable BLENDER_NAVCAP_MOUSE not set")
	}

	return nil
}

// ----------------------------------------------------------------------

func (appCtx *AppContext) openDevices() error {

	var err error

	fmt.Printf("Opening %s\n", appCtx.KeyboardPath)

	appCtx.Keyboard, err = evdev.Open(appCtx.KeyboardPath)

	if err != nil {
		return err
	}

	fmt.Printf("Opening %s\n", appCtx.MousePath)

	appCtx.Mouse, err = evdev.Open(appCtx.MousePath)

	if err != nil {
		return err
	}

	absInfos, err := appCtx.Mouse.AbsInfos()

	if err != nil {
		return err
	}

	fmt.Print("Mouse absInfos:\n")
	fmt.Printf("len(absInfos): %d\n", len(absInfos))

	for i, absInfo := range absInfos {
		// print absInfo.Flat
		fmt.Printf("%d: %d\n", i, absInfo.Flat)
		// print absInfo.Fuzz
		fmt.Printf("%d: %d\n", i, absInfo.Fuzz)
		// print absInfo.Maximum
		fmt.Printf("%d: %d\n", i, absInfo.Maximum)
		// print absInfo.Minimum
		fmt.Printf("%d: %d\n", i, absInfo.Minimum)
		// print absInfo.Resolution
		fmt.Printf("%d: %d\n", i, absInfo.Resolution)
		// print absInfo.Value
		fmt.Printf("%d: %d\n", i, absInfo.Value)
	}

	return nil
}

// ----------------------------------------------------------------------

func (appCtx *AppContext) cloneDevices() error {

	var err error

	fmt.Printf("Cloning %s\n", appCtx.KeyboardPath)

	appCtx.KeyboardOutput, err = evdev.CloneDevice("Blender NavCap keyboard", appCtx.Keyboard)

	if err != nil {
		return err
	}

	fmt.Printf("Cloning %s\n", appCtx.MousePath)

	appCtx.MouseOutput, err = evdev.CloneDevice("Blender NavCap mouse", appCtx.Mouse)

	if err != nil {
		return err
	}

	return nil
}

// ----------------------------------------------------------------------

func (appCtx *AppContext) grabDevices() error {

	fmt.Printf("Grabbing %s\n", appCtx.KeyboardPath)
	fmt.Printf("Grabbing %s\n", appCtx.MousePath)

	return whileNoError(
		appCtx.Keyboard.Grab,
		appCtx.Mouse.Grab,
	)
}

// ----------------------------------------------------------------------

func (appCtx *AppContext) initialize() error {

	appCtx.Args = os.Args

	appCtx.KeyState = newKeyState()

	err := whileNoError(
		appCtx.selectPaths,
		appCtx.openDevices,
		appCtx.cloneDevices,
		appCtx.grabDevices,
	)

	if err != nil {
		return err
	}

	appCtx.KeyboardEvents = make(chan *evdev.InputEvent)
	appCtx.MouseEvents = make(chan *evdev.InputEvent)
	appCtx.Errors = make(chan error)

	appCtx.Context, appCtx.Cancel = context.WithCancel(context.Background())
	appCtx.wg.Add(2)

	return nil
}

// ----------------------------------------------------------------------
