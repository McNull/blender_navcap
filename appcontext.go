package main

import (
	"context"
	"fmt"
	"os"
	"strings"
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

	KeyboardOutputPath string
	MouseOutputPath    string

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

func nameToPath(name string) (string, error) {
	devicePaths, err := evdev.ListDevicePaths()

	if err != nil {
		return "", err
	}

	for _, path := range devicePaths {
		if path.Name == name {
			return path.Path, nil
		}
	}

	return "", fmt.Errorf("device not found: %s", name)
}

// ----------------------------------------------------------------------

func nameOrPath(nameOrPath string) (string, error) {
	if strings.HasPrefix(nameOrPath, "/dev/input/") && fileExists(nameOrPath) {
		return nameOrPath, nil
	} else {
		return nameToPath(nameOrPath)
	}
}

// ----------------------------------------------------------------------

func (appCtx *AppContext) selectPathsFromEnv() error {

	appCtx.KeyboardPath = os.Getenv("BLENDER_NAVCAP_KEYBOARD")
	appCtx.MousePath = os.Getenv("BLENDER_NAVCAP_MOUSE")

	return nil
}

// ----------------------------------------------------------------------

func (appCtx *AppContext) selectPathFromArgs() error {

	for _, arg := range appCtx.Args {
		if strings.HasPrefix(arg, "--keyboard=") {
			appCtx.KeyboardPath = strings.TrimPrefix(arg, "--keyboard=")
		} else if strings.HasPrefix(arg, "--mouse=") {
			appCtx.MousePath = strings.TrimPrefix(arg, "--mouse=")
		}
	}

	return nil
}

// ----------------------------------------------------------------------

func (appCtx *AppContext) selectPaths() error {

	err := appCtx.selectPathsFromEnv()

	if err != nil {
		return err
	}

	err = appCtx.selectPathFromArgs()

	if err != nil {
		return err
	}

	for _, arg := range appCtx.Args {
		if arg == "-i" || arg == "--interactive" {
			return appCtx.selectPathsInteractive()
		}
	}

	if appCtx.KeyboardPath == "" {
		return fmt.Errorf("no keyboard device specified")
	} else {
		if appCtx.KeyboardPath, err = nameOrPath(appCtx.KeyboardPath); err != nil {
			return err
		}
	}

	if appCtx.MousePath == "" {
		return fmt.Errorf("no mouse device specified")
	} else {
		if appCtx.MousePath, err = nameOrPath(appCtx.MousePath); err != nil {
			return err
		}
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

	return nil
}

// ----------------------------------------------------------------------

func (appCtx *AppContext) cloneDevices() error {

	var err error

	cloneDevice := func(cloneName string, device *evdev.InputDevice) (*evdev.InputDevice, string, error) {

		if name, err := device.Name(); err != nil {
			return nil, "", err
		} else {
			fmt.Printf("Cloning %s (%s)\n", name, device.Path())

			if clone, err := evdev.CloneDevice(cloneName, device); err != nil {
				return nil, "", err
			} else {
				path, _ := nameToPath(cloneName)
				fmt.Printf("Cloned to %s (%s)\n", cloneName, path)

				return clone, path, nil
			}
		}
	}

	if appCtx.KeyboardOutput, appCtx.KeyboardOutputPath, err = cloneDevice("Blender NavCap Keyboard", appCtx.Keyboard); err != nil {
		return err
	}

	if appCtx.MouseOutput, appCtx.MouseOutputPath, err = cloneDevice("Blender NavCap Mouse", appCtx.Mouse); err != nil {
		return err
	}

	return nil
}

// ----------------------------------------------------------------------

func (appCtx *AppContext) cloneMouseDPI() error {

	fmt.Printf("Cloning mouse DPI rules ...\n")

	dpi, err := getDeviceDPI(appCtx.MousePath)

	if err != nil {
		return err
	}

	fmt.Printf("Current mouse DPI: %s\n", dpi)

	fmt.Printf("Setting mouse DPI ...\n")

	if err := setMouseDPI(appCtx.MouseOutputPath, dpi); err != nil {
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
		appCtx.cloneMouseDPI,
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
