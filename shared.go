package main

import (
	"fmt"

	"github.com/holoplot/go-evdev"
)

// ----------------------------------------------------------------------

func whileNoError(funcs ...func() error) error {

	for _, f := range funcs {
		err := f()

		if err != nil {
			return err
		}
	}

	return nil
}

// ----------------------------------------------------------------------

func selectDevice(header string) (string, error) {

	fmt.Println(header)
	fmt.Println()

	paths, err := evdev.ListDevicePaths()

	if err != nil {
		return "", err
	}

	if len(paths) == 0 {
		return "", fmt.Errorf("no input devices found")
	}

	for i, path := range paths {
		fmt.Printf("%d: %s: %s\n", i, path.Name, path.Path)
	}

	fmt.Printf("Select a device: [0-%d]: ", len(paths)-1)

	var i int
	fmt.Scanf("%d", &i)

	if i < 0 || i >= len(paths) {
		return "", fmt.Errorf("invalid selection")
	}

	return paths[i].Path, nil
}

// ----------------------------------------------------------------------
