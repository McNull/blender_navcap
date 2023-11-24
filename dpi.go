package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func getDeviceDPI(device_path string) (string, error) {
	cmd := exec.Command("udevadm", "info", "--query=property", "--name="+device_path)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", err
	}

	lines := strings.Split(out.String(), "\n")
	for _, line := range lines {
		if strings.Contains(line, "MOUSE_DPI=") {
			return strings.Split(line, "=")[1], nil
		}
	}

	return "", fmt.Errorf("MOUSE_DPI not found")
}

func setMouseDPI(device_path string, mouse_dpi string) error {
	file, err := os.OpenFile("/etc/udev/rules.d/90-blender-navcap-dpi.rules", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(fmt.Sprintf(`ACTION=="add|change", ENV{DEVNAME}=="%s", ENV{MOUSE_DPI}="%s"`, device_path, mouse_dpi))
	if err != nil {
		return err
	}

	err = exec.Command("udevadm", "control", "--reload-rules").Run()
	if err != nil {
		return err
	}

	err = exec.Command("udevadm", "trigger").Run()
	if err != nil {
		return err
	}

	return nil
}
